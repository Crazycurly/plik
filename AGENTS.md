# AGENTS.md — Plik

> Entry point for AI agents working on this codebase. Not an exhaustive manual — follow pointers to scoped ARCHITECTURE.md files for deeper context.

> [!CAUTION]
> ## Mandatory Review Gate — No Exceptions
>
> **NEVER perform any git or GitHub write action without explicit user approval.** This includes:
> - `git commit`, `git push`, `git push --force-with-lease`
> - Creating branches, pull requests, issues, or comments on GitHub
> - Submitting PR reviews, merging PRs
>
> **Required process for EVERY commit/push (use the `/commit` workflow):**
> 1. `git add -A && git diff --cached --stat` — show the diff summary to the user
> 2. Propose a commit message — wait for approval
> 3. `git commit` — only after user approves both diff and message
> 4. Ask before pushing — the user must explicitly say "push" or "go ahead"
>
> This applies equally to trivial one-line changes and large refactors. There are zero exceptions.

## What is Plik?

Plik is a temporary file upload system (WeTransfer-like) written in Go, with a Vue 3 web UI and a cross-platform CLI client. It supports multiple storage and metadata backends, authentication providers, and features like one-shot downloads, streaming, end-to-end encryption (E2EE via age), and server-side encryption.

## Tech Stack

| Layer     | Tech |
|-----------|------|
| Server    | Go, gorilla/mux, GORM |
| Webapp    | Vue 3, Vite, Tailwind CSS, CodeMirror 6 |
| CLI       | Go, docopt-go |
| Config    | TOML (server), TOML (client `.plikrc`) |
| Data      | File, OpenStack Swift, S3, Google Cloud Storage |
| Metadata  | SQLite3, PostgreSQL, MySQL (via GORM) |
| CI        | GitHub Actions (tests, docker build/deploy on PR comment, release, Helm chart publish) |

## Repo Layout

```
plik/
├── AGENTS.md              ← you are here
├── ARCHITECTURE.md         ← system-wide architecture
├── README.md               ← project README (concise)
├── Makefile                ← build orchestration
├── Dockerfile
├── .agent/                 ← agentic workflows (/review-changes, /prepare-pr, /cut-release)
├── server/                 ← Go server (see server/ARCHITECTURE.md)
│   ├── main.go             ← entry point
│   ├── plikd.cfg           ← default config
│   ├── cmd/                ← CLI commands (cobra)
│   ├── common/             ← shared types, config, feature flags
│   ├── context/            ← custom request context (predates Go stdlib context)
│   ├── data/               ← data backend interface + implementations
│   ├── handlers/           ← HTTP handlers
│   ├── metadata/           ← metadata backend (GORM)
│   ├── middleware/          ← middleware chain (auth, logging, upload/file resolution, CORS, download domain restriction)
│   └── server/             ← HTTP server + router setup
├── client/                 ← CLI client + MCP server (see client/ARCHITECTURE.md)
├── plik/                   ← Go client library (see plik/ARCHITECTURE.md)
├── webapp/                 ← Vue 3 SPA (see webapp/ARCHITECTURE.md)
├── testing/                ← backend integration tests (see testing/ARCHITECTURE.md)
├── charts/                 ← Helm chart for Kubernetes deployment
├── .github/                ← GitHub Actions workflows (see .github/ARCHITECTURE.md)
├── changelog/              ← release changelogs
├── releaser/               ← release build scripts
├── docs/                   ← VitePress documentation site
└── vendor/                 ← Go vendored dependencies
```

## Build & Run

```bash
make                        # Build everything (frontend + clients + server)
make server                 # Build server only → server/plikd
make client                 # Build CLI client only → client/plik
make frontend               # Build Vue webapp → webapp/dist
make docker                 # Build Docker image (rootgg/plik:dev)
make helm                   # Package Helm chart locally (dry-run)
make helm-install           # Package and install Helm chart locally
cd server && ./plikd        # Run server on http://127.0.0.1:8080
```

#### Pull Request Deployments (GitHub Actions)
- `docker build` (comment on PR): Builds and pushes image `rootgg/plik:pr-{PR_NUMBER}`
- `docker deploy` (comment on PR): Deploy PR image to `plik.root.gg` (requires secrets)

## Test

```bash
make test                   # Unit tests + CLI integration tests
make test-frontend           # Webapp unit tests (vitest)
make test-frontend-e2e       # Webapp e2e tests (playwright — builds + starts fresh plikd)
make test-backends           # Docker-based backend integration tests (all)
make test-backend mariadb    # Docker-based test for a single backend
make lint                   # go fmt + go vet + go fix
make gofix                  # Run go fix
make vuln                   # govulncheck (report only)
```

## Key Files

| File | Purpose |
|------|---------|
| `server/plikd.cfg` | Server configuration (TOML) — all options with comments |
| `client/.plikrc` | CLI client configuration template |
| `Makefile` | Build targets for server, client, frontend, docker, release |
| `server/common/config.go` | Config struct + parsing + env var override logic |
| `server/common/file.go` | File model + status constants |
| `server/common/upload.go` | Upload model |
| `server/metadata/upload.go` | Upload queries, `UploadFilters` struct for filtering by user/token/badge settings |
| `server/handlers/misc.go` | Shared helpers: `parseBoolFilter`, `parseBadgeFilters` (badge filter struct from request) |
| `webapp/src/components/UploadControls.vue` | Shared sort/order/badge-filter control bar used by AdminView and HomeView |
| `server/common/feature_flags.go` | Feature flag types (`disabled`/`enabled`/`default`/`forced`) |

## Conventions

- **Configuration**: TOML file + env var override using SCREAMING_SNAKE_CASE (e.g., `PLIKD_DEBUG_REQUESTS=true`)
- **Feature flags**: Four states — `disabled`, `enabled` (opt-in), `default` (opt-out), `forced`
- **Special values**: `0` = use server default, `-1` = unlimited (for file size, TTL, etc.)
- **Error handling**: Handlers return HTTP errors; middleware chain panics on missing required context values
- **ID generation**: Random hex strings (16 chars for files, 16 chars for uploads)
- **Backend interface**: `data.Backend` is the storage abstraction; implementations are swappable via config

## Best Practices

- **Always update docs**: When changing code, update the relevant `ARCHITECTURE.md` and VitePress docs
- **Keep Helm chart in sync with plikd config**: When adding, removing, or renaming configuration fields in `server/common/config.go` or `server/plikd.cfg`, you **must** also update the Helm chart:
  - `charts/plik/values.yaml` — add/update the field under `plikd:` (non-sensitive) or `secrets:` (sensitive)
  - `charts/plik/templates/configmap.yaml` — add/update the explicit key in the template (non-sensitive config only; never put secrets here)
  - `charts/plik/templates/secret.yaml` — if the field is a credential, add it under `secrets:` in `values.yaml` and a corresponding key in `secret.yaml`
- **Helm secrets pattern**: All sensitive credentials must live in the `secrets:` top-level block of `values.yaml`. They are rendered into a `Secret` resource by `secret.yaml`, and injected into the pod via `envFrom.secretRef` (`optional: true`). Never put secrets in the ConfigMap.
- **BYO Secret (existingSecret)**: Set `secrets.existingSecret: "my-secret-name"` to skip Secret creation and reference an external secret (e.g., Vault, Sealed Secrets, ESO). Use the `plik.secretName` helper in templates to resolve the correct name.
- **Helm persistence**: the chart has two independent PVCs — `persistence` for uploaded files (`/home/plik/server/files`) and `dbPersistence` for the SQLite database (`/home/plik/server/db`). Both default to `emptyDir` when disabled. The default `MetadataBackendConfig.ConnectionString` is `/home/plik/server/db/plik.db`.
- **Run tests before committing**: `make lint && make test`
- **Keep ARCHITECTURE.md files in sync**: Each root folder has its own — update the one closest to your change
- **Release process**: Before creating a GitHub release, update the version in `README.md` and move `charts/plik/CHANGELOG.md` entries from `[Unreleased]` to the new version heading

## Documentation

The documentation lives in two places:

1. **For agents**: Scoped `ARCHITECTURE.md` files in each root folder
2. **For humans**: VitePress site in `docs/` — preview locally with `cd docs && npm run dev`

### Updating docs

```bash
cd docs && npm install       # First time only
cd docs && npm run dev       # Preview at localhost:5173
make docs                    # Build docs (validates links, injects version)
```

**Important**: Always run `make docs` when you touch documentation files to catch build errors (dead links, etc.) before committing.

## Scoped Architecture Docs

| File | Scope |
|------|-------|
| [ARCHITECTURE.md](ARCHITECTURE.md) | System-wide: package layering, data flow, API, auth, config |
| [server/ARCHITECTURE.md](server/ARCHITECTURE.md) | Server internals: packages, middleware chain, handlers |
| [client/ARCHITECTURE.md](client/ARCHITECTURE.md) | CLI client: commands, config, archive/crypto |
| [plik/ARCHITECTURE.md](plik/ARCHITECTURE.md) | Go library: public API, types, test harness |
| [webapp/ARCHITECTURE.md](webapp/ARCHITECTURE.md) | Vue 3 SPA: components, routing, API layer, state |
| [testing/ARCHITECTURE.md](testing/ARCHITECTURE.md) | Backend integration tests: docker-based test scripts |
| [releaser/ARCHITECTURE.md](releaser/ARCHITECTURE.md) | Release tooling: build pipeline, Docker stages, client/server compilation |
| [charts/plik/ARCHITECTURE.md](charts/plik/ARCHITECTURE.md) | Helm chart: structure, config/secrets separation, persistence, versioning |
| [.github/ARCHITECTURE.md](.github/ARCHITECTURE.md) | GitHub Actions workflows, CI/CD, Helm chart release flow |
