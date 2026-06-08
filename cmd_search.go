package main

import (
	"context"
	"flag"
	"fmt"
	"strings"
)

func runSearch(args []string) error {
	fs := flag.NewFlagSet("search", flag.ExitOnError)
	common, err := registerCommon(fs)
	if err != nil {
		return err
	}
	var (
		jql     = fs.String("jql", "", "raw JQL query (overrides --project/--text/--status)")
		project = fs.String("project", common.defaultProject, "restrict to a project key (default: JIRA_PROJECT)")
		text    = fs.String("text", "", "free-text search (text ~ ...)")
		status  = fs.String("status", "", "restrict to a status (e.g. \"In Progress\")")
		max     = fs.Int("max", 50, "maximum results")
		fields  = fs.String("fields", "summary,status,assignee", "comma-separated fields to return")
	)
	fs.Usage = func() {
		fmt.Fprintln(fs.Output(), "Usage: jira-cli search [--jql Q | --project KEY --text T --status S] [flags]")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		return err
	}

	query := *jql
	if query == "" {
		query = buildJQL(*project, *text, *status)
	}
	if query == "" {
		return fmt.Errorf("provide --jql, or --project/--text/--status")
	}

	cl, err := common.client()
	if err != nil {
		return err
	}

	res, err := cl.Search(context.Background(), query, splitCSV(*fields), *max, 0)
	if err != nil {
		return err
	}

	return emit(common.output, res, func() {
		if len(res.Issues) == 0 {
			fmt.Println("no issues")
			return
		}
		for _, it := range res.Issues {
			issueLine(it)
		}
		fmt.Printf("\n%d of %d issue(s)\n", len(res.Issues), res.Total)
	})
}

// buildJQL composes a simple JQL query from a project, text, and status.
func buildJQL(project, text, status string) string {
	var clauses []string
	if project != "" {
		clauses = append(clauses, "project = "+jqlQuote(project))
	}
	if status != "" {
		clauses = append(clauses, "status = "+jqlQuote(status))
	}
	if text != "" {
		clauses = append(clauses, "text ~ "+jqlQuote(text))
	}
	if len(clauses) == 0 {
		return ""
	}
	return strings.Join(clauses, " AND ") + " ORDER BY updated DESC"
}

// jqlQuote wraps a value in double quotes, escaping embedded quotes/backslashes.
func jqlQuote(v string) string {
	v = strings.ReplaceAll(v, `\`, `\\`)
	v = strings.ReplaceAll(v, `"`, `\"`)
	return `"` + v + `"`
}
