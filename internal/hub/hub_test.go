package hub

import (
	"testing"
	"time"
)

func TestRegisterSendAndUnregister(t *testing.T) {
	h := New()
	conn, unregister := h.Register("m1")
	if !h.Connected("m1") {
		t.Fatal("machine not connected")
	}
	if !h.Send("m1", Message{Type: "hello", RunID: "run_1"}) {
		t.Fatal("send failed")
	}
	select {
	case msg := <-conn.Messages():
		if msg.Type != "hello" || msg.RunID != "run_1" {
			t.Fatalf("unexpected message: %+v", msg)
		}
	case <-time.After(time.Second):
		t.Fatal("message not delivered")
	}
	unregister()
	if h.Connected("m1") {
		t.Fatal("machine still connected")
	}
	if h.Send("m1", Message{Type: "hello"}) {
		t.Fatal("send to closed connection succeeded")
	}
}

func TestRegisterReplacesOldConnection(t *testing.T) {
	h := New()
	old, _ := h.Register("m1")
	fresh, _ := h.Register("m1")

	select {
	case <-old.Done():
	case <-time.After(time.Second):
		t.Fatal("old connection was not closed")
	}
	if !h.Send("m1", Message{Type: "lease.offer"}) {
		t.Fatal("send to replacement failed")
	}
	select {
	case msg := <-fresh.Messages():
		if msg.Type != "lease.offer" {
			t.Fatalf("unexpected message: %+v", msg)
		}
	case <-time.After(time.Second):
		t.Fatal("replacement did not receive message")
	}
}
