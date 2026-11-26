package api

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/coder/websocket"

	"github.com/OpenSlash/tandemesh/internal/config"
	"github.com/OpenSlash/tandemesh/internal/domain"
	"github.com/OpenSlash/tandemesh/internal/hub"
	"github.com/OpenSlash/tandemesh/internal/store"
)

func TestWebSocketRunLifecycle(t *testing.T) {
	st, err := store.Open(filepath.Join(t.TempDir(), "ws.db"))
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })
	ctx := context.Background()
	if err := st.Migrate(ctx); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	run := &domain.Run{
		ID:        "run_ws",
		Work:      domain.WorkRef{Provider: "manual", Instance: "test", Type: "task", ID: "1"},
		Status:    domain.RunQueued,
		CreatedAt: time.Now().UTC(),
	}
	if err := st.CreateRun(ctx, run); err != nil {
		t.Fatalf("create run: %v", err)
	}
	if _, err := st.DB().ExecContext(ctx, `UPDATE runs SET status='leased', machine_id='m1', fence=1 WHERE id=?`, run.ID); err != nil {
		t.Fatalf("lease run: %v", err)
	}

	cfg := config.Config{JoinToken: "test-token"}
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	srv := New(cfg, st, nil, hub.New(), testRegistry(), nil, log)
	ts := httptest.NewServer(srv.Router())
	t.Cleanup(ts.Close)

	wsURL := "ws" + strings.TrimPrefix(ts.URL, "http") + "/ws?machineId=m1&token=test-token"
	conn, _, err := websocket.Dial(ctx, wsURL, nil)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	t.Cleanup(func() { _ = conn.Close(websocket.StatusNormalClosure, "") })

	// The server greets the daemon.
	hello := readMessage(t, ctx, conn)
	if hello.Type != "hello.ok" {
		t.Fatalf("expected hello.ok, got %q", hello.Type)
	}

	writeMessage(t, ctx, conn, hub.Message{Type: "run.started", RunID: run.ID, Fence: 1})
	raw, _ := json.Marshal(map[string]any{"type": "agent.message", "payload": map[string]any{"text": "hi"}})
	writeMessage(t, ctx, conn, hub.Message{Type: "run.event", RunID: run.ID, Fence: 1, Payload: raw})
	finish, _ := json.Marshal(map[string]any{"status": "succeeded", "code": "", "message": ""})
	writeMessage(t, ctx, conn, hub.Message{Type: "run.finished", RunID: run.ID, Fence: 1, Payload: finish})

	deadline := time.Now().Add(3 * time.Second)
	var got *domain.Run
	for time.Now().Before(deadline) {
		got, err = st.GetRun(ctx, run.ID)
		if err == nil && got != nil && got.Status == domain.RunSucceeded {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if got == nil || got.Status != domain.RunSucceeded {
		t.Fatalf("run status = %+v", got)
	}

	events, err := st.ListEvents(ctx, run.ID, 0)
	if err != nil {
		t.Fatalf("list events: %v", err)
	}
	var sawMessage bool
	for _, ev := range events {
		if ev.Type == "agent.message" {
			sawMessage = true
		}
	}
	if !sawMessage {
		t.Fatalf("agent.message not recorded: %+v", events)
	}
}

func TestWebSocketRejectsBadToken(t *testing.T) {
	st, err := store.Open(filepath.Join(t.TempDir(), "ws2.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	_ = st.Migrate(context.Background())

	cfg := config.Config{JoinToken: "good"}
	srv := New(cfg, st, nil, hub.New(), testRegistry(), nil, slog.New(slog.NewTextHandler(io.Discard, nil)))
	ts := httptest.NewServer(srv.Router())
	t.Cleanup(ts.Close)

	wsURL := "ws" + strings.TrimPrefix(ts.URL, "http") + "/ws?machineId=m1&token=bad"
	if _, _, err := websocket.Dial(context.Background(), wsURL, nil); err == nil {
		t.Fatal("bad token accepted")
	}
}

func readMessage(t *testing.T, ctx context.Context, conn *websocket.Conn) hub.Message {
	t.Helper()
	rctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	_, data, err := conn.Read(rctx)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	var msg hub.Message
	if err := json.Unmarshal(data, &msg); err != nil {
		t.Fatalf("decode: %v", err)
	}
	return msg
}

func writeMessage(t *testing.T, ctx context.Context, conn *websocket.Conn, msg hub.Message) {
	t.Helper()
	data, _ := json.Marshal(msg)
	if err := conn.Write(ctx, websocket.MessageText, data); err != nil {
		t.Fatalf("write: %v", err)
	}
}

func TestWebSocketJoinTokenWithLocalToken(t *testing.T) {
	st, err := store.Open(filepath.Join(t.TempDir(), "ws3.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	_ = st.Migrate(context.Background())

	cfg := config.Config{JoinToken: "join", LocalToken: "desk-token"}
	srv := New(cfg, st, nil, hub.New(), testRegistry(), nil, slog.New(slog.NewTextHandler(io.Discard, nil)))
	ts := httptest.NewServer(srv.Router())
	t.Cleanup(ts.Close)
	base := "ws" + strings.TrimPrefix(ts.URL, "http") + "/ws?machineId=m1&token="

	tests := []struct {
		name  string
		token string
		ok    bool
	}{
		{"join token accepted", "join", true},
		{"local token is not a join token", "desk-token", false},
		{"bad token rejected", "bad", false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			conn, _, err := websocket.Dial(context.Background(), base+tc.token, nil)
			if conn != nil {
				_ = conn.Close(websocket.StatusNormalClosure, "")
			}
			if (err == nil) != tc.ok {
				t.Fatalf("dial err = %v, want ok=%v", err, tc.ok)
			}
		})
	}
}
