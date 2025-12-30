package httpapi

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

type searchResult struct {
	Title      string
	URL        string
	Canonical  string
	Snippet    string
	Engine     string
	Raw        json.RawMessage
	Rank       int
	QueryIndex int
	Score      float64
}

type sourceRecord struct {
	ID       string
	URL      string
	Title    string
	Domain   string
	Favicon  string
	Snippets []snippetRecord
}

type snippetRecord struct {
	URL   string
	Quote string
	Ref   int
}

type chatMessage struct {
	Role    string
	Content string
}

type cachedPage struct {
	Title     string
	Content   string
	Snippets  []string
	FetchedAt time.Time
}
type openRouterChunk struct {
	Choices []struct {
		Delta struct {
			Content string `json:"content"`
		} `json:"delta"`
	} `json:"choices"`
}

func (s *Server) runPipeline(ctx context.Context, runID, query, model string) {
	ctx, cancel := context.WithTimeout(ctx, s.cfg.PipelineTimeout)
	defer cancel()

	s.logger.Debug().Str("run_id", runID).Str("model", model).Msg("pipeline start")
	s.publishStep(ctx, runID, "run.started", "Запуск", map[string]any{"model": model, "query": query})

	s.publishStep(ctx, runID, "plan.ready", "План", map[string]any{
		"items": []string{"Сформировать поисковый запрос", "Найти источники", "Прочитать страницы", "Извлечь цитаты", "Сгенерировать ответ"},
	})

	results, err := s.searchAll(ctx, runID, query)
	if err != nil {
		s.logger.Error().Err(err).Str("run_id", runID).Msg("search failed")
		s.finalizeRun(ctx, runID, "search error: "+err.Error())
		return
	}

	selected := selectSources(results, s.cfg.SearchMaxSources)
	s.publishStep(ctx, runID, "sources.selected", "Выбраны источники", map[string]any{
		"urls": urlsFromResults(selected),
	})

	sources, err := s.persistSources(ctx, runID, selected)
	if err != nil {
		s.logger.Error().Err(err).Str("run_id", runID).Msg("persist sources failed")
		s.finalizeRun(ctx, runID, "sources persist error: "+err.Error())
		return
	}

	snippets, err := s.readAndSnippet(ctx, runID, sources)
	if err != nil {
		s.logger.Error().Err(err).Str("run_id", runID).Msg("reader failed")
		s.finalizeRun(ctx, runID, "reader error: "+err.Error())
		return
	}

	history, err := s.loadChatHistory(ctx, runID, s.cfg.ChatHistoryLimit)
	if err != nil {
		s.logger.Warn().Err(err).Str("run_id", runID).Msg("load chat history failed")
	}
	history = trimHistory(history, query)

	answer, err := s.generateAnswer(ctx, runID, query, model, sources, snippets, history)
	if err != nil {
		s.logger.Error().Err(err).Str("run_id", runID).Msg("answer generation failed")
		s.finalizeRun(ctx, runID, "answer error: "+err.Error())
		return
	}

	s.publishFinal(runID, answer)
	if err := s.storeAssistantMessage(ctx, runID, answer); err != nil {
		s.logger.Error().Err(err).Str("run_id", runID).Msg("store assistant message failed")
		s.finalizeRun(ctx, runID, "store message error: "+err.Error())
		return
	}

	_, _ = s.pool.Exec(ctx, `update runs set status='finished', finished_at=now() where id=$1`, runID)
	s.logger.Info().Str("run_id", runID).Msg("pipeline finished")
}

func (s *Server) searchAll(ctx context.Context, runID, query string) ([]searchResult, error) {
	queries := buildSearchQueries(query, s.cfg.SearchMaxQueries)
	all := make([]searchResult, 0, len(queries)*10)
	for idx, q := range queries {
		s.logger.Debug().Str("run_id", runID).Int("query_index", idx+1).Str("query", q).Msg("search query")
		results, err := s.searchProvider(ctx, runID, q, idx+1, len(queries))
		if err != nil {
			return nil, err
		}
		all = append(all, results...)
	}
	ranked := rankAndDedup(all)
	s.logger.Debug().Str("run_id", runID).Int("results", len(ranked)).Msg("search ranked results")
	return ranked, nil
}

func (s *Server) searchProvider(ctx context.Context, runID, query string, queryIndex, totalQueries int) ([]searchResult, error) {
	switch strings.ToLower(strings.TrimSpace(s.cfg.SearchProvider)) {
	case "", "searxng", "searx":
		return s.searchSearx(ctx, runID, query, queryIndex, totalQueries)
	case "serper":
		return s.searchSerper(ctx, runID, query, queryIndex, totalQueries)
	default:
		return nil, fmt.Errorf("unknown SEARCH_PROVIDER: %s", s.cfg.SearchProvider)
	}
}

func (s *Server) searchSearx(ctx context.Context, runID, query string, queryIndex, totalQueries int) ([]searchResult, error) {
	s.publishStep(ctx, runID, "search.query", "Поиск", map[string]any{
		"query":       query,
		"category":    "general",
		"query_index": queryIndex,
		"total":       totalQueries,
	})

	var queryID string
	if err := s.pool.QueryRow(ctx, `insert into search_queries(run_id, query, category) values ($1,$2,'general') returning id`, runID, query).Scan(&queryID); err != nil {
		return nil, err
	}

	endpoint := strings.TrimRight(s.cfg.SearxNGBaseURL, "/") + "/search"
	reqURL, err := url.Parse(endpoint)
	if err != nil {
		return nil, err
	}
	q := reqURL.Query()
	q.Set("format", "json")
	q.Set("q", query)
	reqURL.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL.String(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "gosearch-ai/0.1")

	client := &http.Client{Timeout: s.cfg.SearchTimeout}
	resp, err := client.Do(req)
	if err != nil {
		s.logger.Error().Err(err).Str("run_id", runID).Msg("searxng request failed")
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		s.logger.Error().Int("status", resp.StatusCode).Str("run_id", runID).Msg("searxng non-200")
		return nil, fmt.Errorf("searxng status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var payload struct {
		Results []map[string]any `json:"results"`
	}
	dec := json.NewDecoder(resp.Body)
	if err := dec.Decode(&payload); err != nil {
		s.logger.Error().Err(err).Str("run_id", runID).Msg("searxng decode failed")
		return nil, err
	}

	results := make([]searchResult, 0, len(payload.Results))
	rank := 1
	for _, item := range payload.Results {
		title, _ := item["title"].(string)
		rawURL, _ := item["url"].(string)
		content, _ := item["content"].(string)
		engine, _ := item["engine"].(string)
		if rawURL == "" {
			continue
		}
		canonical := canonicalizeURL(rawURL)
		rawJSON, _ := json.Marshal(item)
		results = append(results, searchResult{
			Title:      title,
			URL:        rawURL,
			Canonical:  canonical,
			Snippet:    content,
			Engine:     engine,
			Raw:        rawJSON,
			Rank:       rank,
			QueryIndex: queryIndex,
			Score:      scoreResult(rank, queryIndex, rawURL, title, content),
		})
		rank++
	}

	if err := s.storeSearchResults(ctx, queryID, results); err != nil {
		s.logger.Error().Err(err).Str("run_id", runID).Msg("store search results failed")
		return nil, err
	}

	s.publishStep(ctx, runID, "search.results", "Результаты поиска", map[string]any{
		"count":       len(results),
		"query":       query,
		"query_index": queryIndex,
		"total":       totalQueries,
		"results":     normalizeResults(results),
	})

	return results, nil
}

func (s *Server) searchSerper(ctx context.Context, runID, query string, queryIndex, totalQueries int) ([]searchResult, error) {
	if strings.TrimSpace(s.cfg.SerperAPIKey) == "" {
		return nil, fmt.Errorf("SERPER_API_KEY is required for serper provider")
	}

	s.publishStep(ctx, runID, "search.query", "Поиск", map[string]any{
		"query":       query,
		"category":    "general",
		"query_index": queryIndex,
		"total":       totalQueries,
		"provider":    "serper",
	})

	var queryID string
	if err := s.pool.QueryRow(ctx, `insert into search_queries(run_id, query, category) values ($1,$2,'general') returning id`, runID, query).Scan(&queryID); err != nil {
		return nil, err
	}

	payload := map[string]any{
		"q":   query,
		"num": s.cfg.SerperNum,
		"hl":  s.cfg.SerperHL,
		"gl":  s.cfg.SerperGL,
	}
	body, _ := json.Marshal(payload)

	endpoint := strings.TrimRight(s.cfg.SerperBaseURL, "/") + "/search"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", s.cfg.SerperAPIKey)
	req.Header.Set("User-Agent", "gosearch-ai/0.1")

	client := &http.Client{Timeout: s.cfg.SearchTimeout}
	resp, err := client.Do(req)
	if err != nil {
		s.logger.Error().Err(err).Str("run_id", runID).Msg("serper request failed")
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		s.logger.Error().Int("status", resp.StatusCode).Str("run_id", runID).Msg("serper non-200")
		return nil, fmt.Errorf("serper status %d: %s", resp.StatusCode, strings.TrimSpace(string(respBody)))
	}

	var payloadResp struct {
		Organic []struct {
			Title    string `json:"title"`
			Link     string `json:"link"`
			Snippet  string `json:"snippet"`
			Position int    `json:"position"`
		} `json:"organic"`
	}
	dec := json.NewDecoder(resp.Body)
	if err := dec.Decode(&payloadResp); err != nil {
		s.logger.Error().Err(err).Str("run_id", runID).Msg("serper decode failed")
		return nil, err
	}

	results := make([]searchResult, 0, len(payloadResp.Organic))
	for i, item := range payloadResp.Organic {
		rawURL := strings.TrimSpace(item.Link)
		if rawURL == "" {
			continue
		}
		rank := item.Position
		if rank <= 0 {
			rank = i + 1
		}
		canonical := canonicalizeURL(rawURL)
		rawJSON, _ := json.Marshal(item)
		results = append(results, searchResult{
			Title:      item.Title,
			URL:        rawURL,
			Canonical:  canonical,
			Snippet:    item.Snippet,
			Engine:     "serper",
			Raw:        rawJSON,
			Rank:       rank,
			QueryIndex: queryIndex,
			Score:      scoreResult(rank, queryIndex, rawURL, item.Title, item.Snippet),
		})
	}

	if err := s.storeSearchResults(ctx, queryID, results); err != nil {
		s.logger.Error().Err(err).Str("run_id", runID).Msg("store search results failed")
		return nil, err
	}

	s.publishStep(ctx, runID, "search.results", "Результаты поиска", map[string]any{
		"count":       len(results),
		"query":       query,
		"query_index": queryIndex,
		"total":       totalQueries,
		"results":     normalizeResults(results),
	})

	return results, nil
}

func (s *Server) storeSearchResults(ctx context.Context, queryID string, results []searchResult) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback(ctx)
		}
	}()

	for _, res := range results {
		_, err = tx.Exec(
			ctx,
			`insert into search_results(query_id, rank, title, url, snippet, engine, raw) values ($1,$2,$3,$4,$5,$6,$7)`,
			queryID,
			res.Rank,
			res.Title,
			res.URL,
			res.Snippet,
			res.Engine,
			res.Raw,
		)
		if err != nil {
			return err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return err
	}
	return nil
}

func (s *Server) persistSources(ctx context.Context, runID string, selected []searchResult) ([]sourceRecord, error) {
	records := make([]sourceRecord, 0, len(selected))
	for _, res := range selected {
		domain := domainFromURL(res.URL)
		favicon := faviconFromURL(res.URL)
		var sourceID string
		if err := s.pool.QueryRow(
			ctx,
			`insert into sources(run_id, url, title, domain, favicon_url) values ($1,$2,$3,$4,$5) returning id`,
			runID,
			res.URL,
			res.Title,
			domain,
			favicon,
		).Scan(&sourceID); err != nil {
			return nil, err
		}

		records = append(records, sourceRecord{
			ID:      sourceID,
			URL:     res.URL,
			Title:   res.Title,
			Domain:  domain,
			Favicon: favicon,
		})
	}
	return records, nil
}

func (s *Server) readAndSnippet(ctx context.Context, runID string, sources []sourceRecord) ([]snippetRecord, error) {
	client := &http.Client{Timeout: s.cfg.FetchTimeout}
	snippets := make([]snippetRecord, 0, 12)
	ref := 1
	cacheTTL := s.cfg.PageCacheTTL

	for i := range sources {
		source := &sources[i]
		s.publishStep(ctx, runID, "page.fetch.started", "Запрос страницы", map[string]any{"url": source.URL})

		cached, ok, err := s.loadCachedPage(ctx, source.URL)
		if err == nil && ok && cached.Content != "" && time.Since(cached.FetchedAt) < cacheTTL {
			s.logger.Debug().Str("run_id", runID).Str("url", source.URL).Msg("page cache hit")
			cached.Title = sanitizeUTF8(cached.Title)
			cached.Content = sanitizeUTF8(cached.Content)
			if cached.Title != "" && source.Title == "" {
				source.Title = cached.Title
				_, _ = s.pool.Exec(ctx, `update sources set title=$1 where id=$2`, cached.Title, source.ID)
			}

			s.publishStep(ctx, runID, "page.fetch.ok", "Кэш страницы", map[string]any{
				"url":         source.URL,
				"cached":      true,
				"age_seconds": int(time.Since(cached.FetchedAt).Seconds()),
			})

			s.publishStep(ctx, runID, "page.readability.ready", "Страница прочитана", map[string]any{
				"url":    source.URL,
				"title":  cached.Title,
				"length": len(cached.Content),
			})

			snips := sanitizeSnippets(cached.Snippets)
			if len(snips) == 0 {
				snips = extractSnippets(cached.Content, s.cfg.SnippetMaxPerSource)
				_ = s.upsertPageCache(ctx, source.URL, cached.Title, cached.Content, snips)
			}
			for _, snip := range snips {
				if err := s.storeSnippet(ctx, source.ID, snip); err != nil {
					return nil, err
				}
				record := snippetRecord{URL: source.URL, Quote: snip, Ref: ref}
				source.Snippets = append(source.Snippets, record)
				snippets = append(snippets, record)
				ref++
			}
			continue
		}

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, source.URL, nil)
		if err != nil {
			s.logger.Warn().Err(err).Str("run_id", runID).Str("url", source.URL).Msg("build page request failed")
			s.publishStep(ctx, runID, "page.fetch.error", "Ошибка запроса", map[string]any{"url": source.URL, "error": err.Error()})
			continue
		}
		req.Header.Set("User-Agent", "gosearch-ai/0.1")

		resp, err := client.Do(req)
		if err != nil {
			s.logger.Warn().Err(err).Str("run_id", runID).Str("url", source.URL).Msg("page fetch failed")
			s.publishStep(ctx, runID, "page.fetch.error", "Ошибка запроса", map[string]any{"url": source.URL, "error": err.Error()})
			continue
		}
		body, err := io.ReadAll(io.LimitReader(resp.Body, 2*1024*1024))
		_ = resp.Body.Close()
		if err != nil || resp.StatusCode < 200 || resp.StatusCode >= 300 {
			errMsg := err
			if errMsg == nil {
				errMsg = fmt.Errorf("status %d", resp.StatusCode)
			}
			s.logger.Warn().Err(errMsg).Str("run_id", runID).Str("url", source.URL).Msg("page fetch non-200")
			s.publishStep(ctx, runID, "page.fetch.error", "Ошибка запроса", map[string]any{"url": source.URL, "error": errMsg.Error()})
			continue
		}

		contentType := resp.Header.Get("Content-Type")
		if !isTextContentType(contentType, source.URL) {
			// TODO: add PDF reader to extract text/snippets from application/pdf.
			s.logger.Warn().Str("run_id", runID).Str("url", source.URL).Str("content_type", contentType).Msg("page fetch skipped")
			s.publishStep(ctx, runID, "page.fetch.skipped", "Пропущен неподдерживаемый тип", map[string]any{
				"url":          source.URL,
				"content_type": contentType,
			})
			continue
		}

		s.publishStep(ctx, runID, "page.fetch.ok", "Страница получена", map[string]any{"url": source.URL, "bytes": len(body), "cached": false})

		title, text := extractText(body)
		title = sanitizeUTF8(title)
		text = sanitizeUTF8(text)
		if title != "" && source.Title == "" {
			source.Title = title
			_, _ = s.pool.Exec(ctx, `update sources set title=$1 where id=$2`, title, source.ID)
		}

		s.publishStep(ctx, runID, "page.readability.ready", "Страница прочитана", map[string]any{
			"url":    source.URL,
			"title":  title,
			"length": len(text),
		})

		snips := extractSnippets(text, s.cfg.SnippetMaxPerSource)
		snips = sanitizeSnippets(snips)
		if err := s.upsertPageCache(ctx, source.URL, title, text, snips); err != nil {
			s.logger.Warn().Err(err).Str("run_id", runID).Str("url", source.URL).Msg("cache upsert failed")
		}
		if len(snips) == 0 {
			continue
		}

		for _, snip := range snips {
			if err := s.storeSnippet(ctx, source.ID, snip); err != nil {
				s.logger.Error().Err(err).Str("run_id", runID).Str("url", source.URL).Msg("store snippet failed")
				return nil, err
			}
			record := snippetRecord{URL: source.URL, Quote: snip, Ref: ref}
			source.Snippets = append(source.Snippets, record)
			snippets = append(snippets, record)
			ref++
		}
	}

	if len(snippets) > 0 {
		s.publishStep(ctx, runID, "snippets.extracted", "Извлечены цитаты", map[string]any{"snippets": snippets})
	}

	return snippets, nil
}

func (s *Server) loadCachedPage(ctx context.Context, pageURL string) (cachedPage, bool, error) {
	var page cachedPage
	var rawSnips []byte
	err := s.pool.QueryRow(
		ctx,
		`select title, content, snippets, fetched_at from page_cache where url=$1`,
		pageURL,
	).Scan(&page.Title, &page.Content, &rawSnips, &page.FetchedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return cachedPage{}, false, nil
		}
		return cachedPage{}, false, err
	}
	if len(rawSnips) > 0 {
		_ = json.Unmarshal(rawSnips, &page.Snippets)
	}
	return page, true, nil
}

func (s *Server) upsertPageCache(ctx context.Context, pageURL, title, content string, snippets []string) error {
	title = sanitizeUTF8(title)
	content = sanitizeUTF8(content)
	snippets = sanitizeSnippets(snippets)
	raw, _ := json.Marshal(snippets)
	_, err := s.pool.Exec(
		ctx,
		`insert into page_cache(url, title, content, snippets, fetched_at)
		 values ($1,$2,$3,$4,now())
		 on conflict (url) do update set title=excluded.title, content=excluded.content, snippets=excluded.snippets, fetched_at=excluded.fetched_at`,
		pageURL,
		title,
		content,
		raw,
	)
	return err
}

func (s *Server) storeSnippet(ctx context.Context, sourceID, quote string) error {
	quote = sanitizeUTF8(quote)
	_, err := s.pool.Exec(ctx, `insert into page_snippets(source_id, quote) values ($1,$2)`, sourceID, quote)
	return err
}

func (s *Server) generateAnswer(ctx context.Context, runID, query, model string, sources []sourceRecord, snippets []snippetRecord, history []chatMessage) (string, error) {
	if strings.TrimSpace(s.cfg.OpenRouterAPIKey) == "" {
		s.logger.Debug().Str("run_id", runID).Msg("openrouter disabled, using fallback answer")
		return fallbackAnswer(query, snippets), nil
	}

	messages := make([]map[string]any, 0, len(history)+2)
	messages = append(messages, map[string]any{
		"role":    "system",
		"content": "You are a research assistant. Use only the provided sources and cite them as [n]. Keep the answer concise and structured in Markdown.\n\nMath: use LaTeX delimiters. Inline formulas with $...$, display formulas with $$...$$. Do not put formulas in square brackets.\nExamples: Inline $x_{n+1}=r x_n(1-x_n)$. Display:\n$$x_{n+1}=r x_n(1-x_n)$$",
	})
	for _, msg := range history {
		role := strings.TrimSpace(msg.Role)
		if role != "user" && role != "assistant" && role != "system" {
			continue
		}
		content := strings.TrimSpace(msg.Content)
		if content == "" {
			continue
		}
		messages = append(messages, map[string]any{
			"role":    role,
			"content": content,
		})
	}
	messages = append(messages, map[string]any{
		"role":    "user",
		"content": buildPrompt(query, sources, snippets),
	})

	reqBody := map[string]any{
		"model":    model,
		"stream":   true,
		"messages": messages,
	}
	payload, _ := json.Marshal(reqBody)

	reqURL := strings.TrimRight(s.cfg.OpenRouterBaseURL, "/") + "/chat/completions"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, reqURL, bytes.NewReader(payload))
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+s.cfg.OpenRouterAPIKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("HTTP-Referer", s.cfg.BaseURL)
	req.Header.Set("X-Title", "gosearch-ai")

	client := &http.Client{Timeout: s.cfg.OpenRouterTimeout}
	resp, err := client.Do(req)
	if err != nil {
		s.logger.Error().Err(err).Str("run_id", runID).Msg("openrouter request failed")
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		s.logger.Error().Int("status", resp.StatusCode).Str("run_id", runID).Msg("openrouter non-200")
		return "", fmt.Errorf("openrouter status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	answer := strings.Builder{}
	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || !strings.HasPrefix(line, "data:") {
			continue
		}
		data := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if data == "[DONE]" {
			break
		}

		var chunk openRouterChunk
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			continue
		}
		if len(chunk.Choices) == 0 {
			continue
		}
		delta := chunk.Choices[0].Delta.Content
		if delta == "" {
			continue
		}
		answer.WriteString(delta)
		s.publishAnswerDelta(runID, delta)
	}

	if err := scanner.Err(); err != nil && !errors.Is(err, context.Canceled) {
		s.logger.Error().Err(err).Str("run_id", runID).Msg("openrouter stream error")
		return "", err
	}

	if answer.Len() == 0 {
		s.logger.Warn().Str("run_id", runID).Msg("openrouter empty answer, using fallback")
		return fallbackAnswer(query, snippets), nil
	}
	return answer.String(), nil
}

func (s *Server) storeAssistantMessage(ctx context.Context, runID, answer string) error {
	var chatID string
	if err := s.pool.QueryRow(ctx, `select chat_id from runs where id=$1`, runID).Scan(&chatID); err != nil {
		return err
	}
	_, err := s.pool.Exec(ctx, `insert into messages(chat_id, user_id, role, content) select $1, user_id, 'assistant', $2 from runs where id=$3`, chatID, answer, runID)
	return err
}

func (s *Server) loadChatHistory(ctx context.Context, runID string, limit int) ([]chatMessage, error) {
	if limit <= 0 {
		return nil, nil
	}
	rows, err := s.pool.Query(
		ctx,
		`select role, content from (
			select m.role, m.content, m.created_at
			from messages m
			join runs r on r.chat_id = m.chat_id
			where r.id=$1
			order by m.created_at desc
			limit $2
		) t
		order by created_at asc`,
		runID,
		limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]chatMessage, 0, limit)
	for rows.Next() {
		var item chatMessage
		if err := rows.Scan(&item.Role, &item.Content); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, nil
}

func trimHistory(history []chatMessage, query string) []chatMessage {
	if len(history) == 0 {
		return history
	}
	last := history[len(history)-1]
	if strings.TrimSpace(last.Role) == "user" && strings.TrimSpace(last.Content) == strings.TrimSpace(query) {
		return history[:len(history)-1]
	}
	return history
}

func (s *Server) finalizeRun(ctx context.Context, runID, errMsg string) {
	_, _ = s.pool.Exec(ctx, `update runs set status='failed', finished_at=now(), error=$2 where id=$1`, runID, errMsg)
}

func selectSources(results []searchResult, maxSources int) []searchResult {
	out := make([]searchResult, 0, maxSources)
	seenDomains := map[string]struct{}{}
	seenURLs := map[string]struct{}{}
	for _, res := range results {
		domain := domainFromURL(res.URL)
		if domain == "" {
			continue
		}
		key := res.Canonical
		if key == "" {
			key = res.URL
		}
		if _, ok := seenURLs[key]; ok {
			continue
		}
		if _, ok := seenDomains[domain]; ok {
			continue
		}
		seenDomains[domain] = struct{}{}
		seenURLs[key] = struct{}{}
		out = append(out, res)
		if len(out) >= maxSources {
			break
		}
	}
	return out
}

func urlsFromResults(results []searchResult) []string {
	out := make([]string, 0, len(results))
	for _, res := range results {
		out = append(out, res.URL)
	}
	return out
}

func normalizeResults(results []searchResult) []map[string]any {
	out := make([]map[string]any, 0, len(results))
	for _, res := range results {
		out = append(out, map[string]any{
			"title":   res.Title,
			"url":     res.URL,
			"snippet": res.Snippet,
			"engine":  res.Engine,
			"score":   res.Score,
		})
	}
	return out
}

func rankAndDedup(results []searchResult) []searchResult {
	dedup := map[string]searchResult{}
	for _, res := range results {
		key := res.Canonical
		if key == "" {
			key = res.URL
		}
		if key == "" {
			continue
		}
		if existing, ok := dedup[key]; ok {
			if res.Score > existing.Score {
				dedup[key] = res
			} else {
				if existing.Title == "" && res.Title != "" {
					existing.Title = res.Title
				}
				if existing.Snippet == "" && res.Snippet != "" {
					existing.Snippet = res.Snippet
				}
				dedup[key] = existing
			}
			continue
		}
		dedup[key] = res
	}

	sorted := make([]searchResult, 0, len(dedup))
	for _, res := range dedup {
		sorted = append(sorted, res)
	}
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Score > sorted[j].Score
	})
	return sorted
}

func domainFromURL(raw string) string {
	parsed, err := url.Parse(raw)
	if err != nil || parsed.Hostname() == "" {
		return ""
	}
	return parsed.Hostname()
}

func canonicalizeURL(raw string) string {
	parsed, err := url.Parse(raw)
	if err != nil || parsed.Hostname() == "" {
		return ""
	}

	parsed.Fragment = ""
	parsed.Host = strings.ToLower(parsed.Host)
	parsed.Scheme = strings.ToLower(parsed.Scheme)
	if parsed.Scheme == "" {
		parsed.Scheme = "https"
	}

	q := parsed.Query()
	for key := range q {
		lower := strings.ToLower(key)
		if strings.HasPrefix(lower, "utm_") || lower == "fbclid" || lower == "gclid" || lower == "ref" {
			q.Del(key)
		}
	}
	parsed.RawQuery = q.Encode()

	parsed.Path = strings.TrimSuffix(parsed.Path, "/")
	if parsed.Path == "" {
		parsed.Path = "/"
	}

	return parsed.String()
}

func scoreResult(rank, queryIndex int, rawURL, title, snippet string) float64 {
	score := 100.0 - float64(rank*2)
	score -= float64(queryIndex-1) * 5.0
	if strings.HasPrefix(rawURL, "https://") {
		score += 3.0
	}
	if len(title) > 8 {
		score += 2.0
	}
	if len(snippet) > 80 {
		score += 2.0
	}
	return score
}

func buildSearchQueries(query string, maxQueries int) []string {
	out := []string{query}
	if maxQueries <= 1 {
		return out
	}

	if looksCyrillic(query) {
		out = append(out, query+" обзор")
	} else {
		out = append(out, query+" overview")
	}
	if maxQueries <= 2 {
		return out
	}

	if looksCyrillic(query) {
		out = append(out, query+" руководство")
	} else {
		out = append(out, query+" guide")
	}
	return out
}

func looksCyrillic(text string) bool {
	for _, r := range text {
		if r >= 'А' && r <= 'я' {
			return true
		}
	}
	return false
}

func faviconFromURL(raw string) string {
	domain := domainFromURL(raw)
	if domain == "" {
		return ""
	}
	return fmt.Sprintf("https://www.google.com/s2/favicons?domain=%s&sz=64", url.QueryEscape(domain))
}

func extractText(input []byte) (string, string) {
	raw := string(input)

	title := ""
	titleRe := regexp.MustCompile(`(?is)<title[^>]*>(.*?)</title>`)
	if match := titleRe.FindStringSubmatch(raw); len(match) > 1 {
		title = strings.TrimSpace(stripTags(match[1]))
	}

	clean := raw
	clean = regexp.MustCompile(`(?is)<script[^>]*>.*?</script>`).ReplaceAllString(clean, " ")
	clean = regexp.MustCompile(`(?is)<style[^>]*>.*?</style>`).ReplaceAllString(clean, " ")
	clean = regexp.MustCompile(`(?is)<noscript[^>]*>.*?</noscript>`).ReplaceAllString(clean, " ")
	clean = stripTags(clean)

	return title, normalizeWhitespace(clean)
}

func stripTags(input string) string {
	tagRe := regexp.MustCompile(`(?s)<[^>]+>`)
	return tagRe.ReplaceAllString(input, " ")
}

func extractSnippets(text string, max int) []string {
	if max <= 0 {
		return nil
	}
	parts := strings.FieldsFunc(text, func(r rune) bool {
		return r == '.' || r == '!' || r == '?' || r == '\n'
	})
	out := make([]string, 0, max)
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		if len(part) > 240 {
			part = part[:240] + "…"
		}
		out = append(out, part)
		if len(out) >= max {
			break
		}
	}
	return out
}

func buildPrompt(query string, sources []sourceRecord, snippets []snippetRecord) string {
	var b strings.Builder
	b.WriteString("Question:\n")
	b.WriteString(query)
	b.WriteString("\n\nSources:\n")
	for i, snip := range snippets {
		title := ""
		for _, src := range sources {
			if src.URL == snip.URL {
				title = src.Title
				break
			}
		}
		label := fmt.Sprintf("[%d]", i+1)
		b.WriteString(label)
		if title != "" {
			b.WriteString(" ")
			b.WriteString(title)
		}
		b.WriteString("\n")
		b.WriteString("- URL: ")
		b.WriteString(snip.URL)
		b.WriteString("\n")
		b.WriteString("- Quote: ")
		b.WriteString(snip.Quote)
		b.WriteString("\n\n")
	}
	return b.String()
}

func fallbackAnswer(query string, snippets []snippetRecord) string {
	if len(snippets) == 0 {
		return fmt.Sprintf("Пока нет данных для ответа на запрос: %s", query)
	}
	var b strings.Builder
	b.WriteString("Черновик ответа на основе найденных источников:\n\n")
	for i, snip := range snippets {
		b.WriteString(fmt.Sprintf("- %s [%d]\n", snip.Quote, i+1))
	}
	return b.String()
}

func normalizeWhitespace(text string) string {
	return strings.Join(strings.Fields(text), " ")
}

func sanitizeUTF8(input string) string {
	if input == "" {
		return input
	}
	cleaned := strings.Map(func(r rune) rune {
		if r == 0 {
			return -1
		}
		return r
	}, input)
	return strings.ToValidUTF8(cleaned, " ")
}

func sanitizeSnippets(snippets []string) []string {
	if len(snippets) == 0 {
		return snippets
	}
	out := make([]string, 0, len(snippets))
	for _, snip := range snippets {
		snip = sanitizeUTF8(snip)
		snip = strings.TrimSpace(snip)
		if snip == "" {
			continue
		}
		out = append(out, snip)
	}
	return out
}

func isTextContentType(contentType, rawURL string) bool {
	ct := strings.ToLower(strings.TrimSpace(contentType))
	if strings.Contains(ct, "text/html") || strings.Contains(ct, "text/plain") || strings.Contains(ct, "application/xhtml+xml") {
		return true
	}
	if strings.HasSuffix(strings.ToLower(rawURL), ".pdf") || strings.Contains(ct, "application/pdf") {
		return false
	}
	if ct == "" {
		return true
	}
	if strings.HasPrefix(ct, "text/") {
		return true
	}
	return false
}
