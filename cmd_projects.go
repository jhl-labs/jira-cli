package main

import (
	"context"
	"flag"
	"fmt"
)

func runProjects(args []string) error {
	fs := flag.NewFlagSet("projects", flag.ExitOnError)
	common, err := registerCommon(fs)
	if err != nil {
		return err
	}
	fs.Usage = func() {
		fmt.Fprintln(fs.Output(), "Usage: jira-cli projects [flags]")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		return err
	}

	cl, err := common.client()
	if err != nil {
		return err
	}

	projects, err := cl.ListProjects(context.Background())
	if err != nil {
		return err
	}

	return emit(common.output, projects, func() {
		if len(projects) == 0 {
			fmt.Println("no projects")
			return
		}
		for _, p := range projects {
			fmt.Printf("%-14s %-12s %s\n", p.Key, p.ProjectTypeKey, p.Name)
		}
		fmt.Printf("\n%d project(s)\n", len(projects))
	})
}
