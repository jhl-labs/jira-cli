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
		user     = fs.String("user", "", "assignee username")
		unassign = fs.Bool("unassign", false, "remove the current assignee")
	)
	fs.Usage = func() {
		fmt.Fprintln(fs.Output(), "Usage: jira-cli assign --key PROJ-123 (--user USERNAME | --unassign)")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		return err
	}
	if err := requireFlag("key", *key); err != nil {
		return err
	}
	if *user == "" && !*unassign {
		return fmt.Errorf("provide --user USERNAME or --unassign")
	}
	if *user != "" && *unassign {
		return fmt.Errorf("--user and --unassign are mutually exclusive")
	}

	cl, err := common.client()
	if err != nil {
		return err
	}

	name := *user // empty string => unassign
	if err := cl.AssignIssue(context.Background(), *key, name); err != nil {
		return err
	}

	who := *user
	if *unassign {
		who = "(unassigned)"
	}
	result := map[string]string{"key": *key, "assignee": who, "status": "assigned"}
	return emit(common.output, result, func() {
		fmt.Printf("assigned %s to %s\n", *key, who)
	})
}
