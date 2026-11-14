package api

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/OpenSlash/tandemesh/internal/config"
	"github.com/OpenSlash/tandemesh/internal/hub"
	"github.com/OpenSlash/tandemesh/internal/store"
	"log/slog"
	"path/filepath"
)

func TestSlackEventsVerification(t *testing.T) {
	st, err := store.Open(filepath.Join(t.TempDir(), "slack.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	if err := st.Migrate(t.Context()); err != nil {
		t.Fatal(err)
	}
	srv := New(config.Config{User: "alice", SlackSigningSecret: "secret"}, st, nil, hub.New(), testRegistry(), nil, slog.New(slog.NewTextHandler(io.Discard, nil)))
	ts := httptest.NewServer(srv.Router())
	defer ts.Close()
	body := []byte(`{"type":"url_verification","challenge":"abc"}`)
	mac := hmac.New(sha256.New, []byte("secret"))
	tsv := strconv.FormatInt(time.Now().Unix(), 10)
	mac.Write([]byte("v0:" + tsv + ":"))
	mac.Write(body)
	req, _ := http.NewRequest(http.MethodPost, ts.URL+"/webhooks/slack/events", bytes.NewReader(body))
	req.Header.Set("X-Slack-Request-Timestamp", tsv)
	req.Header.Set("X-Slack-Signature", "v0="+hex.EncodeToString(mac.Sum(nil)))
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d", resp.StatusCode)
	}
}
