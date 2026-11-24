package api

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/oklog/ulid/v2"

	"github.com/OpenSlash/tandemesh/internal/domain"
	"github.com/OpenSlash/tandemesh/internal/integration"
	"github.com/OpenSlash/tandemesh/internal/store"
)

type importJiraRequest struct {
	ConnectionID string `json:"connectionId"`
	JQL          string `json:"jql"`
	Limit        int    `json:"limit"`
	StartAt      int    `json:"startAt"`
}

func (s *Server) createJiraImport(w http.ResponseWriter, r *http.Request) {
	workspaceID := chi.URLParam(r, "workspace")
	if workspaceID == "" {
		workspaceID = "ws_default"
	}
	if !s.requireWorkspace(w, r, workspaceID) {
		return
	}
	var req importJiraRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeErr(w, http.StatusBadRequest, "E_BODY", err.Error())
		return
	}
	req.ConnectionID = strings.TrimSpace(req.ConnectionID)
	req.JQL = strings.TrimSpace(req.JQL)
	if req.Limit <= 0 || req.Limit > 500 {
		req.Limit = 100
	}
	if req.StartAt < 0 {
		req.StartAt = 0
	}
	if req.ConnectionID == "" || req.JQL == "" {
		writeErr(w, http.StatusBadRequest, "E_FIELDS", "connectionId and jql are required")
		return
	}
	conn, err := s.store.GetConnection(r.Context(), req.ConnectionID)
	if err != nil || conn == nil || conn.WorkspaceID != workspaceID {
		writeErr(w, http.StatusNotFound, "E_CONNECTION", "connection not found")
		return
	}
	ad, ok := s.integrations.Get(conn.Provider)
	if !ok {
		writeErr(w, http.StatusBadRequest, "E_PROVIDER", "provider not registered")
		return
	}
	if _, ok := ad.Reader.(integration.ImportReader); !ok {
		writeErr(w, http.StatusBadRequest, "E_UNSUPPORTED", "provider does not support bulk import")
		return
	}
	batch := &domain.ImportBatch{
		ID: "imp_" + ulid.Make().String(), WorkspaceID: workspaceID, Provider: conn.Provider,
		ConnectionID: conn.ID, Query: req.JQL, Status: "queued", NextStartAt: req.StartAt,
		Limit: req.Limit, CreatedAt: time.Now().UTC(),
	}
	if err := s.store.CreateImportBatch(r.Context(), batch); err != nil {
		writeErr(w, http.StatusInternalServerError, "E_DB", err.Error())
		return
	}
	go s.runJiraImport(batch.ID)
	writeJSON(w, http.StatusAccepted, map[string]any{"batch": batch})
}

// runJiraImport pages through the saved JQL and projects tasks, attachments and
// comments until the batch limit or the end of the result set.
func (s *Server) runJiraImport(batchID string) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()
	batch, err := s.store.GetImportBatch(ctx, batchID)
	if err != nil || batch == nil {
		return
	}
	if batch.Status == "running" || batch.Status == "rolled_back" {
		return
	}
	conn, err := s.store.GetConnection(ctx, batch.ConnectionID)
	if err != nil || conn == nil {
		batch.Status, batch.Error = "failed", "connection not found"
		_ = s.store.FinishImportBatch(ctx, batch)
		return
	}
	releaseConn := s.acquireImportConn(conn.ID)
	defer releaseConn()
	if s.importSlots != nil {
		s.importSlots <- struct{}{}
		defer func() { <-s.importSlots }()
	}
	ad, ok := s.integrations.Get(conn.Provider)
	if !ok {
		batch.Status, batch.Error = "failed", "provider not registered"
		_ = s.store.FinishImportBatch(ctx, batch)
		return
	}
	reader, ok := ad.Reader.(integration.ImportReader)
	if !ok {
		batch.Status, batch.Error = "failed", "provider does not support bulk import"
		_ = s.store.FinishImportBatch(ctx, batch)
		return
	}
	_ = s.store.SetImportStatus(ctx, batch.ID, "running", "")
	cred := s.providerCredential(ctx, conn.Provider, conn.ID)
	process := func(items []integration.WorkItem) {
		for _, item := range items {
			if item.Ref.Instance == "" {
				item.Ref.Instance = conn.Instance
			}
			existing, _ := s.store.GetTask(ctx, store.TaskID(item.Ref))
			if existing != nil {
				continue
			}
			snap := domain.Snapshot{
				Title: item.Title, Body: item.Body, Labels: item.Labels, Author: item.Author,
				Assignee: item.Assignee, Status: item.Status, StatusGroup: item.StatusGroup,
			}
			task, err := s.store.EnsureTaskIn(ctx, batch.WorkspaceID, item.Ref, snap)
			if err != nil || task == nil {
				batch.Failed++
				continue
			}
			_ = s.store.AddImportItem(ctx, batch.ID, task.ID, item.Ref.ID)
			attachments := make([]*domain.TaskAttachment, 0, len(item.Attachments))
			for _, attachment := range item.Attachments {
				attachments = append(attachments, &domain.TaskAttachment{
					ID: "att_" + task.ID + "_" + attachment.ID, TaskID: task.ID, Provider: task.Provider,
					ExternalID: attachment.ID, Name: attachment.Name, URL: attachment.URL,
					Mime: attachment.Mime, Size: attachment.Size, Author: attachment.Author,
				})
			}
			_ = s.store.ReplaceTaskAttachments(ctx, task.ID, task.Provider, "issue", attachments)
			_ = s.store.ReplaceTaskFields(ctx, task.ID, task.Provider, item.Fields)
			s.syncPlatformComments(ctx, ad, cred, conn.Provider, task.ID, item.Ref)
			batch.Imported++
		}
	}
	if pager, ok := ad.Reader.(integration.ImportPager); ok {
		for batch.NextStartAt < batch.Limit {
			if current, err := s.store.GetImportBatch(ctx, batch.ID); err == nil && current != nil && current.Status == "cancelled" {
				return
			}
			pageSize := 100
			if remaining := batch.Limit - batch.NextStartAt; remaining < pageSize {
				pageSize = remaining
			}
			items, err := pager.SearchIssuesPage(ctx, cred, conn.Instance, batch.Query, batch.NextStartAt, pageSize)
			if err != nil {
				batch.Status, batch.Error = "failed", err.Error()
				_ = s.store.FinishImportBatch(ctx, batch)
				return
			}
			if len(items) == 0 {
				break
			}
			process(items)
			batch.NextStartAt += len(items)
			_ = s.store.SetImportProgress(ctx, batch.ID, batch.NextStartAt, batch.Imported, batch.Failed)
			if len(items) < pageSize {
				break
			}
		}
	} else {
		items, err := reader.SearchIssues(ctx, cred, conn.Instance, batch.Query, batch.Limit)
		if err != nil {
			batch.Status, batch.Error = "failed", err.Error()
			_ = s.store.FinishImportBatch(ctx, batch)
			return
		}
		process(items)
	}
	batch.Status = "completed"
	if batch.Failed > 0 {
		batch.Status = "partial"
	}
	_ = s.store.FinishImportBatch(ctx, batch)
}

func (s *Server) listImports(w http.ResponseWriter, r *http.Request) {
	workspaceID := chi.URLParam(r, "workspace")
	if workspaceID == "" {
		workspaceID = "ws_default"
	}
	if !s.requireWorkspace(w, r, workspaceID) {
		return
	}
	items, err := s.store.ListImportBatches(r.Context(), workspaceID, 50)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "E_DB", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (s *Server) resumeImport(w http.ResponseWriter, r *http.Request) {
	batch, err := s.store.GetImportBatch(r.Context(), chi.URLParam(r, "id"))
	if err != nil || batch == nil {
		writeErr(w, http.StatusNotFound, "E_NOT_FOUND", "import batch not found")
		return
	}
	if !s.requireWorkspace(w, r, batch.WorkspaceID) {
		return
	}
	if batch.Status == "running" || batch.Status == "queued" {
		writeErr(w, http.StatusConflict, "E_RUNNING", "import is already running")
		return
	}
	if batch.Status == "rolled_back" {
		writeErr(w, http.StatusConflict, "E_ROLLED_BACK", "import was rolled back")
		return
	}
	if batch.NextStartAt >= batch.Limit {
		writeErr(w, http.StatusConflict, "E_DONE", "import has no remaining items")
		return
	}
	_ = s.store.SetImportStatus(r.Context(), batch.ID, "queued", "")
	go s.runJiraImport(batch.ID)
	writeJSON(w, http.StatusAccepted, map[string]any{"batch": batch})
}

func (s *Server) rollbackImport(w http.ResponseWriter, r *http.Request) {
	batch, err := s.store.GetImportBatch(r.Context(), chi.URLParam(r, "id"))
	if err != nil || batch == nil {
		writeErr(w, http.StatusNotFound, "E_NOT_FOUND", "import batch not found")
		return
	}
	if !s.requireWorkspace(w, r, batch.WorkspaceID) {
		return
	}
	taskIDs, err := s.store.ListImportItems(r.Context(), batch.ID)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "E_DB", err.Error())
		return
	}
	deleted, skipped := 0, 0
	for _, taskID := range taskIDs {
		ok, err := s.store.DeleteImportedTask(r.Context(), taskID)
		if err != nil {
			writeErr(w, http.StatusInternalServerError, "E_DB", err.Error())
			return
		}
		if ok {
			deleted++
		} else {
			skipped++
		}
	}
	batch.Status = "rolled_back"
	_ = s.store.FinishImportBatch(r.Context(), batch)
	writeJSON(w, http.StatusOK, map[string]any{"deleted": deleted, "skipped": skipped, "batch": batch})
}

// ResumeImports re-queues batches left running by a previous process.
func (s *Server) ResumeImports(ctx context.Context) {
	batches, err := s.store.ListResumableImports(ctx)
	if err != nil {
		return
	}
	for _, batch := range batches {
		_ = s.store.SetImportStatus(ctx, batch.ID, "queued", "")
		go s.runJiraImport(batch.ID)
	}
}

func (s *Server) cancelImport(w http.ResponseWriter, r *http.Request) {
	batch, err := s.store.GetImportBatch(r.Context(), chi.URLParam(r, "id"))
	if err != nil || batch == nil {
		writeErr(w, http.StatusNotFound, "E_NOT_FOUND", "import batch not found")
		return
	}
	if !s.requireWorkspace(w, r, batch.WorkspaceID) {
		return
	}
	if batch.Status != "queued" && batch.Status != "running" {
		writeErr(w, http.StatusConflict, "E_STATUS", "import is not running")
		return
	}
	batch.Status = "cancelled"
	_ = s.store.FinishImportBatch(r.Context(), batch)
	writeJSON(w, http.StatusOK, map[string]any{"status": "cancelled"})
}

func (s *Server) importRuntime(w http.ResponseWriter, r *http.Request) {
	batches, _ := s.store.ListResumableImports(r.Context())
	queued, running := 0, 0
	for _, batch := range batches {
		if batch.Status == "queued" {
			queued++
		} else if batch.Status == "running" {
			running++
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"concurrency": cap(s.importSlots),
		"active":      len(s.importSlots),
		"queued":      queued,
		"running":     running,
	})
}

// acquireImportConn serializes imports per connection while allowing different
// connections to run concurrently.
func (s *Server) acquireImportConn(connectionID string) func() {
	s.importConnMu.Lock()
	ch := s.importConns[connectionID]
	if ch == nil {
		ch = make(chan struct{}, 1)
		s.importConns[connectionID] = ch
	}
	s.importConnMu.Unlock()
	ch <- struct{}{}
	return func() { <-ch }
}
