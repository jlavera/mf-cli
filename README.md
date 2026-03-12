# mf — a CLI for docker-compose projects

`mf` is a standalone CLI tool that replaces project Makefiles for docker-compose based projects. It provides organized commands with proper help, flags, and shell completions — all driven by a simple `mf.yaml` config file.

## Requirements

This project requires **Go 1.26.1**. Use [mise](https://mise.jdx.dev/) to manage Go versions:

```bash
# Install mise (if you don't have it)
curl https://mise.run | sh

# Install the right Go version (reads .go-version or go.mod)
mise install go@1.26.1
mise use go@1.26.1
```

## Requirements

This project requires **Go 1.26.1**. Use [mise](https://mise.jdx.dev/) to manage Go versions:

```bash
# Install mise (if you don't have it)
curl https://mise.run | sh

# Install the right Go version (reads .go-version or go.mod)
mise install go@1.26.1
mise use go@1.26.1
```

## Install

```bash
# Build and install (overwrites existing install)
make install

# Or install directly from source
go install github.com/jlavera/mf-cli@latest
```

## Quick Start

```bash
# In your project directory (where docker-compose.yml lives)
mf init

# This scans your compose file and generates mf.yaml
# Then you can use:
mf up                  # start all services
mf logs                # follow logs
mf shell               # bash into backend container
mf test                # run tests
mf down                # stop everything
```

## `mf init`

Scans your docker-compose file and auto-generates `mf.yaml`. Detects:

- **Databases** — Postgres, MySQL, MongoDB (extracts DB name, user from env vars)
- **Caches** — Redis, Memcached, Valkey
- **Workers** — Celery workers/beat/flower (from command or service name)
- **Proxies** — Nginx, Traefik, Caddy
- **Search** — Elasticsearch, OpenSearch
- **Storage** — MinIO, LocalStack
- **Apps** — Backend/frontend classification from ports and build context

```bash
mf init                          # auto-detect compose file in CWD
mf init --file path/to/compose.yml   # specify compose file
mf init --force                  # overwrite existing mf.yaml
```

## Commands

### Docker Lifecycle
```bash
mf up [services...]              # start containers (detached)
mf stop [services...]            # stop without removing
mf build [services...] [--no-cache]  # build images
mf down                          # stop and remove
mf logs [services...]            # follow logs
mf restart [services...]         # full restart (down + up)
mf clean                         # remove containers + volumes
mf rebuild [services...]         # down + build --no-cache + up
```

### Shell Access
```bash
mf shell [service]               # bash into container (default: backend)
mf psql                          # database shell (postgres/mysql/mongo)
mf redis-cli                     # redis-cli
```

### Testing
```bash
mf test [apps...]                # run all or specific app tests
mf test -f path/to/test.py       # specific test file
mf test -m "TestClass.method"    # specific test method
mf test --debug                  # run with debugpy (attach from VS Code)
```

### Celery
```bash
mf celery start                  # start workers
mf celery stop                   # stop workers
mf celery restart                # restart workers
mf celery logs                   # follow worker logs
mf flower logs                   # follow flower logs
```

### Frontend
```bash
mf frontend install              # npm/yarn/pnpm install
mf frontend dev                  # start dev server
mf frontend build [--prod]       # build for production
mf frontend preview              # preview production build
mf frontend lint                 # run ESLint
mf frontend type-check           # TypeScript check
mf frontend check-all            # all checks
mf frontend restart              # clear cache + restart dev
```

### E2E Testing
```bash
mf e2e install                   # install deps + browsers
mf e2e run                       # run all tests
mf e2e run -f tests/smoke.spec.ts    # specific file
mf e2e run --project approval    # specific project
mf e2e ui                        # interactive UI mode
mf e2e headed                    # visible browser
mf e2e debug                     # debug mode
mf e2e report                    # view HTML report
```

### Code Quality
```bash
mf format [--check]              # format code
mf lint                          # run linter
mf sort-imports                  # sort Python imports
mf format-all                    # format + lint + sort-imports
mf pre-commit [--all] [--local]  # run pre-commit hooks
```

### Debug Utilities
```bash
mf debug check                   # check if debug ports are in use
mf debug clean                   # kill debug port processes
```

### Shell Completions
```bash
# Bash
mf completion bash > /etc/bash_completion.d/mf

# Zsh
mf completion zsh > "${fpath[1]}/_mf"

# Fish
mf completion fish > ~/.config/fish/completions/mf.fish
```

## Configuration

See [`mf.example.yaml`](mf.example.yaml) for a fully commented example.

The config is organized into sections:

| Section | Purpose |
|---------|---------|
| `project` | Project name |
| `compose_file` | Path to docker-compose file |
| `services` | Maps roles (backend, db, redis, workers, flower) to compose service names |
| `database` | DB type, name, user for `mf psql` |
| `frontend` | Path and package manager for frontend commands |
| `e2e` | Path, framework, browser for e2e commands |
| `scripts` | Paths to project scripts (format, lint, pre-commit) |
| `test` | Test runner, env vars, debug port |

## Adding New Image Matchers

The `mf init` scanner uses a pluggable registry of image matchers. To detect new service types, add an entry to `DefaultMatchers` in [`internal/compose/parser.go`](internal/compose/parser.go):

```go
var DefaultMatchers = []ImageMatcher{
    // ... existing matchers ...
    {
        Patterns:    []string{"my-custom-image"},
        Role:        "custom-role",
        ServiceType: "custom",
        EnvMappings: map[string]string{"MY_VAR": "db_name"},
    },
}
```

## License

MIT
