package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"os"
	"strings"
)

func runDelete(args []string) error {
	fs := flag.NewFlagSet("delete", flag.ExitOnError)
	common, err := registerCommon(fs)
	if err != nil {
		return err
	}
	var (
		key = fs.String("key", "", "issue key to delete (required)")
		yes = fs.Bool("yes", false, "skip the confirmation prompt")
	)
	fs.Usage = func() {
		fmt.Fprintln(fs.Output(), "Usage: jira-cli delete --key PROJ-123 [--yes]")
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
	ctx := context.Background()

	if !*yes {
		issue, err := cl.GetIssue(ctx, *key, []string{"summary"}, nil)
		if err != nil {
			return err
		}
		summary := ""
		if issue.Fields != nil {
			summary = issue.Fields.Summary
		}
		fmt.Fprintf(os.Stderr, "About to delete %s: %q\nType 'yes' to confirm: ", *key, summary)
		line, _ := bufio.NewReader(os.Stdin).ReadString('\n')
		if strings.TrimSpace(line) != "yes" {
			return fmt.Errorf("aborted")
		}
	}

	if err := cl.DeleteIssue(ctx, *key); err != nil {
		return err
	}

	return emit(common.output, map[string]string{"key": *key, "status": "deleted"}, func() {
		fmt.Printf("deleted issue %s\n", *key)
	})
}
