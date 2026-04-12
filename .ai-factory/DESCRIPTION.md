# Project: gosearch-ai

## Overview
Gosearch-ai is a Perplexity-style AI research assistant that combines web search with LLM-powered synthesis. Users submit queries through a chat interface; the backend orchestrates a multi-step agent pipeline — generating search queries, fetching and parsing web pages, extracting relevant snippets, and streaming a cited answer back via Server-Sent Events (SSE). The system supports multiple LLM models through OpenRouter and multiple search providers (SearxNG self-hosted or Serper API).

## Core Features
- Multi-step agent pipeline: query generation → web search → page fetch/parse → snippet extraction → LLM synthesis
- Real-time SSE streaming of intermediate steps and final answer
- Tool-calling LLM loop (search + fetch tools) with configurable retries
- Chat history with bookmarks, per-user sessions
- Configurable model selection (OpenRouter gateway with multiple providers)
- Dual search provider support: SearxNG (self-hosted) and Serper API
- HTML-to-Markdown and PDF content extraction
- Page-level caching with TTL
- Docker Compose orchestration for full-stack local development

## Tech Stack
- **Backend Language:** Go 1.24
- **Backend Framework:** chi v5 (HTTP router)
- **Frontend Language:** TypeScript
- **Frontend Framework:** Vue 3 (Composition API, Vite, Pinia)
- **Database:** PostgreSQL 16 (pgx v5 driver, goose migrations)
- **Search:** SearxNG (self-hosted) / Serper API
- **LLM Gateway:** OpenRouter API
- **Auth:** JWT (dev mode with auto-seed user)
- **Logging:** zerolog
- **Containerization:** Docker, Docker Compose
- **Package Manager (frontend):** bun

## Architecture Notes
- Backend follows a clean package layout: `cmd/api` (entry), `internal/config`, `internal/db`, `internal/httpapi`, `internal/log`
- Pipeline runs as a goroutine per request, publishing SSE events to the client
- Frontend is a single-page app with Vue Router (`/` home, `/chat/:chatId` chat)
- All services communicate over a Docker internal network
- A proxy_mux service sits in front for request routing
- Database schema managed via embedded SQL migrations (goose)

## Architecture
See `.ai-factory/ARCHITECTURE.md` for detailed architecture guidelines.
Pattern: Layered Architecture

## Non-Functional Requirements
- Logging: Structured JSON via zerolog, configurable level
- Error handling: Structured JSON error responses from API
- Security: JWT auth, secrets via environment variables, no hardcoded keys
- Performance: Configurable timeouts per pipeline stage, page caching
- Deployment: Single `docker compose up --build` for full stack
