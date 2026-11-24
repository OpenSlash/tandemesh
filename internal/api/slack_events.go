package api

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/oklog/ulid/v2"

	"github.com/OpenSlash/tandemesh/internal/domain"
)

type slackEventEnvelope struct {
	Type      string `json:"type"`
	Challenge string `json:"challenge"`
	TeamID    string `json:"team_id"`
	Event     struct {
		Type    string `json:"type"`
		Text    string `json:"text"`
		User    string `json:"user"`
		Channel string `json:"channel"`
		TS      string `json:"ts"`
	} `json:"event"`
}

func (s *Server) slackEvents(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		writeErr(w, http.StatusBadRequest, "E_BODY", "body too large")
		return
	}
	if !verifySlackSignature(s.cfg.SlackSigningSecret, body, r.Header.Get("X-Slack-Request-Timestamp"), r.Header.Get("X-Slack-Signature")) {
		writeErr(w, http.StatusUnauthorized, "E_SIG", "signature verification failed")
		return
	}
	var envelope slackEventEnvelope
	if err := json.Unmarshal(body, &envelope); err != nil {
		writeErr(w, http.StatusBadRequest, "E_BODY", err.Error())
		return
	}
	if envelope.Type == "url_verification" {
		writeJSON(w, http.StatusOK, map[string]string{"challenge": envelope.Challenge})
		return
	}
	kind := envelope.Event.Type
	if kind == "" {
		kind = envelope.Type
	}
	id := "whd_" + ulid.Make().String()
	_ = s.store.CreateWebhookDelivery(r.Context(), &domain.WebhookDelivery{
		ID: id, Provider: "slack", Delivery: r.Header.Get("X-Slack-Retry-Num"), Kind: kind, Body: string(body),
	})
	_ = s.store.UpdateWebhookDelivery(r.Context(), id, "accepted", "", "")
	writeJSON(w, http.StatusOK, map[string]string{"status": "accepted"})
}

func verifySlackSignature(secret string, body []byte, timestamp, signature string) bool {
	if secret == "" || timestamp == "" || signature == "" {
		return false
	}
	seconds, err := strconv.ParseInt(timestamp, 10, 64)
	if err != nil || time.Since(time.Unix(seconds, 0)).Abs() > 5*time.Minute {
		return false
	}
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte("v0:" + timestamp + ":"))
	mac.Write(body)
	want := "v0=" + hex.EncodeToString(mac.Sum(nil))
	return hmac.Equal([]byte(want), []byte(strings.TrimSpace(signature)))
}
