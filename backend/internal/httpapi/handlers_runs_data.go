package httpapi

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"time"

	htmltomarkdown "github.com/JohannesKaufmann/html-to-markdown/v2"
	"github.com/JohannesKaufmann/html-to-markdown/v2/converter"
	"github.com/go-chi/chi/v5"
)

type runStepItem struct {
	Type      string          `json:"type"`
	Title     string          `json:"title"`
	Payload   json.RawMessage `json:"payload"`
	CreatedAt time.Time       `json:"created_at"`
}

type runSourceItem struct {
	ID              string    `json:"id"`
	URL             string    `json:"url"`
	Title           string    `json:"title"`
	Domain          string    `json:"domain"`
	Favicon         string    `json:"favicon_url"`
	MarkdownContent string    `json:"markdown_content,omitempty"`
	CreatedAt       time.Time `json:"created_at"`
}

func (s *Server) handleListRunSteps(w http.ResponseWriter, r *http.Request) {
	user := userFromCtx(r.Context())
	if user == nil {
		writeErr(w, http.StatusUnauthorized, "auth required")
		return
	}

	runID := chi.URLParam(r, "runID")
	if runID == "" {
		writeErr(w, http.StatusBadRequest, "runID is required")
		return
	}

	rows, err := s.pool.Query(
		r.Context(),
		`select rs.type, rs.title, rs.payload, rs.created_at
		 from run_steps rs
		 join runs r on r.id=rs.run_id
		 where rs.run_id=$1 and r.user_id=$2
		 order by rs.created_at asc`,
		runID,
		user.ID,
	)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer rows.Close()

	items := []runStepItem{}
	for rows.Next() {
		var item runStepItem
		if err := rows.Scan(&item.Type, &item.Title, &item.Payload, &item.CreatedAt); err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		items = append(items, item)
	}

	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (s *Server) handleListRunSources(w http.ResponseWriter, r *http.Request) {
	user := userFromCtx(r.Context())
	if user == nil {
		writeErr(w, http.StatusUnauthorized, "auth required")
		return
	}

	runID := chi.URLParam(r, "runID")
	if runID == "" {
		writeErr(w, http.StatusBadRequest, "runID is required")
		return
	}

	// Query sources with page_cache content for markdown conversion
	rows, err := s.pool.Query(
		r.Context(),
		`SELECT s.id, s.url, s.title, s.domain, s.favicon_url, s.created_at,
		        COALESCE(pc.content, '') as content
		 FROM sources s
		 LEFT JOIN page_cache pc ON pc.url = s.url
		 JOIN runs r ON r.id = s.run_id
		 WHERE s.run_id = $1 AND r.user_id = $2
		 ORDER BY s.created_at ASC`,
		runID,
		user.ID,
	)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer rows.Close()

	items := []runSourceItem{}
	for rows.Next() {
		var item runSourceItem
		var content sql.NullString
		if err := rows.Scan(&item.ID, &item.URL, &item.Title, &item.Domain, &item.Favicon, &item.CreatedAt, &content); err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}

		// Convert HTML content to Markdown on the fly
		if content.Valid && content.String != "" {
			markdown, err := htmltomarkdown.ConvertString(
				content.String,
				converter.WithDomain(item.URL),
			)
			if err == nil {
				item.MarkdownContent = markdown
			} else {
				// Fallback: use raw content (already extracted text)
				item.MarkdownContent = content.String
			}
		}

		items = append(items, item)
	}

	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}
