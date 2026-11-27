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
		if memberID == "" {
			memberID = s.memberForAccount(ctx, task.WorkspaceID, task.Provider, task.Assignee)
		}
		cache[key] = memberID
		task.AssigneeMemberID = memberID
	}
}

// memberForAccount finds the member whose linked platform account (usually
// recorded when they connected it) is login.
func (s *Server) memberForAccount(ctx context.Context, workspaceID, provider, login string) string {
	items, err := s.store.ListMemberIdentities(ctx, workspaceID, "")
	if err != nil {
		return ""
	}
	for _, item := range items {
		if item.Provider == provider && (item.Login == login || item.ExternalID == login) {
			return item.MemberID
		}
	}
	return ""
}
