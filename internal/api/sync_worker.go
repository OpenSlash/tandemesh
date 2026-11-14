package api

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/OpenSlash/tandemesh/internal/domain"
	"github.com/OpenSlash/tandemesh/internal/integration"
	"github.com/OpenSlash/tandemesh/internal/store"
)

// syncOwner identifies this control-plane process for shared leases.
func syncOwner() string {
	host, _ := os.Hostname()
	return fmt.Sprintf("%s:%d", host, os.Getpid())
}

// SyncLoop periodically pulls connected task sources as a webhook fallback.
func (s *Server) SyncLoop(ctx context.Context) {
	interval := 5 * time.Minute
	if raw := os.Getenv("ORCA_SYNC_INTERVAL"); raw != "" {
		if d, err := time.ParseDuration(raw); err == nil && d >= 30*time.Second {
			interval = d
		}
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.SyncConnections(ctx)
		}
	}
}

// SyncConnections refreshes tasks from every enabled connection.
func (s *Server) SyncConnections(ctx context.Context) int {
	owner := syncOwner()
	locked, err := s.store.AcquireSyncLock(ctx, "sync:connections", owner, 30*time.Minute)
	if err != nil || !locked {
		return 0
	}
	defer s.store.ReleaseSyncLock(ctx, "sync:connections", owner)
	_, _ = s.store.CleanupOAuthNonces(ctx)
	conns, err := s.store.ListAllConnections(ctx)
	if err != nil {
		return 0
	}
	synced := 0
	for _, conn := range conns {
		if !conn.Enabled || conn.Status == domain.ConnectionRevoked {
			continue
		}
		ad, ok := s.integrations.Get(conn.Provider)
		if !ok || ad.Reader == nil {
			continue
		}
		cred := s.providerCredential(ctx, conn.Provider, conn.ID)
		if cred.AccessToken == "" {
			continue
		}
		cursor, _ := s.store.GetSyncCursor(ctx, conn.ID)
		customFields := map[string]string{}
		if automations, err := s.store.ListAutomations(ctx, conn.WorkspaceID); err == nil {
			for _, automation := range automations {
				if automation.ConnectionID != conn.ID || automation.Writeback == nil {
					continue
				}
				if raw, ok := automation.Writeback["customFields"].(string); ok {
					customFields = parseCustomFields(raw)
				}
				break
			}
		}
		opts := integration.ListOptions{Instance: conn.Instance, Limit: 100, Fields: customFields}
		if !cursor.Cursor.IsZero() {
			opts.Since = cursor.Cursor
		}
		items, err := ad.Reader.ListAssigned(ctx, cred, opts)
		if err != nil {
			cursor.LastError = err.Error()
			cursor.LastSyncedAt = time.Now().UTC()
			_ = s.store.SaveSyncCursor(ctx, cursor)
			if integration.IsAuthError(err) {
				_ = s.store.SetConnectionStatus(ctx, conn.ID, domain.ConnectionNeedsReauth)
			}
			continue
		}
		stateMap := map[string]string{}
		conflict := "platform"
		if automations, err := s.store.ListAutomations(ctx, conn.WorkspaceID); err == nil {
			for _, automation := range automations {
				if automation.ConnectionID == conn.ID {
					stateMap = automation.StateMap
					conflict = conflictPolicy(automation)
					break
				}
			}
		}
		latest := cursor.Cursor
		for _, item := range items {
			if !cursor.Cursor.IsZero() && !item.Updated.After(cursor.Cursor) {
				continue
			}
			if existing, err := s.store.GetTask(ctx, store.TaskID(item.Ref)); err == nil && existing != nil && existing.Revision == item.Ref.Revision && existing.PlatformState == item.Status && existing.PlatformStateGroup == item.StatusGroup {
				if item.Updated.After(latest) {
					latest = item.Updated
				}
				continue
			}
			snap := domain.Snapshot{
				Title: item.Title, Body: item.Body, Labels: item.Labels,
				Author: item.Author, Assignee: item.Assignee,
				Status: item.Status, StatusGroup: item.StatusGroup,
			}
			task, err := s.store.EnsureTaskIn(ctx, conn.WorkspaceID, item.Ref, snap)
			if err != nil {
				continue
			}
			s.applyPlatformState(ctx, task, snap, stateMap, conflict)
			attachments := make([]*domain.TaskAttachment, 0, len(item.Attachments))
			for _, attachment := range item.Attachments {
				attachments = append(attachments, &domain.TaskAttachment{
					ID: "att_" + task.ID + "_" + attachment.ID, TaskID: task.ID, Provider: conn.Provider,
					ExternalID: attachment.ID, Name: attachment.Name, URL: attachment.URL,
					Mime: attachment.Mime, Size: attachment.Size, Author: attachment.Author,
					CreatedAt: attachment.CreatedAt,
				})
			}
			_ = s.store.ReplaceTaskAttachments(ctx, task.ID, conn.Provider, "issue", attachments)
			_ = s.store.ReplaceTaskFields(ctx, task.ID, conn.Provider, item.Fields)
			s.applyAssignmentRules(ctx, task)
			s.syncPlatformComments(ctx, ad, cred, conn.Provider, task.ID, item.Ref)
			_ = s.store.TouchTaskComments(ctx, task.ID)
			if item.Updated.After(latest) {
				latest = item.Updated
			}
			synced++
		}
		if _, ok := ad.Reader.(integration.CommentReader); ok {
			if pending, err := s.store.ListTasksForComments(ctx, conn.Provider, conn.Instance, time.Now().Add(-15*time.Minute), 20); err == nil {
				for _, task := range pending {
					ref := domain.WorkRef{Provider: task.Provider, Instance: task.Instance, Type: task.WorkType, ID: task.WorkID}
					s.syncPlatformComments(ctx, ad, cred, conn.Provider, task.ID, ref)
					_ = s.store.TouchTaskComments(ctx, task.ID)
				}
			}
		}
		// Full-fetch providers can reconcile deletions once a day. Providers that
		// only return updated rows (e.g. GitHub since) must not mark missing rows deleted.
		fullFetch := opts.Since.IsZero() || conn.Provider == "plane"
		if fullFetch && len(items) < 100 && time.Since(cursor.LastFullAt) >= 24*time.Hour {
			if all, err := s.store.ListTaskWorkIDs(ctx, conn.Provider, conn.Instance); err == nil {
				seen := map[string]bool{}
				for _, item := range items {
					seen[item.Ref.ID] = true
				}
				for workID, taskID := range all {
					if !seen[workID] {
						_ = s.store.SetTaskPlatformState(ctx, taskID, "deleted", "cancelled", "")
					}
				}
				for _, item := range items {
					if taskID, ok := all[item.Ref.ID]; ok {
						s.syncPlatformComments(ctx, ad, cred, conn.Provider, taskID, item.Ref)
					}
				}
				cursor.LastFullAt = time.Now().UTC()
			}
		}
		cursor.Cursor = latest
		cursor.LastSyncedAt = time.Now().UTC()
		cursor.LastError = ""
		_ = s.store.SaveSyncCursor(ctx, cursor)
		if bindings, err := s.store.ListBindings(ctx, conn.WorkspaceID, conn.ID); err == nil {
			for _, binding := range bindings {
				if !binding.Enabled || binding.ExternalID == "" || binding.ExternalID == conn.Instance {
					continue
				}
				synced += s.syncBinding(ctx, conn, ad, cred, binding)
			}
		}
	}
	if synced > 0 {
		s.log.Info("task sync", "items", synced, "connections", len(conns), "interval", os.Getenv("ORCA_SYNC_INTERVAL"))
	}
	return synced
}

func (s *Server) syncBinding(ctx context.Context, conn *domain.Connection, ad *integration.Integration, cred integration.Credential, binding *domain.ResourceBinding) int {
	cred = s.accountCredential(ctx, conn.Provider, binding.AccountID, cred)
	cursor, _ := s.store.GetBindingSyncCursor(ctx, conn.ID, binding.ID)
	opts := integration.ListOptions{Instance: binding.ExternalID, Limit: 100}
	if !cursor.Cursor.IsZero() {
		opts.Since = cursor.Cursor
	}
	items, err := ad.Reader.ListAssigned(ctx, cred, opts)
	if err != nil {
		cursor.LastError = err.Error()
		cursor.LastSyncedAt = time.Now().UTC()
		_ = s.store.SaveBindingSyncCursor(ctx, cursor)
		_ = s.store.SetBindingSyncState(ctx, binding.ID, cursor.LastSyncedAt, err.Error())
		s.log.Warn("binding sync", "connection", conn.ID, "binding", binding.ID, "err", err)
		return 0
	}
	stateMap := map[string]string{}
	conflict := "platform"
	if automations, err := s.store.ListAutomations(ctx, conn.WorkspaceID); err == nil {
		for _, automation := range automations {
			if automation.ConnectionID == conn.ID {
				stateMap = automation.StateMap
				conflict = conflictPolicy(automation)
				break
			}
		}
	}
	synced := 0
	latest := cursor.Cursor
	for _, item := range items {
		if !cursor.Cursor.IsZero() && !item.Updated.After(cursor.Cursor) {
			continue
		}
		snap := domain.Snapshot{
			Title: item.Title, Body: item.Body, Labels: item.Labels,
			Author: item.Author, Assignee: item.Assignee,
			Status: item.Status, StatusGroup: item.StatusGroup,
		}
		task, err := s.store.EnsureTaskIn(ctx, conn.WorkspaceID, item.Ref, snap)
		if err != nil {
			continue
		}
		s.applyPlatformState(ctx, task, snap, stateMap, conflict)
		s.applyAssignmentRules(ctx, task)
		if item.Updated.After(latest) {
			latest = item.Updated
		}
		synced++
	}
	cursor.Cursor = latest
	cursor.LastSyncedAt = time.Now().UTC()
	cursor.LastError = ""
	_ = s.store.SaveBindingSyncCursor(ctx, cursor)
	_ = s.store.SetBindingSyncState(ctx, binding.ID, cursor.LastSyncedAt, "")
	return synced
}

func envInt(name string, fallback int) int {
	if raw := os.Getenv(name); raw != "" {
		if value, err := strconv.Atoi(raw); err == nil {
			return value
		}
	}
	return fallback
}

func (s *Server) syncPlatformComments(ctx context.Context, ad *integration.Integration, cred integration.Credential, provider, taskID string, ref domain.WorkRef) {
	reader, ok := ad.Reader.(integration.CommentReader)
	if !ok {
		return
	}
	comments, err := reader.ListComments(ctx, cred, ref)
	if err != nil {
		return
	}
	workspaceID := ""
	if task, err := s.store.GetTask(ctx, taskID); err == nil && task != nil {
		workspaceID = task.WorkspaceID
	}
	thread, _ := s.store.EnsureCommentThread(ctx, &domain.CommentThread{
		ID: "ct_" + taskID + "_" + provider, WorkspaceID: workspaceID, TaskID: taskID,
		Provider: provider, ExternalID: provider + ":" + ref.Instance + ":" + ref.Type + ":" + ref.ID,
		Source: provider,
	})
	threadID := ""
	if thread != nil {
		threadID = thread.ID
	}
	commentAttachments := make([]*domain.TaskAttachment, 0)
	for _, comment := range comments {
		if workspaceID != "" && comment.Author != "" {
			_, _ = s.store.EnsureExternalIdentity(ctx, workspaceID, provider, comment.Author, comment.Author, "")
		}
		_ = s.store.UpsertExternalComment(ctx, &domain.ExternalComment{
			ID:               "pc_" + taskID + "_" + comment.ID,
			TaskID:           taskID,
			Provider:         provider,
			ExternalID:       comment.ID,
			Author:           comment.Author,
			Body:             comment.Body,
			ThreadID:         threadID,
			AuthorExternalID: comment.Author,
			Direction:        "inbound",
			CreatedAt:        comment.CreatedAt,
		})
		for _, attachment := range comment.Attachments {
			commentAttachments = append(commentAttachments, &domain.TaskAttachment{
				ID: "att_" + taskID + "_" + attachment.ID, TaskID: taskID, Provider: provider,
				ExternalID: attachment.ID, Name: attachment.Name, URL: attachment.URL,
				Mime: attachment.Mime, Size: attachment.Size, Author: attachment.Author,
				Source: "comment",
			})
		}
	}
	_ = s.store.ReplaceTaskAttachments(ctx, taskID, provider, "comment", commentAttachments)
}

func parseCustomFields(raw string) map[string]string {
	out := map[string]string{}
	for _, part := range strings.Split(raw, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		key, name, ok := strings.Cut(part, "=")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		name = strings.TrimSpace(name)
		if key != "" && name != "" {
			out[key] = name
		}
	}
	return out
}
