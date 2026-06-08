package jira

import (
	"context"
	"fmt"
	"net/url"
	"strings"
)

// Transition is an available workflow transition for an issue.
type Transition struct {
	ID   string    `json:"id"`
	Name string    `json:"name"`
	To   *NamedRef `json:"to,omitempty"`
}

type transitionsResp struct {
	Transitions []Transition `json:"transitions"`
}

// GetTransitions lists the transitions available from an issue's current status.
func (c *Client) GetTransitions(ctx context.Context, key string) ([]Transition, error) {
	var out transitionsResp
	if err := c.doJSON(ctx, "GET", "/rest/api/2/issue/"+url.PathEscape(key)+"/transitions", nil, nil, &out); err != nil {
		return nil, err
	}
	return out.Transitions, nil
}

// DoTransition applies a transition by ID, optionally adding a comment and/or
// reassigning as part of the transition.
func (c *Client) DoTransition(ctx context.Context, key, transitionID, comment, assignee string) error {
	body := map[string]any{
		"transition": map[string]string{"id": transitionID},
	}
	if assignee != "" {
		body["fields"] = map[string]any{"assignee": map[string]string{"name": assignee}}
	}
	if comment != "" {
		body["update"] = map[string]any{
			"comment": []map[string]any{{"add": map[string]string{"body": comment}}},
		}
	}
	return c.doJSON(ctx, "POST", "/rest/api/2/issue/"+url.PathEscape(key)+"/transitions", nil, body, nil)
}

// ResolveTransition finds a transition by ID or (case-insensitive) name.
func ResolveTransition(transitions []Transition, idOrName string) (*Transition, error) {
	for i := range transitions {
		if transitions[i].ID == idOrName {
			return &transitions[i], nil
		}
	}
	for i := range transitions {
		if strings.EqualFold(transitions[i].Name, idOrName) {
			return &transitions[i], nil
		}
	}
	names := make([]string, 0, len(transitions))
	for _, t := range transitions {
		names = append(names, fmt.Sprintf("%q (id %s)", t.Name, t.ID))
	}
	return nil, fmt.Errorf("no transition matching %q; available: %s", idOrName, strings.Join(names, ", "))
}
