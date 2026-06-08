package main

import (
	"flag"
	"fmt"
	"os"
	"sort"
	"strings"
)

// skillFlavors are the supported agent platforms. "" and "generic" both map to
// the universal, frontmatter-less skill.
var skillFlavors = map[string]bool{
	"":         true,
	"generic":  true,
	"claude":   true,
	"codex":    true,
	"gemini":   true,
	"opencode": true,
}

func runGenerateSkill(args []string) error {
	var flavor string
	rest := args
	if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		flavor, rest = args[0], args[1:]
	}

	fs := flag.NewFlagSet("generate-skill", flag.ExitOnError)
	var (
		out   = fs.String("out", "jira-skill.md", "output file path")
		toOut = fs.Bool("stdout", false, "write to stdout instead of a file")
		force = fs.Bool("force", false, "overwrite the output file if it exists")
	)
	fs.Usage = func() {
		fmt.Fprintln(fs.Output(), "Usage: jira-cli generate-skill [flavor] [flags]")
		fmt.Fprintf(fs.Output(), "\nFlavors: %s\n", strings.Join(flavorNames(), ", "))
		fmt.Fprintln(fs.Output(), "  (no flavor = generic / universal skill)")
		fs.PrintDefaults()
	}
	if err := fs.Parse(rest); err != nil {
		return err
	}
	if flavor == "" && fs.NArg() > 0 {
		flavor = fs.Arg(0)
	}
	if !skillFlavors[flavor] {
		return fmt.Errorf("unknown flavor %q (supported: %s)", flavor, strings.Join(flavorNames(), ", "))
	}

	content := buildSkill(flavor)

	if *toOut {
		_, err := os.Stdout.WriteString(content)
		return err
	}
	if !*force {
		if _, err := os.Stat(*out); err == nil {
			return fmt.Errorf("%s already exists (use --force to overwrite or --stdout to print)", *out)
		}
	}
	if err := os.WriteFile(*out, []byte(content), 0o644); err != nil {
		return err
	}
	label := flavor
	if label == "" {
		label = "generic"
	}
	fmt.Printf("wrote %s (%s flavor, %d bytes)\n", *out, label, len(content))
	return nil
}

func flavorNames() []string {
	names := make([]string, 0, len(skillFlavors))
	for k := range skillFlavors {
		if k == "" || k == "generic" {
			continue
		}
		names = append(names, k)
	}
	sort.Strings(names)
	return names
}

func buildSkill(flavor string) string {
	var preamble string
	switch flavor {
	case "", "generic":
		preamble = genericPreamble
	case "claude":
		preamble = claudePreamble
	case "codex":
		preamble = codexPreamble
	case "gemini":
		preamble = geminiPreamble
	case "opencode":
		preamble = opencodePreamble
	}
	return preamble + skillReference
}

const genericPreamble = `# Jira CLI Skill

> Universal skill describing how an AI agent should use the ` + "`jira-cli`" + ` tool.
> Place or import this file wherever your agent reads project instructions.

`

const claudePreamble = `---
name: jira-cli
description: Work with issues on a self-hosted Jira (Server/Data Center) instance via the jira-cli command-line tool. Use when the user wants to search, read, create, update, comment on, transition, assign, or delete Jira issues.
---

# Jira CLI Skill

> **Install (Claude):** rename this file to ` + "`SKILL.md`" + ` and place it under a folder
> matching the skill name, e.g. ` + "`.claude/skills/jira-cli/SKILL.md`" + `
> (the folder name must match the ` + "`name`" + ` field above).

`

const codexPreamble = `# Jira CLI — Agent Instructions

> **Install (Codex):** append this content to your project ` + "`AGENTS.md`" + ` (or
> ` + "`~/.codex/AGENTS.md`" + ` for global use). Codex reads AGENTS.md as standard Markdown.

`

const geminiPreamble = `# Jira CLI — Context

> **Install (Gemini CLI):** add this content to ` + "`GEMINI.md`" + ` at your project root
> (or ` + "`~/.gemini/GEMINI.md`" + ` for global), or import it with ` + "`@jira-skill.md`" + `.

`

const opencodePreamble = `# Jira CLI — Agent Rules

> **Install (opencode):** opencode reads ` + "`AGENTS.md`" + `; append this content there,
> or reference this ` + "`jira-skill.md`" + ` from it.

`

const skillReference = `## What this is

` + "`jira-cli`" + ` is a small, dependency-free CLI for working with issues on a
**self-hosted Jira Server/Data Center** instance. Prefer it over calling the REST
API directly: it handles authentication, retries on transient errors, JQL search,
and workflow transitions.

## Setup (authentication)

Set these environment variables before invoking the CLI:

| Variable | Required | Purpose |
|---|---|---|
| ` + "`JIRA_BASE_URL`" + ` | yes | Site base URL, e.g. ` + "`https://jira.example.com`" + ` |
| ` + "`JIRA_TOKEN`" + ` | yes* | Personal Access Token (Bearer). Preferred. |
| ` + "`JIRA_USER`" + ` / ` + "`JIRA_PASSWORD`" + ` | yes* | Basic-auth fallback when no token |
| ` + "`JIRA_PROJECT`" + ` | no | Default project key for ` + "`search`" + `/` + "`create`" + ` (lets you omit ` + "`--project`" + `) |

\* Provide **either** a token **or** user + password.

## Commands

Every command prints **JSON by default** (ideal for parsing); add ` + "`--output text`" + ` for a
human-readable summary.

### search — find issues with JQL
` + "```bash" + `
jira-cli search --project PROJ --status "In Progress" --max 20
jira-cli search --jql 'assignee = currentUser() AND resolution = Unresolved ORDER BY updated DESC'
` + "```" + `

### get — read an issue
` + "```bash" + `
jira-cli get --key PROJ-123 --output text
jira-cli get --key PROJ-123 --description   # print only the description
` + "```" + `

### create — new issue
` + "```bash" + `
jira-cli create --project PROJ --summary "Fix login bug" --type Bug --priority High
echo "Steps to reproduce..." | jira-cli create --summary "From stdin" --description-file -
` + "```" + `

### update — edit fields
` + "```bash" + `
jira-cli update --key PROJ-123 --summary "New title" --priority Low
` + "```" + `

### transition — move through the workflow
` + "```bash" + `
jira-cli transition --key PROJ-123                 # list available transitions
jira-cli transition --key PROJ-123 --to "Done" --comment "Shipped in 1.2.0"
` + "```" + `

### assign — set or clear the assignee
` + "```bash" + `
jira-cli assign --key PROJ-123 --user alice
jira-cli assign --key PROJ-123 --unassign
` + "```" + `

### comment / labels / delete / projects
` + "```bash" + `
jira-cli comment --key PROJ-123 --body "Looking into this."
jira-cli labels --key PROJ-123 --add "backend,urgent" --remove "triage"
jira-cli delete --key PROJ-123 --yes
jira-cli projects
` + "```" + `

## Field formats (important)

On Server/Data Center, the issue **description and comment bodies are plain text /
wiki markup strings** — not Markdown and not Cloud's ADF JSON. Send plain strings.

## Usage guidance for agents

- Use ` + "`--output json`" + ` when you need to parse results; ` + "`--output text`" + ` for quick checks.
- For ` + "`transition`" + `, list transitions first — the valid set depends on the issue's
  current status and the project's workflow.
- ` + "`search`" + ` accepts raw JQL via ` + "`--jql`" + ` for anything the simple flags can't express.
- Returned issues include a ` + "`key`" + `; surface the issue URL (` + "`<base>/browse/<KEY>`" + `) to the user.
- Never hardcode credentials; rely on the environment variables above.
- Confirm destructive or outward-facing writes (create/update/transition/delete on
  shared projects) with the user first.
`
