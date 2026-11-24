package agent

import (
	"context"
	"testing"
	"time"
)

func TestMockEmitsEvents(t *testing.T) {
	r := NewMock(time.Millisecond)
	var got []string
	res, err := r.Run(context.Background(), Spec{RunID: "run_test"}, func(ev Event) {
		got = append(got, ev.Type)
	})
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if res.Status != "succeeded" {
		t.Fatalf("status = %q", res.Status)
	}
	want := []string{"agent.message", "tool.call", "tool.result", "agent.message"}
	if len(got) != len(want) {
		t.Fatalf("events = %v", got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("event %d = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestMockHonoursCancellation(t *testing.T) {
	r := NewMock(50 * time.Millisecond)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	res, err := r.Run(ctx, Spec{}, func(Event) {})
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if res.Code != "E_CANCELLED" {
		t.Fatalf("code = %q", res.Code)
	}
}
