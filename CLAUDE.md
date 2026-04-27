# Project Instructions

This is a Go 1.24+ project using standard library only. Third-party dependencies require explicit approval before introduction.

## AI Workflow Protocol

Every task — feature, bug fix, or refactor — must follow this five-phase cycle. **Do not skip or reorder phases.**

### Phase 1: Understand

- Read the task description in full before touching any file.
- Use search tools (`grep`, `glob`, file reads) to identify every file that will need to change.
- Read the current code in each affected area — never modify code you have not read.
- Check whether tests already exist for the code you are about to change.

### Phase 2: Plan

- State, in plain text, which files will change and why.
- Identify risks and side-effects (e.g., interface changes, DB migrations, breaking API).
- For ambiguous requirements, choose the most conservative interpretation and record the assumption.
- **Do not write any code yet.**

### Phase 3: Implement

- Make the smallest change that fully satisfies the requirement.
- Follow project architecture and coding standards exactly (see sections below).
- Implement everything completely — no `// TODO: implement` placeholders.
- One logical unit of change per commit.

### Phase 4: Verify — Pipeline Gates

Run all gates in order. **Every gate must pass before proceeding to the next.** If a gate fails, fix the root cause and restart from Gate 1.

| # | Gate | Command | Pass condition |
|---|------|---------|----------------|
| 1 | Vet | `go vet ./...` | Zero output |
| 2 | Build | `go build ./...` | Zero output |
| 3 | Test | `go test ./...` | All tests pass |
| 4 | Lint | `golangci-lint run` | No new violations |

### Phase 5: Commit

- Use Conventional Commits (`type(scope): subject`).
- Document any assumptions and CVE-check results in the commit message when introducing dependencies.
- Never push directly to `main`.

## Debug Protocol

When a pipeline gate fails:

1. **Read** the full error output — do not skim.
2. **Identify** the root cause (compile error, test assertion failure, lint rule).
3. **Locate** the minimal code change needed to fix it.
4. **Fix** only what is necessary — do not refactor surrounding code.
5. **Re-run** from Gate 1 through the previously failing gate and all subsequent gates.
6. **Repeat** until all four gates pass.

### Self-Correction Rules

- **NEVER** comment out or delete a failing test — fix the implementation instead.
- **NEVER** add `//nolint` directives to silence lint errors — fix the code instead.
- **NEVER** use `_ = err` to discard errors you introduced.
- If a gate still fails after three focused fix attempts, stop and report the full error output with a clear explanation of what was tried.

## Project Types

This template supports two project types. Determine which one applies based on the current project structure.

### CLI / Library Project

Flat structure under project root:

```
├── main.go          # Entry point
├── internal/        # Internal packages (flat)
│   └── xxx.go
├── go.mod
└── Makefile
```

### Web Project (Clean Architecture)

Architecture layers, from outermost to innermost:

- **route** — composition root; registers HTTP routes and wires all layers together
- **controller** — HTTP delivery; parses requests, calls usecase interfaces, writes responses
- **usecase** — business logic; orchestrates domain operations through repository interfaces
- **repository** — data access; implements persistence using external driver packages
- **domain** — entities + all interfaces; imports nothing from this project

Folder structure (layers may sit at project root or under `internal/`; group HTTP-related code under `api/` when preferred):

```
├── cmd/
│   └── main.go              # Entry point
├── domain/                  # Entities, repository interfaces, usecase interfaces
│   ├── entity.go
│   ├── repository.go
│   └── usecase.go
├── usecase/                 # Business logic (implements domain.XxxUsecase)
│   └── xxx_usecase.go
├── repository/              # Data access (implements domain.XxxRepository)
│   └── xxx_repository.go
├── api/
│   ├── controller/          # HTTP handlers (calls domain usecase interfaces)
│   │   └── xxx_controller.go
│   ├── middleware/          # HTTP middleware
│   └── route/               # Route registration and dependency wiring
│       └── route.go
├── bootstrap/               # App initialization (config, DB setup)
├── internal/                # Shared utilities (tokenutil, etc.)
├── mocks/                   # Generated mocks (mockery)
├── go.mod
└── Makefile
```

#### Dependency Direction

Dependencies flow inward: `route` → `controller` → `usecase` → `repository` → `domain`.

- `domain` — defines entity types + all interfaces (XxxRepository, XxxUsecase); imports nothing from this project
- `usecase` — implements `domain.XxxUsecase`; imports `domain` only; applies `context.WithTimeout`
- `repository` — implements `domain.XxxRepository`; imports `domain` + external driver packages
- `controller` — imports `domain` interfaces only; never imports `usecase` package directly
- `route` — imports all layers; instantiates concrete types and wires them through `domain` interfaces

#### Key Conventions

1. **Interfaces in domain, implementations in outer layers** — dependency inversion
2. **Unexported concrete types, exported constructors** — `type xxxUsecase struct` (unexported), `func NewXxxUsecase() domain.XxxUsecase` (exported)
3. **Manual dependency injection** — no DI framework, assembly in route layer
4. **Context timeout in usecases** — `context.WithTimeout` set in usecase layer
5. **Mocks generated by tooling** — use mockery, stored in `mocks/`
6. **Black-box testing** — `package xxx_test`, test only public API

## Go Coding Standards

### Naming

- Package names: lowercase single word, no underscores or camelCase (`usecase` not `useCase`)
- Interfaces: `-er` suffix for behavior (`Repository`, `Usecase` are exceptions per project convention)
- Exported functions: verb prefix (`NewUserUsecase`, `FetchByUserID`)
- Constants: camelCase, not ALL_CAPS
- Error variables: `Err` prefix (`ErrUserNotFound`)

### Error Handling

- Never ignore `error` return values
- Propagate errors without wrapping — keep original messages
- Controller layer maps errors to HTTP status codes:
  - 400: request binding / validation failure
  - 401: authentication failure
  - 404: resource not found
  - 409: conflict (e.g. duplicate creation)
  - 500: internal error
- Startup fatal errors use `log.Fatal()`

### Context

- Every DB-related function accepts `context.Context` as first parameter
- Usecase layer uses `context.WithTimeout` uniformly
- Never store context in structs

### Concurrency

- Prefer channels over shared memory
- Goroutines must have an exit mechanism
- No unmanaged goroutines

### Logging

- Use `log/slog` for structured logging (stdlib, Go 1.21+)
- Never use `fmt.Println` for logging

## Testing

### Organization

- Test files co-located with source (`xxx_test.go` next to `xxx.go`)
- Use `package xxx_test` (black-box testing)

### Conventions

- Prefer table-driven tests
- Test naming: `TestXxx_Scenario_ExpectedResult`
- Use `t.Parallel()` for parallelizable tests
- Use `t.Run()` for sub-tests

### Mocks

- Use `mockery` to generate — never hand-write
- Mock files in `mocks/` with `// Code generated by mockery. DO NOT EDIT.` header
- Only mock external deps and cross-layer interfaces
- Controller tests: `httptest.NewRecorder` + real route engine

### Coverage

- Usecase and Repository layers require coverage
- Controller: at least happy path + major error paths
- Core business logic must have tests

### TDD Strategy

Apply Test-Driven Development selectively based on code complexity:

**Use TDD (Red → Green → Refactor) for:**

- Business logic in the `usecase` layer (non-trivial orchestration, branching, error paths)
- Domain validation rules embedded in entity constructors or value objects
- Any function whose correct behaviour is hard to verify by inspection alone

TDD cycle for these cases:
1. Write a failing test that captures the requirement (`go test` must fail — Red)
2. Write the minimal implementation to make it pass (Green)
3. Refactor without changing observable behaviour; keep tests green (Refactor)

**Coverage-only (write test alongside or after implementation) for:**

- Simple helper / utility functions with a single, obvious code path
- Pure data transformations with no branching
- Boilerplate constructors (`NewXxx`) that only set fields
- Infrastructure wiring in `route` (tested by integration / e2e)

**Decision rule:** if you can state the expected behaviour in a table of ≤3 rows, coverage-only is sufficient. If the logic involves conditionals, loops, or error-path branching, apply TDD.

## Git Workflow

### Branches

- `main`: stable, protected, no direct push
- `feat/xxx`: new features
- `fix/xxx`: bug fixes
- `refactor/xxx`: refactoring
- `chore/xxx`: misc

### Commit Messages

Conventional Commits:
```
type(scope): subject
```
- type: `feat`, `fix`, `refactor`, `test`, `docs`, `chore`, `build`, `ci`
- scope: affected module (optional)
- subject: imperative, max 72 chars, no period

## Issue-Driven Automation

When a server-side Claude Code instance receives an issue webhook request, follow this procedure:

### Trigger Convention

- Only issues with titles ending in `[claude bot]` are forwarded to the server webhook.
- The issue title (excluding the `[claude bot]` suffix) is the task description.
- The issue body contains detailed requirements, acceptance criteria, or additional context.

### Execution Procedure

1. **Pull latest code** — `git pull origin main` to ensure working on the latest state.
2. **Create feature branch** — branch name follows `feat/issue-{number}-{short-description}` or `fix/issue-{number}-{short-description}` based on task type.
3. **Execute the task** — follow the five-phase AI Workflow Protocol (Understand → Plan → Implement → Verify → Commit).
4. **Run all pipeline gates** — `go vet` → `go build` → `go test` → `golangci-lint run` must all pass.
5. **Push branch and create PR** — push to remote and open a pull request targeting `main`:
   - PR title: copy from issue title, remove the `[claude bot]` suffix.
   - PR body: reference the original issue with `Closes #{issue_number}`, summarize changes.
6. **Comment on the issue** — leave a comment with the PR link confirming the task is done.

### Failure Handling

- If pipeline gates fail after three fix attempts, comment on the issue with the full error output and a clear explanation. Do not force-push or skip gates.
- If the task description is ambiguous, apply the most conservative interpretation and document assumptions in the PR description.

## Hard Boundaries

### Architecture

- **NEVER** import other project packages in `domain`
- **NEVER** let `usecase` depend on `repository` concrete types — use `domain` interfaces
- **NEVER** let `controller` import the `usecase` package directly — call use cases only through `domain` interfaces
- **NEVER** break dependency direction (outer → inner only)
- **NEVER** put business logic implementations in `domain`

### Code Generation

- **NEVER** generate empty placeholder implementations (`// TODO: implement`)
- **NEVER** generate code with hardcoded secrets or passwords
- **NEVER** generate code without error handling
- **NEVER** generate `init()` functions without very good reason

### Dependencies

- **NEVER** introduce third-party dependencies without documenting the purpose and checking for known CVEs in the commit message
- **NEVER** replace existing chosen dependencies
- **NEVER** change the Go version

### Structure

- **NEVER** modify the module path in `go.mod`
- **NEVER** create stray files in project root
- **NEVER** rename or move existing files unless explicitly asked

### Common Mistakes to Avoid

- Do not use `panic` for expected errors — return `error`
- Do not use `time.Sleep` for test synchronization — use channels or sync primitives
- Do not write `_ = err` to ignore errors
- Do not use `defer` in loops
- Do not use `interface{}` — use generics (Go 1.24+)
- Do not generate redundant type conversions
- Do not omit `json` struct tags on API response structs

### Behavior

- Read existing code in the affected area before modifying — match established patterns
- After every change: run `go vet ./...` and `go build ./...`; fix all reported errors before proceeding
- After completing a full change set: run `go test ./...` to confirm no regressions
- Follow existing project style — do not introduce new patterns or mix styles
- When requirements are ambiguous, apply the most conservative interpretation; document assumptions in the commit message
- Make the smallest change that satisfies the requirement — do not refactor unrelated code
- When a build or test fails, diagnose the root cause from the error output, fix it, and re-verify; never skip or comment out failing tests

## Recommended Libraries (not pre-installed)

| Scenario | Library | Notes |
|----------|---------|-------|
| HTTP routing | `net/http` (stdlib), `chi` | Go 1.22+ stdlib has route params |
| Web framework | `gin`, `echo` | Rich middleware ecosystem |
| Configuration | `github.com/spf13/viper` | .env / yaml / json |
| CLI | `github.com/spf13/cobra` | Complex CLI tools |
| Database ORM | `gorm`, `ent` | Per project need |
| Database Driver | `go-sql-driver/mysql`, `lib/pq`, `mongo-driver` | Per database |
| Logging | `log/slog` (stdlib) | Preferred, built-in since Go 1.21 |
| JWT | `github.com/golang-jwt/jwt/v5` | Authentication |
| Testing | `github.com/stretchr/testify` | Assertions + mock |
| Mock generation | `github.com/vektra/mockery/v2` | Interface mocks |
| Password hashing | `golang.org/x/crypto/bcrypt` | User authentication |
| Linting | `golangci-lint` | Code quality |
