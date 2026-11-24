package api

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/OpenSlash/tandemesh/internal/domain"
	"github.com/OpenSlash/tandemesh/internal/integration"
)

const (
	reminderAfter  = 5 * time.Minute
	reminderRepeat = 10 * time.Minute
)

// StartReminders nudges chat channels about approvals that nobody handled.
func (s *Server) StartReminders(ctx context.Context) {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.RemindOnce(ctx, time.Now().UTC())
		}
	}
}

// RemindOnce sends at most one reminder per approval per interval.
func (s *Server) RemindOnce(ctx context.Context, now time.Time) {
	items, err := s.store.ListApprovals(ctx, domain.ApprovalPending, "")
	if err != nil {
		s.log.Warn("list approvals for reminders", "err", err)
		return
	}
	s.remindersMu.Lock()
	defer s.remindersMu.Unlock()
	if s.reminded == nil {
		s.reminded = map[string]time.Time{}
	}
	for _, a := range items {
		if now.Sub(a.RequestedAt) < reminderAfter {
			continue
		}
		if last, ok := s.reminded[a.ID]; ok && now.Sub(last) < reminderRepeat {
			continue
		}
		if s.sendReminder(ctx, a) {
			s.reminded[a.ID] = now
		}
	}
}

func (s *Server) sendReminder(ctx context.Context, a *domain.Approval) bool {
	text := fmt.Sprintf(
		"Reminder: approval `%s` for run `%s` is still pending.\nApprove: %s\nDeny: %s",
		a.Tool, a.RunID, s.approvalURL(a.ID, "approved"), s.approvalURL(a.ID, "denied"),
	)
	sent := false

	slackCred := s.slackCredential(ctx)
	channel := s.cfg.SlackChannel
	if value := slackCred.Fields["channel"]; value != "" {
		channel = value
	}
	if channel != "" && slackCred.AccessToken != "" {
		if ad, ok := s.integrations.Get("slack"); ok && ad.Chat != nil {
			if _, err := ad.Chat.Send(ctx, slackCred, integration.ChatMessage{
				Channel: channel,
				Text:    text,
			}); err != nil {
				s.log.Warn("slack reminder", "approval", a.ID, "err", err)
			} else {
				sent = true
			}
		}
	}

	feishuCred := s.providerCredential(ctx, "feishu", "default")
	if s.cfg.FeishuWebhook != "" || feishuCred.Fields["webhookUrl"] != "" {
		if ad, ok := s.integrations.Get("feishu"); ok && ad.Chat != nil {
			if _, err := ad.Chat.Send(ctx, feishuCred, integration.ChatMessage{Text: text}); err != nil {
				s.log.Warn("feishu reminder", "approval", a.ID, "err", err)
			} else {
				sent = true
			}
		}
	}
	return sent
}

// reminderState is embedded in Server.
type reminderState struct {
	remindersMu sync.Mutex
	reminded    map[string]time.Time
}
