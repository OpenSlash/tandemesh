package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/OpenSlash/tandemesh/internal/domain"
)

func TestBatchTaskInbox(t *testing.T) {
	ts, st := newTestServerWithStore(t)
	ctx := t.Context()
	mk := func(id, status string) string {
		task, err := st.EnsureTaskIn(ctx, "ws_default", domain.WorkRef{Provider: "plane", Instance: "acme/p1", Type: "issue", ID: id}, domain.Snapshot{Title: id})
		if err != nil {
			t.Fatal(err)
		}
		if status != "" {
			_ = st.SetTaskStatus(ctx, task.ID, status)
		}
		return task.ID
	}
	open1, open2 := mk("o1", ""), mk("o2", "")
	done, cancelled := mk("d", domain.TaskDone), mk("c", domain.TaskCancelled)

	post := func(body string) int {
		resp, err := http.Post(ts.URL+"/api/v1/tasks/inbox", "application/json", bytes.NewReader([]byte(body)))
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		var out struct{ Updated int }
		_ = json.NewDecoder(resp.Body).Decode(&out)
		return out.Updated
	}
	if n := post(`{"ids":["` + open1 + `","` + open2 + `"],"state":"later","read":true}`); n != 2 {
		t.Fatalf("by ids updated %d", n)
	}
	if n := post(`{"closed":true,"workspace":"all","state":"cleared"}`); n != 2 {
		t.Fatalf("closed updated %d", n)
	}
	states, _ := st.TaskInboxStates(ctx, "alice", []string{open1, open2, done, cancelled})
	if states[open1].State != "later" || states[open1].ReadAt == nil || states[done].State != "cleared" || states[cancelled].State != "cleared" {
		t.Fatalf("states = %+v %+v %+v", states[open1], states[done], states[cancelled])
	}
}
