# Repository Guidelines

## Project Structure & Module Organization
- `backend/`: Go API service (HTTP + SSE). Entry point at `backend/cmd/api/main.go`; DB and migrations in `backend/internal/db/` and `backend/internal/db/migrations/`.
- `frontend/`: Vue 3 app (Vite). Main UI in `frontend/src/` with routes in `frontend/src/router.ts`.
- `docker/`: Local orchestration (`docker/docker-compose.yml`) and SearxNG config at `docker/searxng/settings.yml`.
- `screenshots/`, `plan/`: Reference assets and planning notes.

## Build, Test, and Development Commands
- `cp .env.example .env`: Create env file and set `OPENROUTER_API_KEY`.
- `docker compose -f docker/docker-compose.yml up --build`: Run full stack (frontend, backend, SearxNG, Postgres).
- `cd frontend && npm install`: Install UI dependencies.
- `cd frontend && npm run dev`: Run Vite dev server.
- `cd frontend && npm run build`: Build production frontend bundle.
- `cd backend && go run ./cmd/api`: Run the API directly (expects `.env` and Postgres/SearxNG reachable).

## Coding Style & Naming Conventions
- Go: use standard `gofmt` formatting; keep package names short and lowercase.
- Vue/TS: follow existing file naming (`PascalCase.vue`, `camelCase` for TS identifiers).
- Config files live under `docker/`; avoid hardcoding secrets in code.

## Testing Guidelines
- No automated tests are committed currently.
- If adding Go tests, place `*_test.go` next to the package and run `go test ./...` from `backend/`.
- If adding frontend tests, introduce a test runner in `frontend/package.json` and document its command here.

## Commit & Pull Request Guidelines
- No commit message conventions are documented in this repository. Use concise, imperative summaries (optionally with a scope, e.g., `backend: add runs endpoint`).
- PRs should include: a clear summary, testing notes (`docker compose ...` / `go test ./...`), and screenshots for UI changes.

## Security & Configuration Tips
- Store secrets in `.env`; never commit API keys.
- Verify ports when running locally: frontend `3000`, backend `8084`, SearxNG `8083`, Postgres `5434`.
