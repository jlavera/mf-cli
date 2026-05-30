# mf — a CLI for docker-compose projects

`mf` is a standalone CLI tool that replaces project Makefiles for docker-compose based projects. It provides organized commands with proper help, flags, and shell completions — all driven by a simple `mf.yaml` config file.

## Install

### Homebrew (macOS/Linux)

```bash
brew install jlavera/tap/mf
```

### Shell script

```bash
bash <(curl -fsSL https://raw.githubusercontent.com/jlavera/mf-cli/main/install.sh)
```

### From source

Requires Go 1.26.1. Use [mise](https://mise.jdx.dev/) to manage it:

```bash
# Install mise (if you don't have it)
curl https://mise.run | sh

# Add mise to your shell (restart your terminal after this)
echo 'eval "$(~/.local/bin/mise activate)"' >> ~/.zshrc

# Install Go and build mf
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
- **Apps** — Python, Node.js, Ruby, Go, Java detection from commands and images

It also asks whether to enable [local DNS & HTTPS](#local-dns--https-macos) (macOS), letting you reach services as `https://web.my-project.mf`.

```bash
mf init                              # auto-detect compose file in CWD
mf init --file path/to/compose.yml   # specify compose file
mf init --env-file .env.dev          # custom env file (default: .env; passed as docker-compose --env-file)
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
mf rebuild [services...] [-c]        # down + build --no-cache + up (-c also removes volumes)

mf shell [service]                   # shell into container (default: first app-type service)
mf psql [service]                    # database shell (postgres/mysql/mongo); omit service if only one DB
mf redis-cli [service]               # redis-cli (default: first redis service)

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
mf update                            # update mf to the latest version

mf rule                              # print an AI-agent usage guide (e.g. > .cursor/rules/mf.md)
mf rule --copy                       # copy it to the clipboard instead
```

`mf rule` gives you a ready-to-paste guide that teaches an AI assistant how to drive this project with `mf`. Drop it into a [Cursor rule](https://docs.cursor.com/context/rules), `AGENTS.md`, `CLAUDE.md`, or any system/project prompt.

### Stack Commands

```bash
mf celery start|stop|restart|logs    # manage Celery workers
mf flower logs                       # follow Flower logs

mf run <service> <script> [args...]  # run package.json script for services with path: set
mf run frontend dev                  # npm run dev in ./frontend
mf run api test                      # npm run test in ./api

mf e2e install                       # install deps + browsers
mf e2e run [-f file] [--project name]  # run tests
mf e2e ui                            # interactive UI mode
mf e2e headed                        # visible browser
mf e2e debug                         # debug mode
mf e2e report                        # view HTML report
```

## Local DNS & HTTPS (macOS)

Reach your services by name — `https://web.my-project.mf` instead of `localhost:3000`. This is **macOS only** and built entirely into the `mf` binary (no Homebrew, dnsmasq, or external tools).

It has two pieces, both installed once per machine and shared by every project:

- **DNS** — a tiny resolver that points `*.<tld>` (default `*.mf`) at `127.0.0.1`, wired up through macOS's `/etc/resolver`.
- **Proxy** — a reverse proxy on ports 80/443 that routes by hostname to the right container port, with automatic HTTPS via a locally-trusted certificate authority.

### One-time machine setup

```bash
sudo mf dns install      # resolve *.mf to 127.0.0.1 (writes /etc/resolver/mf + runs a DNS daemon)
sudo mf proxy install    # trust a local HTTPS CA + run the reverse proxy on :80/:443
```

Both run as root system services (LaunchDaemons), so they need `sudo`. If you forget it, `mf` explains why and re-runs itself under `sudo` for you. You only do this once — the services start automatically on boot.

### Enable it for a project

Add a `dns:` block to the project's `mf.yaml` (or answer "yes" to the prompt during `mf init`):

```yaml
dns:
  enabled: true      # required to activate (default: false)
  tld: mf            # hostname suffix (default: mf)
  address: 127.0.0.1 # IP the DNS server returns (default: 127.0.0.1)
```

Then start your stack as usual:

```bash
mf up
```

`mf up` registers a route for every service that publishes a port and prints the URLs:

```
🌐 Local DNS routes:
   https://web.my-project.mf → http://localhost:3000
   https://api.my-project.mf → http://localhost:3001
```

`mf down` removes the routes again. The hostname format is `<service>.<compose-project>.<tld>`, so two projects can each have a `web` service without colliding (`web.shop.mf` vs `web.blog.mf`).

#### Per-service hostnames

```yaml
services:
  - name: admin
    type: nodejs
    hostname: backoffice.my-project.mf   # override the auto-derived name
  - name: worker
    type: celery_worker
    hostname: false                      # opt this service out
```

Services without a published port are skipped automatically (there's nothing to proxy to).

### Managing the daemons

```bash
mf dns status              # is the DNS daemon running? show resolver config + a test lookup
mf dns stop                # stop the DNS daemon (sudo; restarts on next boot)
mf dns uninstall           # remove the resolver file + DNS daemon (sudo)

mf proxy status            # is the proxy running? list active routes
mf proxy stop              # stop the proxy daemon (sudo; restarts on next boot)
mf proxy uninstall         # untrust the CA + remove the proxy daemon (sudo)
```

See [USAGE.md](USAGE.md#local-dns--https-macos) for a deeper walkthrough and troubleshooting.

## Configuration

`mf init` generates the config. See `[mf.example.yaml](mf.example.yaml)` for a fully commented example.


| Section        | Purpose                                                                                                  |
| -------------- | -------------------------------------------------------------------------------------------------------- |
| `project`      | Project name                                                                                             |
| `compose_file` | Path to docker-compose file                                                                              |
| `services`     | List of services, each with `name`, `type`, and optional `hostname`, `db_name`, `db_user`, `path`, `package_manager` |
| `dns`          | Local DNS + HTTPS: `enabled`, `tld`, `address` (macOS only)                                              |
| `e2e`          | Path, framework, browser for e2e commands                                                                |
| `scripts`      | Paths to project scripts (format, lint, pre-commit)                                                      |
| `test`         | Test runner, env vars, debug port                                                                        |


## Releasing (maintainers)

```bash
make release VERSION=v1.2.3
```

This tags the commit and pushes the tag. GitHub Actions builds binaries for all platforms and publishes a GitHub Release automatically.

## Adding New Image Matchers

Add entries to `DefaultMatchers` in `[internal/compose/parser.go](internal/compose/parser.go)`:

```go
{
    Patterns:    []string{"my-custom-image"},
    ServiceType: "custom",
    EnvMappings: map[string]string{"MY_DB": "db_name"},
},
```

## License

MIT