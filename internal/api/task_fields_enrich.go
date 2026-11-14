package api

import (
	"context"

	"github.com/OpenSlash/tandemesh/internal/domain"
)

// enrichTaskFields attaches structured provider fields to task responses.
func (s *Server) enrichTaskFields(ctx context.Context, tasks []*domain.Task) {
	for _, task := range tasks {
		if task == nil {
			continue
		}
		items, err := s.store.ListTaskFields(ctx, task.ID)
		if err != nil || len(items) == 0 {
			continue
		}
		fields := map[string][]string{}
		for _, item := range items {
			fields[item.Key] = append(fields[item.Key], item.Value)
		}
		task.Fields = fields
	}
}
