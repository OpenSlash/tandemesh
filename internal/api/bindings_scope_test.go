package api

import (
	"bytes"
	"net/http"
	"testing"

	"github.com/OpenSlash/tandemesh/internal/domain"
)

func TestBindingBelongsToOneWorkspace(t *testing.T) {
	ts, st := newTestServerWithStore(t)
	ctx := t.Context()
	if err := st.CreateWorkspace(ctx, &domain.Workspace{ID: "ws_other", Name: "Other", Slug: "other"}); err != nil {
		t.Fatal(err)
	}
	for _, c := range []*domain.Connection{
		{ID: "conn_a", WorkspaceID: "ws_default", Provider: "plane", Instance: "acme/p1", Enabled: true},
		{ID: "conn_b", WorkspaceID: "ws_other", Provider: "plane", Instance: "acme/p2", Enabled: true},
	} {
		if err := st.UpsertConnection(ctx, c); err != nil {
			t.Fatal(err)
		}
	}
	bind := func(conn, externalID string) int {
		resp, err := http.Post(ts.URL+"/api/v1/connections/"+conn+"/bindings", "application/json",
			bytes.NewReader([]byte(`{"externalId":"`+externalID+`"}`)))
		if err != nil {
			t.Fatal(err)
		}
		resp.Body.Close()
		return resp.StatusCode
	}
	if code := bind("conn_a", "acme/p1"); code != http.StatusCreated {
		t.Fatalf("first binding: %d", code)
	}
	if code := bind("conn_b", "acme/p1"); code != http.StatusConflict {
		t.Fatalf("same project in another workspace: %d", code)
	}
	if code := bind("conn_b", "acme"); code != http.StatusConflict {
		t.Fatalf("whole workspace overlapping a bound project: %d", code)
	}
	if code := bind("conn_b", "acme/p2"); code != http.StatusCreated {
		t.Fatalf("other project: %d", code)
	}
}

func TestRemovingLastBindingStopsConnectionSync(t *testing.T) {
	ts, st := newTestServerWithStore(t)
	ctx := t.Context()
	if err := st.UpsertConnection(ctx, &domain.Connection{ID: "conn_a", WorkspaceID: "ws_default", Provider: "plane", Instance: "acme/p1", Enabled: true}); err != nil {
		t.Fatal(err)
	}
	if err := st.UpsertBinding(ctx, &domain.ResourceBinding{ID: "bind_a", WorkspaceID: "ws_default", ConnectionID: "conn_a", Provider: "plane", ExternalID: "acme/p1", Enabled: true}); err != nil {
		t.Fatal(err)
	}
	req, _ := http.NewRequest(http.MethodDelete, ts.URL+"/api/v1/bindings/bind_a", nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("delete: %d", resp.StatusCode)
	}
	conn, err := st.GetConnection(ctx, "conn_a")
	if err != nil || conn == nil || conn.Instance != "" {
		t.Fatalf("connection should no longer sync its own instance: %+v err=%v", conn, err)
	}
}
