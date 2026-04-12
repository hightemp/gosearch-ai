# Technical Debt & Improvement Tasks

This document tracks technical improvements, bug fixes, and enhancements prioritized by an experienced developer.

---

## Legend

- 🔴 **P0 (Critical)** — Security/stability issues, fix immediately
- 🟠 **P1 (High)** — Should be fixed this week
- 🟡 **P2 (Medium)** — Plan for next sprint
- 🟢 **P3 (Low)** — Nice to have

---

## 🔴 P0 — Critical

### Security & Stability

- [ ] **Implement JWT authentication** — Currently dev mode only; production blocks all requests except `/healthz`
  - File: `backend/internal/httpapi/auth_dev.go`
  - Implement proper JWT token verification
  - Add refresh token rotation
  - Add logout/revoke endpoints

- [ ] **SSE Hub memory leak** — `globalHub` broadcasters are not cleaned up on errors
  - File: `backend/internal/httpapi/runs.go:65`
  - Add TTL-based cleanup for inactive channels
  - Implement proper resource cleanup on connection close

- [ ] **Graceful shutdown for goroutines** — Background goroutines not tied to server context
  - File: `backend/internal/httpapi/runs.go:175`
  - Pass cancellable context to `runPipeline`
  - Track active pipelines and wait on shutdown

- [ ] **Add HTTP server timeouts** — Missing ReadTimeout/WriteTimeout
  - File: `backend/cmd/api/main.go:37`
  - Add `ReadTimeout`, `WriteTimeout`, `IdleTimeout`

---

## 🟠 P1 — High Priority

### Backend

- [ ] **Add rate limiting** — OpenRouter API called without limits; risk of quota overrun
  - File: `backend/internal/httpapi/server.go`
  - Add per-user rate limits
  - Add global rate limits for external API calls

- [ ] **Add CORS middleware** — Frontend on different domain won't work in production
  - File: `backend/internal/httpapi/server.go`
  - Use `chi/cors` middleware with proper origin configuration

- [ ] **Log ignored errors** — Database errors silently discarded
  - File: `backend/internal/httpapi/runs.go:168-169`
  - Replace `_, _ = s.pool.Exec(...)` with proper error logging

- [ ] **Add request logging middleware** — No visibility into incoming requests
  - File: `backend/internal/httpapi/server.go`
  - Add `chi/middleware.Logger` or custom structured logger

- [ ] **Input validation** — No sanitization of user queries
  - Files: `runs.go`, `pipeline.go`
  - Add max length validation
  - Sanitize for XSS in user content

### Frontend

- [ ] **Split ChatPage.vue** — 1393 lines, too complex to maintain
  - File: `frontend/src/pages/ChatPage.vue`
  - Extract components: `MessageList`, `StepsList`, `SourcesPanel`, `ComposerInput`

- [ ] **Add ESLint + Prettier** — No linting configured
  - File: `frontend/package.json`
  - Add eslint, prettier, eslint-plugin-vue
  - Configure pre-commit hooks

- [ ] **SSE reconnection logic** — No auto-reconnect on connection loss
  - File: `frontend/src/pages/ChatPage.vue`
  - Implement exponential backoff reconnection

### Testing

- [ ] **Add Go unit tests** — No tests exist
  - Priority files: `pipeline.go`, `runs.go`, handlers
  - Add `*_test.go` files
  - Target 60%+ coverage for critical paths

- [ ] **Add frontend tests** — No tests exist
  - Add Vitest configuration
  - Test critical user flows

---

## 🟡 P2 — Medium Priority

### Architecture

- [ ] **Separate concerns into layers** — All logic in `httpapi` package
  ```
  internal/
    ├── handler/     (HTTP handlers only)
    ├── service/     (business logic)
    │   ├── pipeline/
    │   ├── search/
    │   └── llm/
    ├── repository/  (database access)
    └── model/       (domain types)
  ```

- [ ] **Add OpenAPI/Swagger documentation** — No API docs
  - Generate from code annotations
  - Serve Swagger UI at `/docs`

- [ ] **Centralized error handling** — Inconsistent error responses
  - Create `AppError` type with codes
  - Add error handling middleware

### Infrastructure

- [ ] **Add Prometheus metrics** — No observability
  - Pipeline latency histograms
  - Run counts by status
  - External API call metrics

- [ ] **Enable SearxNG healthcheck** — Currently commented out
  - File: `docker/docker-compose.yml:43-46`
  - Uncomment and verify

- [ ] **Configure pgxpool limits** — Using defaults
  - File: `backend/internal/db/db.go`
  - Set `MaxConns`, `MinConns`, `MaxConnLifetime`

### Frontend

- [ ] **Cache models list** — `/models` called on every render
  - Add localStorage cache with TTL

- [ ] **Memoize Markdown rendering** — Re-renders on every change
  - Use computed with proper dependencies

- [ ] **Lazy load components** — All pages load upfront
  - Use `defineAsyncComponent` for routes

---

## 🟢 P3 — Low Priority / Nice to Have

### Developer Experience

- [ ] **Add .editorconfig** — Inconsistent formatting across editors
- [ ] **Add pre-commit hooks** — Use husky + lint-staged
- [ ] **API versioning** — Add `/v1/` prefix for future compatibility

### Features

- [ ] **Dark/Light theme toggle**
- [ ] **Internationalization (i18n)** — Currently Russian only
- [ ] **Image search support** — Tab exists but not implemented

### Documentation

- [ ] **Add CONTRIBUTING.md**
- [ ] **Add architecture diagram**
- [ ] **Document environment variables**

---

## ✅ Already Implemented

The following items from the original review have been completed:

- [x] TypeScript strict mode (`tsconfig.json`)
- [x] Pinia for state management
- [x] PDF reading support
- [x] Iterative search with multiple queries
- [x] URL deduplication and filtering
- [x] Source ranking
- [x] Follow-up questions in same chat
- [x] Page caching
- [x] UTF-8 sanitization
- [x] Bookmarks functionality
- [x] Chat history in sidebar
- [x] Streaming markdown responses with citations
- [x] Source snippets display

---

## Quick Wins (< 1 hour each)

1. Add `chi/middleware.Logger` to `server.go`
2. Log errors instead of `_, _ =` pattern
3. Add `.editorconfig` file
4. Uncomment SearxNG healthcheck
5. Add `ReadTimeout: 30*time.Second` to HTTP server

---

## References

- [Original TODO](plan/TODO.md) — Feature roadmap
- [Project Plan](plan/project.md) — Architecture decisions
- [AGENTS.md](AGENTS.md) — Repository guidelines
