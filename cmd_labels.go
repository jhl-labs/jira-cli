package main

import (
	"context"
	"flag"
	"fmt"
	"strings"
)

func runLabels(args []string) error {
	fs := flag.NewFlagSet("labels", flag.ExitOnError)
	common, err := registerCommon(fs)
	if err != nil {
		return err
	}
	var (
		key    = fs.String("key", "", "issue key (required)")
		add    = fs.String("add", "", "comma-separated labels to add")
		remove = fs.String("remove", "", "comma-separated labels to remove")
	)
	fs.Usage = func() {
		fmt.Fprintln(fs.Output(), "Usage: jira-cli labels --key PROJ-123 [--add a,b] [--remove c]")
		fmt.Fprintln(fs.Output(), "  With no --add/--remove, lists current labels.")
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

	adds, removes := splitCSV(*add), splitCSV(*remove)
	if len(adds) > 0 || len(removes) > 0 {
		ops := make([]map[string]string, 0, len(adds)+len(removes))
		for _, l := range adds {
			ops = append(ops, map[string]string{"add": l})
		}
		for _, l := range removes {
			ops = append(ops, map[string]string{"remove": l})
		}
		if err := cl.UpdateOps(ctx, *key, map[string]any{"labels": ops}); err != nil {
			return err
		}
	}

	// Report the resulting label set.
	issue, err := cl.GetIssue(ctx, *key, []string{"labels"}, nil)
	if err != nil {
		return err
	}
	var labels []string
	if issue.Fields != nil {
		labels = issue.Fields.Labels
	}
	return emit(common.output, map[string]any{"key": *key, "labels": labels}, func() {
		if len(labels) == 0 {
			fmt.Println("(no labels)")
			return
		}
		fmt.Println(strings.Join(labels, ", "))
	})
}
