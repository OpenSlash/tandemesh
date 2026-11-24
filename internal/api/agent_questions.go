package api

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/OpenSlash/tandemesh/internal/domain"
	"github.com/OpenSlash/tandemesh/internal/hub"
)

func (s *Server) listSessionQuestions(w http.ResponseWriter, r *http.Request) {
	sess, err := s.store.GetAgentSession(r.Context(), chi.URLParam(r, "id"))
	if err != nil || sess == nil {
		writeErr(w, http.StatusNotFound, "E_NOT_FOUND", "session not found")
		return
	}
	if !s.requireWorkspace(w, r, sess.WorkspaceID) {
		return
	}
	items, err := s.store.ListAgentQuestions(r.Context(), sess.ID, r.URL.Query().Get("status"))
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "E_DB", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

type answerQuestionRequest struct {
	Answer string `json:"answer"`
}

func (s *Server) answerAgentQuestion(w http.ResponseWriter, r *http.Request) {
	question, err := s.store.GetAgentQuestion(r.Context(), chi.URLParam(r, "id"))
	if err != nil || question == nil {
		writeErr(w, http.StatusNotFound, "E_NOT_FOUND", "question not found")
		return
	}
	sess, err := s.store.GetAgentSession(r.Context(), question.SessionID)
	if err != nil || sess == nil {
		writeErr(w, http.StatusNotFound, "E_NOT_FOUND", "session not found")
		return
	}
	if !s.requireWorkspace(w, r, sess.WorkspaceID) {
		return
	}
	if question.Status != "pending" {
		writeErr(w, http.StatusConflict, "E_STATE", "question is already answered")
		return
	}
	if question.ExpiresAt != nil && !time.Now().UTC().Before(*question.ExpiresAt) {
		writeErr(w, http.StatusConflict, "E_EXPIRED", "question expired")
		return
	}
	var req answerQuestionRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeErr(w, http.StatusBadRequest, "E_BODY", err.Error())
		return
	}
	answer := strings.TrimSpace(req.Answer)
	if answer == "" {
		writeErr(w, http.StatusBadRequest, "E_ANSWER", "answer is required")
		return
	}
	updated, err := s.store.AnswerAgentQuestion(r.Context(), question.ID, answer)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "E_DB", err.Error())
		return
	}
	if updated == nil {
		writeErr(w, http.StatusConflict, "E_STATE", "question is already answered")
		return
	}
	text := "Q: " + question.Question
	if len(question.Options) > 0 {
		text += "\nOptions: " + strings.Join(question.Options, ", ")
	}
	text += "\nA: " + answer
	_, _ = s.store.AppendSessionMessage(r.Context(), &domain.SessionMessage{
		SessionID: sess.ID, RunID: question.RunID, Role: "system", Kind: "question", Text: text,
	})
	if run, err := s.store.GetRun(r.Context(), question.RunID); err == nil && run != nil && run.MachineID != "" {
		payload, _ := json.Marshal(map[string]string{"questionId": question.ID, "answer": answer})
		s.hub.Send(run.MachineID, hub.Message{Type: "question.answer", RunID: run.ID, Payload: payload})
	}
	writeJSON(w, http.StatusOK, updated)
}

type agentdQuestionRequest struct {
	ID       string   `json:"id"`
	Question string   `json:"question"`
	Options  []string `json:"options"`
}

func (s *Server) createAgentdQuestion(w http.ResponseWriter, r *http.Request) {
	run, err := s.store.GetRun(r.Context(), chi.URLParam(r, "id"))
	if err != nil || run == nil || run.SessionID == "" {
		writeErr(w, http.StatusNotFound, "E_RUN", "run not found")
		return
	}
	var req agentdQuestionRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeErr(w, http.StatusBadRequest, "E_BODY", err.Error())
		return
	}
	if strings.TrimSpace(req.Question) == "" {
		writeErr(w, http.StatusBadRequest, "E_QUESTION", "question is required")
		return
	}
	question, err := s.store.CreateAgentQuestion(r.Context(), &domain.AgentQuestion{
		ID: req.ID, SessionID: run.SessionID, RunID: run.ID, Question: req.Question, Options: req.Options,
	})
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "E_DB", err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, question)
}

func (s *Server) getAgentdQuestion(w http.ResponseWriter, r *http.Request) {
	question, err := s.store.GetAgentQuestion(r.Context(), chi.URLParam(r, "id"))
	if err != nil || question == nil {
		writeErr(w, http.StatusNotFound, "E_NOT_FOUND", "question not found")
		return
	}
	writeJSON(w, http.StatusOK, question)
}
