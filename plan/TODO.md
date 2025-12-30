# План работ / TODO

## 0) Срез проекта (сейчас)

### Уже сделано

- Docker Compose с сервисами postgres/searxng/backend/frontend: [`docker/docker-compose.yml`](../docker/docker-compose.yml:1)
- Конфигурация env: [`./.env.example`](../.env.example:1)
- README переведен на английский: [`README.md`](../README.md:1)
- Postgres схема под чаты/раны/шаги/поиск/источники/цитаты/закладки: [`backend/internal/db/migrations/00001_init.sql`](../backend/internal/db/migrations/00001_init.sql:1)
- Backend:
  - chi сервер и маршруты: [`Router()`](../backend/internal/httpapi/server.go:27)
  - dev user middleware: [`withUser()`](../backend/internal/httpapi/auth_dev.go:16)
  - SSE хаб и стрим run: [`handleRunStream()`](../backend/internal/httpapi/runs.go:128)
  - старт run: [`handleRunStart()`](../backend/internal/httpapi/runs.go:81)
  - демо-пайплайн (эмулирует Perplexity-стадии): [`runDemoPipeline()`](../backend/internal/httpapi/runs.go:213)
  - список моделей: [`handleListModels()`](../backend/internal/httpapi/handlers_models.go:5)
- Frontend:
  - базовый «perplexity-like» макет: [`frontend/src/App.vue`](../frontend/src/App.vue:1)
  - главная страница (инпут + model select, пока мок): [`frontend/src/pages/HomePage.vue`](../frontend/src/pages/HomePage.vue:1)
  - страница чата подключает SSE и показывает шаги/источники: [`frontend/src/pages/ChatPage.vue`](../frontend/src/pages/ChatPage.vue:1)

### Что уже работает (демо)

- Можно ввести вопрос на главной, перейти на чат и получить:
  - шаги `step` (план/поиск/результаты/выбор источников/цитаты)
  - `answer.delta`/`answer.final` (пока не отображается в UI)

## 1) MVP: «как Perplexity» (приоритет 1)

### 1.1 Backend — интеграция реальных агентов

- [x] Реализовать клиент SearxNG (HTTP) и шаги:
  - [x] `search.query` (логируем *все* запросы)
  - [x] `search.results` (логируем сырые результаты + нормализованные поля)
  - [x] запись в таблицы `search_queries` / `search_results`
- [x] Реализовать Reader (скачивание страниц) + шаги:
  - [x] `page.fetch.started` / `page.fetch.ok` / `page.fetch.error`
  - [x] `page.readability.ready` (текст + заголовок, длина, язык)
  - [x] запись в `sources`
- [x] Реализовать извлечение цитат (SnippetAgent):
  - [x] `snippets.extracted` (список цитат + ref)
  - [x] запись в `page_snippets`
- [x] Реализовать OpenRouter client (OpenAI-compatible):
  - [x] выбор модели из UI (передаётся в [`handleRunStart()`](../backend/internal/httpapi/runs.go:81))
  - [x] streaming completion → `answer.delta`
  - [x] финал → `answer.final`
- [x] Реализовать генерацию ответа с цитированием:
  - [x] соглашение `[n]` ↔ `sources[n]` (или стабильно по ID)
  - [x] гарантировать: все шаги поиска/чтения/цитирования отправлены *до* `answer.final`
- [ ] SSE хаб: чистка `globalHub` при отсутствии подписчиков (избежать утечки памяти)
- [x] Reader: добавить чтение PDF (извлечение текста/цитат) при `application/pdf`

### 1.2 Backend — API для UI «Библиотеки»

- [x] `GET /chats` (recent) + пагинация
- [x] `GET /chats/{id}/messages`
- [x] `POST /bookmarks/{chatID}` / `DELETE /bookmarks/{chatID}`
- [x] `GET /bookmarks`

### 1.3 Frontend — довести UI до оригинального поведения

- [x] На главной загрузка моделей с backend: `GET /api/models` (заменить мок в [`HomePage.vue`](../frontend/src/pages/HomePage.vue:35))
- [ ] Страница чата:
  - [x] Показ потокового ответа (слушать `answer.delta` и `answer.final`)
  - [x] Рендер markdown (минимум: параграфы/списки/код)
  - [x] Превратить `[n]` в кликабельные ссылки на источник
  - [x] UI блока «Просмотр источников» расширить: показывать прочитанные страницы + цитаты
  - [x] Tab «Ссылки»: список источников, домен, фавикон (если есть)
- [ ] Страница чата: корректная привязка `[n]` к snippets (а не `sources[n-1]`)
- [ ] Страница чата: явный error state при сбое SSE (не показывать «Завершено» без ответа)
- [ ] Страница чата: объяснить, что источники относятся к последнему run (история может ссылаться на другое)
- [ ] Все страницы: использовать `VITE_API_BASE_URL` вместо хардкода `/api`
- [x] Сайдбар: история основных запросов (recent queries) для отображения списков
- [x] Sidebar «Недавние» и «Закладки»:
  - [x] список recent chats (title + timestamp)
  - [x] переход к чату по клику
  - [x] pin/bookmark toggle
  - [x] рабочие «Закладки»: загрузка/снятие, отображение в списке

### 1.4 Docker/инфра

- [ ] Привести URL-ы к единой схеме:
  - [ ] backend знает `SEARXNG_BASE_URL` (уже есть в [`config.go`](../backend/internal/config/config.go:25))
  - [ ] frontend проксирует `/api` на backend (проверить `nginx.conf`)
- [ ] Добавить healthchecks (postgres/searxng/backend)

## 2) Perplexity Pro-like качество (приоритет 2)

- [x] Итеративный поиск (несколько запросов с уточнениями)
- [x] Дедуп результатов (по URL/canonical/domain) + фильтрация мусора
- [x] Ранжирование источников (качество/свежесть/разнообразие)
- [x] Поддержка «follow-up» в том же чате (контекст + уточнение)
- [x] Кеширование прочитанных страниц и цитат
- [x] Санитизация текста в UTF-8 перед записью в БД

## 3) Прод-готовность (приоритет 3)

- [ ] JWT auth (заменить dev-path в [`withUser()`](../backend/internal/httpapi/auth_dev.go:16))
- [ ] Rate limiting / timeouts / retries
- [ ] Санитайзинг HTML/markdown и защита от SSRF при чтении страниц
- [ ] Наблюдаемость: трассировка шагов, метрики, structured logs
