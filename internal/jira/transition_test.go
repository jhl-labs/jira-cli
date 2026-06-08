package jira

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGetAndDoTransition(t *testing.T) {
	var postBody map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			json.NewEncoder(w).Encode(transitionsResp{Transitions: []Transition{
				{ID: "21", Name: "Done", To: &NamedRef{Name: "Done"}},
				{ID: "11", Name: "In Progress"},
			}})
		case http.MethodPost:
			data, _ := io.ReadAll(r.Body)
			json.Unmarshal(data, &postBody)
			w.WriteHeader(http.StatusNoContent)
		}
	}))
	defer srv.Close()

	cl := testClient(t, srv)
	ts, err := cl.GetTransitions(context.Background(), "PROJ-1")
	if err != nil {
		t.Fatalf("GetTransitions: %v", err)
	}

	target, err := ResolveTransition(ts, "done") // case-insensitive name match
	if err != nil {
		t.Fatalf("ResolveTransition: %v", err)
	}
	if target.ID != "21" {
		t.Errorf("resolved id = %s, want 21", target.ID)
	}

	if err := cl.DoTransition(context.Background(), "PROJ-1", target.ID, "shipped", ""); err != nil {
		t.Fatalf("DoTransition: %v", err)
	}
	tr, _ := postBody["transition"].(map[string]any)
	if tr == nil || tr["id"] != "21" {
		t.Errorf("posted transition = %v", postBody["transition"])
	}
	if _, ok := postBody["update"]; !ok {
		t.Errorf("expected comment update in body, got %v", postBody)
	}
}

func TestResolveTransitionNotFound(t *testing.T) {
	_, err := ResolveTransition([]Transition{{ID: "1", Name: "Open"}}, "Closed")
	if err == nil {
		t.Fatal("expected error for unknown transition")
	}
}
