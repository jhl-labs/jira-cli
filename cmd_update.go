package main

import (
	"context"
	"flag"
	"fmt"
)

func runUpdate(args []string) error {
	fs := flag.NewFlagSet("update", flag.ExitOnError)
	common, err := registerCommon(fs)
	if err != nil {
		return err
	}
	var (
		key      = fs.String("key", "", "issue key (required)")
		summary  = fs.String("summary", "", "new summary")
		descVal  = fs.String("description", "", "new description")
		descFile = fs.String("description-file", "", `read description from file ("-" for stdin)`)
		priority = fs.String("priority", "", "new priority name")
	)
	fs.Usage = func() {
		fmt.Fprintln(fs.Output(), "Usage: jira-cli update --key PROJ-123 [--summary S] [--description D] [--priority P]")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		return err
	}
	if err := requireFlag("key", *key); err != nil {
		return err
	}

	fields := map[string]any{}
	if *summary != "" {
		fields["summary"] = *summary
	}
	if *descVal != "" || *descFile != "" {
		description, err := readBody(*descVal, *descFile)
		if err != nil {
			return err
		}
		fields["description"] = description
	}
	if *priority != "" {
		fields["priority"] = map[string]string{"name": *priority}
	}
	if len(fields) == 0 {
		return fmt.Errorf("nothing to update (set --summary, --description, or --priority)")
	}

	cl, err := common.client()
	if err != nil {
		return err
	}

	if err := cl.UpdateFields(context.Background(), *key, fields); err != nil {
		return err
	}

	result := map[string]string{"key": *key, "status": "updated"}
	return emit(common.output, result, func() {
		fmt.Printf("updated issue %s\n", *key)
		if u := cl.WebURL(*key); u != "" {
			fmt.Println(u)
		}
	})
}
