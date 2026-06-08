package main

import (
	"encoding/json"
	"fmt"
	"os"

	"jira-cli/internal/jira"
)

// emit writes either pretty JSON or a human-readable text summary.
func emit(format string, v any, textFn func()) error {
	switch format {
	case "json":
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(v)
	case "text":
		textFn()
		return nil
	default:
		return fmt.Errorf("unknown output format %q (use json or text)", format)
	}
}

// issueLine prints a one-line summary of an issue (for search/get text output).
func issueLine(it jira.Issue) {
	status, assignee, summary := "-", "-", ""
	if it.Fields != nil {
		if it.Fields.Status != nil {
			status = it.Fields.Status.Name
		}
		if it.Fields.Assignee != nil {
			assignee = it.Fields.Assignee.Name
		}
		summary = it.Fields.Summary
	}
	fmt.Printf("%-14s %-14s %-14s %s\n", it.Key, status, assignee, summary)
}
