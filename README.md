# mf — a CLI for docker-compose projects

`mf` is a standalone CLI tool that replaces project Makefiles for docker-compose based projects. It provides organized commands with proper help, flags, and shell completions — all driven by a simple `mf.yaml` config file.

## Install

### Team members (private repo)

```bash
GITHUB_TOKEN=ghp_xxx bash <(curl -fsSL https://raw.githubusercontent.com/your-org/mf-cli/main/install.sh)
```

This always installs the latest release. Run the same command to update.

> Your `GITHUB_TOKEN` only needs `repo` scope. Create one at GitHub → Settings → Developer Settings → Personal Access Tokens.

### From source (contributors)

Requires Go 1.26.1. Use [mise](https://mise.jdx.dev/) to manage it:

```bash
mise install && make install
```

## Quick Start

```bash
# In your project directory (where docker-compose.yml lives)
mf init

# This scans your compose file, generates mf.yaml, and installs shell completions
mf up                  # start all services
mf logs                # follow logs
mf shell               # shell into backend container
mf test                # run tests
mf down                # stop everything
```

## `mf init`

Scans your docker-compose file and auto-generates `mf.yaml`. Also installs shell completions automatically.

Detects:

- **Databases** — Postgres, MySQL, MongoDB (extracts DB name and user from env vars)
- **Caches** — Redis, Memcached, Valkey
- **Workers** — Celery workers/beat/flower (from command or service name)
- **Proxies** — Nginx, Traefik, Caddy
- **Search** — Elasticsearch, OpenSearch
- **Storage** — MinIO, LocalStack
- **Apps** — Backend/frontend classification from ports and build context

```bash
mf init                              # auto-detect compose file in CWD
mf init --file path/to/compose.yml   # specify compose file
mf init --force                      # overwrite existing mf.yaml
```

## Commands

### General Commands

```bash
mf up [services...]                  # start containers (detached)
mf stop [services...]                # stop without removing
mf build [services...] [--no-cache]  # build images
mf down                              # stop and remove
mf logs [services...]                # follow logs
mf restart [services...]             # restart (with args: docker-compose restart; no args: down + up)
mf clean                             # remove containers + volumes
mf rebuild [services...]             # down + build --no-cache + up

mf shell [service]                   # shell into container (default: backend; tries bash, falls back to sh)
mf psql [service]                    # database shell (postgres/mysql/mongo); omit service if only one DB
mf redis-cli [service]               # redis-cli (default: services.redis)

mf test [apps...]                    # run all or specific app tests
mf test -f path/to/test.py           # specific test file
mf test -m "TestClass.method"        # specific test method
mf test --debug                      # run with debugpy (attach from VS Code)

mf format [--check]                  # format code
mf lint                              # run linter
mf sort-imports                      # sort Python imports
mf format-all                        # format + lint + sort-imports
mf pre-commit [--all] [--local]      # run pre-commit hooks

mf debug check                       # check if debug ports are in use
mf debug clean                       # kill debug port processes
mf update                            # update mf to the latest release (requires GITHUB_TOKEN)
```

### Stack Commands

```bash
mf celery start|stop|restart|logs    # manage Celery workers
mf flower logs                       # follow Flower logs

mf frontend install                  # npm/yarn/pnpm install
mf frontend dev                      # start dev server
mf frontend build [--prod]           # build (--prod for production)
mf frontend preview                  # preview production build
mf frontend lint                     # run ESLint
mf frontend type-check               # TypeScript check
mf frontend check-all                # all checks
mf frontend restart                  # clear cache + restart dev

mf e2e install                       # install deps + browsers
mf e2e run [-f file] [--project name]  # run tests
mf e2e ui                            # interactive UI mode
mf e2e headed                        # visible browser
mf e2e debug                         # debug mode
mf e2e report                        # view HTML report
```

## Configuration

`mf init` generates the config. See [`mf.example.yaml`](mf.example.yaml) for a fully commented example.

| Section | Purpose |
|---------|---------|
| `project` | Project name |
| `compose_file` | Path to docker-compose file |
| `services.backend` | Compose service name for the backend |
| `services.databases` | List of DB services with `type`, `db_name`, `db_user` |
| `services.redis` | Redis service name |
| `services.workers` | Celery worker service names |
| `services.flower` | Flower service name |
| `frontend` | Path and package manager for frontend commands |
| `e2e` | Path, framework, browser for e2e commands |
| `scripts` | Paths to project scripts (format, lint, pre-commit) |
| `test` | Test runner, env vars, debug port |

## Releasing (maintainers)

```bash
make release VERSION=v1.2.3
```

This tags the commit and pushes the tag. GitHub Actions builds binaries for all platforms and publishes a GitHub Release automatically.

## Adding New Image Matchers

Add entries to `DefaultMatchers` in [`internal/compose/parser.go`](internal/compose/parser.go):

```go
{
    Patterns:    []string{"my-custom-image"},
    Role:        "custom-role",
    ServiceType: "custom",
    EnvMappings: map[string]string{"MY_DB": "db_name"},
},
```

## License

MIT
