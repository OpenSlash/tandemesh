package api

import (
	"testing"

	"github.com/OpenSlash/tandemesh/internal/domain"
)

func TestMapPlatformStatus(t *testing.T) {
	tests := []struct {
		group, state string
		override     map[string]string
		want         string
	}{
		// Groups decide; renamed states ("已完成") still map correctly.
		{"backlog", "需求池", nil, domain.TaskOpen},
		{"unstarted", "待开始", nil, domain.TaskOpen},
		{"started", "开发中", nil, domain.TaskInProgress},
		{"completed", "已完成", nil, domain.TaskDone},
		{"cancelled", "已取消", nil, domain.TaskCancelled},
		// Names only matter when there is no group.
		{"", "Canceled", nil, domain.TaskCancelled},
		{"", "Done", nil, domain.TaskDone},
		{"", "Triage", nil, ""},
		// An override wins over the group.
		{"started", "待验收", map[string]string{"待验收": domain.TaskDone}, domain.TaskDone},
	}
	for _, tt := range tests {
		if got := mapPlatformStatusWith(tt.override, tt.group, tt.state); got != tt.want {
			t.Errorf("map(%q, %q) = %q, want %q", tt.group, tt.state, got, tt.want)
		}
	}
}
