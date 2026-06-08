// Package jira is a thin client for the Jira Server/Data Center REST API
// (base path /rest/api/2). It handles auth (Personal Access Token or Basic),
// retries on transient failures, and structured error reporting.
package jira

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"jira-cli/internal/config"
)

// Client talks to a single Jira instance.
type Client struct {
	baseURL    string
	httpClient *http.Client
	authHeader string
	userAgent  string

	MaxRetries int
	RetryWait  time.Duration

	lastRetryAfter time.Duration
}

// APIError is a non-2xx response from Jira.
type APIError struct {
	StatusCode int
	Message    string
	Body       string
}

func (e *APIError) Error() string {
	if e.Message != "" {
		return fmt.Sprintf("jira API %d: %s", e.StatusCode, e.Message)
	}
	return fmt.Sprintf("jira API %d", e.StatusCode)
}

// New builds a Client from config.
func New(cfg config.Config, timeout time.Duration) (*Client, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	auth, err := authHeader(cfg)
	if err != nil {
		return nil, err
	}

	transport := http.DefaultTransport.(*http.Transport).Clone()
	if cfg.Insecure {
		transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	}

	return &Client{
		baseURL:    cfg.BaseURL,
		httpClient: &http.Client{Timeout: timeout, Transport: transport},
		authHeader: auth,
		userAgent:  "jira-cli",
		MaxRetries: 3,
		RetryWait:  500 * time.Millisecond,
	}, nil
}

func authHeader(cfg config.Config) (string, error) {
	if cfg.Token != "" {
		return "Bearer " + cfg.Token, nil
	}
	if cfg.User != "" {
		raw := cfg.User + ":" + cfg.Password
		return "Basic " + base64.StdEncoding.EncodeToString([]byte(raw)), nil
	}
	return "", errors.New("no credentials configured")
}

// doJSON performs an API request, decoding a JSON response into out (may be nil).
func (c *Client) doJSON(ctx context.Context, method, path string, query url.Values, body, out any) error {
	var payload []byte
	if body != nil {
		var err error
		payload, err = json.Marshal(body)
		if err != nil {
			return fmt.Errorf("encoding request body: %w", err)
		}
	}

	endpoint := c.baseURL + path
	if len(query) > 0 {
		endpoint += "?" + query.Encode()
	}

	var lastErr error
	for attempt := 0; attempt <= c.MaxRetries; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(c.backoff(attempt)):
			}
		}

		var reader io.Reader
		if payload != nil {
			reader = bytes.NewReader(payload)
		}
		req, err := http.NewRequestWithContext(ctx, method, endpoint, reader)
		if err != nil {
			return fmt.Errorf("building request: %w", err)
		}
		req.Header.Set("Authorization", c.authHeader)
		req.Header.Set("Accept", "application/json")
		req.Header.Set("User-Agent", c.userAgent)
		if payload != nil {
			req.Header.Set("Content-Type", "application/json")
		}

		resp, err := c.httpClient.Do(req)
		if err != nil {
			lastErr = err
			if attempt < c.MaxRetries {
				continue
			}
			return fmt.Errorf("request failed: %w", err)
		}

		data, readErr := io.ReadAll(resp.Body)
		resp.Body.Close()
		if readErr != nil {
			lastErr = readErr
			continue
		}

		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			if out == nil || len(data) == 0 {
				return nil
			}
			if err := json.Unmarshal(data, out); err != nil {
				return fmt.Errorf("decoding response: %w", err)
			}
			return nil
		}

		apiErr := parseAPIError(resp.StatusCode, data)
		if isRetryable(resp.StatusCode) && attempt < c.MaxRetries {
			lastErr = apiErr
			c.lastRetryAfter = retryAfter(resp.Header)
			continue
		}
		return apiErr
	}

	if lastErr != nil {
		return lastErr
	}
	return errors.New("request failed after retries")
}

func (c *Client) backoff(attempt int) time.Duration {
	if c.lastRetryAfter > 0 {
		d := c.lastRetryAfter
		c.lastRetryAfter = 0
		return d
	}
	return c.RetryWait * time.Duration(1<<(attempt-1))
}

func isRetryable(status int) bool {
	switch status {
	case http.StatusTooManyRequests,
		http.StatusBadGateway,
		http.StatusServiceUnavailable,
		http.StatusGatewayTimeout:
		return true
	}
	return false
}

func retryAfter(h http.Header) time.Duration {
	v := h.Get("Retry-After")
	if v == "" {
		return 0
	}
	if secs, err := strconv.Atoi(strings.TrimSpace(v)); err == nil {
		return time.Duration(secs) * time.Second
	}
	return 0
}

// parseAPIError handles Jira's error shape:
//
//	{"errorMessages":["..."],"errors":{"field":"msg"}}
func parseAPIError(status int, data []byte) *APIError {
	e := &APIError{StatusCode: status, Body: string(data)}
	var parsed struct {
		ErrorMessages []string          `json:"errorMessages"`
		Errors        map[string]string `json:"errors"`
		Message       string            `json:"message"`
	}
	if json.Unmarshal(data, &parsed) == nil {
		var parts []string
		parts = append(parts, parsed.ErrorMessages...)
		for k, v := range parsed.Errors {
			parts = append(parts, k+": "+v)
		}
		if parsed.Message != "" {
			parts = append(parts, parsed.Message)
		}
		if len(parts) > 0 {
			e.Message = strings.Join(parts, "; ")
			return e
		}
	}
	if len(data) > 0 && len(data) < 300 {
		e.Message = strings.TrimSpace(string(data))
	}
	return e
}
