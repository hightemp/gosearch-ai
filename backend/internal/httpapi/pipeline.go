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
	"github.com/ledongthuc/pdf"
)

const maxPDFSizeBytes = 25 << 20

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
	URL   string `json:"url"`
	Quote string `json:"quote"`
	Ref   int    `json:"ref"`
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

type openRouterToolChoice struct {
	Message struct {
		Content          string     `json:"content"`
		ToolCalls        []toolCall `json:"tool_calls"`
		Reasoning        string     `json:"reasoning"`
		ReasoningDetails []struct {
			Type      string `json:"type"`
			Text      string `json:"text"`
			Summary   string `json:"summary"`
			Encrypted string `json:"encrypted"`
		} `json:"reasoning_details"`
	} `json:"message"`
}

type openRouterToolResponse struct {
	Choices []openRouterToolChoice `json:"choices"`
}

type toolCall struct {
	ID       string `json:"id"`
	Type     string `json:"type"`
	Function struct {
		Name      string `json:"name"`
		Arguments string `json:"arguments"`
	} `json:"function"`
}

type toolStepResponse struct {
	Content   string
	ToolCalls []toolCall
	Reasoning string
}

type toolSearchArgs struct {
	Query      string `json:"query"`
	MaxResults int    `json:"max_results"`
}

type toolFetchArgs struct {
	URLs []struct {
		URL   string `json:"url"`
		Title string `json:"title"`
	} `json:"urls"`
}

func (s *Server) runPipeline(ctx context.Context, runID, query, model string) {
	ctx, cancel := context.WithTimeout(ctx, s.cfg.PipelineTimeout)
	defer cancel()

	s.logger.Debug().Str("run_id", runID).Str("model", model).Msg("pipeline start")
	s.publishStep(ctx, runID, "run.started", "Запуск", map[string]any{"model": model, "query": query})

	s.publishStep(ctx, runID, "plan.ready", "План", map[string]any{
		"items": []string{"Сформировать поисковый запрос", "Найти источники", "Прочитать страницы", "Извлечь цитаты", "Сгенерировать ответ"},
	})

	var (
		answer   string
		sources  []sourceRecord
		snippets []snippetRecord
		err      error
	)
	if strings.TrimSpace(s.cfg.OpenRouterAPIKey) == "" {
		sources, snippets, err = s.searchAndReadIteratively(ctx, runID, query, model)
		if err != nil {
			errMsg := "search error: " + err.Error()
			s.logger.Error().Err(err).Str("run_id", runID).Msg("search pipeline failed")
			s.finalizeRun(ctx, runID, errMsg)
			s.publishRunError(runID, errMsg)
			return
		}
		answer = fallbackAnswer(query, snippets)
	} else {
		answer, sources, snippets, err = s.runAgentPipeline(ctx, runID, query, model)
		if err != nil {
			errMsg := "agent error: " + err.Error()
			s.logger.Error().Err(err).Str("run_id", runID).Msg("agent pipeline failed")
			s.finalizeRun(ctx, runID, errMsg)
			s.publishRunError(runID, errMsg)
			return
		}
	}

	s.publishFinal(runID, answer)
	if err := s.storeAssistantMessage(ctx, runID, answer); err != nil {
		errMsg := "store message error: " + err.Error()
		s.logger.Error().Err(err).Str("run_id", runID).Msg("store assistant message failed")
		s.finalizeRun(ctx, runID, errMsg)
		s.publishRunError(runID, errMsg)
		return
	}

	_, _ = s.pool.Exec(ctx, `update runs set status='finished', finished_at=now() where id=$1`, runID)
	s.logger.Info().Str("run_id", runID).Int("sources", len(sources)).Int("snippets", len(snippets)).Msg("pipeline finished")
}

func (s *Server) searchAndReadIteratively(ctx context.Context, runID, query, model string) ([]sourceRecord, []snippetRecord, error) {
	queue := []string{query}
	used := map[string]struct{}{}
	allResults := make([]searchResult, 0, 32)
	allSources := make([]sourceRecord, 0, s.cfg.SearchMaxSources)
	allSnippets := make([]snippetRecord, 0, s.cfg.SearchMaxSources*s.cfg.SnippetMaxPerSource)
	seenURLs := map[string]struct{}{}
	targetSnippets := s.cfg.SearchMaxSources
	if targetSnippets < 3 {
		targetSnippets = 3
	}

	queryIndex := 1
	for len(queue) > 0 && len(used) < s.cfg.SearchMaxQueries {
		q := strings.TrimSpace(queue[0])
		queue = queue[1:]
		if q == "" {
			continue
		}
		key := strings.ToLower(q)
		if _, ok := used[key]; ok {
			continue
		}
		used[key] = struct{}{}

		s.logger.Debug().Str("run_id", runID).Int("query_index", queryIndex).Str("query", q).Msg("search query")
		results, err := s.searchProvider(ctx, runID, q, queryIndex, s.cfg.SearchMaxQueries)
		if err != nil {
			return nil, nil, err
		}
		allResults = append(allResults, results...)
		ranked := rankAndDedup(allResults)
		s.logger.Debug().Str("run_id", runID).Int("results", len(ranked)).Msg("search ranked results")

		remaining := s.cfg.SearchMaxSources - len(seenURLs)
		if remaining <= 0 {
			break
		}

		selected := selectSourcesExcluding(ranked, remaining, seenURLs)
		if len(selected) == 0 {
			queryIndex++
			continue
		}

		s.publishStep(ctx, runID, "sources.selected", "Выбраны источники", map[string]any{
			"urls": urlsFromResults(selected),
		})

		sources, err := s.persistSources(ctx, runID, selected)
		if err != nil {
			return nil, nil, err
		}
		for _, src := range sources {
			seenURLs[src.URL] = struct{}{}
		}
		allSources = append(allSources, sources...)

		snips, err := s.readAndSnippet(ctx, runID, sources)
		if err != nil {
			return nil, nil, err
		}
		allSnippets = append(allSnippets, snips...)

		if len(allSnippets) >= targetSnippets {
			break
		}

		if len(used) < s.cfg.SearchMaxQueries {
			more, err := s.generateFollowupQueries(ctx, query, allSources, allSnippets, model, s.cfg.SearchMaxQueries-len(used))
			if err != nil {
				s.logger.Warn().Err(err).Str("run_id", runID).Msg("generate followup queries failed")
			} else {
				for _, candidate := range more {
					clean := strings.TrimSpace(candidate)
					if clean == "" {
						continue
					}
					cKey := strings.ToLower(clean)
					if _, ok := used[cKey]; ok {
						continue
					}
					queue = append(queue, clean)
				}
			}
		}

		queryIndex++
	}

	return allSources, allSnippets, nil
}

func (s *Server) runAgentPipeline(ctx context.Context, runID, query, model string) (string, []sourceRecord, []snippetRecord, error) {
	history, err := s.loadChatHistory(ctx, runID, s.cfg.ChatHistoryLimit)
	if err != nil {
		s.logger.Warn().Err(err).Str("run_id", runID).Msg("load chat history failed")
	}
	history = trimHistory(history, query)
	if strings.TrimSpace(model) == "" {
		model = s.cfg.OpenRouterModels[0]
	}

	messages := make([]map[string]any, 0, len(history)+2)
	messages = append(messages, map[string]any{
		"role": "system",
		"content": "You are the primary research agent. Decide whether to answer directly or use tools.\n" +
			"Use tools to search and fetch sources when needed. Keep all tool usage in this single conversation.\n" +
			"Rules:\n" +
			"- Cite sources as [n].\n" +
			"- If you need more info, call the search tool with a focused query.\n" +
			"- If you have URLs to read, call the fetch tool.\n" +
			"- When enough evidence is collected, call final_answer with the full answer in Markdown.\n" +
			"- Do not answer directly in plain content.\n\n" +
			"Math: use $...$ for inline and $$...$$ for display math.",
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
	messages = append(messages, map[string]any{"role": "user", "content": query})

	tools := []map[string]any{
		{
			"type": "function",
			"function": map[string]any{
				"name":        "search",
				"description": "Search the web for relevant sources.",
				"parameters": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"query":       map[string]any{"type": "string"},
						"max_results": map[string]any{"type": "integer"},
					},
					"required": []string{"query"},
				},
			},
		},
		{
			"type": "function",
			"function": map[string]any{
				"name":        "fetch",
				"description": "Fetch and extract snippets from a list of URLs.",
				"parameters": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"urls": map[string]any{
							"type": "array",
							"items": map[string]any{
								"type": "object",
								"properties": map[string]any{
									"url":   map[string]any{"type": "string"},
									"title": map[string]any{"type": "string"},
								},
								"required": []string{"url"},
							},
						},
					},
					"required": []string{"urls"},
				},
			},
		},
		{
			"type": "function",
			"function": map[string]any{
				"name":        "final_answer",
				"description": "Return the final answer in Markdown with citations.",
				"parameters": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"answer": map[string]any{"type": "string"},
					},
					"required": []string{"answer"},
				},
			},
		},
	}

	seenURLs := map[string]struct{}{}
	collectedSources := make([]sourceRecord, 0, s.cfg.SearchMaxSources)
	collectedSnippets := make([]snippetRecord, 0, s.cfg.SearchMaxSources*s.cfg.SnippetMaxPerSource)
	searchCalls := 0

	maxIterations := s.cfg.SearchMaxQueries + 4
	if maxIterations < 6 {
		maxIterations = 6
	}

	for i := 0; i < maxIterations; i++ {
		resp, err := s.openRouterToolStep(ctx, model, messages, tools)
		if err != nil {
			return "", collectedSources, collectedSnippets, err
		}

		if strings.TrimSpace(resp.Reasoning) != "" {
			s.publishStep(ctx, runID, "agent.reasoning", "Рассуждение агента", map[string]any{
				"content": truncateRunes(resp.Reasoning, 2000),
			})
		}

		if len(resp.ToolCalls) == 0 {
			if strings.TrimSpace(resp.Content) != "" {
				s.publishStep(ctx, runID, "agent.message", "Сообщение агента", map[string]any{
					"content": resp.Content,
				})
			}
			continue
		}

		if strings.TrimSpace(resp.Content) != "" {
			s.publishStep(ctx, runID, "agent.message", "Сообщение агента", map[string]any{
				"content": resp.Content,
			})
		}

		messages = append(messages, map[string]any{
			"role":       "assistant",
			"content":    resp.Content,
			"tool_calls": resp.ToolCalls,
		})

		for _, call := range resp.ToolCalls {
			name := call.Function.Name
			args := call.Function.Arguments
			var result any
			var callErr error

			switch name {
			case "search":
				if searchCalls >= s.cfg.SearchMaxQueries {
					callErr = fmt.Errorf("search limit reached")
					break
				}
				var parsed toolSearchArgs
				if err := json.Unmarshal([]byte(args), &parsed); err != nil {
					callErr = err
					break
				}
				parsed.Query = strings.TrimSpace(parsed.Query)
				if parsed.Query == "" {
					callErr = fmt.Errorf("query is required")
					break
				}
				searchCalls++
				results, err := s.searchProvider(ctx, runID, parsed.Query, searchCalls, s.cfg.SearchMaxQueries)
				if err != nil {
					callErr = err
					break
				}
				if parsed.MaxResults > 0 && parsed.MaxResults < len(results) {
					results = results[:parsed.MaxResults]
				}
				result = normalizeResults(results)

			case "fetch":
				var parsed toolFetchArgs
				if err := json.Unmarshal([]byte(args), &parsed); err != nil {
					callErr = err
					break
				}
				items := make([]searchResult, 0, len(parsed.URLs))
				for _, item := range parsed.URLs {
					urlStr := strings.TrimSpace(item.URL)
					if urlStr == "" {
						continue
					}
					key := canonicalizeURL(urlStr)
					if key == "" {
						key = urlStr
					}
					if _, ok := seenURLs[key]; ok {
						continue
					}
					if len(items) >= s.cfg.SearchMaxSources {
						break
					}
					seenURLs[key] = struct{}{}
					items = append(items, searchResult{
						Title:     strings.TrimSpace(item.Title),
						URL:       urlStr,
						Canonical: key,
					})
				}
				if len(items) == 0 {
					result = map[string]any{"items": []any{}}
					break
				}
				s.publishStep(ctx, runID, "agent.fetch", "Чтение источников", map[string]any{
					"items": normalizeResults(items),
				})
				s.publishStep(ctx, runID, "sources.selected", "Выбраны источники", map[string]any{
					"urls": urlsFromResults(items),
				})
				sources, err := s.persistSources(ctx, runID, items)
				if err != nil {
					callErr = err
					break
				}
				snips, err := s.readAndSnippet(ctx, runID, sources)
				if err != nil {
					callErr = err
					break
				}
				collectedSources = append(collectedSources, sources...)
				collectedSnippets = append(collectedSnippets, snips...)
				result = map[string]any{
					"sources":  sources,
					"snippets": snips,
				}

			case "final_answer":
				var payload struct {
					Answer string `json:"answer"`
				}
				if err := json.Unmarshal([]byte(args), &payload); err != nil {
					callErr = err
					break
				}
				answer := strings.TrimSpace(payload.Answer)
				if answer == "" {
					callErr = fmt.Errorf("answer is empty")
					break
				}
				return answer, collectedSources, collectedSnippets, nil

			default:
				callErr = fmt.Errorf("unknown tool: %s", name)
			}

			toolPayload := map[string]any{"ok": callErr == nil}
			if callErr != nil {
				toolPayload["error"] = callErr.Error()
			} else if result != nil {
				toolPayload["result"] = result
			}
			toolJSON, _ := json.Marshal(toolPayload)
			messages = append(messages, map[string]any{
				"role":         "tool",
				"tool_call_id": call.ID,
				"name":         name,
				"content":      string(toolJSON),
			})
		}
	}

	return fallbackAnswer(query, collectedSnippets), collectedSources, collectedSnippets, nil
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
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			_ = resp.Body.Close()
			errMsg := fmt.Errorf("status %d", resp.StatusCode)
			s.logger.Warn().Err(errMsg).Str("run_id", runID).Str("url", source.URL).Msg("page fetch non-200")
			s.publishStep(ctx, runID, "page.fetch.error", "Ошибка запроса", map[string]any{"url": source.URL, "error": errMsg.Error()})
			continue
		}

		contentType := resp.Header.Get("Content-Type")
		if isPDFContentType(contentType, source.URL) {
			s.publishStep(ctx, runID, "page.fetch.pdf", "PDF получен", map[string]any{"url": source.URL, "cached": false})
			text, err := extractPDFText(resp.Body, resp.ContentLength)
			_ = resp.Body.Close()
			if err != nil {
				s.logger.Warn().Err(err).Str("run_id", runID).Str("url", source.URL).Msg("pdf extract failed")
				s.publishStep(ctx, runID, "page.fetch.error", "Ошибка PDF", map[string]any{"url": source.URL, "error": err.Error()})
				continue
			}

			s.publishStep(ctx, runID, "page.fetch.ok", "PDF извлечен", map[string]any{"url": source.URL, "bytes": len(text), "cached": false})

			text = sanitizeUTF8(text)
			s.publishStep(ctx, runID, "page.readability.ready", "PDF прочитан", map[string]any{
				"url":    source.URL,
				"title":  source.Title,
				"length": len(text),
			})

			snips := extractSnippets(text, s.cfg.SnippetMaxPerSource)
			snips = sanitizeSnippets(snips)
			if err := s.upsertPageCache(ctx, source.URL, source.Title, text, snips); err != nil {
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
			continue
		}

		if !isTextContentType(contentType, source.URL) {
			_ = resp.Body.Close()
			s.logger.Warn().Str("run_id", runID).Str("url", source.URL).Str("content_type", contentType).Msg("page fetch skipped")
			s.publishStep(ctx, runID, "page.fetch.skipped", "Пропущен неподдерживаемый тип", map[string]any{
				"url":          source.URL,
				"content_type": contentType,
			})
			continue
		}

		body, err := io.ReadAll(io.LimitReader(resp.Body, 2*1024*1024))
		_ = resp.Body.Close()
		if err != nil {
			errMsg := err
			s.logger.Warn().Err(errMsg).Str("run_id", runID).Str("url", source.URL).Msg("page fetch non-200")
			s.publishStep(ctx, runID, "page.fetch.error", "Ошибка запроса", map[string]any{"url": source.URL, "error": errMsg.Error()})
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

func (s *Server) openRouterToolStep(ctx context.Context, model string, messages []map[string]any, tools []map[string]any) (toolStepResponse, error) {
	if strings.TrimSpace(model) == "" {
		model = s.cfg.OpenRouterModels[0]
	}

	reqBody := map[string]any{
		"model":    model,
		"stream":   false,
		"messages": messages,
		"tools":    tools,
	}
	if s.cfg.OpenRouterReasoning {
		reqBody["reasoning"] = map[string]any{
			"effort":  s.cfg.OpenRouterReasoningEffort,
			"exclude": false,
		}
	}
	payload, _ := json.Marshal(reqBody)

	reqURL := strings.TrimRight(s.cfg.OpenRouterBaseURL, "/") + "/chat/completions"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, reqURL, bytes.NewReader(payload))
	if err != nil {
		return toolStepResponse{}, err
	}
	req.Header.Set("Authorization", "Bearer "+s.cfg.OpenRouterAPIKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("HTTP-Referer", s.cfg.BaseURL)
	req.Header.Set("X-Title", "gosearch-ai")

	client := &http.Client{Timeout: s.cfg.OpenRouterTimeout}
	resp, err := client.Do(req)
	if err != nil {
		return toolStepResponse{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return toolStepResponse{}, fmt.Errorf("openrouter status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var payloadResp openRouterToolResponse
	dec := json.NewDecoder(resp.Body)
	if err := dec.Decode(&payloadResp); err != nil {
		return toolStepResponse{}, err
	}
	if len(payloadResp.Choices) == 0 {
		return toolStepResponse{}, fmt.Errorf("openrouter: empty response")
	}
	msg := payloadResp.Choices[0].Message
	reasoning := strings.TrimSpace(msg.Reasoning)
	if reasoning == "" && len(msg.ReasoningDetails) > 0 {
		var parts []string
		for _, item := range msg.ReasoningDetails {
			switch {
			case item.Summary != "":
				parts = append(parts, item.Summary)
			case item.Text != "":
				parts = append(parts, item.Text)
			case item.Encrypted != "":
				parts = append(parts, item.Encrypted)
			}
		}
		reasoning = strings.Join(parts, "\n")
	}
	return toolStepResponse{Content: msg.Content, ToolCalls: msg.ToolCalls, Reasoning: reasoning}, nil
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

func selectSourcesExcluding(results []searchResult, maxSources int, seenURLs map[string]struct{}) []searchResult {
	out := make([]searchResult, 0, maxSources)
	seenDomains := map[string]struct{}{}
	for _, res := range results {
		if len(out) >= maxSources {
			break
		}
		domain := domainFromURL(res.URL)
		if domain == "" {
			continue
		}
		key := res.Canonical
		if key == "" {
			key = res.URL
		}
		if key == "" {
			continue
		}
		if _, ok := seenURLs[key]; ok {
			continue
		}
		if _, ok := seenDomains[domain]; ok {
			continue
		}
		seenDomains[domain] = struct{}{}
		out = append(out, res)
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
		part = truncateRunes(part, 240)
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

func (s *Server) generateFollowupQueries(ctx context.Context, query string, sources []sourceRecord, snippets []snippetRecord, model string, max int) ([]string, error) {
	if max <= 0 {
		return nil, nil
	}
	if strings.TrimSpace(s.cfg.OpenRouterAPIKey) == "" {
		return nil, nil
	}
	if strings.TrimSpace(model) == "" {
		model = s.cfg.OpenRouterModels[0]
	}

	type quickSource struct {
		Title string `json:"title"`
		URL   string `json:"url"`
	}
	type quickSnippet struct {
		URL   string `json:"url"`
		Quote string `json:"quote"`
	}

	qs := make([]quickSource, 0, len(sources))
	for _, src := range sources {
		qs = append(qs, quickSource{Title: src.Title, URL: src.URL})
	}
	qsSnips := make([]quickSnippet, 0, len(snippets))
	for _, snip := range snippets {
		qsSnips = append(qsSnips, quickSnippet{URL: snip.URL, Quote: snip.Quote})
	}

	prompt := map[string]any{
		"query":    query,
		"sources":  qs,
		"snippets": qsSnips,
		"max":      max,
	}
	promptJSON, _ := json.Marshal(prompt)

	messages := []map[string]any{
		{
			"role":    "system",
			"content": "You propose follow-up web search queries. Return ONLY a JSON array of strings. Do not include explanations.",
		},
		{
			"role":    "user",
			"content": "Generate focused follow-up search queries to fill missing information. Use the context JSON:\n" + string(promptJSON),
		},
	}

	reqBody := map[string]any{
		"model":    model,
		"stream":   false,
		"messages": messages,
	}
	payload, _ := json.Marshal(reqBody)

	reqURL := strings.TrimRight(s.cfg.OpenRouterBaseURL, "/") + "/chat/completions"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, reqURL, bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+s.cfg.OpenRouterAPIKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("HTTP-Referer", s.cfg.BaseURL)
	req.Header.Set("X-Title", "gosearch-ai")

	client := &http.Client{Timeout: s.cfg.OpenRouterTimeout}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return nil, fmt.Errorf("openrouter status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var payloadResp struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	dec := json.NewDecoder(resp.Body)
	if err := dec.Decode(&payloadResp); err != nil {
		return nil, err
	}
	if len(payloadResp.Choices) == 0 {
		return nil, nil
	}
	raw := strings.TrimSpace(payloadResp.Choices[0].Message.Content)
	if raw == "" {
		return nil, nil
	}

	var queries []string
	if err := json.Unmarshal([]byte(raw), &queries); err != nil {
		return nil, err
	}

	out := make([]string, 0, len(queries))
	seen := map[string]struct{}{}
	for _, q := range queries {
		q = strings.TrimSpace(q)
		if q == "" {
			continue
		}
		key := strings.ToLower(q)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, q)
		if len(out) >= max {
			break
		}
	}
	return out, nil
}

func truncateRunes(input string, max int) string {
	if max <= 0 {
		return ""
	}
	count := 0
	for idx := range input {
		if count == max {
			return input[:idx] + "..."
		}
		count++
	}
	return input
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
	if ct == "" {
		return true
	}
	if strings.HasPrefix(ct, "text/") {
		return true
	}
	return false
}

func isPDFContentType(contentType, rawURL string) bool {
	ct := strings.ToLower(strings.TrimSpace(contentType))
	if strings.Contains(ct, "application/pdf") {
		return true
	}
	return strings.HasSuffix(strings.ToLower(rawURL), ".pdf")
}

func extractPDFText(body io.Reader, contentLength int64) (text string, err error) {
	if contentLength > maxPDFSizeBytes {
		return "", fmt.Errorf("pdf too large: %d bytes", contentLength)
	}

	data, err := io.ReadAll(io.LimitReader(body, maxPDFSizeBytes+1))
	if err != nil {
		return "", err
	}
	if int64(len(data)) > maxPDFSizeBytes {
		return "", fmt.Errorf("pdf too large (read): %d bytes", len(data))
	}
	if !bytes.HasPrefix(data, []byte("%PDF-")) {
		return "", errors.New("response is not a PDF (no %PDF- header)")
	}

	defer func() {
		if r := recover(); r != nil {
			text = ""
			err = fmt.Errorf("pdf parse panic: %v", r)
		}
	}()

	reader, err := pdf.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return "", fmt.Errorf("pdf.NewReader: %w", err)
	}

	txtReader, err := reader.GetPlainText()
	if err != nil {
		return "", fmt.Errorf("GetPlainText: %w", err)
	}

	var buf bytes.Buffer
	if _, err := buf.ReadFrom(txtReader); err != nil {
		return "", err
	}
	return buf.String(), nil
}
