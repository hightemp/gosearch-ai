package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"

	"gosearch-ai/backend/internal/config"
)

type Server struct {
	cfg    config.Config
	pool   *pgxpool.Pool
	logger zerolog.Logger
}

func NewServer(cfg config.Config, pool *pgxpool.Pool, logger zerolog.Logger) *Server {
	return &Server{cfg: cfg, pool: pool, logger: logger}
}

func (s *Server) Router() http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(10 * time.Minute))
	r.Use(s.withUser)

	r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true}`))
	})

	r.Get("/models", s.handleListModels)
	r.Post("/runs/start", s.handleRunStart)
	r.Get("/runs/{runID}/stream", s.handleRunStream)
	r.Get("/runs/{runID}/steps", s.handleListRunSteps)
	r.Get("/runs/{runID}/sources", s.handleListRunSources)
	r.Get("/runs/{runID}/snippets", s.handleListRunSnippets)
	r.Get("/chats", s.handleListChats)
	r.Get("/chats/{chatID}", s.handleGetChat)
	r.Delete("/chats/{chatID}", s.handleDeleteChat)
	r.Get("/chats/{chatID}/messages", s.handleListMessages)
	r.Get("/bookmarks", s.handleListBookmarks)
	r.Post("/bookmarks/{chatID}", s.handleCreateBookmark)
	r.Delete("/bookmarks/{chatID}", s.handleDeleteBookmark)

	return r
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeErr(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]any{"error": msg})
}

// userCtxKey is a private type to avoid collisions.
type userCtxKey struct{}

type User struct {
	ID             string
	Email          string
	PreferredModel string
}

func userFromCtx(ctx context.Context) *User {
	v := ctx.Value(userCtxKey{})
	if v == nil {
		return nil
	}
	if u, ok := v.(*User); ok {
		return u
	}
	return nil
}
