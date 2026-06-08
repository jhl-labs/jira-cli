package main

import (
	"context"
	"flag"
	"fmt"

	"jira-cli/internal/jira"
)

func runTransition(args []string) error {
	fs := flag.NewFlagSet("transition", flag.ExitOnError)
	common, err := registerCommon(fs)
	if err != nil {
		return err
	}
	var (
		key      = fs.String("key", "", "issue key (required)")
		to       = fs.String("to", "", "transition to apply (name or id); omit to list available")
		comment  = fs.String("comment", "", "comment to add during the transition")
		assignee = fs.String("assignee", "", "reassign during the transition (username)")
	)
	fs.Usage = func() {
		fmt.Fprintln(fs.Output(), "Usage: jira-cli transition --key PROJ-123 [--to \"Done\"] [--comment C]")
		fmt.Fprintln(fs.Output(), "  With no --to, lists the transitions available from the current status.")
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

	transitions, err := cl.GetTransitions(ctx, *key)
	if err != nil {
		return err
	}

	// No target -> list available transitions.
	if *to == "" {
		return emit(common.output, transitions, func() {
			if len(transitions) == 0 {
				fmt.Println("no transitions available")
				return
			}
			for _, t := range transitions {
				dest := ""
				if t.To != nil {
					dest = " -> " + t.To.Name
				}
				fmt.Printf("%-4s %s%s\n", t.ID, t.Name, dest)
			}
		})
	}

	target, err := jira.ResolveTransition(transitions, *to)
	if err != nil {
		return err
	}
	if err := cl.DoTransition(ctx, *key, target.ID, *comment, *assignee); err != nil {
		return err
	}

	result := map[string]string{"key": *key, "transition": target.Name, "status": "transitioned"}
	return emit(common.output, result, func() {
		fmt.Printf("transitioned %s via %q\n", *key, target.Name)
		if u := cl.WebURL(*key); u != "" {
			fmt.Println(u)
		}
	})
}
