// Package tenancy defines the boundary between the open-source local account
// and the cloud organization model.
package tenancy

import (
	"context"

	"github.com/OpenSlash/tandemesh/internal/store"
)

// Resolver answers workspace access questions. The open-source build uses
// Local; the cloud build provides an organization-backed implementation.
type Resolver interface {
	CanAccess(ctx context.Context, user, workspaceID string) bool
	Role(ctx context.Context, user, workspaceID string) string
	Workspaces(ctx context.Context, user string) ([]string, error)
	IsLocal() bool
}

// Local grants the single local account owner access to every workspace.
type Local struct {
	Store *store.Store
	User  string
}

func (l Local) IsLocal() bool { return true }

func (l Local) CanAccess(_ context.Context, user, _ string) bool {
	return l.User == "" || user == l.User
}

func (l Local) Role(_ context.Context, user, _ string) string {
	if l.User == "" || user == l.User {
		return "owner"
	}
	return ""
}

func (l Local) Workspaces(ctx context.Context, _ string) ([]string, error) {
	items, err := l.Store.ListWorkspaces(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]string, 0, len(items))
	for _, item := range items {
		out = append(out, item.ID)
	}
	return out, nil
}
