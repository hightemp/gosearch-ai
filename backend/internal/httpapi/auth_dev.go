package httpapi

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/google/uuid"
)

// withUser attaches a user to the request context.
//
// Сейчас сделан dev-режим: если нет Authorization, создаём/используем одного
// пользователя dev@local. Это позволяет развивать UI/агентов до внедрения полноценного auth.
func (s *Server) withUser(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authz := strings.TrimSpace(r.Header.Get("Authorization"))
		_ = authz // TODO: внедрить JWT auth

		if s.cfg.Env != "dev" {
			// Пока без auth в prod запретим все, кроме healthz.
			if r.URL.Path != "/healthz" {
				writeErr(w, http.StatusUnauthorized, "auth required")
				return
			}
			next.ServeHTTP(w, r)
			return
		}

		u, err := s.ensureDevUser(r.Context())
		if err != nil {
			writeErr(w, http.StatusInternalServerError, fmt.Sprintf("dev user: %v", err))
			return
		}

		ctx := context.WithValue(r.Context(), userCtxKey{}, u)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (s *Server) ensureDevUser(ctx context.Context) (*User, error) {
	const email = "dev@local"

	var id uuid.UUID
	var preferred string
	err := s.pool.QueryRow(ctx, `select id, preferred_model from users where email=$1`, email).Scan(&id, &preferred)
	if err == nil {
		return &User{ID: id.String(), Email: email, PreferredModel: preferred}, nil
	}

	newID := uuid.New()
	preferred = s.cfg.OpenRouterModels[0]
	_, err = s.pool.Exec(ctx, `insert into users(id, email, password_hash, preferred_model) values ($1,$2,'', $3)`, newID, email, preferred)
	if err != nil {
		// could be concurrent: retry select
		var id2 uuid.UUID
		var preferred2 string
		err2 := s.pool.QueryRow(ctx, `select id, preferred_model from users where email=$1`, email).Scan(&id2, &preferred2)
		if err2 == nil {
			return &User{ID: id2.String(), Email: email, PreferredModel: preferred2}, nil
		}
		return nil, err
	}

	return &User{ID: newID.String(), Email: email, PreferredModel: preferred}, nil
}
