package httpapi

import "net/http"

func (s *Server) handleListModels(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"models": s.cfg.OpenRouterModels,
	})
}
