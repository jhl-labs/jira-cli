package jira

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"jira-cli/internal/config"
)

func testClient(t *testing.T, srv *httptest.Server) *Client {
	t.Helper()
	cl, err := New(config.Config{BaseURL: srv.URL, Token: "secret"}, 5*time.Second)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	cl.RetryWait = time.Millisecond
	return cl
}

func TestSearchSendsBearerAndJQL(t *testing.T) {
	var auth, query string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth = r.Header.Get("Authorization")
		query = r.URL.RawQuery
		json.NewEncoder(w).Encode(SearchResult{
			Issues: []Issue{{Key: "PROJ-1", Fields: &Fields{Summary: "Hi"}}},
			Total:  1,
		})
	}))
	defer srv.Close()

	res, err := testClient(t, srv).Search(context.Background(), "project = PROJ", []string{"summary"}, 10, 0)
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if auth != "Bearer secret" {
		t.Errorf("auth = %q", auth)
	}
	if res.Total != 1 || res.Issues[0].Key != "PROJ-1" {
		t.Errorf("unexpected result: %+v", res)
	}
	if query == "" || !contains(query, "jql=") {
		t.Errorf("query missing jql: %q", query)
	}
}

func TestCreateIssue(t *testing.T) {
	var body map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		data, _ := io.ReadAll(r.Body)
		json.Unmarshal(data, &body)
		json.NewEncoder(w).Encode(Issue{ID: "1001", Key: "PROJ-9"})
	}))
	defer srv.Close()

	issue, err := testClient(t, srv).Create(context.Background(), CreateInput{
		ProjectKey: "PROJ", Summary: "S", IssueType: "Bug",
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if issue.Key != "PROJ-9" {
		t.Errorf("key = %q", issue.Key)
	}
	fields, _ := body["fields"].(map[string]any)
	if fields == nil || fields["summary"] != "S" {
		t.Errorf("posted fields = %v", body["fields"])
	}
	if it, _ := fields["issuetype"].(map[string]any); it == nil || it["name"] != "Bug" {
		t.Errorf("issuetype = %v", fields["issuetype"])
	}
}

func TestParseJiraError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"errorMessages":["Field 'summary' is required"],"errors":{}}`))
	}))
	defer srv.Close()

	_, err := testClient(t, srv).GetIssue(context.Background(), "PROJ-1", nil, nil)
	if err == nil {
		t.Fatal("expected error")
	}
	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("type = %T, want *APIError", err)
	}
	if apiErr.StatusCode != 400 || apiErr.Message != "Field 'summary' is required" {
		t.Errorf("apiErr = %+v", apiErr)
	}
}

func contains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
