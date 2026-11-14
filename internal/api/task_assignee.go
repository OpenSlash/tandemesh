package api

import (
	"context"

	"github.com/OpenSlash/tandemesh/internal/domain"
)

// enrichTaskAssignees maps platform assignee logins to workspace members.
func (s *Server) enrichTaskAssignees(ctx context.Context, tasks []*domain.Task) {
	cache := map[string]string{}
	for _, task := range tasks {
		if task == nil || task.Assignee == "" {
			continue
		}
		key := task.WorkspaceID + "|" + task.Provider + "|" + task.Assignee
		if memberID, ok := cache[key]; ok {
			task.AssigneeMemberID = memberID
			continue
		}
		identity, err := s.store.ResolveExternalIdentity(ctx, task.WorkspaceID, task.Provider, task.Assignee)
		memberID := ""
		if err == nil && identity != nil {
			memberID = identity.MemberID
		}
		cache[key] = memberID
		task.AssigneeMemberID = memberID
	}
}
