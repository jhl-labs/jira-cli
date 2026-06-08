package main

import (
	"context"
	"flag"
	"fmt"

	"jira-cli/internal/jira"
)

func runCreate(args []string) error {
	fs := flag.NewFlagSet("create", flag.ExitOnError)
	common, err := registerCommon(fs)
	if err != nil {
		return err
	}
	var (
		project  = fs.String("project", common.defaultProject, "project key (required; default: JIRA_PROJECT)")
		summary  = fs.String("summary", "", "issue summary/title (required)")
		descVal  = fs.String("description", "", "issue description")
		descFile = fs.String("description-file", "", `read description from file ("-" for stdin)`)
		itype    = fs.String("type", "Task", "issue type name (e.g. Task, Bug, Story)")
		assignee = fs.String("assignee", "", "assignee username (optional)")
		priority = fs.String("priority", "", "priority name (optional)")
		labels   = fs.String("labels", "", "comma-separated labels (optional)")
	)
	fs.Usage = func() {
		fmt.Fprintln(fs.Output(), "Usage: jira-cli create --project KEY --summary S [--type Task] [flags]")
		fmt.Fprintln(fs.Output(), "\nNote: on Server/DC the description is plain text / wiki markup, not Markdown or ADF.")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		return err
	}
	if err := requireFlag("project", *project); err != nil {
		return err
	}
	if err := requireFlag("summary", *summary); err != nil {
		return err
	}

	description, err := readBody(*descVal, *descFile)
	if err != nil {
		return err
	}

	cl, err := common.client()
	if err != nil {
		return err
	}

	issue, err := cl.Create(context.Background(), jira.CreateInput{
		ProjectKey:  *project,
		Summary:     *summary,
		Description: description,
		IssueType:   *itype,
		Assignee:    *assignee,
		Priority:    *priority,
		Labels:      splitCSV(*labels),
	})
	if err != nil {
		return err
	}

	return emit(common.output, issue, func() {
		fmt.Printf("created issue %s\n", issue.Key)
		if u := cl.WebURL(issue.Key); u != "" {
			fmt.Println(u)
		}
	})
}
