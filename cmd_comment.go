package main

import (
	"context"
	"flag"
	"fmt"
)

func runComment(args []string) error {
	fs := flag.NewFlagSet("comment", flag.ExitOnError)
	common, err := registerCommon(fs)
	if err != nil {
		return err
	}
	var (
		key      = fs.String("key", "", "issue key (required)")
		bodyVal  = fs.String("body", "", "comment body")
		bodyFile = fs.String("body-file", "", `read body from file ("-" for stdin)`)
	)
	fs.Usage = func() {
		fmt.Fprintln(fs.Output(), "Usage: jira-cli comment --key PROJ-123 (--body B | --body-file F)")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		return err
	}
	if err := requireFlag("key", *key); err != nil {
		return err
	}

	body, err := readBody(*bodyVal, *bodyFile)
	if err != nil {
		return err
	}
	if body == "" {
		return fmt.Errorf("comment body is empty (use --body or --body-file)")
	}

	cl, err := common.client()
	if err != nil {
		return err
	}

	comment, err := cl.AddComment(context.Background(), *key, body)
	if err != nil {
		return err
	}

	return emit(common.output, comment, func() {
		fmt.Printf("added comment %s on %s\n", comment.ID, *key)
	})
}
