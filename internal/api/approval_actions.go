package api

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/OpenSlash/tandemesh/internal/domain"
	"github.com/OpenSlash/tandemesh/internal/hub"
	"github.com/OpenSlash/tandemesh/internal/integration"
)

const approvalTTL = 24 * time.Hour

func (s *Server) approvalToken(id, decision string, exp int64) string {
	mac := hmac.New(sha256.New, []byte(s.cfg.ApprovalKey))
	fmt.Fprintf(mac, "%s|%s|%d", id, decision, exp)
	return hex.EncodeToString(mac.Sum(nil))
}

func (s *Server) approvalURL(id, decision string) string {
	exp := time.Now().Add(approvalTTL).Unix()
	token := s.approvalToken(id, decision, exp)
	base := strings.TrimRight(s.cfg.BaseURL, "/")
	return fmt.Sprintf("%s/api/v1/approvals/%s/decide?decision=%s&exp=%d&token=%s", base, id, decision, exp, token)
}

// slackCredential prefers the connection credential over the environment.
func (s *Server) slackCredential(ctx context.Context) integration.Credential {
	if s.vault != nil {
		if cred, ok, err := s.vault.Get(ctx, "slack", "default"); err == nil && ok && cred.AccessToken != "" {
			return cred
		}
	}
	return integration.Credential{Provider: "slack", AccessToken: s.cfg.SlackToken}
}

func (s *Server) slackToken(ctx context.Context) string {
	return s.slackCredential(ctx).AccessToken
}

// notifyApproval fans out approval cards to every configured chat integration.
func (s *Server) notifyApproval(ctx context.Context, approval *domain.Approval) {
	if run, err := s.store.GetRun(ctx, approval.RunID); err == nil && run != nil {
		s.pushNotification(ctx, run.WorkspaceID, "approval.requested", "需要审批："+approval.Tool, approval.Command, "/approvals")
	}
	card := integration.ApprovalCard{
		ApprovalID: approval.ID,
		Channel:    s.cfg.SlackChannel,
		RunID:      approval.RunID,
		Tool:       approval.Tool,
		Command:    approval.Command,
		Risk:       approval.Risk,
		ApproveURL: s.approvalURL(approval.ID, "approved"),
		DenyURL:    s.approvalURL(approval.ID, "denied"),
	}

	slackCred := s.slackCredential(ctx)
	if channel := slackCred.Fields["channel"]; channel != "" {
		card.Channel = channel
	}
	if card.Channel != "" && slackCred.AccessToken != "" {
		if ad, ok := s.integrations.Get("slack"); ok && ad.Approver != nil {
			ref, err := ad.Approver.RequestApproval(ctx, slackCred, card)
			if err != nil {
				s.log.Warn("slack approval card", "approval", approval.ID, "err", err)
			} else {
				approval.MessageRef = ref
				if err := s.store.SetApprovalMessageRef(ctx, approval.ID, ref); err != nil {
					s.log.Warn("store approval message ref", "approval", approval.ID, "err", err)
				}
			}
		}
	}

	feishuCred := s.providerCredential(ctx, "feishu", "default")
	if s.cfg.FeishuWebhook != "" || feishuCred.Fields["webhookUrl"] != "" {
		if ad, ok := s.integrations.Get("feishu"); ok && ad.Approver != nil {
			if _, err := ad.Approver.RequestApproval(ctx, feishuCred, card); err != nil {
				s.log.Warn("feishu approval card", "approval", approval.ID, "err", err)
			}
		}
	}
}

// updateApprovalCard edits the Slack message after a decision.
func (s *Server) updateApprovalCard(ctx context.Context, approval *domain.Approval, by, decision string) {
	if approval.MessageRef == "" {
		return
	}
	token := s.slackToken(ctx)
	if token == "" {
		return
	}
	ad, ok := s.integrations.Get("slack")
	if !ok || ad.Chat == nil {
		return
	}
	text := fmt.Sprintf("%s approval %s by %s", strings.Title(decision), approval.Tool, by)
	cred := integration.Credential{Provider: "slack", AccessToken: token}
	if err := ad.Chat.Update(ctx, cred, approval.MessageRef, integration.ChatMessage{Text: text}); err != nil {
		s.log.Warn("slack card update", "approval", approval.ID, "err", err)
	}
}

// approvalCallback handles the signed links embedded in the Slack card.
func (s *Server) approvalCallback(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	decision := r.URL.Query().Get("decision")
	expRaw := r.URL.Query().Get("exp")
	token := r.URL.Query().Get("token")

	if decision != domain.ApprovalApproved && decision != domain.ApprovalDenied {
		http.Error(w, "invalid decision", http.StatusBadRequest)
		return
	}
	exp, err := strconv.ParseInt(expRaw, 10, 64)
	if err != nil || time.Now().Unix() > exp {
		http.Error(w, "link expired", http.StatusGone)
		return
	}
	want := s.approvalToken(id, decision, exp)
	if !hmac.Equal([]byte(want), []byte(token)) {
		http.Error(w, "invalid token", http.StatusUnauthorized)
		return
	}

	approval, err := s.store.GetApproval(r.Context(), id)
	if err != nil || approval == nil {
		http.Error(w, "approval not found", http.StatusNotFound)
		return
	}
	if approval.Status != domain.ApprovalPending {
		writeApprovalPage(w, "Already decided", "This approval is "+approval.Status+".")
		return
	}

	ok, err := s.store.DecideApproval(r.Context(), id, decision, "slack-link")
	if err != nil || !ok {
		http.Error(w, "could not record decision", http.StatusConflict)
		return
	}
	eventType := "approval.denied"
	if decision == domain.ApprovalApproved {
		eventType = "approval.granted"
	}
	_, _ = s.store.AppendEvent(r.Context(), approval.RunID, eventType, map[string]any{
		"approvalId": id,
		"by":         "slack-link",
	})
	s.recordApprovalTranscript(r.Context(), approval, decision, "slack-link")
	if run, err := s.store.GetRun(r.Context(), approval.RunID); err == nil && run != nil && run.MachineID != "" {
		payload, _ := json.Marshal(map[string]string{"approvalId": id, "decision": decision})
		s.hub.Send(run.MachineID, hub.Message{Type: "approval.decision", RunID: run.ID, Payload: payload})
	}
	s.updateApprovalCard(r.Context(), approval, "slack", decision)

	writeApprovalPage(w, strings.Title(decision), "Run "+approval.RunID+" will continue.")
}

func writeApprovalPage(w http.ResponseWriter, title, body string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprintf(w, `<!doctype html><html><head><meta charset="utf-8"><title>%s</title>
<style>body{font-family:system-ui;background:#08090a;color:#e8e8e8;display:flex;align-items:center;justify-content:center;height:100vh;margin:0}
main{border:1px solid #2a2d31;padding:32px 40px;border-radius:12px;background:#0e0f11}h1{font-size:18px;margin:0 0 8px}p{color:#8a8f98;margin:0}</style>
</head><body><main><h1>%s</h1><p>%s</p></main></body></html>`, title, title, body)
}
