package httpapi

import (
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
)

type chatListItem struct {
	ID         string    `json:"id"`
	Title      string    `json:"title"`
	Pinned     bool      `json:"pinned"`
	Bookmarked bool      `json:"bookmarked"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

type chatMeta struct {
	ID         string    `json:"id"`
	Title      string    `json:"title"`
	Pinned     bool      `json:"pinned"`
	Bookmarked bool      `json:"bookmarked"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
	LastRunID  string    `json:"last_run_id"`
}

type bookmarkItem struct {
	ID           string    `json:"id"`
	Title        string    `json:"title"`
	Pinned       bool      `json:"pinned"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
	BookmarkedAt time.Time `json:"bookmarked_at"`
}

type messageItem struct {
	ID        string    `json:"id"`
	Role      string    `json:"role"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
}

func (s *Server) handleListChats(w http.ResponseWriter, r *http.Request) {
	user := userFromCtx(r.Context())
	if user == nil {
		writeErr(w, http.StatusUnauthorized, "auth required")
		return
	}

	limit, offset := parseLimitOffset(r, 20, 100)
	rows, err := s.pool.Query(
		r.Context(),
		`select c.id, c.title, c.pinned, c.created_at, c.updated_at, (b.id is not null) as bookmarked
		 from chats c
		 left join bookmarks b on b.chat_id=c.id and b.user_id=$1
		 where c.user_id=$1 and c.deleted_at is null
		 order by c.pinned desc, c.updated_at desc
		 limit $2 offset $3`,
		user.ID,
		limit,
		offset,
	)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer rows.Close()

	items := make([]chatListItem, 0, limit)
	for rows.Next() {
		var item chatListItem
		if err := rows.Scan(&item.ID, &item.Title, &item.Pinned, &item.CreatedAt, &item.UpdatedAt, &item.Bookmarked); err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		items = append(items, item)
	}

	writeJSON(w, http.StatusOK, map[string]any{"items": items, "limit": limit, "offset": offset})
}

func (s *Server) handleListMessages(w http.ResponseWriter, r *http.Request) {
	user := userFromCtx(r.Context())
	if user == nil {
		writeErr(w, http.StatusUnauthorized, "auth required")
		return
	}

	chatID := chi.URLParam(r, "chatID")
	if chatID == "" {
		writeErr(w, http.StatusBadRequest, "chatID is required")
		return
	}

	limit, offset := parseLimitOffset(r, 50, 200)
	rows, err := s.pool.Query(
		r.Context(),
		`select m.id, m.role, m.content, m.created_at
		 from messages m
		 join chats c on c.id=m.chat_id
		 where m.chat_id=$1 and c.user_id=$2 and c.deleted_at is null
		 order by m.created_at asc
		 limit $3 offset $4`,
		chatID,
		user.ID,
		limit,
		offset,
	)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer rows.Close()

	items := make([]messageItem, 0, limit)
	for rows.Next() {
		var item messageItem
		if err := rows.Scan(&item.ID, &item.Role, &item.Content, &item.CreatedAt); err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		items = append(items, item)
	}

	writeJSON(w, http.StatusOK, map[string]any{"items": items, "limit": limit, "offset": offset})
}

func (s *Server) handleGetChat(w http.ResponseWriter, r *http.Request) {
	user := userFromCtx(r.Context())
	if user == nil {
		writeErr(w, http.StatusUnauthorized, "auth required")
		return
	}

	chatID := chi.URLParam(r, "chatID")
	if chatID == "" {
		writeErr(w, http.StatusBadRequest, "chatID is required")
		return
	}

	var item chatMeta
	var lastRunID *string
	err := s.pool.QueryRow(
		r.Context(),
		`select c.id, c.title, c.pinned, c.created_at, c.updated_at,
			(select r.id from runs r where r.chat_id=c.id order by r.started_at desc limit 1) as last_run_id,
			(select 1 from bookmarks b where b.chat_id=c.id and b.user_id=$2 limit 1) is not null as bookmarked
		 from chats c
		 where c.id=$1 and c.user_id=$2 and c.deleted_at is null`,
		chatID,
		user.ID,
	).Scan(&item.ID, &item.Title, &item.Pinned, &item.CreatedAt, &item.UpdatedAt, &lastRunID, &item.Bookmarked)
	if err != nil {
		writeErr(w, http.StatusNotFound, "chat not found")
		return
	}
	if lastRunID != nil {
		item.LastRunID = *lastRunID
	}

	writeJSON(w, http.StatusOK, item)
}

func (s *Server) handleCreateBookmark(w http.ResponseWriter, r *http.Request) {
	user := userFromCtx(r.Context())
	if user == nil {
		writeErr(w, http.StatusUnauthorized, "auth required")
		return
	}

	chatID := chi.URLParam(r, "chatID")
	if chatID == "" {
		writeErr(w, http.StatusBadRequest, "chatID is required")
		return
	}

	var exists string
	if err := s.pool.QueryRow(
		r.Context(),
		`select id from chats where id=$1 and user_id=$2 and deleted_at is null`,
		chatID,
		user.ID,
	).Scan(&exists); err != nil {
		writeErr(w, http.StatusNotFound, "chat not found")
		return
	}

	_, err := s.pool.Exec(
		r.Context(),
		`insert into bookmarks(user_id, chat_id) values ($1,$2) on conflict do nothing`,
		user.ID,
		chatID,
	)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (s *Server) handleDeleteBookmark(w http.ResponseWriter, r *http.Request) {
	user := userFromCtx(r.Context())
	if user == nil {
		writeErr(w, http.StatusUnauthorized, "auth required")
		return
	}

	chatID := chi.URLParam(r, "chatID")
	if chatID == "" {
		writeErr(w, http.StatusBadRequest, "chatID is required")
		return
	}

	_, err := s.pool.Exec(
		r.Context(),
		`delete from bookmarks where user_id=$1 and chat_id=$2`,
		user.ID,
		chatID,
	)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (s *Server) handleDeleteChat(w http.ResponseWriter, r *http.Request) {
	user := userFromCtx(r.Context())
	if user == nil {
		writeErr(w, http.StatusUnauthorized, "auth required")
		return
	}

	chatID := chi.URLParam(r, "chatID")
	if chatID == "" {
		writeErr(w, http.StatusBadRequest, "chatID is required")
		return
	}

	result, err := s.pool.Exec(
		r.Context(),
		`update chats set deleted_at = now() where id=$1 and user_id=$2 and deleted_at is null`,
		chatID,
		user.ID,
	)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}

	if result.RowsAffected() == 0 {
		writeErr(w, http.StatusNotFound, "chat not found")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (s *Server) handleListBookmarks(w http.ResponseWriter, r *http.Request) {
	user := userFromCtx(r.Context())
	if user == nil {
		writeErr(w, http.StatusUnauthorized, "auth required")
		return
	}

	limit, offset := parseLimitOffset(r, 20, 100)
	rows, err := s.pool.Query(
		r.Context(),
		`select c.id, c.title, c.pinned, c.created_at, c.updated_at, b.created_at
		 from bookmarks b
		 join chats c on c.id=b.chat_id
		 where b.user_id=$1 and c.deleted_at is null
		 order by b.created_at desc
		 limit $2 offset $3`,
		user.ID,
		limit,
		offset,
	)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer rows.Close()

	items := make([]bookmarkItem, 0, limit)
	for rows.Next() {
		var item bookmarkItem
		if err := rows.Scan(&item.ID, &item.Title, &item.Pinned, &item.CreatedAt, &item.UpdatedAt, &item.BookmarkedAt); err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		items = append(items, item)
	}

	writeJSON(w, http.StatusOK, map[string]any{"items": items, "limit": limit, "offset": offset})
}

func parseLimitOffset(r *http.Request, defaultLimit, maxLimit int) (int, int) {
	limit := defaultLimit
	offset := 0

	if raw := r.URL.Query().Get("limit"); raw != "" {
		if v, err := strconv.Atoi(raw); err == nil {
			limit = v
		}
	}
	if raw := r.URL.Query().Get("offset"); raw != "" {
		if v, err := strconv.Atoi(raw); err == nil {
			offset = v
		}
	}

	if limit <= 0 {
		limit = defaultLimit
	}
	if limit > maxLimit {
		limit = maxLimit
	}
	if offset < 0 {
		offset = 0
	}

	return limit, offset
}
