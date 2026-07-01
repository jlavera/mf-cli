# Using `mf` in this project

This project uses **`mf`** — a CLI that wraps `docker-compose` — for all container and
project operations. Configuration lives in `mf.yaml` at the repo root.

## Core rules

- **Always use `mf`** for container/project tasks. Never run raw `docker` or
  `docker-compose` commands directly.
- **Read `mf.yaml` first.** It is the source of truth for service names, types, and the
  commands available in this project, and may define more than what's listed below.
- Unsure about flags? Run `mf --help` or `mf <command> --help`.

## Containers & services

| Goal                       | Command                                       |
| -------------------------- | --------------------------------------------- |
| Start services             | `mf up [services...]`                          |
| Stop (keep containers)     | `mf stop [services...]`                        |
| Stop + remove containers   | `mf down`                                      |
| Restart                    | `mf restart [services...]`                     |
| Rebuild image(s)           | `mf rebuild [services...]` (`-c` wipes volumes)|
| Build image(s)             | `mf build [services...] [--no-cache]`          |
| Full reset (volumes too)   | `mf clean`                                     |
| Follow logs                | `mf logs [services...]`                        |
| Show status of each service| `mf status`                                    |

### Compose profiles

Services behind a docker-compose `profiles:` key are inactive by default. To include them,
pass the global `--profile` flag (comma-separated) to any command, e.g.
`mf up --profile debug,tools`. Profiles listed under `profiles:` in `mf.yaml` are always
enabled and merged with the flag.

## Shells & data

- `mf shell [service]` — open a shell in a container (defaults to the main app service).
- `mf psql [service]` — database shell (postgres/mysql/mongo).
- `mf redis-cli [service]` — Redis client.
- `mf run <service> <script>` — run a `package.json` script for a service with `path:` set.

## Tests & code quality

- `mf test [apps...]` — run tests (`-f <file>`, `-m <method>`, `--debug`).
- `mf format [--check]`, `mf lint`, `mf sort-imports`, `mf format-all`.
- `mf pre-commit [--all] [--local]`.

## Local DNS & HTTPS (macOS)

If `mf.yaml` sets `dns.enabled: true`, services with a published port are reachable by
name over HTTPS at `https://<service>.<project>.<tld>` (default tld: `mf`). `mf up` prints
the URLs and registers routes; `mf down` removes them. One-time machine setup:
`sudo mf dns install` and `sudo mf proxy install`.

## Conventions for agents

- Prefer the narrowest command — pass explicit service names instead of acting on everything.
- Never suggest `docker` / `docker-compose` equivalents; use `mf`.
- After changing dependencies, use `mf rebuild <service>` (not just `mf up`).
- Verify state with `mf status` instead of guessing.
