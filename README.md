# CGate

Self-hosted automation gateway that connects GitHub issues to [Claude Code](https://docs.anthropic.com/en/docs/claude-code). Create an issue titled `... [claude bot]`, and CGate spins up an isolated Docker container that runs Claude Code to implement the feature, run tests, and open a pull request — automatically.

## How It Works

```
GitHub Issue [... claude bot]
  → GitHub Actions sends webhook to CGate
    → CGate creates a Task, enqueues it
      → Scheduler launches an isolated Docker container
        → Container clones repo, runs Claude Code
          → Claude implements, tests, commits, opens a PR
```

## Features

- **Issue-driven automation** — open a GitHub issue, get a PR
- **Docker-isolated execution** — each task runs in its own container
- **Concurrency control** — configurable max parallel tasks (default: 3)
- **Task lifecycle** — pending → running → succeeded / failed / cancelled
- **REST API** — list, inspect, cancel, and read logs for tasks
- **Persistence** — SQLite storage, survives restarts
- **Recovery** — re-enqueues pending tasks and re-attaches to running containers on restart

## Quick Start

### Prerequisites

- Go 1.24+
- Docker
- GitHub PAT with repo permissions
- Anthropic API key

### 1. Build

```bash
make docker-build-all
```

This builds both the server image and the runner image (`claude-code-runner`).

### 2. Configure

Copy `config.yaml` and adjust as needed:

```yaml
server:
  port: 8080

docker:
  runner_image: claude-code-runner:latest
  max_concurrency: 3

database:
  path: ./data/cgate.db
```

### 3. Run

```bash
docker compose up -d
```

Set the following environment variables (via `docker-compose.yml` or `.env`):

| Variable | Description |
|----------|-------------|
| `GITHUB_WEBHOOK_SECRET` | Secret for webhook authentication |
| `GITHUB_PAT` | GitHub personal access token (injected into runner containers) |
| `ANTHROPIC_API_KEY` | Anthropic API key (injected into runner containers) |

### 4. Set up the GitHub webhook

In your target repository, configure a GitHub Actions workflow that POSTs issue data to CGate. See `.github/workflows/issue-webhook.yml` for a ready-made workflow.

Alternatively, use the manual trigger workflow (`.github/workflows/claude.yml`) from the Actions tab.

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
make docker-test        # Run Docker image integration tests
```

## License

MIT
