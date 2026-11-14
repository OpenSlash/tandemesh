package api

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/OpenSlash/tandemesh/internal/domain"
)

type transcriptBlock struct {
	Kind string `json:"kind"` // message|thought|tool|result|status
	Text string `json:"text"`
	TS   int64  `json:"tsMs"`
	Seq  int    `json:"seq"`
}

// runTranscript aggregates streamed ACP chunks into readable blocks.
func (s *Server) runTranscript(w http.ResponseWriter, r *http.Request) {
	run, err := s.store.GetRun(r.Context(), chi.URLParam(r, "id"))
	if err != nil || run == nil {
		writeErr(w, http.StatusNotFound, "E_NOT_FOUND", "run not found")
		return
	}
	if !s.requireWorkspace(w, r, run.WorkspaceID) {
		return
	}
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit <= 0 || limit > 500 {
		limit = 120
	}
	before, _ := strconv.Atoi(r.URL.Query().Get("before"))
	var events []domain.Event
	if before > 0 {
		events, err = s.store.ListEventsBefore(r.Context(), run.ID, before, limit)
	} else {
		events, err = s.store.ListEventsTail(r.Context(), run.ID, limit)
	}
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "E_DB", err.Error())
		return
	}
	hasComplete := false
	for _, ev := range events {
		if ev.Type == "agent.message.complete" {
			hasComplete = true
			break
		}
	}
	blocks := make([]transcriptBlock, 0, len(events))
	for _, ev := range events {
		if hasComplete && ev.Type == "agent.message" {
			continue
		}
		block := aggregateEvent(ev)
		if block == nil {
			continue
		}
		if len(blocks) > 0 {
			last := &blocks[len(blocks)-1]
			if block.Kind == "message" && last.Kind == "message" {
				last.Text += block.Text
				continue
			}
			if block.Kind == "thought" && last.Kind == "thought" {
				last.Text += block.Text
				continue
			}
		}
		blocks = append(blocks, *block)
	}
	nextBefore := 0
	if len(events) > 0 {
		nextBefore = events[0].Seq
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"items": blocks, "runId": run.ID, "total": len(blocks),
		"hasMore": len(events) == limit, "nextBefore": nextBefore,
	})
}

func aggregateEvent(ev domain.Event) *transcriptBlock {
	p := ev.Payload
	switch ev.Type {
	case "agent.message", "agent.message.complete":
		text, _ := p["text"].(string)
		if text == "" || strings.HasPrefix(text, "prompt finished:") {
			return nil
		}
		kind := "message"
		if role, _ := p["role"].(string); role == "thought" {
			kind = "thought"
		}
		if chunk, _ := p["chunk"].(bool); chunk && kind == "message" {
			kind = "message.chunk"
		}
		return &transcriptBlock{Kind: kind, Text: text, TS: ev.TS.UnixMilli(), Seq: ev.Seq}
	case "tool.call":
		name, _ := p["name"].(string)
		args, _ := p["args"].(string)
		return &transcriptBlock{Kind: "tool", Text: joinNonEmpty(name, args), TS: ev.TS.UnixMilli(), Seq: ev.Seq}
	case "tool.result":
		status, _ := p["status"].(string)
		output, _ := p["output"].(string)
		text := status
		if output != "" {
			text = strings.TrimSpace(status + ": " + output)
		}
		return &transcriptBlock{Kind: "result", Text: text, TS: ev.TS.UnixMilli(), Seq: ev.Seq}
	case "run.finished":
		status, _ := p["status"].(string)
		return &transcriptBlock{Kind: "status", Text: status, TS: ev.TS.UnixMilli(), Seq: ev.Seq}
	}
	return nil
}

func joinNonEmpty(parts ...string) string {
	out := ""
	for _, part := range parts {
		if part == "" {
			continue
		}
		if out != "" {
			out += " "
		}
		out += part
	}
	return out
}
