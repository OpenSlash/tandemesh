package api

import (
	"context"
	"strings"

	"github.com/OpenSlash/tandemesh/internal/domain"
	"github.com/OpenSlash/tandemesh/internal/integration"
)

// backfillBindingNames replaces UUID-like binding names with catalog names.
func (s *Server) backfillBindingNames(ctx context.Context, conn *domain.Connection, bindings []*domain.ResourceBinding) {
	if conn == nil || len(bindings) == 0 {
		return
	}
	ad, ok := s.integrations.Get(conn.Provider)
	if !ok {
		return
	}
	reader, ok := ad.Reader.(integration.CatalogReader)
	if !ok {
		return
	}
	cred := s.providerCredential(ctx, conn.Provider, conn.ID)
	if cred.AccessToken == "" {
		return
	}
	byWorkspace := map[string][]*domain.ResourceBinding{}
	for _, binding := range bindings {
		if binding.Name != "" && binding.Name != binding.ExternalID {
			continue
		}
		parts := strings.SplitN(binding.ExternalID, "/", 2)
		if len(parts) != 2 {
			continue
		}
		byWorkspace[parts[0]] = append(byWorkspace[parts[0]], binding)
	}
	for providerWorkspace, items := range byWorkspace {
		projects, err := reader.ListProjects(ctx, cred, providerWorkspace)
		if err != nil {
			continue
		}
		names := map[string]string{}
		for _, project := range projects {
			names[project.ID] = project.Name
		}
		for _, binding := range items {
			parts := strings.SplitN(binding.ExternalID, "/", 2)
			if len(parts) != 2 {
				continue
			}
			if name := names[parts[1]]; name != "" {
				binding.Name = name
				_ = s.store.UpsertBinding(ctx, binding)
			}
		}
	}
}
