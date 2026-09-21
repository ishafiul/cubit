package http

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/ishaf/cubit/internal/domain"
	"github.com/ishaf/cubit/internal/usecase"
)

// SSELogStreamer streams deployment logs live to connected web clients.
type SSELogStreamer struct {
	depRepo usecase.DeploymentRepository
}

// NewSSELogStreamer creates a new SSELogStreamer.
func NewSSELogStreamer(depRepo usecase.DeploymentRepository) *SSELogStreamer {
	return &SSELogStreamer{depRepo: depRepo}
}

// HandleStream streams new deployment logs over Server-Sent Events.
func (s *SSELogStreamer) HandleStream(w http.ResponseWriter, r *http.Request) {
	depID := chi.URLParam(r, "id")
	if depID == "" {
		http.Error(w, "missing deployment id", http.StatusBadRequest)
		return
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Minute)
	defer cancel()

	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	lastCount := 0

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			logs, err := s.depRepo.GetLogs(ctx, depID)
			if err != nil {
				return
			}

			if len(logs) > lastCount {
				for _, entry := range logs[lastCount:] {
					data, _ := json.Marshal(entry)
					fmt.Fprintf(w, "data: %s\n\n", data)
				}
				lastCount = len(logs)
				flusher.Flush()
			}

			dep, err := s.depRepo.GetByID(ctx, depID)
			if err == nil && (dep.Status == domain.DeploymentStatusActive || dep.Status == domain.DeploymentStatusFailed) {
				fmt.Fprintf(w, "event: complete\ndata: {\"status\":\"%s\"}\n\n", dep.Status)
				flusher.Flush()
				return
			}
		}
	}
}
