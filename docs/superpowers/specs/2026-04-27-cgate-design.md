# CGate Design Spec

CGate is a backend service that receives GitHub Issue webhooks and manages parallel Claude Code instances in Docker containers to automatically implement code changes.

## Requirements Summary

- **Deployment**: Docker in Docker (DinD), single server
- **Execution**: Parallel with configurable concurrency
- **Repos**: Multi-repository support
- **Auth**: Shared GitHub PAT
- **Storage**: SQLite for task persistence
- **Observability**: REST API for task management, no UI initially
- **Config**: Claude Code settings.json mounted into containers

## Architecture

Pattern: Event-driven with task queue.

```
GitHub Issue → [Webhook Handler]
                    ↓
              [In-memory Queue]
                    ↓
         [Scheduler/Dispatcher] ← maxConcurrency control
                    ↓
         [Docker Container Pool]
         ┌───────────────────────────┐
         │ Container: claude + git   │ → clone → implement → push → PR
         │ Container: claude + git   │
         │ Container: ...           │
         └───────────────────────────┘
                    ↓
              [SQLite Persistence]
```

## Project Structure

```
cgate/
├── cmd/main.go
├── domain/
│   ├── task.go
│   ├── task_test.go
│   ├── repository.go
│   ├── usecase.go
│   └── config.go
├── usecase/
│   ├── task_usecase.go
│   └── task_usecase_test.go
├── repository/
│   ├── sqlite.go
│   ├── sqlite_test.go
│   ├── task_repository.go
│   └── task_repository_test.go
├── api/
│   ├── controller/
│   │   ├── webhook_controller.go
│   │   ├── webhook_controller_test.go
│   │   ├── task_controller.go
│   │   └── task_controller_test.go
│   ├── middleware/
│   │   ├── auth.go
│   │   └── auth_test.go
│   └── route/
│       └── route.go
├── internal/
│   ├── docker/
│   │   ├── runner.go
│   │   └── runner_test.go
│   └── queue/
│       ├── queue.go
│       └── queue_test.go
├── mocks/
│   ├── mock_task_repository.go
│   ├── mock_task_usecase.go
│   ├── mock_docker_runner.go
│   └── ...
├── bootstrap/
│   └── bootstrap.go
├── config.yaml
├── Dockerfile
├── docker-compose.yml
├── go.mod
└── Makefile
```

Dependency direction (outer → inner): `route` → `controller` → `usecase` → `repository` → `domain`.

`internal/docker` and `internal/queue` are infrastructure, called by `usecase` through interfaces defined in `domain`.

## Data Model

### Task

```go
type TaskStatus string

const (
    TaskStatusPending   TaskStatus = "pending"
    TaskStatusRunning   TaskStatus = "running"
    TaskStatusSucceeded TaskStatus = "succeeded"
    TaskStatusFailed    TaskStatus = "failed"
)

type Task struct {
    ID           string
    IssueNumber  int
    Title        string
    Body         string
    Author       string
    Repository   string     // "owner/repo"
    HTMLURL      string
    Status       TaskStatus
    ContainerID  string
    Log          string
    CreatedAt    time.Time
    StartedAt    *time.Time
    FinishedAt   *time.Time
}
```

### Config

```go
type Config struct {
    Server   ServerConfig
    Docker   DockerConfig
    Queue    QueueConfig
    GitHub   GitHubConfig
}

type ServerConfig struct {
    Port int
}

type DockerConfig struct {
    Image          string
    MaxConcurrency int
    Timeout        time.Duration
    SettingsPath   string
}

type QueueConfig struct {
    MaxRetries int
}

type GitHubConfig struct {
    PAT           string
    WebhookSecret string
}
```

## API Endpoints

| Method | Path | Description |
|--------|------|-------------|
| POST | `/webhook/github` | Receive GitHub Issue webhook |
| GET | `/api/tasks` | List tasks (supports `?status=` filter) |
| GET | `/api/tasks/:id` | Get task detail + logs |
| POST | `/api/tasks/:id/cancel` | Cancel a running task |
| GET | `/api/tasks/:id/logs` | Get task execution logs |

HTTP routing uses Go 1.22+ `net/http` standard library (no third-party router).

## Core Flows

### Webhook Processing

1. GitHub Issue webhook → `POST /webhook/github`
2. Middleware validates `X-Webhook-Secret` header
3. Controller parses payload → constructs `Task` entity (status=pending)
4. Usecase enqueues task, persists to SQLite
5. Returns `202 Accepted` with task ID

### Scheduler (background goroutine)

1. Loop: check running containers < maxConcurrency?
2. Yes → dequeue pending task
3. Start Docker container:
   ```
   docker run -d \
     -e ANTHROPIC_API_KEY=xxx \
     -e GITHUB_TOKEN=xxx \
     -e CGATE_URL=xxx \
     -v /tmp/cgate/repos/{taskID}:/workspace \
     -v settings.json:/root/.claude/settings.json:ro \
     claude-code-runner:latest \
     claude --max-turns 15 \
       --prompt "处理 Issue #{N}: {title}\n{body}"
   ```
4. Update task: status=running, containerID=xxx
5. Start log collector goroutine (`docker logs -f`)
6. Start completion watcher goroutine: monitor container exit, update task status, release concurrency slot

### Container Lifecycle

- Each task gets an isolated Docker container
- Container image includes: Claude Code CLI, git, Go toolchain, golangci-lint
- Claude Code settings.json mounted read-only from host
- Container exit → task terminal status (succeeded if exit 0, failed otherwise)
- Timeout enforcement: `docker stop` after configurable duration (default 30m)

### Crash Recovery

On cgate restart:
1. Load pending tasks from SQLite → re-enqueue
2. Load running tasks → check container status via Docker API
3. If container still running → resume monitoring
4. If container exited → update task status accordingly

## Error Handling

| Scenario | Behavior |
|----------|----------|
| Invalid webhook secret | Return 401 |
| Duplicate issue (pending/running task exists) | Return 409 |
| Container start failure | Task status → failed, log error |
| Container timeout | Force `docker stop`, task status → failed |
| Queue empty | Scheduler blocks until new task arrives |

## Dependencies

| Package | Purpose |
|---------|---------|
| `github.com/docker/docker` | Docker Engine API for container management |
| `github.com/mattn/go-sqlite3` | SQLite driver (CGO required) |
| `github.com/spf13/viper` | Configuration management (yaml + env) |
| `github.com/stretchr/testify` | Test assertions |

HTTP routing: Go 1.22+ `net/http` stdlib.

## Configuration

```yaml
server:
  port: 8080

docker:
  image: "claude-code-runner:latest"
  max_concurrency: 3
  timeout: "30m"
  settings_path: "./settings.json"

github:
  webhook_secret: "${GITHUB_WEBHOOK_SECRET}"
  pat: "${GITHUB_PAT}"

queue:
  max_retries: 1
```

Environment variables: `GITHUB_WEBHOOK_SECRET`, `GITHUB_PAT`, `ANTHROPIC_API_KEY`, `CGATE_URL`.

## Deployment

```yaml
# docker-compose.yml
services:
  cgate:
    build: .
    ports:
      - "8080:8080"
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
      - ./settings.json:/app/settings.json:ro
      - ./data:/app/data
    environment:
      - GITHUB_WEBHOOK_SECRET=${GITHUB_WEBHOOK_SECRET}
      - GITHUB_PAT=${GITHUB_PAT}
      - ANTHROPIC_API_KEY=${ANTHROPIC_API_KEY}
      - CGATE_URL=${CGATE_URL}
```

## Testing Strategy

- **usecase layer**: TDD for scheduler logic, concurrency control, error paths
- **repository layer**: TDD for SQLite CRUD and state transitions
- **controller layer**: Coverage for happy path + error paths, using `httptest.NewRecorder`
- **internal/docker**: Core logic tests with mocked Docker API
- **internal/queue**: Queue enqueue/dequeue/concurrency tests
- **domain/entity**: Coverage-only for constructors
- **mocks**: Generated by mockery, stored in `mocks/`
