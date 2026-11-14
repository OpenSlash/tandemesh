package api

import (
	"context"
	"net/http"

	"github.com/OpenSlash/tandemesh/internal/domain"
)

func (s *Server) actingUser(r *http.Request) string {
	if user := userOf(r); user != "" {
		return user
	}
	if s.cfg.User != "" {
		return s.cfg.User
	}
	return "me"
}

// applyTaskInbox overlays each task with the acting user's triage state.
// Missing rows default to inbox/unread.
func (s *Server) applyTaskInbox(ctx context.Context, user string, tasks []*domain.Task) error {
	ids := make([]string, 0, len(tasks))
	for _, task := range tasks {
		task.InboxState = "inbox"
		task.InboxReadAt = nil
		ids = append(ids, task.ID)
	}
	states, err := s.store.TaskInboxStates(ctx, user, ids)
	if err != nil {
		return err
	}
	for _, task := range tasks {
		if state, ok := states[task.ID]; ok {
			task.InboxState = state.State
			task.InboxReadAt = state.ReadAt
		}
	}
	return nil
}
