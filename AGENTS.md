# AGENTS.md

## Project Overview

`mf-cli` is a Go CLI tool built with Cobra that wraps docker-compose for project management.

## Build & Run

```bash
# Build
go build -o mf .

# Run
./mf --help
./mf init --file path/to/docker-compose.yml
```

## Test

```bash
# Run all tests
go test ./...

# Run with verbose output
go test -v ./...

# Run specific package tests
go test -v ./internal/compose/
go test -v ./internal/config/
```

## Project Structure

- `main.go` — entry point
- `cmd/` — Cobra commands (one file per command/group)
- `internal/config/` — mf.yaml config struct, loader, writer
- `internal/compose/` — docker-compose wrapper and compose file parser
- `internal/runner/` — generic shell command runner

## Key Files

- `internal/compose/parser.go` — Compose file parser + service classifier with pluggable `DefaultMatchers` registry
- `internal/config/config.go` — Config struct matching mf.yaml shape
- `cmd/root.go` — Root command with config loading (skipped for commands with `skipConfig` annotation)
- `cmd/init.go` — `mf init` command that scans compose files

## Adding New Commands

1. Create a new file in `cmd/` (e.g. `cmd/mycommand.go`)
2. Define a `cobra.Command` with `Use`, `Short`, and `RunE`
3. In `init()`, add it to the parent command with `rootCmd.AddCommand()`
4. Commands that don't need mf.yaml should add `Annotations: map[string]string{"skipConfig": "true"}`

## Adding New Image Matchers

Add entries to `DefaultMatchers` in `internal/compose/parser.go`. Each matcher has:
- `Patterns` — image name prefixes to match
- `ServiceType` — type identifier (e.g. "postgres", "redis", "flower")
- `EnvMappings` — optional env var → config field mappings
