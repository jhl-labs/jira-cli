package main

import (
	"context"
	"flag"
	"fmt"
)

func runAssign(args []string) error {
	fs := flag.NewFlagSet("assign", flag.ExitOnError)
	common, err := registerCommon(fs)
	if err != nil {
		return err
	}
	var (
		key      = fs.String("key", "", "issue key (required)")
		assignee = fs.String("assignee", "", "assignee username")
		unassign = fs.Bool("unassign", false, "remove the current assignee")
	)
	fs.Usage = func() {
		fmt.Fprintln(fs.Output(), "Usage: jira-cli assign --key PROJ-123 (--assignee USERNAME | --unassign)")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		return err
	}
	if err := requireFlag("key", *key); err != nil {
		return err
	}
	if *assignee == "" && !*unassign {
		return fmt.Errorf("provide --assignee USERNAME or --unassign")
	}
	if *assignee != "" && *unassign {
		return fmt.Errorf("--assignee and --unassign are mutually exclusive")
	}

	cl, err := common.client()
	if err != nil {
		return err
	}

	name := *assignee // empty string => unassign
	if err := cl.AssignIssue(context.Background(), *key, name); err != nil {
		return err
	}

	who := *assignee
	if *unassign {
		who = "(unassigned)"
	}
	result := map[string]string{"key": *key, "assignee": who, "status": "assigned"}
	return emit(common.output, result, func() {
		fmt.Printf("assigned %s to %s\n", *key, who)
	})
}
