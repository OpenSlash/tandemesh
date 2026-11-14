package api

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/oklog/ulid/v2"

	"github.com/OpenSlash/tandemesh/internal/domain"
)

type annotationRequest struct {
	RunID    string `json:"runId"`
	FilePath string `json:"filePath"`
	Line     int    `json:"line"`
	Side     string `json:"side"`
	Body     string `json:"body"`
}

func (s *Server) listAnnotations(w http.ResponseWriter, r *http.Request) {
	task, err := s.store.GetTask(r.Context(), chi.URLParam(r, "id"))
	if err != nil || task == nil {
		writeErr(w, http.StatusNotFound, "E_NOT_FOUND", "task not found")
		return
	}
	if !s.requireTask(w, r, task) {
		return
	}
	items, err := s.store.ListDiffAnnotations(r.Context(), task.ID, r.URL.Query().Get("runId"), r.URL.Query().Get("status"))
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "E_DB", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"annotations": items})
}

func (s *Server) createAnnotation(w http.ResponseWriter, r *http.Request) {
	task, err := s.store.GetTask(r.Context(), chi.URLParam(r, "id"))
	if err != nil || task == nil {
		writeErr(w, http.StatusNotFound, "E_NOT_FOUND", "task not found")
		return
	}
	if !s.requireTask(w, r, task) {
		return
	}
	var req annotationRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeErr(w, http.StatusBadRequest, "E_BODY", err.Error())
		return
	}
	if strings.TrimSpace(req.FilePath) == "" || strings.TrimSpace(req.Body) == "" {
		writeErr(w, http.StatusBadRequest, "E_ANNOTATION", "filePath and body are required")
		return
	}
	a := &domain.DiffAnnotation{
		WorkspaceID: task.WorkspaceID,
		TaskID:      task.ID,
		RunID:       req.RunID,
		FilePath:    req.FilePath,
		Line:        req.Line,
		Side:        req.Side,
		Body:        req.Body,
		Author:      userOf(r),
		Status:      "open",
	}
	if err := s.store.AddDiffAnnotation(r.Context(), a); err != nil {
		writeErr(w, http.StatusInternalServerError, "E_DB", err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, a)
}

type sendAnnotationsRequest struct {
	RunID string `json:"runId"`
}

func (s *Server) sendAnnotations(w http.ResponseWriter, r *http.Request) {
	task, err := s.store.GetTask(r.Context(), chi.URLParam(r, "id"))
	if err != nil || task == nil {
		writeErr(w, http.StatusNotFound, "E_NOT_FOUND", "task not found")
		return
	}
	if !s.requireTask(w, r, task) {
		return
	}
	var req sendAnnotationsRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeErr(w, http.StatusBadRequest, "E_BODY", err.Error())
		return
	}
	notes, err := s.store.ListDiffAnnotations(r.Context(), task.ID, req.RunID, "open")
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "E_DB", err.Error())
		return
	}
	if len(notes) == 0 {
		writeErr(w, http.StatusBadRequest, "E_ANNOTATIONS", "no open annotations to send")
		return
	}
	runs, err := s.store.ListRunsByTask(r.Context(), task.ID)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "E_DB", err.Error())
		return
	}
	target := (*domain.Run)(nil)
	for _, run := range runs {
		if req.RunID != "" && run.ID == req.RunID {
			target = run
			break
		}
	}
	if target == nil && req.RunID == "" {
		for i := len(runs) - 1; i >= 0; i-- {
			if !runs[i].Status.Terminal() {
				target = runs[i]
				break
			}
		}
	}
	if target == nil && len(runs) > 0 && req.RunID == "" {
		target = runs[len(runs)-1]
	}
	if target == nil {
		writeErr(w, http.StatusBadRequest, "E_RUN", "runId is required when the task has no runs")
		return
	}

	var b strings.Builder
	b.WriteString("请根据下面的 diff 行级评审意见修改代码。每条意见都锚定到文件和行，请逐条处理；完成后说明修改点。\n\n")
	for _, n := range notes {
		fmt.Fprintf(&b, "- %s:%d", n.FilePath, n.Line)
		if n.Side != "" {
			fmt.Fprintf(&b, " (%s)", n.Side)
		}
		fmt.Fprintf(&b, " — %s\n", n.Body)
	}
	batchID := "batch_" + ulid.Make().String()
	prompt := b.String()
	m := &domain.AgentMessage{
		WorkspaceID: task.WorkspaceID,
		TaskID:      task.ID,
		RunID:       target.ID,
		Sender:      userOf(r),
		Recipient:   target.ID,
		Type:        "review",
		Subject:     "Diff 评审意见",
		Body:        prompt,
		Status:      "unread",
	}
	if err := s.store.SendAgentMessage(r.Context(), m); err != nil {
		writeErr(w, http.StatusInternalServerError, "E_DB", err.Error())
		return
	}
	ids := make([]string, 0, len(notes))
	for _, n := range notes {
		ids = append(ids, n.ID)
	}
	sent, err := s.store.MarkDiffAnnotationsSent(r.Context(), ids, batchID)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "E_DB", err.Error())
		return
	}
	s.recordTaskEvent(r.Context(), task.ID, "task.annotations.sent", map[string]any{
		"batchId": batchID, "count": sent, "runId": target.ID,
	})
	writeJSON(w, http.StatusOK, map[string]any{
		"batchId": batchID,
		"count":   sent,
		"runId":   target.ID,
		"message": m,
		"prompt":  prompt,
	})
}

func (s *Server) resolveAnnotation(w http.ResponseWriter, r *http.Request) {
	a, err := s.store.GetDiffAnnotation(r.Context(), chi.URLParam(r, "id"))
	if err != nil || a == nil {
		writeErr(w, http.StatusNotFound, "E_NOT_FOUND", "annotation not found")
		return
	}
	task, _ := s.store.GetTask(r.Context(), a.TaskID)
	if !s.requireTask(w, r, task) {
		return
	}
	updated, err := s.store.ResolveDiffAnnotation(r.Context(), a.ID)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "E_DB", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, updated)
}
