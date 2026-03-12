# Using `mf` — Practical Guide

## Getting Started

### 1. Build the binary

```bash
go build -o mf .
```

Move it somewhere in your `$PATH` for global access:

```bash
sudo mv mf /usr/local/bin/
```

### 2. Initialize your project

Navigate to your project directory (where your `docker-compose.yml` lives) and run:

```bash
mf init
```

This scans your compose file, detects all services, and generates an `mf.yaml` config file. You'll see output like:

```
📄 Found compose file: /home/user/my-project/docker-compose.yml

✅ Scanned docker-compose.yml — found 6 services

  web           → backend (main service)
  db            → db (postgres, db: myapp, user: postgres)
  redis         → redis
  celery_worker → celery_worker
  celery_flower → flower
  nginx         → proxy

✅ Generated mf.yaml
```

#### Options

```bash
# Point to a specific compose file
mf init --file path/to/docker-compose.prod.yml

# Overwrite existing mf.yaml without prompting
mf init --force
```

### 3. Review and tweak `mf.yaml`

The generated config will look something like this:

```yaml
project: my-project
compose_file: docker-compose.yml
services:
    backend: web
    db: db
    redis: redis
    workers:
        - celery_worker
    flower: celery_flower
database:
    type: postgres
    name: myapp
    user: postgres
frontend:
    path: ./frontend
    package_manager: npm
e2e:
    path: ./e2e
    framework: playwright
    browser: chromium
```

You can add script paths, test settings, or adjust any detected values:

```yaml
scripts:
    pre_commit: ./scripts/pre-commit-docker.sh
    format: ./scripts/docker-black.sh
    lint: ./scripts/docker-pylint.sh
    ruff: ./scripts/docker-ruff.sh

test:
    runner: pytest
    env:
        ENV: test
    debug_port: 5679
```

---

## Daily Workflow

### Starting your dev environment

```bash
# Start everything
mf up

# Start specific services only
mf up web db redis

# Check logs
mf logs

# Logs for a specific service
mf logs web
```

### Stopping work

```bash
# Stop containers (keeps them around for quick restart)
mf stop

# Stop and remove containers
mf down

# Nuclear option — remove containers AND volumes (data reset)
mf clean
```

### Rebuilding after dependency changes

```bash
# Rebuild all services
mf rebuild

# Rebuild just the backend
mf rebuild web

# Build without the nuclear option
mf build web
mf build --no-cache web
```

---

## Working with Containers

### Getting a shell

```bash
# Shell into the backend service
mf shell

# Shell into a specific service
mf shell db
```

### Database access

```bash
# Open psql (or mysql/mongosh depending on your db type)
mf psql
```

This uses the `database.type`, `database.name`, and `database.user` from your `mf.yaml` to build the right command. For a postgres config, it runs:

```
docker-compose exec db psql -U postgres -d myapp
```

### Redis

```bash
mf redis-cli
```

---

## Running Tests

```bash
# Run all tests
mf test

# Run tests for specific apps/modules
mf test users payments

# Run a specific test file
mf test -f users/tests/test_models.py

# Run a specific test method
mf test -m "TestUserModel.test_create_user"

# Combine file + method
mf test -m "TestUserModel.test_create_user" -f users/tests/test_models.py

# Debug mode — starts debugpy and waits for VS Code/Cursor to attach
mf test --debug
mf test --debug -f users/tests/test_models.py
```

The test command uses the configured `test.runner` (default: `pytest`) and passes `test.env` variables (default: `ENV=test`).

---

## Celery Workers

```bash
mf celery start       # Start all workers defined in services.workers
mf celery stop        # Stop workers
mf celery restart     # Restart workers
mf celery logs        # Follow worker logs

mf flower logs        # Follow Flower dashboard logs
```

---

## Frontend Development

All frontend commands run in the directory specified by `frontend.path` using the configured `frontend.package_manager`.

```bash
mf frontend install       # npm install (or yarn/pnpm)
mf frontend dev           # Start dev server (npm run dev)
mf frontend build         # Production build
mf frontend build --prod  # Production build with optimizations (npm run build:prod)
mf frontend lint          # ESLint
mf frontend type-check    # TypeScript checking
mf frontend check-all     # All checks at once
mf frontend preview       # Preview production build

# Stuck with stale cache?
mf frontend restart       # Clears Vite cache and restarts dev server
```

---

## E2E Testing

All commands run in the directory specified by `e2e.path`.

```bash
# First time setup
mf e2e install            # Installs deps + Playwright browsers

# Run tests
mf e2e run                # All tests
mf e2e run -f tests/smoke.spec.ts        # Specific file
mf e2e run --project approval            # Specific Playwright project

# Interactive modes
mf e2e ui                 # Playwright UI mode
mf e2e headed             # Visible browser
mf e2e debug              # Debug mode

# View results
mf e2e report             # Open the HTML report
```

---

## Code Quality

```bash
# Format code
mf format                 # Run formatter (uses scripts.format or ruff)
mf format --check         # Check only, don't modify files

# Lint
mf lint                   # Run linter (uses scripts.lint, scripts.ruff, or ruff in container)

# Import sorting
mf sort-imports           # Sort Python imports with ruff

# All at once
mf format-all             # Runs format + lint + sort-imports

# Pre-commit hooks
mf pre-commit             # Run pre-commit
mf pre-commit --all       # Run on all files
mf pre-commit --local     # Run local Docker-based pre-commit
```

---

## Debug Utilities

```bash
# Check if debug ports are in use
mf debug check

# Kill processes on debug ports
mf debug clean
```

---

## Shell Completions

Enable tab completion for your shell:

```bash
# Bash
mf completion bash > /etc/bash_completion.d/mf

# Zsh (add to your .zshrc)
mf completion zsh > "${fpath[1]}/_mf"

# Fish
mf completion fish > ~/.config/fish/completions/mf.fish
```

After sourcing, you get tab completion for all commands and flags:

```
$ mf <TAB>
build    celery   clean    completion  debug   down   e2e ...

$ mf celery <TAB>
logs   restart   start   stop

$ mf test --<TAB>
--debug   --file   --method
```

---

## Using a Custom Config Path

By default, `mf` looks for `mf.yaml` in the current directory. You can point to a different config:

```bash
mf --config path/to/my-config.yaml up
```

---

## Extending the Image Matcher

When `mf init` doesn't recognize a service image, you can add support by editing `internal/compose/parser.go`. Add an entry to `DefaultMatchers`:

```go
var DefaultMatchers = []ImageMatcher{
    // ... existing matchers ...

    // Add your custom matcher
    {
        Patterns:    []string{"clickhouse", "yandex/clickhouse-server"},
        Role:        "db",
        ServiceType: "clickhouse",
        EnvMappings: map[string]string{
            "CLICKHOUSE_DB":   "db_name",
            "CLICKHOUSE_USER": "db_user",
        },
    },
}
```

Rebuild with `go build -o mf .` and your new service type will be detected.

---

## Tips

- **Commit `mf.yaml`** to your repo so the whole team uses the same config
- **Add `mf` binary to `.gitignore`** if you build it locally
- Run `mf --help` or `mf <command> --help` anytime — every command has built-in documentation
- The CLI passes through to `docker-compose`, so all standard compose behavior applies
