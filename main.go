// Command jira-cli is a small, dependency-free CLI for working with issues on a
// self-hosted Jira Server/Data Center instance.
package main

import (
	"fmt"
	"os"
)

// version is overridable at build time:
//
//	go build -ldflags "-X main.version=1.2.3"
var version = "0.1.0-dev"

func main() {
	if len(os.Args) < 2 {
		usage(os.Stderr)
		os.Exit(2)
	}

	cmd, args := os.Args[1], os.Args[2:]
	var err error
	switch cmd {
	case "search":
		err = runSearch(args)
	case "get":
		err = runGet(args)
	case "create":
		err = runCreate(args)
	case "update":
		err = runUpdate(args)
	case "comment":
		err = runComment(args)
	case "transition":
		err = runTransition(args)
	case "assign":
		err = runAssign(args)
	case "labels":
		err = runLabels(args)
	case "delete":
		err = runDelete(args)
	case "projects":
		err = runProjects(args)
	case "generate-skill":
		err = runGenerateSkill(args)
	case "version", "-v", "--version":
		fmt.Printf("jira-cli %s\n", version)
	case "help", "-h", "--help":
		usage(os.Stdout)
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n\n", cmd)
		usage(os.Stderr)
		os.Exit(2)
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func usage(w *os.File) {
	fmt.Fprintf(w, `jira-cli %s — CLI for self-hosted Jira (Server/Data Center)

Usage:
  jira-cli <command> [flags]

Commands:
  search      Search issues with JQL
  get         Fetch an issue by key
  create      Create a new issue
  update      Update issue fields (summary, description, priority, ...)
  comment     Add a comment to an issue
  transition  List or apply workflow transitions (e.g. To Do -> Done)
  assign      Assign or unassign an issue
  labels      Add or remove labels on an issue
  delete      Delete an issue (asks for confirmation)
  projects    List projects
  generate-skill  Write a jira-skill.md for an AI agent
                  (flavors: claude, codex, gemini, opencode; none = generic)
  version     Print version
  help        Show this help

Run "jira-cli <command> -h" for command-specific flags.

Authentication (Server/Data Center):
  Personal Access Token (preferred):  --token / JIRA_TOKEN
  Basic auth:                          --user + --password / JIRA_USER + JIRA_PASSWORD
  Site base URL:                       --base-url / JIRA_BASE_URL
  Default project (search/create):     --project / JIRA_PROJECT
`, version)
}
