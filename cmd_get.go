package main

import (
	"context"
	"flag"
	"fmt"
)

func runGet(args []string) error {
	fs := flag.NewFlagSet("get", flag.ExitOnError)
	common, err := registerCommon(fs)
	if err != nil {
		return err
	}
	var (
		key    = fs.String("key", "", "issue key, e.g. PROJ-123 (required)")
		fields = fs.String("fields", "", "comma-separated fields to return (default: all)")
		desc   = fs.Bool("description", false, "print only the description (text output)")
	)
	fs.Usage = func() {
		fmt.Fprintln(fs.Output(), "Usage: jira-cli get --key PROJ-123 [flags]")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		return err
	}
	if err := requireFlag("key", *key); err != nil {
		return err
	}

	cl, err := common.client()
	if err != nil {
		return err
	}

	issue, err := cl.GetIssue(context.Background(), *key, splitCSV(*fields), nil)
	if err != nil {
		return err
	}

	return emit(common.output, issue, func() {
		if *desc {
			if issue.Fields != nil {
				fmt.Println(issue.Fields.Description)
			}
			return
		}
		issueLine(*issue)
		if issue.Fields != nil {
			if issue.Fields.IssueType != nil {
				fmt.Printf("type:      %s\n", issue.Fields.IssueType.Name)
			}
			if issue.Fields.Priority != nil {
				fmt.Printf("priority:  %s\n", issue.Fields.Priority.Name)
			}
			if len(issue.Fields.Labels) > 0 {
				fmt.Printf("labels:    %v\n", issue.Fields.Labels)
			}
		}
		if u := cl.WebURL(issue.Key); u != "" {
			fmt.Println(u)
		}
	})
}
