package main

import (
	"context"
	"testing"
	"time"
)

func TestApprovalBrokerResolve(t *testing.T) {
	b := newApprovalBroker()
	done := make(chan string, 1)
	go func() {
		decision, err := b.wait(context.Background(), "apr_1", time.Second)
		if err != nil {
			t.Errorf("wait: %v", err)
		}
		done <- decision
	}()
	time.Sleep(10 * time.Millisecond)
	b.resolve("apr_1", "approved")
	if got := <-done; got != "approved" {
		t.Fatalf("decision = %q", got)
	}
}

func TestApprovalBrokerDenyAll(t *testing.T) {
	b := newApprovalBroker()
	done := make(chan string, 1)
	go func() {
		decision, _ := b.wait(context.Background(), "apr_2", time.Second)
		done <- decision
	}()
	time.Sleep(10 * time.Millisecond)
	b.denyAll()
	if got := <-done; got != "denied" {
		t.Fatalf("decision = %q", got)
	}
}

func TestApprovalBrokerTimeout(t *testing.T) {
	b := newApprovalBroker()
	decision, err := b.wait(context.Background(), "apr_3", 20*time.Millisecond)
	if err != nil {
		t.Fatalf("wait: %v", err)
	}
	if decision != "denied" {
		t.Fatalf("decision = %q", decision)
	}
}
