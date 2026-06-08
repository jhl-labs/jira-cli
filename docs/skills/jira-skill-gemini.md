# Jira CLI — Context

> **Install (Gemini CLI):** add this content to `GEMINI.md` at your project root
> (or `~/.gemini/GEMINI.md` for global), or import it with `@jira-skill.md`.

## What this is

`jira-cli` is a small, dependency-free CLI for working with issues on a
**self-hosted Jira Server/Data Center** instance. Prefer it over calling the REST
API directly: it handles authentication, retries on transient errors, JQL search,
and workflow transitions.

## Setup (authentication)

Set these environment variables before invoking the CLI:

| Variable | Required | Purpose |
|---|---|---|
| `JIRA_BASE_URL` | yes | Site base URL, e.g. `https://jira.example.com` |
| `JIRA_TOKEN` | yes* | Personal Access Token (Bearer). Preferred. |
| `JIRA_USER` / `JIRA_PASSWORD` | yes* | Basic-auth fallback when no token |
| `JIRA_PROJECT` | no | Default project key for `search`/`create` (lets you omit `--project`) |

\* Provide **either** a token **or** user + password.

## Commands

Every command prints **JSON by default** (ideal for parsing); add `--output text` for a
human-readable summary.

### search — find issues with JQL
```bash
jira-cli search --project PROJ --status "In Progress" --max 20
jira-cli search --jql 'assignee = currentUser() AND resolution = Unresolved ORDER BY updated DESC'
```

### get — read an issue
```bash
jira-cli get --key PROJ-123 --output text
jira-cli get --key PROJ-123 --description   # print only the description
```

### create — new issue
```bash
jira-cli create --project PROJ --summary "Fix login bug" --type Bug --priority High
echo "Steps to reproduce..." | jira-cli create --summary "From stdin" --description-file -
```

### update — edit fields
```bash
jira-cli update --key PROJ-123 --summary "New title" --priority Low
```

### transition — move through the workflow
```bash
jira-cli transition --key PROJ-123                 # list available transitions
jira-cli transition --key PROJ-123 --to "Done" --comment "Shipped in 1.2.0"
```

### assign — set or clear the assignee
```bash
jira-cli assign --key PROJ-123 --assignee alice
jira-cli assign --key PROJ-123 --unassign
```

### comment / labels / delete / projects
```bash
jira-cli comment --key PROJ-123 --body "Looking into this."
jira-cli labels --key PROJ-123 --add "backend,urgent" --remove "triage"
jira-cli delete --key PROJ-123 --yes
jira-cli projects
```

## Field formats (important)

On Server/Data Center, the issue **description and comment bodies are plain text /
wiki markup strings** — not Markdown and not Cloud's ADF JSON. Send plain strings.

## Usage guidance for agents

- Use `--output json` when you need to parse results; `--output text` for quick checks.
- For `transition`, list transitions first — the valid set depends on the issue's
  current status and the project's workflow.
- `search` accepts raw JQL via `--jql` for anything the simple flags can't express.
- Returned issues include a `key`; surface the issue URL (`<base>/browse/<KEY>`) to the user.
- Never hardcode credentials; rely on the environment variables above.
- Confirm destructive or outward-facing writes (create/update/transition/delete on
  shared projects) with the user first.
