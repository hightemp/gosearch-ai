# Project Base Rules

> Auto-detected conventions from codebase analysis. Edit as needed.

## Naming Conventions

- **Go files:** lowercase, snake_case (`config.go`, `auth_dev.go`, `handlers_library.go`)
- **Go packages:** short, lowercase, single-word (`config`, `db`, `httpapi`, `log`)
- **Go variables/functions:** camelCase / PascalCase per Go conventions
- **Go types:** PascalCase (`Config`, `Server`, `User`)
- **Vue components:** PascalCase (`ChatPage.vue`, `MessageItem.vue`, `SettingsModal.vue`)
- **TypeScript files:** camelCase (`api.ts`, `settingsStore.ts`, `modelStore.ts`)
- **CSS:** single `styles.css` in `src/`
- **SQL migrations:** numbered prefix `00001_description.sql`

## Module Structure

- `backend/cmd/api/` — Application entry point
- `backend/internal/config/` — Configuration loading from env + YAML
- `backend/internal/db/` — Database connection pool, migrations
- `backend/internal/httpapi/` — HTTP handlers, router, pipeline, auth
- `backend/internal/log/` — Logging setup (zerolog)
- `frontend/src/components/` — Reusable Vue components (organized by domain: `chat/`, `common/`, `modals/`, `sidebar/`)
- `frontend/src/pages/` — Route-level page components
- `frontend/src/stores/` — Pinia stores
- `docker/` — Docker Compose, SearxNG config, proxy config

## Error Handling

- Go: errors wrapped with `fmt.Errorf("context: %w", err)`, fatal on startup failures via `logger.Fatal()`
- HTTP: structured JSON error responses via `writeErr(w, status, msg)`
- Frontend: fetch-based API calls, no centralized error handler observed

## Logging

- Backend: zerolog structured JSON logging
- Log levels via `zerolog.Logger`
- Pattern: `logger.Info().Str("key", val).Msg("event")`

## Database

- PostgreSQL with pgx v5 connection pool
- Migrations via goose (embedded SQL files in `internal/db/migrations/`)
- UUID primary keys via `gen_random_uuid()`
- Timestamps with `timestamptz`

## HTTP Router

- chi v5 with standard middleware (RequestID, RealIP, Recoverer, Timeout)
- RESTful routes: `/runs`, `/chats`, `/bookmarks`, `/models`
- SSE streaming via `/runs/{runID}/stream`

## Frontend Patterns

- Vue 3 Composition API with `<script setup>` (Vite + TypeScript)
- Pinia for state management
- Vue Router for navigation
- markdown-it + KaTeX for rendering
- highlight.js for code highlighting
- lucide-vue-next for icons
