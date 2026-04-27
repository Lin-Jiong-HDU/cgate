# Go Project Template

A Go 1.24+ project template with built-in AI coding assistant instructions, CI/CD pipelines, and automated issue processing. Start a new project with conventions, tooling, and AI support already wired up.

## Features

- **Zero dependencies** — pure Go standard library skeleton
- **AI-ready** — instruction files for Claude Code, GitHub Copilot, and Cursor
- **Two project modes** — CLI/Library (flat) or Web (Clean Architecture)
- **CI/CD pipeline** — automated vet, build, test, and lint on every push
- **Issue-driven automation** — create an issue to trigger a remote AI agent
- **Makefile** — common commands out of the box

## Issue-Driven Automation

This template's core feature: **create a GitHub issue, and a remote AI agent implements it for you.**

### How it works

```
GitHub Issue  ──[claude bot]──►  GitHub Actions  ──webhook──►  Your Server (Claude Code)
                                                                        │
                                                                        ▼
                                                                 Pull → Branch → Implement → Test → PR
```

1. Create an issue with title ending in `[claude bot]`, e.g.: `Add user login API [claude bot]`
2. GitHub Actions sends the issue content to your server webhook
3. Your server-side Claude Code receives it and:
   - Pulls the latest code from `main`
   - Creates a feature branch (`feat/issue-{number}-{desc}`)
   - Implements the task following the five-phase workflow
   - Runs all pipeline gates (vet, build, test, lint)
   - Pushes the branch and opens a pull request
   - Comments on the issue with the PR link

### Setup

1. **Deploy Claude Code on your server** with a webhook endpoint that accepts POST requests
2. **Add repository secrets** in GitHub (Settings → Secrets and variables → Actions):
   - `WEBHOOK_URL` — your server endpoint (e.g. `https://your-server.com/webhook/issue`)
   - `WEBHOOK_SECRET` — shared secret for request verification
   - `ANTHROPIC_API_KEY` — your Anthropic API key
3. **Create an issue** with `[claude bot]` at the end of the title — that's it

### Webhook payload

Your server receives a JSON POST like this:

```json
{
  "action": "opened",
  "issue_number": 42,
  "title": "Add user login API [claude bot]",
  "body": "Use JWT for authentication...",
  "author": "username",
  "labels": ["enhancement"],
  "url": "https://github.com/owner/repo/issues/42",
  "repository": "owner/repo",
  "created_at": "2026-04-27T12:00:00Z"
}
```

Verify the `X-Webhook-Secret` header matches your configured secret before processing.

### Claude Code Action (manual trigger)

`.github/workflows/claude.yml` provides an on-demand Claude Code run. Go to **Actions → Claude Code → Run workflow** and optionally pass an issue number or custom prompt.

## Quick Start

### 1. Create your project

Click **"Use this template"** on GitHub, or:

```bash
git clone https://github.com/Lin-Jiong-HDU/go-project-template.git my-project
cd my-project
rm -rf .git
git init
```

### 2. Update module path

Edit `go.mod` and replace the module path:

```
module github.com/YOUR_USERNAME/YOUR_PROJECT
```

### 3. Choose project type

**CLI / Library** — keep the flat structure as-is:

```
├── main.go
├── internal/
├── go.mod
└── Makefile
```

**Web (Clean Architecture)** — restructure into layers:

```
├── cmd/main.go               # Entry point
├── domain/                   # Entities + interfaces (imports nothing)
├── usecase/                  # Business logic
├── repository/               # Data access
├── api/controller/           # HTTP handlers
├── api/middleware/            # HTTP middleware
├── api/route/                # Route registration + DI wiring
├── bootstrap/                # App initialization
├── internal/                 # Shared utilities
├── mocks/                    # Generated mocks
├── go.mod
└── Makefile
```

Dependency direction is strictly enforced: `route → controller → usecase → repository → domain`.

### 4. Start coding

```bash
make build     # Build the binary
make test      # Run tests with coverage
make lint      # Run linter (requires golangci-lint)
make run       # Run the application
```

## AI Assistant Support

The template ships with instruction files so AI tools understand your project conventions immediately:

| File | Tool | Purpose |
|------|------|---------|
| `CLAUDE.md` | Claude Code | Full project rules (source of truth) |
| `AGENTS.md` | GitHub Copilot | Same rules in Copilot format |
| `.cursorrules` | Cursor | Same rules in Cursor format |
| `.github/copilot-instructions.md` | Copilot Cloud Agent | Condensed operational summary |

### What the AI rules cover

- **Five-phase workflow** — Understand → Plan → Implement → Verify → Commit
- **Pipeline gates** — vet, build, test, lint must all pass before committing
- **Architecture rules** — dependency direction, interface segregation, DI patterns
- **Go coding standards** — naming, error handling, context, concurrency, logging
- **Testing strategy** — table-driven tests, mockery mocks, TDD for complex logic
- **Hard boundaries** — explicit "NEVER" rules that prevent common AI mistakes

## CI/CD Pipeline

`.github/workflows/ci.yml` runs on every push and pull request to `main`:

| Stage | Command |
|-------|---------|
| Tidy check | `go mod tidy` + diff check |
| Vet | `go vet ./...` |
| Build | `go build ./...` |
| Test | `go test -race -coverprofile=coverage.out ./...` |
| Lint | `golangci-lint run` |

## Customization

- Edit `CLAUDE.md` to adjust rules, then sync to `AGENTS.md`, `.cursorrules`, and `.github/copilot-instructions.md`
- Add your preferred libraries from the recommended list in the AI instructions
- Extend the architecture sections to match your domain patterns
- Adjust CI pipeline gates to add project-specific checks

## License

MIT
