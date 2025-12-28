# gosearch-ai (Perplexity-like)

Stack: Go backend (agents + SSE), Vue 3 frontend, `searxng` search, Postgres storage, Docker Compose for local runs.

## Quick start

1) Copy environment variables:

```bash
cp .env.example .env
```

2) Set `OPENROUTER_API_KEY` in `.env`.

3) Configure models in `config.yaml`:

```yaml
openrouter:
  models:
    - openai/gpt-4.1-mini
```

4) Run:

```bash
docker compose -f docker/docker-compose.yml up --build
```

Expected ports:

- Frontend: http://localhost:3000
- Backend: http://localhost:8084 (`/healthz`)
- SearxNG: http://localhost:8083
- Postgres: localhost:5434
