package main

import (
	"context"
	"sync"
	"time"
)

// approvalBroker routes decisions from the control plane back to the run that
// is waiting on them.
type approvalBroker struct {
	mu      sync.Mutex
	waiters map[string]chan string
}

func newApprovalBroker() *approvalBroker {
	return &approvalBroker{waiters: map[string]chan string{}}
}

// wait blocks until a decision arrives, the context ends or the timeout fires.
func (b *approvalBroker) wait(ctx context.Context, id string, timeout time.Duration) (string, error) {
	ch := make(chan string, 1)
	b.mu.Lock()
	b.waiters[id] = ch
	b.mu.Unlock()
	defer func() {
		b.mu.Lock()
		delete(b.waiters, id)
		b.mu.Unlock()
	}()

	timer := time.NewTimer(timeout)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return "", ctx.Err()
	case <-timer.C:
		return "denied", nil
	case decision := <-ch:
		return decision, nil
	}
}

// resolve delivers a decision to a waiter. Unknown ids are ignored.
func (b *approvalBroker) resolve(id, decision string) {
	b.mu.Lock()
	ch := b.waiters[id]
	b.mu.Unlock()
	if ch == nil {
		return
	}
	select {
	case ch <- decision:
	default:
	}
}

// denyAll fails every waiter, e.g. when the connection drops.
func (b *approvalBroker) denyAll() {
	b.mu.Lock()
	waiters := b.waiters
	b.waiters = map[string]chan string{}
	b.mu.Unlock()
	for _, ch := range waiters {
		select {
		case ch <- "denied":
		default:
		}
	}
}
