# gosearch-ai — Perplexity-like (агентный поиск + ответы с цитатами)

## 1) Цель и ключевые принципы

Цель проекта — реализовать локально разворачиваемый аналог Perplexity:

- **Поиск + синтез ответа**: на входе пользовательский вопрос, на выходе — структурированный ответ, подкреплённый **цитатами/ссылками**.
- **Прозрачный трейс выполнения**: пользователь видит **все поисковые запросы**, **все результаты**, **все прочитанные страницы** и **извлечённые цитаты** *до* того, как система публикует финальный ответ.
- **Streaming UI как в оригинале**: прогресс, шаги, источники, потоковая генерация ответа.
- **Выбор модели в интерфейсе**: модели берутся из OpenRouter, список задаётся в `docker/config.yaml`.

Важно: «ход рассуждений» показываем **в безопасном формате**:

- показываем **инструментальные действия** (queries, URLs, excerpts/snippets, ранжирование, правила выбора источников);
- показываем **краткие объяснения/обоснования** (короткие rationale), но не пытаемся экспонировать внутренний chain-of-thought.

## 2) Технологии

- Backend: Go (HTTP API + SSE)
- Frontend: Vue 3 + Vite
- Поиск: SearxNG
- Хранилище: Postgres
- Инфраструктура: Docker Compose + Proxy Mux
- LLM: OpenRouter (совместимый с OpenAI API)

Скелет уже поднят (см. [`docker/docker-compose.yml`](../docker/docker-compose.yml:1)).

## 3) Текущее состояние (что уже есть)

### Backend

- HTTP сервер на chi: [`Router()`](../backend/internal/httpapi/server.go:27)
- DEV-auth middleware (dev@local): [`withUser()`](../backend/internal/httpapi/auth_dev.go:16)
- БД + миграции (таблицы users/chats/messages/runs/run_steps/…): [`00001_init.sql`](../backend/internal/db/migrations/00001_init.sql:1)
- Запуск run и SSE-стрим:
  - [`handleRunStart()`](../backend/internal/httpapi/runs.go:81) создаёт chat/run/message
  - [`handleRunStream()`](../backend/internal/httpapi/runs.go:128) стримит шаги и answer.*
  - Реальный пайплайн: поиск → чтение → цитаты → ответ (см. [`pipeline.go`](../backend/internal/httpapi/pipeline.go:1))
- Список моделей (из config.yaml): [`handleListModels()`](../backend/internal/httpapi/handlers_models.go:5)
- API библиотеки: `GET /chats`, `GET /chats/{id}/messages`, `GET /bookmarks`, `POST/DELETE /bookmarks/{chatID}`

### Frontend

- Макет, близкий к «Perplexity pro» (sidebar + центр): [`frontend/src/App.vue`](../frontend/src/App.vue:1)
- Главная страница с инпутом и выбором модели: [`frontend/src/pages/HomePage.vue`](../frontend/src/pages/HomePage.vue:1)
- Страница чата: SSE, шаги, источники, потоковый ответ, рендер markdown и ссылки: [`frontend/src/pages/ChatPage.vue`](../frontend/src/pages/ChatPage.vue:1)

## 4) UX/поведение (ориентир на оригинал)

### 4.1 Главная (скриншот 1)

- Центрированный логотип, карточка ввода, выбор модели, кнопка отправки.
- Навигация слева: «Библиотека / Закладки / Недавние».

Реализовано частично: [`frontend/src/pages/HomePage.vue`](../frontend/src/pages/HomePage.vue:1), [`frontend/src/App.vue`](../frontend/src/App.vue:1).

### 4.2 Экран выполнения (скриншот 2 — процесс)

Во время run пользователь должен видеть:

1) статус/прогресс (живой)
2) список поисковых запросов (все попытки)
3) результаты поиска
4) какие источники выбраны и почему (коротко)
5) какие страницы прочитаны (URLs + заголовки)
6) извлечённые цитаты/фрагменты
7) потоковую генерацию ответа

### 4.3 Экран результата (скриншот 3 — ответ с ссылками)

- Ответ отрендерен (markdown), цитаты вставлены как **ссылки на источники**.
- Нумерация источников, как минимум формат `[1] [2] …`.
- Табы (Ответ / Ссылки / Изображения) — в MVP: «Ответ» и «Ссылки», «Изображения» позже.

## 5) Архитектура: агентный пайплайн

Минимальный набор агентов (внутри backend, оркестрация — один run):

1) **PlannerAgent** — формирует план: какие подзадачи, сколько итераций поиска.
2) **QueryRewriterAgent** — превращает вопрос в 1..N запросов для SearxNG.
3) **SearchAgent** — обращается к SearxNG, собирает результаты.
4) **Ranker/SelectorAgent** — выбирает top-K источников (дедуп по домену/URL, антиспам, типы сайтов).
5) **ReaderAgent** — скачивает страницы (HTTP GET), базовая очистка HTML.
6) **SnippetAgent** — извлекает цитаты/факты, присваивает `ref_id`.
7) **AnswerAgent** — генерирует ответ с цитированием и ссылками.

Оркестрация должна стримить «шаги» в SSE через [`publishStep()`](../backend/internal/httpapi/runs.go:192).

## 6) Протокол стриминга (SSE)

Сейчас используются события:

- `step` (универсальный)
- `answer.delta`
- `answer.final`

Нужно закрепить контракт:

### 6.1 `step`

Форма:

```json
{
  "type": "search.query",
  "title": "Поиск",
  "payload": {"query": "...", "category": "general"},
  "created_at": "..."
}
```

Рекомендуемые `type` (MVP):

- `run.started`
- `plan.ready`
- `search.query`
- `search.results`
- `sources.selected`
- `page.fetch.started`
- `page.fetch.ok` / `page.fetch.error`
- `page.readability.ready`
- `snippets.extracted`
- `answer.started`
- `answer.completed`

### 6.2 `answer.delta`

```json
{"delta":"..."}
```

### 6.3 `answer.final`

```json
{"answer":"markdown with [1] refs"}
```

## 7) API (MVP)

Уже есть:

- `GET /models` — список моделей из config.yaml (см. [`handleListModels()`](../backend/internal/httpapi/handlers_models.go:5))
- `POST /runs/start` — старт выполнения (см. [`handleRunStart()`](../backend/internal/httpapi/runs.go:81))
- `GET /runs/{runID}/stream` — SSE (см. [`handleRunStream()`](../backend/internal/httpapi/runs.go:128))

Нужно добавить (MVP):

- `GET /chats/{id}` (мета)

Опционально (для детального UI по источникам):

- `GET /runs/{runID}/sources`
- `GET /runs/{runID}/snippets`

## 8) Данные и хранение

Схема уже подготовлена в Postgres (см. [`00001_init.sql`](../backend/internal/db/migrations/00001_init.sql:1)).

Ключевая связка для трассируемости:

- `runs` — одна попытка ответа
- `run_steps` — сериализованные события/шаги
- `search_queries` + `search_results` — все запросы и результаты
- `sources` + `page_snippets` — прочитанные страницы и извлечённые цитаты
- `page_cache` — кэш прочитанных страниц и цитат

## 9) План реализации (этапы)

### Этап A — MVP «как Perplexity» (без аккаунтов)

Backend:

- Реальный клиент SearxNG + сохранение search_*.
- ReaderAgent (fetch + очистка HTML) + сохранение sources/page_snippets.
- OpenRouter клиент + streaming completion (через Proxy Mux).
- Генерация ответа с привязкой `[n]` → `sources[n]`.
- SSE шаги соответствуют UX.

Frontend:

- `GET /api/models` для выпадающего списка.
- UI шагов/источников/прочитанных страниц.
- Рендер markdown ответа + кликабельные ссылки `[n]`.
- Библиотека: «Недавние» чаты, «Закладки».

### Этап B — качество

- Ранжирование источников (доменный дедуп, качество, свежесть).
- Мульти-запросы (итеративный поиск).
- Более строгие цитаты (показывать excerpt вокруг цитаты).

### Этап C — безопасность/прод

- JWT auth (заменить dev-auth).
- Rate limiting, timeouts, caching.
- Sanitization HTML/markdown, allowlist доменов (опционально).
