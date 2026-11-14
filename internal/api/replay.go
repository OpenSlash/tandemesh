package api

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/OpenSlash/tandemesh/internal/domain"
)

type replayContextKey struct{}

func isReplayRequest(r *http.Request) bool {
	value, _ := r.Context().Value(replayContextKey{}).(bool)
	return value
}

// replayWebhookDelivery re-processes a stored delivery without a fresh signature.
func (s *Server) replayWebhookDelivery(w http.ResponseWriter, r *http.Request) {
	d, err := s.store.GetWebhookDelivery(r.Context(), chi.URLParam(r, "id"))
	if err != nil || d == nil {
		writeErr(w, http.StatusNotFound, "E_NOT_FOUND", "delivery not found")
		return
	}
	b, err := s.store.GetAutomation(r.Context(), d.AutomationID)
	if err != nil || b == nil {
		writeErr(w, http.StatusNotFound, "E_AUTOMATION", "automation not found")
		return
	}
	if !s.requireWorkspace(w, r, b.WorkspaceID) {
		return
	}
	code, result := s.replayDelivery(r.Context(), d)
	writeJSON(w, http.StatusOK, map[string]any{
		"delivery": d,
		"status":   code,
		"result":   result,
	})
}

// replayDelivery feeds a stored body through the normal webhook handler.
func (s *Server) replayDelivery(ctx context.Context, d *domain.WebhookDelivery) (int, string) {
	req := httptest.NewRequest(http.MethodPost, "/webhooks/"+d.Provider+"/"+d.AutomationID, bytes.NewReader([]byte(d.Body)))
	routeCtx := chi.NewRouteContext()
	routeCtx.URLParams.Add("provider", d.Provider)
	routeCtx.URLParams.Add("automation", d.AutomationID)
	ctx = context.WithValue(ctx, chi.RouteCtxKey, routeCtx)
	ctx = context.WithValue(ctx, replayContextKey{}, true)
	req = req.WithContext(ctx)
	req.Header.Set("X-Orca-Replay", d.ID)

	rec := httptest.NewRecorder()
	s.webhook(rec, req)
	return rec.Code, rec.Body.String()
}

// RetryDeliveriesLoop retries failed deliveries once after a short delay.
func (s *Server) RetryDeliveriesLoop(ctx context.Context) {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			owner := syncOwner()
			locked, lockErr := s.store.AcquireSyncLock(ctx, "retry:deliveries", owner, 2*time.Minute)
			if lockErr != nil || !locked {
				continue
			}
			items, err := s.store.ListRetryDeliveries(ctx, 5, 20)
			if err != nil {
				continue
			}
			for _, delivery := range items {
				code, result := s.replayDelivery(ctx, delivery)
				if code < 300 {
					_ = s.store.MarkWebhookDeliveryStatus(ctx, delivery.ID, "retried")
				} else {
					_ = s.store.BumpDeliveryRetry(ctx, delivery.ID)
				}
				s.log.Info("retry webhook delivery", "id", delivery.ID, "attempt", delivery.Attempts+1, "code", code, "result", result)
			}
			s.store.ReleaseSyncLock(ctx, "retry:deliveries", owner)
		}
	}
}
