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

  web           → python
  db            → postgres (db: myapp, user: postgres)
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

# Use a custom env file (default: .env; stored in mf.yaml; passed as docker-compose --env-file)
mf init --env-file .env.dev

# Overwrite existing mf.yaml without prompting
mf init --force
```

### 3. Review and tweak `mf.yaml`

The generated config will look something like this:

```yaml
project: my-project
compose_file: docker-compose.yml
services:
  - name: web
    type: python
  - name: db
    type: postgres
    db_name: myapp
    db_user: postgres
  - name: redis
    type: redis
  - name: celery_worker
    type: celery_worker
  - name: celery_flower
    type: flower
  - name: frontend
    type: nodejs
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

# Rebuild with a full reset (removes volumes first)
mf rebuild --clean
mf rebuild -c

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

# Forward extra args to the shell (flags starting with `-` pass through automatically)
mf shell -c "ls /app"
mf shell web -c "env"
```

### Database access

```bash
# Open psql (or mysql/mongosh depending on your db type)
mf psql

# Run a one-off query — extra flags are forwarded to the underlying client
mf psql -c "\dt"
mf psql mydb -c "SELECT 1"

# Use `--` to forward a positional arg (otherwise it's treated as a service name)
mf psql -- some-positional
```

This uses the service's `type`, `db_name`, and `db_user` from your `mf.yaml` to build the right command. For a postgres service, it runs:

```
docker-compose exec db psql -U postgres -d myapp
```

### Redis

```bash
mf redis-cli

# Forward flags or commands to redis-cli
mf redis-cli -n 1
mf redis-cli PING
```

---

## Local DNS & HTTPS (macOS)

Instead of remembering `localhost:3000`, `localhost:3001`, etc., reach each service by name over HTTPS: `https://web.my-project.mf`, `https://api.my-project.mf`. This feature is **macOS only** and ships entirely inside the `mf` binary — no Homebrew, dnsmasq, mkcert, or Docker proxy containers required.

### How it works (the 30-second version)

```
Browser → https://web.my-project.mf
   │
   │  1. macOS sees ".mf" and (via /etc/resolver/mf) asks mf's DNS server
   │  2. DNS server answers 127.0.0.1
   │  3. Browser connects to 127.0.0.1:443 with Host: web.my-project.mf
   │  4. mf's reverse proxy matches the host and forwards to localhost:3000
   ▼
Your container
```

- A small **DNS daemon** resolves `*.<tld>` (default `*.mf`) to `127.0.0.1`.
- A **reverse proxy** on ports 80/443 routes by hostname to the published container port, terminating HTTPS with a locally-trusted certificate.

Both run as root system services (macOS LaunchDaemons) so they survive reboots and can bind privileged ports / write to `/etc/resolver`.

### Step 1 — Install the machine-wide services (once)

```bash
sudo mf dns install
sudo mf proxy install
```

These require administrator access because they:

- `mf dns install` — write `/etc/resolver/mf` (tells macOS to send `*.mf` lookups to mf) and install a DNS LaunchDaemon.
- `mf proxy install` — add a local HTTPS certificate authority to your System keychain and install a proxy LaunchDaemon that binds ports 80/443.

If you run them without `sudo`, `mf` prints exactly what it needs and why, then re-runs itself under `sudo` (you'll get the standard password prompt). You only do this once per machine; the services auto-start on every boot.

### Step 2 — Enable DNS for a project

During `mf init` you'll be asked "Enable local DNS?" — answering yes adds this to your `mf.yaml`:

```yaml
dns:
  enabled: true       # required to turn the feature on
  tld: mf             # hostname suffix → web.my-project.mf  (default: mf)
  address: 127.0.0.1  # IP the DNS server returns           (default: 127.0.0.1)
```

You can also add the block by hand. If `dns.enabled` is absent or `false`, nothing changes — `mf up`/`down` behave exactly as before.

### Step 3 — Start your stack

```bash
mf up
```

For every service that publishes a port, `mf` registers a route and prints it:

```
🌐 Local DNS routes:
   https://web.my-project.mf → http://localhost:3000
   https://api.my-project.mf → http://localhost:3001
```

Open `https://web.my-project.mf` in your browser — the certificate is already trusted. `mf down` removes the project's routes. The proxy picks up route changes within a couple of seconds (it polls the routes directory), so there's no daemon restart on `up`/`down`.

### Hostnames

The auto-derived format is `<service>.<compose-project>.<tld>`. The compose-project segment comes from the `name:` field in your compose file, or the directory name. This keeps multiple projects from colliding:

```
project "shop":  web.shop.mf,  api.shop.mf
project "blog":  web.blog.mf,  api.blog.mf
```

Override or opt out per service in `mf.yaml`:

```yaml
services:
  - name: admin
    type: nodejs
    hostname: backoffice.my-project.mf   # use this exact hostname
  - name: worker
    type: celery_worker
    hostname: false                      # never give this service a hostname
```

Services without a published port are skipped automatically — there's nothing to proxy to.

### Multiple projects & TLD conflicts

The DNS resolver file (`/etc/resolver/<tld>`) is the single source of truth and is shared by all projects. If a second project tries to `mf dns install` a `tld` that's already mapped to a different address, `mf` refuses with a clear error instead of silently clobbering it. Projects that share the same `tld`/`address` just work together.

### Managing the daemons

```bash
# DNS
mf dns status              # running? show resolver config + a live test lookup
mf dns stop                # stop the daemon now (sudo; comes back on next boot)
mf dns uninstall           # remove resolver file + daemon entirely (sudo)

# Proxy
mf proxy status            # running? list every active route across all projects
mf proxy stop              # stop the daemon now (sudo; comes back on next boot)
mf proxy uninstall         # untrust the CA + remove the daemon (sudo)
```

`stop` is a temporary pause (handy if you need port 80/443 for something else); `uninstall` is the full teardown.

### Where things live

| Path                                         | What                                        |
| -------------------------------------------- | ------------------------------------------- |
| `/etc/resolver/<tld>`                        | Tells macOS to resolve `*.<tld>` via mf     |
| `/Library/LaunchDaemons/com.mf-cli.dns.plist`   | DNS service definition                   |
| `/Library/LaunchDaemons/com.mf-cli.proxy.plist` | Proxy service definition                 |
| `/Library/Application Support/mf/ca/`        | Local HTTPS certificate authority           |
| `/Library/Application Support/mf/routes/`    | One JSON file per project with its routes   |

### Troubleshooting

- **`https://…mf` doesn't resolve** — run `mf dns status`. If the daemon isn't running, `sudo mf dns start` or re-run `sudo mf dns install`. Confirm `/etc/resolver/<tld>` exists.
- **Browser shows a certificate warning** — re-run `sudo mf proxy install` to (re)trust the CA, and fully restart the browser.
- **`502 no route for host`** — the service isn't started or doesn't publish a port. Check `mf status` and `mf proxy status`; make sure the service has a `ports:` mapping in compose.
- **Logs** — the daemons log to `/tmp/mf-dns.log` and `/tmp/mf-proxy.log`.

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
mf celery start       # Start all services with type: celery_worker/celery_beat
mf celery stop        # Stop workers
mf celery restart     # Restart workers
mf celery logs        # Follow worker logs

mf flower logs        # Follow Flower dashboard logs
```

---

## Running Package Scripts

For services with a `path:` set, use `mf run` to execute package.json scripts:

```bash
mf run frontend install   # npm install (or yarn/pnpm)
mf run frontend dev       # npm run dev
mf run frontend build     # npm run build
mf run frontend lint      # npm run lint

# Works with any service that has path: set
mf run api test           # npm run test (in the api service dir)
mf run api dev            # npm run dev (in the api service dir)
```

Tab completion lists available services, then available scripts from `package.json`.

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

## AI Agent Rule

If you use an AI coding assistant (Cursor, Claude, etc.), `mf rule` gives you a ready-to-paste
guide that teaches it to drive this project through `mf` instead of raw `docker`/`docker-compose`.

```bash
# Print the rule (write it straight to a file)
mf rule > .cursor/rules/mf.md
mf rule > AGENTS.md

# Or copy it to your clipboard, then paste into your assistant's setup
mf rule --copy
```

Good places to put it: a [Cursor rule](https://docs.cursor.com/context/rules) under `.cursor/rules/`,
an `AGENTS.md` / `CLAUDE.md` at the repo root, or any system/project prompt. If the clipboard isn't
available (e.g. over SSH), `mf rule --copy` falls back to printing so you can copy it manually.

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
    {
        Patterns:    []string{"clickhouse", "yandex/clickhouse-server"},
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
