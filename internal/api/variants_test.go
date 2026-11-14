package api

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/OpenSlash/tandemesh/internal/config"
	"github.com/OpenSlash/tandemesh/internal/domain"
	"github.com/OpenSlash/tandemesh/internal/hub"
	"github.com/OpenSlash/tandemesh/internal/store"
)

func TestRunVariants(t *testing.T) {
	st, err := store.Open(filepath.Join(t.TempDir(), "variants.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	ctx := t.Context()
	if err := st.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	origin := &domain.Run{
		ID: "run_origin", Work: domain.WorkRef{Provider: "manual", Instance: "web", Type: "task", ID: "1"},
		Status: domain.RunQueued, Repo: "acme/api", Base: "main", Prompt: "fix it",
		CreatedAt: time.Now().UTC(),
	}
	if err := st.CreateRun(ctx, origin); err != nil {
		t.Fatal(err)
	}

	srv := New(config.Config{}, st, nil, hub.New(), testRegistry(), nil, slog.New(slog.NewTextHandler(io.Discard, nil)))
	ts := httptest.NewServer(srv.Router())
	t.Cleanup(ts.Close)

	body, _ := json.Marshal(map[string]any{"agents": []string{"claude", "codex"}})
	resp, err := http.Post(ts.URL+"/api/v1/runs/run_origin/variants", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("create status = %d body=%s", resp.StatusCode, raw)
	}
	var created struct {
		Items []domain.Run `json:"items"`
	}
	if err := json.Unmarshal(raw, &created); err != nil {
		t.Fatal(err)
	}
	if len(created.Items) != 2 || created.Items[0].VariantOf != "run_origin" {
		t.Fatalf("variants = %+v", created.Items)
	}

	listResp, err := http.Get(ts.URL + "/api/v1/runs/run_origin/variants")
	if err != nil {
		t.Fatal(err)
	}
	listRaw, _ := io.ReadAll(listResp.Body)
	listResp.Body.Close()
	var listed struct {
		Items []domain.Run `json:"items"`
		Root  string       `json:"root"`
	}
	if err := json.Unmarshal(listRaw, &listed); err != nil {
		t.Fatal(err)
	}
	if len(listed.Items) != 3 || listed.Root != "run_origin" {
		t.Fatalf("listed = %+v root=%s", listed.Items, listed.Root)
	}
}
