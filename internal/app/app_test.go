package app

import (
	"context"
	"io"
	"log/slog"
	"path/filepath"
	"testing"

	"github.com/OpenSlash/tandemesh/internal/config"
)

func TestAppStarts(t *testing.T) {
	cfg := config.Config{
		DBPath:    filepath.Join(t.TempDir(), "app.db"),
		JoinToken: "t",
		LeaseTTL:  0,
	}
	a, err := New(context.Background(), cfg, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatalf("new: %v", err)
	}
	defer a.Close()

	if a.Handler == nil {
		t.Fatal("nil handler")
	}
	if got := len(a.Registry.List()); got != 10 {
		t.Fatalf("providers = %d", got)
	}
}
