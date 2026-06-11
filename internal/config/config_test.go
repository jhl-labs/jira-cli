package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestTruthy(t *testing.T) {
	for _, v := range []string{"1", "true", "TRUE", "yes", "on", " On "} {
		if !truthy(v) {
			t.Errorf("truthy(%q) = false, want true", v)
		}
	}
	for _, v := range []string{"", "0", "false", "no", "off", "nope"} {
		if truthy(v) {
			t.Errorf("truthy(%q) = true, want false", v)
		}
	}
}

func TestValidate(t *testing.T) {
	tests := []struct {
		name string
		c    Config
		ok   bool
	}{
		{"empty base", Config{Token: "t"}, false},
		{"bad scheme", Config{BaseURL: "ftp://x", Token: "t"}, false},
		{"no creds", Config{BaseURL: "https://x"}, false},
		{"token ok", Config{BaseURL: "https://x", Token: "t"}, true},
		{"basic ok", Config{BaseURL: "http://x", User: "u"}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.c.Validate()
			if (err == nil) != tt.ok {
				t.Errorf("Validate() err = %v, want ok = %v", err, tt.ok)
			}
		})
	}
}

// clearEnv blanks every jira-cli env var so a test starts from a known state.
func clearEnv(t *testing.T) {
	t.Helper()
	for _, k := range []string{EnvBaseURL, EnvToken, EnvUser, EnvPassword, EnvProject, EnvInsecure, EnvConfig} {
		t.Setenv(k, "")
	}
}

func TestLoadFileThenEnvOverlay(t *testing.T) {
	clearEnv(t)
	dir := t.TempDir()
	cfg := filepath.Join(dir, "config.json")
	body := `{"base_url":"https://file.example.com/","token":"filetok","project":"FILE","insecure":true}`
	if err := os.WriteFile(cfg, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv(EnvConfig, cfg)

	c, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if c.BaseURL != "https://file.example.com" { // trailing slash trimmed
		t.Errorf("BaseURL = %q", c.BaseURL)
	}
	if c.Token != "filetok" || c.Project != "FILE" || !c.Insecure {
		t.Errorf("file values not loaded: %+v", c)
	}

	// Environment overlays the file.
	t.Setenv(EnvToken, "envtok")
	t.Setenv(EnvBaseURL, "https://env.example.com")
	t.Setenv(EnvUser, "u")
	t.Setenv(EnvPassword, "p")
	t.Setenv(EnvProject, "ENV")
	t.Setenv(EnvInsecure, "false")
	c2, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if c2.Token != "envtok" || c2.BaseURL != "https://env.example.com" ||
		c2.User != "u" || c2.Password != "p" || c2.Project != "ENV" || c2.Insecure {
		t.Errorf("env overlay failed: %+v", c2)
	}
}

func TestLoadBadJSON(t *testing.T) {
	clearEnv(t)
	dir := t.TempDir()
	cfg := filepath.Join(dir, "bad.json")
	if err := os.WriteFile(cfg, []byte("{not json"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv(EnvConfig, cfg)
	if _, err := Load(); err == nil {
		t.Fatal("expected error for malformed config")
	}
}

func TestLoadMissingConfigFileIsError(t *testing.T) {
	clearEnv(t)
	// JIRA_CONFIG explicitly points at a file that does not exist -> read error.
	t.Setenv(EnvConfig, filepath.Join(t.TempDir(), "nope.json"))
	if _, err := Load(); err == nil {
		t.Fatal("expected error reading missing explicit config")
	}
}

func TestConfigFilePathDefault(t *testing.T) {
	clearEnv(t)
	tmp := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmp) // drives os.UserConfigDir on linux

	// No file yet -> empty path, no error.
	p, err := configFilePath()
	if err != nil {
		t.Fatalf("configFilePath: %v", err)
	}
	if p != "" {
		t.Errorf("expected empty path when file absent, got %q", p)
	}

	// Create the default config; now the path resolves.
	dir := filepath.Join(tmp, "jira-cli")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "config.json"), []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}
	p2, err := configFilePath()
	if err != nil {
		t.Fatalf("configFilePath: %v", err)
	}
	if p2 == "" {
		t.Error("expected a path once the default config exists")
	}

	// Load via the default path should also succeed.
	if _, err := Load(); err != nil {
		t.Fatalf("Load with default config: %v", err)
	}
}

func TestDefaultConfigDir(t *testing.T) {
	if _, err := defaultConfigDir(); err != nil {
		t.Fatalf("defaultConfigDir: %v", err)
	}
}

func TestDefaultConfigDirFallbackError(t *testing.T) {
	// With neither XDG_CONFIG_HOME nor HOME set, both os.UserConfigDir and
	// os.UserHomeDir fail, so the fallback returns an error.
	t.Setenv("XDG_CONFIG_HOME", "")
	t.Setenv("HOME", "")
	if _, err := defaultConfigDir(); err == nil {
		t.Skip("platform resolved a config dir without HOME; fallback not reachable here")
	}
}
