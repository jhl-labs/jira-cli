package jira

import (
	"context"
	"net/url"
	"strconv"
	"strings"
)

// Issue is a Jira issue.
type Issue struct {
	ID     string  `json:"id,omitempty"`
	Key    string  `json:"key,omitempty"`
	Self   string  `json:"self,omitempty"`
	Fields *Fields `json:"fields,omitempty"`
}

// Fields holds the commonly used issue fields. On Server/Data Center the
// description is a plain string (wiki markup), unlike Cloud's ADF object.
type Fields struct {
	Summary     string    `json:"summary,omitempty"`
	Description string    `json:"description,omitempty"`
	IssueType   *NamedRef `json:"issuetype,omitempty"`
	Project     *Project  `json:"project,omitempty"`
	Status      *NamedRef `json:"status,omitempty"`
	Priority    *NamedRef `json:"priority,omitempty"`
	Assignee    *User     `json:"assignee,omitempty"`
	Reporter    *User     `json:"reporter,omitempty"`
	Labels      []string  `json:"labels,omitempty"`
	Created     string    `json:"created,omitempty"`
	Updated     string    `json:"updated,omitempty"`
}

// NamedRef is a {id,name} reference (status, issue type, priority).
type NamedRef struct {
	ID   string `json:"id,omitempty"`
	Name string `json:"name,omitempty"`
}

// Project is a Jira project.
type Project struct {
	ID             string `json:"id,omitempty"`
	Key            string `json:"key,omitempty"`
	Name           string `json:"name,omitempty"`
	ProjectTypeKey string `json:"projectTypeKey,omitempty"`
}

// User is a Jira user (Server/DC identifies users by name).
type User struct {
	Name         string `json:"name,omitempty"`
	Key          string `json:"key,omitempty"`
	DisplayName  string `json:"displayName,omitempty"`
	EmailAddress string `json:"emailAddress,omitempty"`
}

// SearchResult is the response from a JQL search.
type SearchResult struct {
	Issues     []Issue `json:"issues"`
	Total      int     `json:"total"`
	MaxResults int     `json:"maxResults"`
	StartAt    int     `json:"startAt"`
}

// WebURL returns the browser URL for an issue.
func (c *Client) WebURL(key string) string {
	if key == "" {
		return ""
	}
	return c.baseURL + "/browse/" + key
}

// Search runs a JQL query.
func (c *Client) Search(ctx context.Context, jql string, fields []string, maxResults, startAt int) (*SearchResult, error) {
	q := url.Values{}
	q.Set("jql", jql)
	if maxResults > 0 {
		q.Set("maxResults", strconv.Itoa(maxResults))
	}
	if startAt > 0 {
		q.Set("startAt", strconv.Itoa(startAt))
	}
	if len(fields) > 0 {
		q.Set("fields", strings.Join(fields, ","))
	}
	var out SearchResult
	if err := c.doJSON(ctx, "GET", "/rest/api/2/search", q, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// GetIssue fetches a single issue by key or ID.
func (c *Client) GetIssue(ctx context.Context, key string, fields, expand []string) (*Issue, error) {
	q := url.Values{}
	if len(fields) > 0 {
		q.Set("fields", strings.Join(fields, ","))
	}
	if len(expand) > 0 {
		q.Set("expand", strings.Join(expand, ","))
	}
	var out Issue
	if err := c.doJSON(ctx, "GET", "/rest/api/2/issue/"+url.PathEscape(key), q, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// CreateInput describes a new issue.
type CreateInput struct {
	ProjectKey  string
	Summary     string
	Description string
	IssueType   string // type name, e.g. "Task", "Bug"
	Assignee    string // optional username
	Priority    string // optional priority name
	Labels      []string
}

// Create makes a new issue and returns its key.
func (c *Client) Create(ctx context.Context, in CreateInput) (*Issue, error) {
	fields := map[string]any{
		"project":   map[string]string{"key": in.ProjectKey},
		"summary":   in.Summary,
		"issuetype": map[string]string{"name": in.IssueType},
	}
	if in.Description != "" {
		fields["description"] = in.Description
	}
	if in.Assignee != "" {
		fields["assignee"] = map[string]string{"name": in.Assignee}
	}
	if in.Priority != "" {
		fields["priority"] = map[string]string{"name": in.Priority}
	}
	if len(in.Labels) > 0 {
		fields["labels"] = in.Labels
	}

	var out Issue
	if err := c.doJSON(ctx, "POST", "/rest/api/2/issue", nil, map[string]any{"fields": fields}, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// UpdateFields sets issue fields (PUT with the "fields" verb).
func (c *Client) UpdateFields(ctx context.Context, key string, fields map[string]any) error {
	body := map[string]any{"fields": fields}
	return c.doJSON(ctx, "PUT", "/rest/api/2/issue/"+url.PathEscape(key), nil, body, nil)
}

// UpdateOps applies field operations (PUT with the "update" verb), e.g.
// adding/removing labels: {"labels":[{"add":"x"},{"remove":"y"}]}.
func (c *Client) UpdateOps(ctx context.Context, key string, update map[string]any) error {
	body := map[string]any{"update": update}
	return c.doJSON(ctx, "PUT", "/rest/api/2/issue/"+url.PathEscape(key), nil, body, nil)
}

// AddComment posts a comment on an issue.
func (c *Client) AddComment(ctx context.Context, key, body string) (*Comment, error) {
	var out Comment
	if err := c.doJSON(ctx, "POST", "/rest/api/2/issue/"+url.PathEscape(key)+"/comment", nil, map[string]string{"body": body}, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// Comment is an issue comment.
type Comment struct {
	ID     string `json:"id,omitempty"`
	Body   string `json:"body,omitempty"`
	Author *User  `json:"author,omitempty"`
}

// AssignIssue sets the assignee. Pass "-1" to assign to the default assignee,
// or an empty string to unassign.
func (c *Client) AssignIssue(ctx context.Context, key, username string) error {
	var name any
	if username == "" {
		name = nil // unassign
	} else {
		name = username
	}
	body := map[string]any{"name": name}
	return c.doJSON(ctx, "PUT", "/rest/api/2/issue/"+url.PathEscape(key)+"/assignee", nil, body, nil)
}

// DeleteIssue deletes an issue.
func (c *Client) DeleteIssue(ctx context.Context, key string) error {
	return c.doJSON(ctx, "DELETE", "/rest/api/2/issue/"+url.PathEscape(key), nil, nil, nil)
}

// ListProjects returns all projects visible to the user.
func (c *Client) ListProjects(ctx context.Context) ([]Project, error) {
	var out []Project
	if err := c.doJSON(ctx, "GET", "/rest/api/2/project", nil, nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}
