package httpapi

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type runStartReq struct {
	ChatID string `json:"chat_id"`
	Query  string `json:"query"`
	Model  string `json:"model"`
}

type runStartResp struct {
	ChatID string `json:"chat_id"`
	RunID  string `json:"run_id"`
}

type sseHub struct {
	mu   sync.Mutex
	pubs map[string]*runBroadcaster
}

type runBroadcaster struct {
	mu   sync.Mutex
	subs map[chan []byte]struct{}
}

func (b *runBroadcaster) subscribe() chan []byte {
	b.mu.Lock()
	defer b.mu.Unlock()
	ch := make(chan []byte, 64)
	if b.subs == nil {
		b.subs = map[chan []byte]struct{}{}
	}
	b.subs[ch] = struct{}{}
	return ch
}

func (b *runBroadcaster) unsubscribe(ch chan []byte) {
	b.mu.Lock()
	defer b.mu.Unlock()
	delete(b.subs, ch)
	close(ch)
}

func (b *runBroadcaster) publish(payload []byte) {
	b.mu.Lock()
	defer b.mu.Unlock()
	for ch := range b.subs {
		select {
		case ch <- payload:
		default:
			// drop if slow
		}
	}
}

var globalHub = &sseHub{pubs: map[string]*runBroadcaster{}}

func (h *sseHub) get(runID string) *runBroadcaster {
	h.mu.Lock()
	defer h.mu.Unlock()
	b, ok := h.pubs[runID]
	if !ok {
		b = &runBroadcaster{subs: map[chan []byte]struct{}{}}
		h.pubs[runID] = b
	}
	return b
}

func (h *sseHub) subscribe(runID string) chan []byte {
	b := h.get(runID)
	return b.subscribe()
}

func (h *sseHub) unsubscribe(runID string, ch chan []byte) {
	h.mu.Lock()
	b := h.pubs[runID]
	h.mu.Unlock()
	if b == nil {
		return
	}

	b.mu.Lock()
	_, exists := b.subs[ch]
	if exists {
		delete(b.subs, ch)
	}
	remaining := len(b.subs)
	b.mu.Unlock()
	if exists {
		close(ch)
	}

	if remaining > 0 {
		return
	}

	h.mu.Lock()
	if current, ok := h.pubs[runID]; ok && current == b {
		delete(h.pubs, runID)
	}
	h.mu.Unlock()
}

func (h *sseHub) publish(runID string, payload []byte) {
	h.mu.Lock()
	b := h.pubs[runID]
	h.mu.Unlock()
	if b == nil {
		return
	}
	b.publish(payload)
}

func (s *Server) handleRunStart(w http.ResponseWriter, r *http.Request) {
	user := userFromCtx(r.Context())
	if user == nil {
		writeErr(w, http.StatusUnauthorized, "auth required")
		return
	}

	var req runStartReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.logger.Warn().Err(err).Msg("decode run start request")
		writeErr(w, http.StatusBadRequest, "invalid json")
		return
	}
	q := strings.TrimSpace(req.Query)
	if q == "" {
		s.logger.Info().Msg("empty query in run start")
		writeErr(w, http.StatusBadRequest, "query is required")
		return
	}

	model := strings.TrimSpace(req.Model)
	if model == "" {
		model = user.PreferredModel
	}

	chatID := strings.TrimSpace(req.ChatID)
	if chatID == "" {
		chatID = uuid.New().String()
		_, err := s.pool.Exec(r.Context(), `insert into chats(id,user_id,title) values ($1,$2,$3)`, chatID, user.ID, q)
		if err != nil {
			s.logger.Error().Err(err).Msg("create chat failed")
			writeErr(w, http.StatusInternalServerError, fmt.Sprintf("create chat: %v", err))
			return
		}
	} else {
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
	}

	runID := uuid.New().String()
	_, err := s.pool.Exec(r.Context(), `insert into runs(id, chat_id, user_id, model, status) values ($1,$2,$3,$4,'running')`, runID, chatID, user.ID, model)
	if err != nil {
		s.logger.Error().Err(err).Msg("create run failed")
		writeErr(w, http.StatusInternalServerError, fmt.Sprintf("create run: %v", err))
		return
	}

	_, _ = s.pool.Exec(r.Context(), `insert into messages(chat_id,user_id,role,content,run_id) values ($1,$2,'user',$3,$4)`, chatID, user.ID, q, runID)
	_, _ = s.pool.Exec(r.Context(), `update chats set updated_at=now() where id=$1`, chatID)

	s.logger.Info().Str("run_id", runID).Str("chat_id", chatID).Str("model", model).Msg("run started")

	go s.runPipeline(context.Background(), runID, q, model)

	writeJSON(w, http.StatusOK, runStartResp{ChatID: chatID, RunID: runID})
}

func (s *Server) handleRunStream(w http.ResponseWriter, r *http.Request) {
	runID := chi.URLParam(r, "runID")
	if runID == "" {
		writeErr(w, http.StatusBadRequest, "runID is required")
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	flusher, ok := w.(http.Flusher)
	if !ok {
		writeErr(w, http.StatusInternalServerError, "stream unsupported")
		return
	}

	sub := globalHub.subscribe(runID)
	defer globalHub.unsubscribe(runID, sub)

	// replay existing steps
	rows, err := s.pool.Query(r.Context(), `select type, title, payload, created_at from run_steps where run_id=$1 order by created_at asc`, runID)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var typ, title string
			var payload []byte
			var created time.Time
			_ = rows.Scan(&typ, &title, &payload, &created)
			s.writeSSE(w, "step", map[string]any{"type": typ, "title": title, "payload": json.RawMessage(payload), "created_at": created})
		}
		flusher.Flush()
	}

	keepAlive := time.NewTicker(15 * time.Second)
	defer keepAlive.Stop()

	bufw := bufio.NewWriter(w)
	for {
		select {
		case <-r.Context().Done():
			_ = bufw.Flush()
			return
		case payload := <-sub:
			_, _ = bufw.Write(payload)
			_ = bufw.Flush()
			flusher.Flush()
		case <-keepAlive.C:
			_, _ = bufw.WriteString(": keep-alive\n\n")
			_ = bufw.Flush()
			flusher.Flush()
		}
	}
}

func (s *Server) writeSSE(w http.ResponseWriter, event string, data any) {
	b, _ := json.Marshal(data)
	_, _ = w.Write([]byte("event: " + event + "\n"))
	_, _ = w.Write([]byte("data: "))
	_, _ = w.Write(b)
	_, _ = w.Write([]byte("\n\n"))
}

func (s *Server) publishStep(ctx context.Context, runID, typ, title string, payload any) {
	jb, _ := json.Marshal(payload)
	_, _ = s.pool.Exec(ctx, `insert into run_steps(run_id,type,title,payload) values ($1,$2,$3,$4)`, runID, typ, title, jb)

	frame, _ := json.Marshal(map[string]any{"type": typ, "title": title, "payload": payload, "created_at": time.Now()})
	sse := []byte("event: step\n" + "data: " + string(frame) + "\n\n")
	globalHub.publish(runID, sse)
}

func (s *Server) publishAnswerDelta(runID string, delta string) {
	frame, _ := json.Marshal(map[string]any{"delta": delta})
	sse := []byte("event: answer.delta\n" + "data: " + string(frame) + "\n\n")
	globalHub.publish(runID, sse)
}

func (s *Server) publishFinal(runID string, answer string, model string) {
	frame, _ := json.Marshal(map[string]any{"answer": answer, "model": model})
	sse := []byte("event: answer.final\n" + "data: " + string(frame) + "\n\n")
	globalHub.publish(runID, sse)
}

func (s *Server) publishRunError(runID string, message string) {
	frame, _ := json.Marshal(map[string]any{"error": message})
	sse := []byte("event: run.error\n" + "data: " + string(frame) + "\n\n")
	globalHub.publish(runID, sse)
}
