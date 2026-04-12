# AGENTS.md

> Project map for AI agents. Keep this file up-to-date as the project evolves.

## Project Overview
Gosearch-ai is a Perplexity-style AI research assistant that combines web search (SearxNG/Serper) with LLM-powered synthesis (OpenRouter). Users chat via a Vue 3 frontend; the Go backend orchestrates a multi-step agent pipeline (search → fetch → extract → synthesize) and streams results back via SSE.

## Tech Stack
- **Backend:** Go 1.24 (chi v5 router, pgx v5, zerolog, goose migrations)
- **Frontend:** Vue 3, TypeScript, Vite, Pinia, Vue Router
- **Database:** PostgreSQL 16
- **Search:** SearxNG (self-hosted) / Serper API
- **LLM Gateway:** OpenRouter API
- **Containerization:** Docker Compose

## Project Structure
```
backend/
  cmd/api/main.go          # HTTP server entry point
  internal/
    config/config.go       # Env + YAML configuration loader
    db/
      db.go                # pgx connection pool
      migrate.go           # goose migration runner
      migrations/          # SQL migration files (00001_init.sql, ...)
    httpapi/
      server.go            # chi router, middleware, route registration
      pipeline.go          # Agent pipeline (search → fetch → synthesize)
      runs.go              # SSE streaming, run lifecycle
      handlers_*.go        # REST handlers (library, models, runs data)
      auth_dev.go          # Dev-mode auto-seed user + JWT
    log/log.go             # zerolog setup
frontend/
  src/
    App.vue                # Root component
    main.ts                # Vue app bootstrap
    router.ts              # Routes: / (home), /chat/:chatId
    api.ts                 # API fetch helpers
    settingsStore.ts       # Settings persistence
    stores/                # Pinia stores (modelStore)
    components/
      chat/                # Chat UI: composer, header, messages, sources, steps
      common/              # Reusable: loading dots, modal backdrop, pagination
      modals/              # Settings, paginated list modals
      sidebar/             # Sidebar, chat list, icon bar
    pages/
      HomePage.vue         # Landing page
      ChatPage.vue         # Chat conversation page
docker/
  docker-compose.yml       # Full-stack orchestration
  config.yaml              # OpenRouter model list
  searxng/settings.yml     # SearxNG engine config
  proxy_mux/config.yaml    # Proxy multiplexer config
plan/                      # Planning notes
screenshots/               # UI screenshots
```

## Key Entry Points
| File | Purpose |
|------|---------|
| `backend/cmd/api/main.go` | Backend HTTP server entry point |
| `backend/internal/httpapi/server.go` | Router setup and route registration |
| `backend/internal/httpapi/pipeline.go` | Agent pipeline logic (search, fetch, LLM) |
| `backend/internal/config/config.go` | Configuration from env vars + YAML |
| `frontend/src/main.ts` | Vue app bootstrap |
| `frontend/src/router.ts` | Frontend route definitions |
| `docker/docker-compose.yml` | Full-stack Docker orchestration |
| `docker/config.yaml` | OpenRouter model configuration |

## Build, Test, and Development Commands
- `cp docker/.env.example docker/.env` — Create env file and set `OPENROUTER_API_KEY`
- `cd docker && docker compose up --build` — Run full stack (frontend, backend, SearxNG, Postgres)
- `cd frontend && bun install` — Install UI dependencies
- `cd frontend && bun run dev` — Run Vite dev server
- `cd frontend && bun run build` — Build production frontend bundle
- `cd backend && go run ./cmd/api` — Run the API directly (expects env vars and Postgres/SearxNG reachable)

## Ports
| Service | Port |
|---------|------|
| Frontend | 3000 |
| Backend | 8084 (host) → 8081 (container) |
| SearxNG | 8083 |
| Postgres | 5434 |
| Proxy Mux | 8380 |

## Documentation
| Document | Path | Description |
|----------|------|-------------|
| README | `README.md` | Project landing page with quick-start |
| Project Plan | `plan/project.md` | Planning notes |
| Tasks | `plan/TASKS.md` | Task tracking |

## AI Context Files
| File | Purpose |
|------|---------|
| `AGENTS.md` | This file — project structure map |
| `.ai-factory/DESCRIPTION.md` | Project specification and tech stack |
| `.ai-factory/ARCHITECTURE.md` | Architecture decisions and guidelines |
| `.ai-factory/config.yaml` | AI Factory configuration |
| `.ai-factory/rules/base.md` | Detected codebase conventions |

## Coding Style & Naming Conventions
- Go: standard `gofmt`; short lowercase package names; camelCase/PascalCase per Go conventions
- Vue/TS: PascalCase `.vue` files, camelCase TS identifiers
- SQL migrations: `00NNN_description.sql` via goose
- Config/secrets in env vars or `docker/` — never hardcode

## Testing Guidelines
- No automated tests committed currently
- Go tests: `*_test.go` next to the package, run `go test ./...` from `backend/`
- Frontend tests: introduce a test runner in `frontend/package.json`

## Security & Configuration
- Store secrets in `.env`; never commit API keys
- JWT auth (dev mode: auto-seed user)

## Agent Rules
- Never combine shell commands with `&&`, `||`, or `;` — execute each command as a separate Bash tool call.
- Use MCP context7 to search for additional information and documentation.