# gosearch-ai (Perplexity-like)

Stack: Go backend (agents + SSE), Vue 3 frontend, SearxNG search, Postgres storage, Proxy Mux for outbound traffic, Docker Compose for local runs.

## Quick start

1) Copy environment variables:

```bash
cp docker/.env.example docker/.env
```

2) Set `OPENROUTER_API_KEY` in `docker/.env`.

3) Configure models in `docker/config.yaml`:

```yaml
openrouter:
  models:
    - openai/gpt-4.1-mini
```

4) Ensure Proxy Mux routes are configured in `docker/proxy_mux/config.yaml`.

5) Run:

```bash
cd docker
docker compose up --build
```

Expected ports:

- Frontend: http://localhost:3000
- Backend: http://localhost:8084 (`/healthz`)
- SearxNG: http://localhost:8083
- Postgres: localhost:5434
