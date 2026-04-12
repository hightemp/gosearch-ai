# Architecture: Layered Architecture

## Overview
Gosearch-ai follows a layered architecture pattern, with clear separation between the transport layer (HTTP/SSE), the application/pipeline layer (agent orchestration), the data access layer (PostgreSQL via pgx), and the configuration layer. This pattern was chosen because the project has a focused domain (search + LLM synthesis), a small codebase, and a single deployment unit orchestrated via Docker Compose. The existing `internal/` package layout already enforces Go's visibility boundaries, making layered architecture a natural fit.

## Decision Rationale
- **Project type:** AI-powered research assistant with agent pipeline
- **Tech stack:** Go 1.24 (chi v5), Vue 3 (Vite), PostgreSQL 16
- **Key factor:** Small team, single deployment, focused domain — complexity doesn't justify DDD or Clean Architecture overhead. The existing package structure already implements clear layers.

## Folder Structure
```
backend/
  cmd/
    api/
      main.go                # Entry point — wires config, DB, HTTP server
  internal/
    config/
      config.go              # Configuration layer — env vars + YAML loading
    db/
      db.go                  # Data layer — pgx connection pool
      migrate.go             # Data layer — goose migration runner
      migrations/            # SQL migration files
        00001_init.sql
        00002_page_cache.sql
        00003_message_run_id.sql
    httpapi/
      server.go              # Transport layer — chi router, middleware, routes
      pipeline.go            # Application layer — agent pipeline (search → fetch → synthesize)
      runs.go                # Transport layer — SSE streaming, run lifecycle
      handlers_library.go    # Transport layer — library REST handlers
      handlers_models.go     # Transport layer — models REST handlers
      handlers_runs_data.go  # Transport layer — runs data REST handlers
      auth_dev.go            # Transport layer — dev-mode auth middleware
    log/
      log.go                 # Cross-cutting — zerolog setup

frontend/
  src/
    main.ts                  # Bootstrap
    router.ts                # Route definitions
    api.ts                   # API client layer
    settingsStore.ts         # Client-side persistence
    stores/                  # State management (Pinia)
    components/              # Presentation layer
      chat/                  # Chat domain components
      common/                # Shared UI components
      modals/                # Modal components
      sidebar/               # Navigation components
    pages/                   # Route-level page components

docker/
  docker-compose.yml         # Infrastructure orchestration
  config.yaml                # Runtime model configuration
```

## Dependency Rules
Dependencies flow downward through the layer stack. Each layer only depends on the layer directly below it.

```
┌──────────────────────────────────┐
│    Transport (httpapi handlers)  │  ← HTTP requests, SSE streams
├──────────────────────────────────┤
│    Application (pipeline.go)     │  ← Agent orchestration logic
├──────────────────────────────────┤
│    Data Access (db package)      │  ← PostgreSQL queries, migrations
├──────────────────────────────────┤
│    Configuration (config)        │  ← Env vars, YAML loading
└──────────────────────────────────┘
```

- ✅ Transport → Application (handlers call pipeline functions)
- ✅ Transport → Data Access (handlers query DB directly for simple CRUD)
- ✅ Application → Data Access (pipeline reads/writes DB)
- ✅ All layers → Configuration (read-only config access)
- ✅ All layers → Log (cross-cutting concern)
- ❌ Data Access → Transport (DB layer must not depend on HTTP concepts)
- ❌ Configuration → any upper layer
- ❌ Application → Transport (pipeline must not know about HTTP)

## Layer/Module Communication
- **HTTP → Handlers:** chi router dispatches to handler methods on `*Server`
- **Handlers → Pipeline:** handlers call pipeline methods, passing context and DB pool
- **Pipeline → External APIs:** pipeline makes HTTP calls to SearxNG/Serper and OpenRouter
- **Pipeline → DB:** pipeline reads/writes page cache, sources, steps, messages via pgx
- **Backend → Frontend:** SSE stream for real-time pipeline progress; REST for CRUD
- **Frontend state:** Pinia stores manage client-side state; `api.ts` handles all fetch calls

## Key Principles
1. **Single `internal/` boundary** — all non-entry-point Go code lives under `internal/`, preventing external imports and keeping the API surface minimal.
2. **Handler methods on Server struct** — all HTTP handlers are methods on `*Server`, which holds config, DB pool, and logger. No global state.
3. **Pipeline as goroutine** — each pipeline run executes in its own goroutine, publishing SSE events. The transport layer streams events; the pipeline layer produces them.
4. **SQL migrations as source of truth** — database schema is defined exclusively in numbered `.sql` files under `migrations/`, run by goose on startup.
5. **Configuration from environment** — all secrets and runtime config come from env vars (with optional YAML overrides for model lists). No hardcoded credentials.

## Code Examples

### Handler pattern (Transport → Data Access)
```go
func (s *Server) handleListChats(w http.ResponseWriter, r *http.Request) {
    user := userFromCtx(r.Context())
    if user == nil {
        writeErr(w, http.StatusUnauthorized, "unauthorized")
        return
    }

    rows, err := s.pool.Query(r.Context(),
        `SELECT id, title, created_at FROM chats
         WHERE user_id = $1 AND deleted_at IS NULL
         ORDER BY updated_at DESC`, user.ID)
    if err != nil {
        writeErr(w, http.StatusInternalServerError, "db error")
        return
    }
    defer rows.Close()

    // ... scan and marshal response
    writeJSON(w, http.StatusOK, chats)
}
```

### Pipeline pattern (Application layer)
```go
// runPipeline executes the search → fetch → synthesize loop.
// It writes progress events to the SSE channel and stores results in the DB.
func (s *Server) runPipeline(ctx context.Context, runID string, query string, ch chan<- sseEvent) {
    // Step 1: Generate search queries via LLM tool call
    // Step 2: Execute searches against SearxNG/Serper
    // Step 3: Fetch and parse page content (HTML→Markdown, PDF)
    // Step 4: Synthesize answer via LLM with context
    // Each step publishes SSE events to `ch`
}
```

### Configuration pattern
```go
cfg, err := config.LoadFromEnv()
if err != nil {
    logger.Fatal().Err(err).Msg("config")
}
// cfg is passed by value to Server — immutable after load
api := httpapi.NewServer(cfg, pool, logger)
```

## Anti-Patterns
- ❌ **Don't add a service/repository abstraction layer** unless the codebase grows significantly. Currently, direct pgx queries in handlers are appropriate for the project's size.
- ❌ **Don't leak HTTP types into the pipeline** — pipeline functions should accept contexts and plain Go types, not `http.Request` or `http.ResponseWriter`.
- ❌ **Don't bypass the migration system** — never modify the database schema manually; always create a new numbered migration file.
- ❌ **Don't add global state** — all shared resources (DB pool, config, logger) are fields on `*Server`, passed via constructor.
- ❌ **Don't create separate Go packages for each handler group** — keep all handlers in `httpapi` to avoid package proliferation in a small codebase.
