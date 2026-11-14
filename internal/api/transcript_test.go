package api

import (
	"testing"
	"time"

	"github.com/OpenSlash/tandemesh/internal/domain"
)

func TestAggregateChunkKind(t *testing.T) {
	ev := domain.Event{Type: "agent.message", TS: time.Now(), Payload: map[string]any{"text": "hi", "chunk": true}}
	block := aggregateEvent(ev)
	if block == nil || block.Kind != "message.chunk" {
		t.Fatalf("block = %+v", block)
	}
}
