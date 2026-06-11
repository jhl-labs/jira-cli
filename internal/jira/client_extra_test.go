package jira

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"jira-cli/internal/config"
)

func TestWebURL(t *testing.T) {
	c := &Client{baseURL: "https://jira.example.com"}
	if got := c.WebURL("PROJ-1"); got != "https://jira.example.com/browse/PROJ-1" {
		t.Errorf("WebURL = %q", got)
	}
	if got := c.WebURL(""); got != "" {
		t.Errorf("WebURL(empty) = %q, want empty", got)
	}
}

func TestAPIErrorError(t *testing.T) {
	if got := (&APIError{StatusCode: 400, Message: "bad"}).Error(); got != "jira API 400: bad" {
		t.Errorf("Error = %q", got)
	}
	if got := (&APIError{StatusCode: 500}).Error(); got != "jira API 500" {
		t.Errorf("Error = %q", got)
	}
}

func TestIsRetryable(t *testing.T) {
	for _, s := range []int{429, 502, 503, 504} {
		if !isRetryable(s) {
			t.Errorf("isRetryable(%d) = false, want true", s)
		}
	}
	for _, s := range []int{200, 400, 404, 500, 501} {
		if isRetryable(s) {
			t.Errorf("isRetryable(%d) = true, want false", s)
		}
	}
}

func TestRetryAfter(t *testing.T) {
	h := http.Header{}
	h.Set("Retry-After", " 3 ")
	if got := retryAfter(h); got != 3*time.Second {
		t.Errorf("retryAfter = %v, want 3s", got)
	}
	if got := retryAfter(http.Header{}); got != 0 {
		t.Errorf("retryAfter(empty) = %v, want 0", got)
	}
	bad := http.Header{}
	bad.Set("Retry-After", "soon")
	if got := retryAfter(bad); got != 0 {
		t.Errorf("retryAfter(non-numeric) = %v, want 0", got)
	}
}

func TestBackoff(t *testing.T) {
	c := &Client{RetryWait: 5 * time.Millisecond, lastRetryAfter: 30 * time.Millisecond}
	if got := c.backoff(1); got != 30*time.Millisecond {
		t.Errorf("backoff with Retry-After = %v, want 30ms", got)
	}
	if c.lastRetryAfter != 0 {
		t.Error("lastRetryAfter should reset after use")
	}
	if got := c.backoff(1); got != 5*time.Millisecond {
		t.Errorf("backoff(1) = %v, want 5ms", got)
	}
	if got := c.backoff(3); got != 20*time.Millisecond {
		t.Errorf("backoff(3) = %v, want 20ms", got)
	}
}

func TestAuthHeader(t *testing.T) {
	got, err := authHeader(config.Config{User: "bot", Password: "pw"})
	if err != nil {
		t.Fatal(err)
	}
	want := "Basic " + base64.StdEncoding.EncodeToString([]byte("bot:pw"))
	if got != want {
		t.Errorf("authHeader basic = %q, want %q", got, want)
	}
	if _, err := authHeader(config.Config{}); err == nil {
		t.Error("expected error with no credentials")
	}
}

func TestParseAPIError(t *testing.T) {
	if e := parseAPIError(400, []byte(`{"errors":{"summary":"required"}}`)); e.Message != "summary: required" {
		t.Errorf("errors map: %q", e.Message)
	}
	if e := parseAPIError(400, []byte(`{"message":"boom"}`)); e.Message != "boom" {
		t.Errorf("message: %q", e.Message)
	}
	if e := parseAPIError(404, []byte(`plain text body`)); e.Message != "plain text body" {
		t.Errorf("raw fallback: %q", e.Message)
	}
	if e := parseAPIError(500, []byte(``)); e.Message != "" {
		t.Errorf("empty body should yield empty message, got %q", e.Message)
	}
}

// lastBody records the most recent decoded request body and method/path.
type recorder struct {
	method, path string
	body         map[string]any
}

func recordingServer(t *testing.T, status int, resp any, rec *recorder) *Client {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rec.method = r.Method
		rec.path = r.URL.Path
		_ = json.NewDecoder(r.Body).Decode(&rec.body)
		if status != 0 {
			w.WriteHeader(status)
		}
		if resp != nil {
			_ = json.NewEncoder(w).Encode(resp)
		}
	}))
	t.Cleanup(srv.Close)
	return testClient(t, srv)
}

func TestUpdateFieldsAndOps(t *testing.T) {
	var rec recorder
	cl := recordingServer(t, http.StatusNoContent, nil, &rec)
	if err := cl.UpdateFields(context.Background(), "PROJ-1", map[string]any{"summary": "x"}); err != nil {
		t.Fatalf("UpdateFields: %v", err)
	}
	if rec.method != "PUT" || rec.body["fields"] == nil {
		t.Errorf("UpdateFields sent %s body=%v", rec.method, rec.body)
	}
	if err := cl.UpdateOps(context.Background(), "PROJ-1", map[string]any{"labels": []any{}}); err != nil {
		t.Fatalf("UpdateOps: %v", err)
	}
	if rec.body["update"] == nil {
		t.Errorf("UpdateOps body = %v", rec.body)
	}
}

func TestAddComment(t *testing.T) {
	var rec recorder
	cl := recordingServer(t, 0, Comment{ID: "55", Body: "hi"}, &rec)
	c, err := cl.AddComment(context.Background(), "PROJ-1", "hi")
	if err != nil {
		t.Fatalf("AddComment: %v", err)
	}
	if c.ID != "55" || rec.method != "POST" || rec.body["body"] != "hi" {
		t.Errorf("AddComment c=%+v rec=%+v", c, rec)
	}
}

func TestAssignIssue(t *testing.T) {
	var rec recorder
	cl := recordingServer(t, http.StatusNoContent, nil, &rec)
	if err := cl.AssignIssue(context.Background(), "PROJ-1", "alice"); err != nil {
		t.Fatalf("AssignIssue: %v", err)
	}
	if rec.method != "PUT" || rec.body["name"] != "alice" {
		t.Errorf("assign body = %v", rec.body)
	}
	// Unassign sends a null name.
	if err := cl.AssignIssue(context.Background(), "PROJ-1", ""); err != nil {
		t.Fatalf("unassign: %v", err)
	}
	if v, ok := rec.body["name"]; !ok || v != nil {
		t.Errorf("unassign name = %v (ok=%v), want nil", v, ok)
	}
}

func TestDeleteIssue(t *testing.T) {
	var rec recorder
	cl := recordingServer(t, http.StatusNoContent, nil, &rec)
	if err := cl.DeleteIssue(context.Background(), "PROJ-1"); err != nil {
		t.Fatalf("DeleteIssue: %v", err)
	}
	if rec.method != "DELETE" {
		t.Errorf("method = %s, want DELETE", rec.method)
	}
}

func TestListProjects(t *testing.T) {
	var rec recorder
	cl := recordingServer(t, 0, []Project{{Key: "TODO", Name: "Todo"}}, &rec)
	ps, err := cl.ListProjects(context.Background())
	if err != nil {
		t.Fatalf("ListProjects: %v", err)
	}
	if len(ps) != 1 || ps[0].Key != "TODO" {
		t.Errorf("projects = %+v", ps)
	}
}

func TestDoJSONRetryThenSuccess(t *testing.T) {
	var calls int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if calls == 1 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		_ = json.NewEncoder(w).Encode(Issue{Key: "PROJ-1"})
	}))
	defer srv.Close()
	cl := testClient(t, srv) // RetryWait = 1ms
	issue, err := cl.GetIssue(context.Background(), "PROJ-1", nil, nil)
	if err != nil {
		t.Fatalf("GetIssue after retry: %v", err)
	}
	if calls != 2 || issue.Key != "PROJ-1" {
		t.Errorf("calls = %d, issue = %+v", calls, issue)
	}
}

func TestNewInsecureAndInvalid(t *testing.T) {
	cl, err := New(config.Config{BaseURL: "https://jira.example.com", Token: "t", Insecure: true}, time.Second)
	if err != nil {
		t.Fatalf("New insecure: %v", err)
	}
	tr := cl.httpClient.Transport.(*http.Transport)
	if tr.TLSClientConfig == nil || !tr.TLSClientConfig.InsecureSkipVerify {
		t.Error("insecure transport not configured")
	}
	if _, err := New(config.Config{}, time.Second); err == nil {
		t.Error("expected error for invalid config")
	}
}

func TestCreateAllFields(t *testing.T) {
	var rec recorder
	cl := recordingServer(t, 0, Issue{Key: "PROJ-9"}, &rec)
	_, err := cl.Create(context.Background(), CreateInput{
		ProjectKey:  "PROJ",
		Summary:     "S",
		Description: "D",
		IssueType:   "Bug",
		Assignee:    "alice",
		Priority:    "High",
		Labels:      []string{"x", "y"},
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	fields, _ := rec.body["fields"].(map[string]any)
	for _, k := range []string{"description", "assignee", "priority", "labels"} {
		if fields[k] == nil {
			t.Errorf("field %q not sent: %v", k, fields)
		}
	}
}

func TestSearchAndGetQueryParams(t *testing.T) {
	var raw string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw = r.URL.RawQuery
		_ = json.NewEncoder(w).Encode(SearchResult{})
	}))
	defer srv.Close()
	cl := testClient(t, srv)
	if _, err := cl.Search(context.Background(), "x", []string{"summary"}, 5, 10); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(raw, "startAt=10") || !strings.Contains(raw, "maxResults=5") {
		t.Errorf("search query = %q", raw)
	}
	if _, err := cl.GetIssue(context.Background(), "PROJ-1", []string{"summary"}, []string{"changelog"}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(raw, "expand=changelog") {
		t.Errorf("get query = %q", raw)
	}
}

func TestDoJSONCanceledContext(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer srv.Close()
	cl := testClient(t, srv)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := cl.GetIssue(ctx, "PROJ-1", nil, nil); err == nil {
		t.Fatal("expected error from canceled context")
	}
}

func TestDoJSONNetworkError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	url := srv.URL
	srv.Close() // nothing is listening now
	cl, err := New(config.Config{BaseURL: url, Token: "t"}, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	cl.MaxRetries = 1
	cl.RetryWait = time.Millisecond
	if _, err := cl.GetIssue(context.Background(), "PROJ-1", nil, nil); err == nil {
		t.Fatal("expected network error")
	}
}

func TestDoJSONExhaustsRetries(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
	}))
	defer srv.Close()
	cl := testClient(t, srv)
	cl.MaxRetries = 2
	_, err := cl.GetIssue(context.Background(), "PROJ-1", nil, nil)
	if err == nil {
		t.Fatal("expected error after exhausting retries")
	}
}
