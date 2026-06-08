// Package config loads jira-cli settings from a config file and environment
// variables. Command-line flags layer on top of the result.
//
// Precedence (low -> high): config file < environment variables < flags.
package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// Config holds the connection and auth settings for a Jira Server/Data Center
// instance.
type Config struct {
	// BaseURL is the site base, e.g. https://jira.example.com
	BaseURL string `json:"base_url"`
	// Token is a Personal Access Token (preferred, Jira 8.14+).
	Token string `json:"token"`
	// User / Password are used for Basic auth when Token is empty.
	User     string `json:"user"`
	Password string `json:"password"`
	// Project is the default project key used by commands that take --project
	// (search, create) when the flag is not given.
	Project string `json:"project"`
	// Insecure skips TLS certificate verification (internal CA only).
	Insecure bool `json:"insecure"`
}

// Env var names.
const (
	EnvBaseURL  = "JIRA_BASE_URL"
	EnvToken    = "JIRA_TOKEN"
	EnvUser     = "JIRA_USER"
	EnvPassword = "JIRA_PASSWORD"
	EnvProject  = "JIRA_PROJECT"
	EnvInsecure = "JIRA_INSECURE"
	EnvConfig   = "JIRA_CONFIG"
)

// Load reads the config file (if any) and overlays environment variables.
// A missing config file is not an error; a malformed one is.
func Load() (Config, error) {
	var c Config

	path, err := configFilePath()
	if err != nil {
		return c, err
	}
	if path != "" {
		if err := loadFile(path, &c); err != nil {
			return c, err
		}
	}

	if v := os.Getenv(EnvBaseURL); v != "" {
		c.BaseURL = v
	}
	if v := os.Getenv(EnvToken); v != "" {
		c.Token = v
	}
	if v := os.Getenv(EnvUser); v != "" {
		c.User = v
	}
	if v := os.Getenv(EnvPassword); v != "" {
		c.Password = v
	}
	if v := os.Getenv(EnvProject); v != "" {
		c.Project = v
	}
	if v := os.Getenv(EnvInsecure); v != "" {
		c.Insecure = truthy(v)
	}

	c.BaseURL = strings.TrimRight(strings.TrimSpace(c.BaseURL), "/")
	return c, nil
}

// Validate ensures the config is usable for making requests.
func (c Config) Validate() error {
	if c.BaseURL == "" {
		return errors.New("base URL is not set (use --base-url or " + EnvBaseURL + ")")
	}
	if !strings.HasPrefix(c.BaseURL, "http://") && !strings.HasPrefix(c.BaseURL, "https://") {
		return fmt.Errorf("base URL must start with http:// or https:// (got %q)", c.BaseURL)
	}
	if c.Token == "" && c.User == "" {
		return errors.New("no credentials: set a Personal Access Token (--token / " +
			EnvToken + ") or Basic auth (--user + --password)")
	}
	return nil
}

func configFilePath() (string, error) {
	if p := os.Getenv(EnvConfig); p != "" {
		return p, nil
	}
	dir, err := defaultConfigDir()
	if err != nil {
		return "", err
	}
	p := filepath.Join(dir, "jira-cli", "config.json")
	if _, err := os.Stat(p); err != nil {
		return "", nil
	}
	return p, nil
}

func defaultConfigDir() (string, error) {
	dir, err := os.UserConfigDir()
	if err == nil {
		return dir, nil
	}
	home, herr := os.UserHomeDir()
	if herr != nil {
		return "", err
	}
	if runtime.GOOS == "windows" {
		return home, nil
	}
	return filepath.Join(home, ".config"), nil
}

func loadFile(path string, c *Config) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("reading config %s: %w", path, err)
	}
	if err := json.Unmarshal(data, c); err != nil {
		return fmt.Errorf("parsing config %s: %w", path, err)
	}
	return nil
}

func truthy(v string) bool {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}
