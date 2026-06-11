package main

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"jira-cli/internal/jira"
)

// testServer returns an httptest server that answers every Jira endpoint the
// CLI touches with canned, valid responses.
func testServer(t *testing.T) *httptest.Server {
	t.Helper()
	issue := jira.Issue{
		ID:  "10000",
		Key: "TODO-1",
		Fields: &jira.Fields{
			Summary:     "First task",
			Description: "DESCRIPTION-BODY",
			Status:      &jira.NamedRef{Name: "To Do"},
			Assignee:    &jira.User{Name: "alice"},
			IssueType:   &jira.NamedRef{Name: "Task"},
			Priority:    &jira.NamedRef{Name: "High"},
			Labels:      []string{"a", "b"},
		},
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/rest/api/2/search", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(jira.SearchResult{Issues: []jira.Issue{issue}, Total: 1})
	})
	mux.HandleFunc("/rest/api/2/project", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode([]jira.Project{{Key: "TODO", Name: "Todo", ProjectTypeKey: "business"}})
	})
	mux.HandleFunc("/rest/api/2/issue", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(jira.Issue{ID: "10100", Key: "TODO-9"})
	})
	mux.HandleFunc("/rest/api/2/issue/", func(w http.ResponseWriter, r *http.Request) {
		p := strings.TrimPrefix(r.URL.Path, "/rest/api/2/issue/")
		switch {
		case strings.HasSuffix(p, "/comment"):
			_ = json.NewEncoder(w).Encode(jira.Comment{ID: "55"})
		case strings.HasSuffix(p, "/assignee"):
			w.WriteHeader(http.StatusNoContent)
		case strings.HasSuffix(p, "/transitions"):
			if r.Method == http.MethodGet {
				_ = json.NewEncoder(w).Encode(map[string]any{"transitions": []jira.Transition{
					{ID: "21", Name: "Done", To: &jira.NamedRef{Name: "Done"}},
				}})
			} else {
				w.WriteHeader(http.StatusNoContent)
			}
		default:
			switch r.Method {
			case http.MethodDelete, http.MethodPut:
				w.WriteHeader(http.StatusNoContent)
			default:
				_ = json.NewEncoder(w).Encode(issue)
			}
		}
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv
}

// setupEnv points the CLI at the test server with a hermetic (empty) config.
func setupEnv(t *testing.T, baseURL string) {
	t.Helper()
	cfg := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(cfg, []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("JIRA_CONFIG", cfg)
	t.Setenv("JIRA_BASE_URL", baseURL)
	t.Setenv("JIRA_TOKEN", "secret")
	t.Setenv("JIRA_PROJECT", "TODO")
	t.Setenv("JIRA_USER", "")
	t.Setenv("JIRA_PASSWORD", "")
	t.Setenv("JIRA_INSECURE", "")
}

// capture swaps os.Stdout for the duration of fn and returns what was written.
func capture(t *testing.T, fn func()) string {
	t.Helper()
	old := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = w
	done := make(chan string)
	go func() {
		var buf bytes.Buffer
		_, _ = io.Copy(&buf, r)
		done <- buf.String()
	}()
	fn()
	_ = w.Close()
	os.Stdout = old
	return <-done
}

// withStdin feeds input as os.Stdin while fn runs.
func withStdin(t *testing.T, input string, fn func()) {
	t.Helper()
	old := os.Stdin
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	go func() { _, _ = io.WriteString(w, input); _ = w.Close() }()
	os.Stdin = r
	defer func() { os.Stdin = old }()
	fn()
}

func TestRunSearch(t *testing.T) {
	srv := testServer(t)
	setupEnv(t, srv.URL)

	out := capture(t, func() {
		if code := run([]string{"search", "--project", "TODO", "--output", "text"}); code != 0 {
			t.Errorf("search text code = %d", code)
		}
	})
	if !strings.Contains(out, "TODO-1") {
		t.Errorf("search text missing issue: %q", out)
	}

	_ = capture(t, func() {
		if code := run([]string{"search", "--jql", "project = TODO"}); code != 0 {
			t.Errorf("search jql code = %d", code)
		}
	})

	// No query at all -> usage error.
	t.Setenv("JIRA_PROJECT", "")
	if code := run([]string{"search"}); code != 1 {
		t.Errorf("search with no query code = %d, want 1", code)
	}
}

func TestRunGet(t *testing.T) {
	srv := testServer(t)
	setupEnv(t, srv.URL)

	out := capture(t, func() {
		if code := run([]string{"get", "--key", "TODO-1", "--output", "text"}); code != 0 {
			t.Errorf("get code = %d", code)
		}
	})
	if !strings.Contains(out, "To Do") || !strings.Contains(out, "Task") {
		t.Errorf("get text output = %q", out)
	}

	desc := capture(t, func() {
		run([]string{"get", "--key", "TODO-1", "--description", "--output", "text"})
	})
	if !strings.Contains(desc, "DESCRIPTION-BODY") {
		t.Errorf("get --description = %q", desc)
	}

	if code := run([]string{"get"}); code != 1 {
		t.Errorf("get without key code = %d, want 1", code)
	}
}

func TestRunCreate(t *testing.T) {
	srv := testServer(t)
	setupEnv(t, srv.URL)

	out := capture(t, func() {
		if code := run([]string{"create", "--project", "TODO", "--summary", "S",
			"--description", "D", "--type", "Task", "--priority", "High",
			"--assignee", "alice", "--labels", "x,y", "--output", "text"}); code != 0 {
			t.Errorf("create code = %d", code)
		}
	})
	if !strings.Contains(out, "TODO-9") {
		t.Errorf("create output = %q", out)
	}

	// Description from stdin.
	withStdin(t, "from stdin", func() {
		_ = capture(t, func() {
			if code := run([]string{"create", "--summary", "S", "--description-file", "-"}); code != 0 {
				t.Errorf("create stdin code = %d", code)
			}
		})
	})

	if code := run([]string{"create"}); code != 1 {
		t.Errorf("create without summary code = %d, want 1", code)
	}
}

func TestRunUpdate(t *testing.T) {
	srv := testServer(t)
	setupEnv(t, srv.URL)

	_ = capture(t, func() {
		if code := run([]string{"update", "--key", "TODO-1", "--summary", "New",
			"--description", "D", "--priority", "Low"}); code != 0 {
			t.Errorf("update code = %d", code)
		}
	})

	if code := run([]string{"update", "--key", "TODO-1"}); code != 1 {
		t.Errorf("empty update code = %d, want 1", code)
	}
	if code := run([]string{"update"}); code != 1 {
		t.Errorf("update without key code = %d, want 1", code)
	}
}

func TestRunComment(t *testing.T) {
	srv := testServer(t)
	setupEnv(t, srv.URL)

	out := capture(t, func() {
		if code := run([]string{"comment", "--key", "TODO-1", "--body", "hi", "--output", "text"}); code != 0 {
			t.Errorf("comment code = %d", code)
		}
	})
	if !strings.Contains(out, "55") {
		t.Errorf("comment output = %q", out)
	}

	if code := run([]string{"comment", "--key", "TODO-1"}); code != 1 {
		t.Errorf("empty comment code = %d, want 1", code)
	}
}

func TestRunTransition(t *testing.T) {
	srv := testServer(t)
	setupEnv(t, srv.URL)

	list := capture(t, func() {
		if code := run([]string{"transition", "--key", "TODO-1", "--output", "text"}); code != 0 {
			t.Errorf("transition list code = %d", code)
		}
	})
	if !strings.Contains(list, "Done") {
		t.Errorf("transition list = %q", list)
	}

	_ = capture(t, func() {
		if code := run([]string{"transition", "--key", "TODO-1", "--to", "Done",
			"--comment", "c", "--output", "text"}); code != 0 {
			t.Errorf("transition apply code = %d", code)
		}
	})

	if code := run([]string{"transition", "--key", "TODO-1", "--to", "Nope"}); code != 1 {
		t.Errorf("unknown transition code = %d, want 1", code)
	}
}

func TestRunAssign(t *testing.T) {
	srv := testServer(t)
	setupEnv(t, srv.URL)

	_ = capture(t, func() {
		if code := run([]string{"assign", "--key", "TODO-1", "--assignee", "alice", "--output", "text"}); code != 0 {
			t.Errorf("assign code = %d", code)
		}
	})
	_ = capture(t, func() {
		if code := run([]string{"assign", "--key", "TODO-1", "--unassign", "--output", "text"}); code != 0 {
			t.Errorf("unassign code = %d", code)
		}
	})
	if code := run([]string{"assign", "--key", "TODO-1", "--assignee", "a", "--unassign"}); code != 1 {
		t.Errorf("mutually exclusive code = %d, want 1", code)
	}
	if code := run([]string{"assign", "--key", "TODO-1"}); code != 1 {
		t.Errorf("assign with neither code = %d, want 1", code)
	}
}

func TestRunLabels(t *testing.T) {
	srv := testServer(t)
	setupEnv(t, srv.URL)

	out := capture(t, func() {
		if code := run([]string{"labels", "--key", "TODO-1", "--add", "x", "--remove", "y", "--output", "text"}); code != 0 {
			t.Errorf("labels add/remove code = %d", code)
		}
	})
	if !strings.Contains(out, "a") {
		t.Errorf("labels output = %q", out)
	}

	_ = capture(t, func() {
		if code := run([]string{"labels", "--key", "TODO-1", "--output", "text"}); code != 0 {
			t.Errorf("labels list code = %d", code)
		}
	})
}

func TestRunDelete(t *testing.T) {
	srv := testServer(t)
	setupEnv(t, srv.URL)

	_ = capture(t, func() {
		if code := run([]string{"delete", "--key", "TODO-1", "--yes", "--output", "text"}); code != 0 {
			t.Errorf("delete --yes code = %d", code)
		}
	})

	// Confirmation prompt: typing "yes" proceeds.
	withStdin(t, "yes\n", func() {
		_ = capture(t, func() {
			if code := run([]string{"delete", "--key", "TODO-1", "--output", "text"}); code != 0 {
				t.Errorf("delete confirmed code = %d", code)
			}
		})
	})

	// Anything else aborts.
	withStdin(t, "no\n", func() {
		_ = capture(t, func() {
			if code := run([]string{"delete", "--key", "TODO-1"}); code != 1 {
				t.Errorf("delete aborted code = %d, want 1", code)
			}
		})
	})
}

func TestRunProjects(t *testing.T) {
	srv := testServer(t)
	setupEnv(t, srv.URL)

	out := capture(t, func() {
		if code := run([]string{"projects", "--output", "text"}); code != 0 {
			t.Errorf("projects code = %d", code)
		}
	})
	if !strings.Contains(out, "TODO") {
		t.Errorf("projects output = %q", out)
	}

	// JSON output path + --insecure flag (covers the TLS branch in jira.New).
	_ = capture(t, func() {
		if code := run([]string{"projects", "--insecure"}); code != 0 {
			t.Errorf("projects insecure code = %d", code)
		}
	})
}

func TestRunGenerateSkill(t *testing.T) {
	// stdout for every flavor (covers buildSkill switch).
	for _, f := range []string{"", "generic", "claude", "codex", "gemini", "opencode"} {
		args := []string{"generate-skill"}
		if f != "" {
			args = append(args, f)
		}
		args = append(args, "--stdout")
		out := capture(t, func() {
			if code := run(args); code != 0 {
				t.Errorf("generate-skill %q code = %d", f, code)
			}
		})
		if !strings.Contains(out, "Jira CLI") {
			t.Errorf("generate-skill %q output = %q", f, out[:min(40, len(out))])
		}
	}

	// File write, no-overwrite, then --force.
	outFile := filepath.Join(t.TempDir(), "skill.md")
	if code := capture2(t, run, []string{"generate-skill", "claude", "--out", outFile}); code != 0 {
		t.Fatalf("write code = %d", code)
	}
	if _, err := os.Stat(outFile); err != nil {
		t.Fatalf("expected file written: %v", err)
	}
	if code := run([]string{"generate-skill", "claude", "--out", outFile}); code != 1 {
		t.Errorf("overwrite-without-force code = %d, want 1", code)
	}
	if code := capture2(t, run, []string{"generate-skill", "claude", "--out", outFile, "--force"}); code != 0 {
		t.Errorf("force code = %d", code)
	}

	// Unknown flavor.
	if code := run([]string{"generate-skill", "bogus", "--stdout"}); code != 1 {
		t.Errorf("unknown flavor code = %d, want 1", code)
	}

	// Flavor given positionally after a flag (covers fs.Arg branch).
	_ = capture(t, func() {
		if code := run([]string{"generate-skill", "--stdout", "gemini"}); code != 0 {
			t.Errorf("positional flavor code = %d", code)
		}
	})
}

// capture2 runs fn(args) while discarding stdout and returns the exit code.
func capture2(t *testing.T, fn func([]string) int, args []string) int {
	t.Helper()
	var code int
	_ = capture(t, func() { code = fn(args) })
	return code
}

func TestRunDispatch(t *testing.T) {
	_ = capture(t, func() {
		if code := run([]string{"version"}); code != 0 {
			t.Errorf("version code = %d", code)
		}
	})
	_ = capture(t, func() {
		if code := run([]string{"help"}); code != 0 {
			t.Errorf("help code = %d", code)
		}
	})
	if code := run([]string{"bogus-cmd"}); code != 2 {
		t.Errorf("unknown command code = %d, want 2", code)
	}
	if code := run(nil); code != 2 {
		t.Errorf("no args code = %d, want 2", code)
	}
}

func TestRegisterCommonAndClientErrors(t *testing.T) {
	// Malformed config makes registerCommon (config.Load) fail.
	bad := filepath.Join(t.TempDir(), "bad.json")
	_ = os.WriteFile(bad, []byte("{nope"), 0o644)
	t.Setenv("JIRA_CONFIG", bad)
	t.Setenv("JIRA_BASE_URL", "")
	t.Setenv("JIRA_TOKEN", "")
	if code := run([]string{"projects"}); code != 1 {
		t.Errorf("bad-config code = %d, want 1", code)
	}

	// Valid config but missing credentials -> client() fails validation.
	good := filepath.Join(t.TempDir(), "ok.json")
	_ = os.WriteFile(good, []byte("{}"), 0o644)
	t.Setenv("JIRA_CONFIG", good)
	t.Setenv("JIRA_BASE_URL", "https://jira.example.com")
	t.Setenv("JIRA_TOKEN", "")
	if code := run([]string{"projects"}); code != 1 {
		t.Errorf("no-creds code = %d, want 1", code)
	}
}

// TestCommandsAPIError drives every command against a server that fails, so the
// "API call returned an error" branch in each run* function is exercised.
func TestCommandsAPIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"errorMessages":["boom"]}`))
	}))
	t.Cleanup(srv.Close)
	setupEnv(t, srv.URL)

	cmds := [][]string{
		{"search", "--project", "TODO"},
		{"get", "--key", "TODO-1"},
		{"create", "--project", "TODO", "--summary", "S"},
		{"update", "--key", "TODO-1", "--summary", "S"},
		{"comment", "--key", "TODO-1", "--body", "b"},
		{"transition", "--key", "TODO-1"},                // GetTransitions fails
		{"assign", "--key", "TODO-1", "--assignee", "a"}, // AssignIssue fails
		{"labels", "--key", "TODO-1", "--add", "x"},      // UpdateOps fails
		{"labels", "--key", "TODO-1"},                    // GetIssue fails (list path)
		{"delete", "--key", "TODO-1", "--yes"},           // DeleteIssue fails
		{"projects"},
	}
	for _, c := range cmds {
		if code := capture2(t, run, c); code != 1 {
			t.Errorf("%v against failing server code = %d, want 1", c, code)
		}
	}

	// delete without --yes: the summary GetIssue fails before the prompt.
	withStdin(t, "yes\n", func() {
		if code := capture2(t, run, []string{"delete", "--key", "TODO-1"}); code != 1 {
			t.Errorf("delete confirm-path API error code = %d, want 1", code)
		}
	})
}

// TestCommandsClientError exercises the common.client() error branch in each
// command by supplying a base URL but no credentials.
func TestCommandsClientError(t *testing.T) {
	cfg := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(cfg, []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("JIRA_CONFIG", cfg)
	t.Setenv("JIRA_BASE_URL", "https://jira.example.com")
	t.Setenv("JIRA_TOKEN", "")
	t.Setenv("JIRA_USER", "")
	t.Setenv("JIRA_PASSWORD", "")
	t.Setenv("JIRA_PROJECT", "TODO")

	cmds := [][]string{
		{"search", "--project", "TODO"},
		{"get", "--key", "TODO-1"},
		{"create", "--project", "TODO", "--summary", "S"},
		{"update", "--key", "TODO-1", "--summary", "S"},
		{"comment", "--key", "TODO-1", "--body", "b"},
		{"transition", "--key", "TODO-1"},
		{"assign", "--key", "TODO-1", "--assignee", "a"},
		{"labels", "--key", "TODO-1", "--add", "x"},
		{"delete", "--key", "TODO-1", "--yes"},
		{"projects"},
	}
	for _, c := range cmds {
		if code := run(c); code != 1 {
			t.Errorf("%v with no creds code = %d, want 1", c, code)
		}
	}
}

// emptyServer answers every endpoint with empty collections, to exercise the
// "no results" text branches.
func emptyServer(t *testing.T) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("/rest/api/2/search", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(jira.SearchResult{})
	})
	mux.HandleFunc("/rest/api/2/project", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode([]jira.Project{})
	})
	mux.HandleFunc("/rest/api/2/issue/", func(w http.ResponseWriter, r *http.Request) {
		p := strings.TrimPrefix(r.URL.Path, "/rest/api/2/issue/")
		switch {
		case strings.HasSuffix(p, "/transitions"):
			_ = json.NewEncoder(w).Encode(map[string]any{"transitions": []jira.Transition{}})
		default:
			_ = json.NewEncoder(w).Encode(jira.Issue{Key: "TODO-1", Fields: &jira.Fields{}})
		}
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv
}

func TestEmptyResultsText(t *testing.T) {
	srv := emptyServer(t)
	setupEnv(t, srv.URL)
	for _, c := range [][]string{
		{"search", "--project", "TODO", "--output", "text"},
		{"projects", "--output", "text"},
		{"transition", "--key", "TODO-1", "--output", "text"},
		{"labels", "--key", "TODO-1", "--output", "text"},
	} {
		if code := capture2(t, run, c); code != 0 {
			t.Errorf("%v code = %d, want 0", c, code)
		}
	}
}

func TestCommandsRegisterCommonError(t *testing.T) {
	bad := filepath.Join(t.TempDir(), "bad.json")
	_ = os.WriteFile(bad, []byte("{nope"), 0o644)
	t.Setenv("JIRA_CONFIG", bad)
	for _, c := range [][]string{
		{"search", "--project", "TODO"}, {"get", "--key", "X"}, {"create", "--summary", "S"},
		{"update", "--key", "X"}, {"comment", "--key", "X"}, {"transition", "--key", "X"},
		{"assign", "--key", "X"}, {"labels", "--key", "X"}, {"delete", "--key", "X"}, {"projects"},
	} {
		if code := run(c); code != 1 {
			t.Errorf("%v bad-config code = %d, want 1", c, code)
		}
	}
}

func TestCommandsMissingRequiredFlag(t *testing.T) {
	srv := testServer(t)
	setupEnv(t, srv.URL)
	// Each command missing its required --key/--summary hits requireFlag.
	for _, c := range [][]string{
		{"get"}, {"update"}, {"comment"}, {"transition"},
		{"assign"}, {"labels"}, {"delete"},
	} {
		if code := run(c); code != 1 {
			t.Errorf("%v missing-flag code = %d, want 1", c, code)
		}
	}
}

func TestRunUpdateText(t *testing.T) {
	srv := testServer(t)
	setupEnv(t, srv.URL)
	out := capture(t, func() {
		if code := run([]string{"update", "--key", "TODO-1", "--summary", "S", "--output", "text"}); code != 0 {
			t.Errorf("update text code = %d", code)
		}
	})
	if !strings.Contains(out, "TODO-1") {
		t.Errorf("update text output = %q", out)
	}

	// --description-file pointing at a missing file -> readBody error.
	if code := run([]string{"update", "--key", "TODO-1", "--description-file",
		filepath.Join(t.TempDir(), "missing")}); code != 1 {
		t.Errorf("update bad description-file code = %d, want 1", code)
	}
}

func TestGenerateSkillGenericFile(t *testing.T) {
	out := filepath.Join(t.TempDir(), "generic.md")
	// generic flavor exercises the label == "" -> "generic" branch.
	if code := capture2(t, run, []string{"generate-skill", "--out", out}); code != 0 {
		t.Errorf("generate-skill generic file code = %d", code)
	}
	if _, err := os.Stat(out); err != nil {
		t.Errorf("expected file: %v", err)
	}
}

func TestEmitUnknownFormat(t *testing.T) {
	if err := emit("xml", map[string]string{"a": "b"}, func() {}); err == nil {
		t.Error("expected error for unknown format")
	}
}

func TestReadBody(t *testing.T) {
	if got, _ := readBody("literal", ""); got != "literal" {
		t.Errorf("value = %q", got)
	}
	f := filepath.Join(t.TempDir(), "b.txt")
	_ = os.WriteFile(f, []byte("file-body"), 0o644)
	if got, _ := readBody("", f); got != "file-body" {
		t.Errorf("file = %q", got)
	}
	if _, err := readBody("", filepath.Join(t.TempDir(), "missing")); err == nil {
		t.Error("expected error reading missing file")
	}
	withStdin(t, "stdin-body", func() {
		if got, _ := readBody("", "-"); got != "stdin-body" {
			t.Errorf("stdin = %q", got)
		}
	})
}

func TestRequireFlag(t *testing.T) {
	if err := requireFlag("key", ""); err == nil {
		t.Error("expected error for empty flag")
	}
	if err := requireFlag("key", "v"); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestUsageAndFlavorNames(t *testing.T) {
	_ = capture(t, func() { usage(os.Stdout) }) // smoke; writes help text
	names := flavorNames()
	if len(names) != 4 {
		t.Errorf("flavorNames = %v", names)
	}
}
