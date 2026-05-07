<div align="center">

# 🍊 CGate

**The Bridge Between GitHub Issues and Autonomous AI Development**

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Go Report Card](https://goreportcard.com/badge/github.com/Lin-Jiong-HDU/cgate)](https://goreportcard.com/report/github.com/Lin-Jiong-HDU/cgate)
[![Docker](https://img.shields.io/badge/Docker-Ready-blue?logo=docker)](https://www.docker.com/)
[![Powered by Claude](https://img.shields.io/badge/Powered%20by-Claude%20Code-orange)](https://www.anthropic.com/)

*Just open a GitHub Issue — CGate handles the rest.*

</div>

---

CGate is a self-hosted automation gateway that connects GitHub Issues to [Claude Code](https://docs.anthropic.com/en/docs/claude-code). Create an issue titled `... [claude bot]`, and CGate spins up an isolated Docker container that runs Claude Code to implement the feature, run tests, and open a pull request — automatically.

## Why CGate?

Tired of the manual cycle of cloning a repo, running AI tools, and submitting PRs by hand? CGate automates the entire workflow:

- 🤖 **Automation** — Turn a GitHub Issue into a merged PR without touching your keyboard.
- 🔒 **Isolation** — Every task runs in its own ephemeral Docker container, keeping your host clean and secure.
- ⚡ **Efficiency** — Run up to N tasks in parallel, with automatic cleanup and full log access via REST API.
- 🔄 **Resilience** — Survives restarts; pending tasks are re-enqueued and running containers are re-attached automatically.

## How It Works

```
📝 Issue [... claude bot]
  → ⚙️  GitHub Actions sends webhook to CGate
    → 📋 CGate creates a Task and enqueues it
      → 🐳 Scheduler launches an isolated Docker container
        → 💻 Container clones repo and runs Claude Code
          → 🚀 Claude implements, tests, commits, and opens a PR
            → 🧹 Container and workspace are cleaned up automatically
```

## Features

- 📌 **Issue-driven automation** — open a GitHub Issue, get a PR
- 🐳 **Docker-isolated execution** — each task runs in its own container
- 🧹 **Auto-cleanup** — containers and workspace directories are removed after task completion
- ⚖️ **Concurrency control** — configurable max parallel tasks (default: 3)
- 🔁 **Task lifecycle** — pending → running → succeeded / failed / cancelled
- 🌐 **REST API** — list, inspect, cancel, and read logs for tasks
- 💾 **Persistence** — SQLite storage, survives restarts
- 🔄 **Recovery** — re-enqueues pending tasks and re-attaches to running containers on restart

## Quick Start

### Prerequisites

- [Docker](https://docs.docker.com/get-docker/) and [Docker Compose](https://docs.docker.com/compose/install/)
- GitHub Personal Access Token with `repo` scope
- Anthropic API key (or compatible endpoint)

### Step 1: Clone and Build Images

```bash
git clone https://github.com/Lin-Jiong-HDU/cgate.git
cd cgate
make docker-build-all
```

This builds two images:
- **`cgate`** — the server (from the project root `Dockerfile`)
- **`claude-code-runner`** — the ephemeral task runner (from `runner-image/Dockerfile`), which includes Go, Node.js, Claude Code CLI, and GitHub CLI

> **Note:** The runner image is large (~2 GB) because it includes the full Go toolchain, Node.js, and Claude Code. The first build takes several minutes.

### Step 2: Create Configuration

Copy the example config and adjust:

```bash
cp config.yaml.example config.yaml
```

```yaml
server:
  port: 8000          # Must match the port in docker-compose.yml

docker:
  image: "claude-code-runner:latest"
  max_concurrency: 3  # Max parallel runner containers
  timeout: "30m"      # Container execution timeout
  max_turns: 15       # Max Claude Code conversation turns
  permission_mode: "permissive"  # "permissive" or "strict"
  settings_path: "./settings.json"
  git_user_name: "cgate-bot"
  git_user_email: "cgate-bot@users.noreply.github.com"
  base_url: ""        # Optional: Anthropic API base URL (e.g. https://open.bigmodel.cn/api/anthropic)
  model: ""           # Optional: Model name (e.g. glm-5.1)

github:
  webhook_secret: "${GITHUB_WEBHOOK_SECRET}"  # From .env
  pat: "${GITHUB_PAT}"                         # From .env
  allowed_authors: []                          # Optional: restrict to specific GitHub usernames (e.g. ["user1", "user2"]); empty = allow all

queue:
  max_retries: 1      # Retry failed tasks once before giving up
```

> **Important:** Set `server.port` to `8000` to match the `docker-compose.yml` port mapping (`8000:8000`).

### Step 3: Create `.env`

Create a `.env` file in the project root (this file is gitignored):

```bash
# Required
GITHUB_WEBHOOK_SECRET=<your-webhook-secret>    # Arbitrary secret, shared with target repo's Actions secrets
GITHUB_PAT=<your-github-pat>                    # GitHub PAT with repo scope
ANTHROPIC_API_KEY=<your-api-key>                # Anthropic API key
CGATE_URL=http://your-server-ip:8000            # Public URL where this server is reachable

# Optional — for non-Anthropic API endpoints
ANTHROPIC_BASE_URL=
ANTHROPIC_MODEL=

# Optional — proxy settings (forwarded to runner containers)
HTTP_PROXY=
HTTPS_PROXY=

# Optional — git identity overrides
GIT_USER_NAME=cgate-bot
GIT_USER_EMAIL=cgate-bot@users.noreply.github.com
```

| Variable | Required | Description |
|----------|----------|-------------|
| `GITHUB_WEBHOOK_SECRET` | Yes | Secret for webhook authentication (shared with target repo) |
| `GITHUB_PAT` | Yes | GitHub PAT with `repo` scope, injected into runner containers |
| `ANTHROPIC_API_KEY` | Yes | Anthropic API key, injected into runner containers |
| `CGATE_URL` | Yes | Public URL of this CGate instance (used by runner containers) |
| `ANTHROPIC_BASE_URL` | No | Alternative API base URL (e.g. `https://open.bigmodel.cn/api/anthropic`) |
| `ANTHROPIC_MODEL` | No | Model name override (e.g. `glm-5.1`) |
| `HTTP_PROXY` | No | HTTP proxy, forwarded to runner containers |
| `HTTPS_PROXY` | No | HTTPS proxy, forwarded to runner containers |
| `GIT_USER_NAME` | No | Git commit author name (default: `cgate-bot`) |
| `GIT_USER_EMAIL` | No | Git commit author email (default: `cgate-bot@users.noreply.github.com`) |

### Step 4: Create `settings.json` (strict mode only)

If `permission_mode` is set to `"strict"`, create a `settings.json` in the project root defining which tools Claude Code is allowed to use:

```json
{
  "permissions": {
    "allow": [
      "Bash(go build:*)",
      "Bash(go test:*)",
      "Bash(go vet:*)",
      "Bash(git add:*)",
      "Bash(git commit:*)",
      "Bash(git push:*)",
      "Read",
      "Write"
    ],
    "deny": []
  }
}
```

If `permission_mode` is `"permissive"`, this file is not required — Claude Code runs with all permissions enabled.

### Step 5: Launch

```bash
docker compose up -d
```

Verify the server is running:

```bash
docker logs cgate-cgate-1
# Expected output:
#   INFO scheduler started max_concurrency=3
#   INFO server starting addr=:8000
```

Test the API:

```bash
curl http://localhost:8000/api/tasks
```

### Step 6: Set up GitHub Webhook

CGate uses a GitHub Actions workflow to forward issue events from your **target repository** (the repo you want Claude to work on) to the CGate server.

1. Copy `.github/workflows/issue-webhook.yml` from this repo into your target repository at `.github/workflows/issue-webhook.yml`.

2. Add the following **repository secrets** in your target repo (Settings → Secrets and variables → Actions):

   | Secret | Value |
   |--------|-------|
   | `WEBHOOK_URL` | Your CGate server URL (e.g. `http://your-server-ip:8000/webhook/github`) |
   | `WEBHOOK_SECRET` | The same value as `GITHUB_WEBHOOK_SECRET` in your `.env` |

3. Create an issue in the target repo with a title ending in `[claude bot]`:

   ```
   Add user authentication [claude bot]
   ```

   The issue body should contain the requirements and acceptance criteria. CGate will pick it up, run Claude Code in an isolated container, and open a pull request.

## API

| Method | Path | Description |
|--------|------|-------------|
| `POST` | `/webhook/github` | Receive GitHub issue webhook |
| `GET` | `/api/tasks` | List tasks (optional `?status=` filter) |
| `GET` | `/api/tasks/{id}` | Get task detail |
| `POST` | `/api/tasks/{id}/cancel` | Cancel a running task |
| `GET` | `/api/tasks/{id}/logs` | Get task container logs |

## Architecture

Clean Architecture with strict one-way dependency:

```
route → controller → usecase → repository → domain
```

```
├── cmd/main.go               # Entry point
├── domain/                   # Entities + interfaces (no project imports)
├── usecase/                  # Business logic (scheduler, webhook handling)
├── repository/               # SQLite persistence
├── api/
│   ├── controller/           # HTTP handlers
│   ├── middleware/           # Webhook authentication
│   └── route/                # Route registration + DI wiring
├── bootstrap/                # App initialization (config, DB, Docker client)
├── internal/
│   ├── docker/               # Docker container management
│   └── queue/                # Buffered channel-based task queue
├── runner-image/             # Runner Dockerfile + entrypoint + prompt template
├── config.yaml
├── docker-compose.yml
└── Dockerfile                # Server multi-stage build
```

## Development

```bash
make build              # Build binary
make run                # Run locally
make test               # Run tests with coverage
make lint               # Run golangci-lint
make docker-build-all   # Build both server and runner Docker images
make docker-up          # Start via docker compose
make docker-down        # Stop via docker compose
```

## License

MIT

---

<div align="center">

If CGate saves you time, please consider giving it a ⭐ — it helps others discover the project!

</div>
