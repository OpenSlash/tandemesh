package api

import (
	"context"
	"os"
	"sync"
	"time"
)

// debounceState suppresses bursts of webhook events for the same work item.
type debounceState struct {
	mu sync.Mutex
	at map[string]time.Time
}

func debounceWindow() time.Duration {
	if raw := os.Getenv("ORCA_WEBHOOK_DEBOUNCE"); raw != "" {
		if d, err := time.ParseDuration(raw); err == nil && d >= 0 {
			return d
		}
	}
	return 3 * time.Second
}

// claimBurst returns true when a matching event was seen within the window.
// The claim lives in the database so multiple control-plane instances share it.
func (s *Server) claimBurst(ctx context.Context, key string) bool {
	window := debounceWindow()
	if window <= 0 {
		return false
	}
	if burst, err := s.store.ClaimDebounce(ctx, key, window); err == nil {
		return burst
	}
	now := time.Now()
	s.debounceState.mu.Lock()
	defer s.debounceState.mu.Unlock()
	if last, ok := s.debounceState.at[key]; ok && now.Sub(last) < window {
		s.debounceState.at[key] = now
		return true
	}
	s.debounceState.at[key] = now
	return false
}
