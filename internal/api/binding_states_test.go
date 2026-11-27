package api

import (
	"context"
	"testing"

	"github.com/OpenSlash/tandemesh/internal/domain"
	"github.com/OpenSlash/tandemesh/internal/integration"
)

type fakeStates []integration.State

func (f fakeStates) ListStates(context.Context, integration.Credential, string) ([]integration.State, error) {
	return f, nil
}

func TestResolveStateTarget(t *testing.T) {
	s := &Server{}
	ad := &integration.Integration{States: fakeStates{
		{ID: "s-todo", Name: "待开始", Group: "unstarted"},
		{ID: "s-dev", Name: "开发中", Group: "started"},
		{ID: "s-test", Name: "测试中", Group: "started"},
		{ID: "s-done", Name: "已完成", Group: "completed"},
		{ID: "s-drop", Name: "已取消", Group: "cancelled"},
	}}
	task := &domain.Task{Instance: "acme/p1"}
	tests := []struct{ target, status, want string }{
		{"", domain.TaskInProgress, "s-dev"}, // first state of the group
		{"", domain.TaskCancelled, "s-drop"}, //
		{"s-test", domain.TaskInProgress, "s-test"},
		{"测试中", domain.TaskInProgress, "s-test"}, // a name resolves to this project's id
		{"", "unknown", ""},
	}
	for _, tt := range tests {
		if got := s.resolveStateTarget(context.Background(), ad, integration.Credential{}, task, tt.target, tt.status); got != tt.want {
			t.Errorf("resolve(%q, %q) = %q, want %q", tt.target, tt.status, got, tt.want)
		}
	}
}

func TestBindingStateOverrides(t *testing.T) {
	_, st := newTestServerWithStore(t)
	s := &Server{store: st}
	if err := st.UpsertBinding(t.Context(), &domain.ResourceBinding{
		ID: "bind_1", WorkspaceID: "ws_default", ConnectionID: "conn", Provider: "plane", ExternalID: "acme/p1", Enabled: true,
		Settings: map[string]string{stateInPrefix + "s-review": domain.TaskDone, settingWritebackStatus: "true"},
	}); err != nil {
		t.Fatal(err)
	}
	task := &domain.Task{WorkspaceID: "ws_default", Provider: "plane", Instance: "acme/p1"}
	merged := s.withBindingStateMap(t.Context(), task, map[string]string{"Backlog": domain.TaskOpen})
	if got := mapPlatformStatusWith(merged, "started", "待验收", "s-review"); got != domain.TaskDone {
		t.Fatalf("binding override ignored: %q", got)
	}
	if got := mapPlatformStatusWith(merged, "backlog", "Backlog", "s-x"); got != domain.TaskOpen {
		t.Fatalf("base map lost: %q", got)
	}
}
