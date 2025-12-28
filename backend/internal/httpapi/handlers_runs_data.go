package httpapi

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
)

type runStepItem struct {
	Type      string          `json:"type"`
	Title     string          `json:"title"`
	Payload   json.RawMessage `json:"payload"`
	CreatedAt time.Time       `json:"created_at"`
}

type runSourceItem struct {
	ID        string    `json:"id"`
	URL       string    `json:"url"`
	Title     string    `json:"title"`
	Domain    string    `json:"domain"`
	Favicon   string    `json:"favicon_url"`
	CreatedAt time.Time `json:"created_at"`
}

type runSnippetItem struct {
	URL       string    `json:"url"`
	Quote     string    `json:"quote"`
	Ref       int       `json:"ref"`
	CreatedAt time.Time `json:"created_at"`
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

	rows, err := s.pool.Query(
		r.Context(),
		`select s.id, s.url, s.title, s.domain, s.favicon_url, s.created_at
		 from sources s
		 join runs r on r.id=s.run_id
		 where s.run_id=$1 and r.user_id=$2
		 order by s.created_at asc`,
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
		if err := rows.Scan(&item.ID, &item.URL, &item.Title, &item.Domain, &item.Favicon, &item.CreatedAt); err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		items = append(items, item)
	}

	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (s *Server) handleListRunSnippets(w http.ResponseWriter, r *http.Request) {
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
		`select s.url, ps.quote, ps.created_at
		 from page_snippets ps
		 join sources s on s.id=ps.source_id
		 join runs r on r.id=s.run_id
		 where s.run_id=$1 and r.user_id=$2
		 order by ps.created_at asc`,
		runID,
		user.ID,
	)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer rows.Close()

	items := []runSnippetItem{}
	ref := 1
	for rows.Next() {
		var item runSnippetItem
		if err := rows.Scan(&item.URL, &item.Quote, &item.CreatedAt); err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		item.Ref = ref
		ref++
		items = append(items, item)
	}

	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}
