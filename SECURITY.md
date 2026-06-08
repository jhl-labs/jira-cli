# Security Policy

## Reporting a vulnerability

Please **do not** open a public issue for security vulnerabilities.

Report privately via GitHub Security Advisories:
<https://github.com/jhl-labs/jira-cli/security/advisories/new>

Include reproduction steps and affected versions where possible. When sharing
logs or configuration, **redact tokens, passwords, and internal hostnames**.

## Supported versions

The latest released version receives security fixes. Older versions are not
maintained.

## Handling secrets

jira-cli authenticates with Jira using a Personal Access Token or Basic
credentials. To keep these safe:

- Provide credentials via environment variables (`JIRA_TOKEN`, `JIRA_USER` /
  `JIRA_PASSWORD`) or a config file outside version control — never hardcode them.
- Keep config files (e.g. `~/.config/jira-cli/config.json`) readable only by
  your user.
- Tokens are sent only to the configured `JIRA_BASE_URL` over HTTPS.
- Rotate any token that may have been exposed (logs, shell history, screen
  sharing) immediately.
