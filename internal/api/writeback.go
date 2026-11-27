package api

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/oklog/ulid/v2"

	"github.com/OpenSlash/tandemesh/internal/domain"
	"github.com/OpenSlash/tandemesh/internal/integration"
)

// enqueueWritebacks records the result publications for a finished run. The
// outbox worker delivers them, so the API request does not wait on the network.
func (s *Server) enqueueWritebacks(ctx context.Context, run *domain.Run) {
	if run == nil {
		return
	}
	var b *domain.Automation
	if run.AutomationID != "" {
		loaded, err := s.store.GetAutomation(ctx, run.AutomationID)
		if err != nil || loaded == nil {
			return
		}
		b = loaded
	} else if run.TaskID != "" {
		// Task runs have no automation id; use an event automation that matches the source.
		automations, err := s.store.ListAutomations(ctx, run.WorkspaceID)
		if err != nil {
			return
		}
		for _, candidate := range automations {
			if candidate.Schedule != "" || candidate.Provider != run.Work.Provider || !candidate.Enabled {
				continue
			}
			if candidate.Instance == "" || domain.InstanceCovers(candidate.Instance, run.Work.Instance) {
				b = candidate
				break
			}
		}
		if b == nil {
			return
		}
	} else {
		return
	}
	ad, ok := s.integrations.Get(b.Provider)
	if !ok || ad.Work == nil {
		return
	}

	baseURL := s.cfg.BaseURL
	if s.cfg.AppURL != "" {
		baseURL = s.cfg.AppURL
	}
	runURL := fmt.Sprintf("%s/runs/%s", strings.TrimRight(baseURL, "/"), run.ID)
	now := time.Now().UTC()
	accountID := s.store.BindingAccountID(ctx, b.Provider, b.ConnectionID, run.Work.Instance)
	var items []*domain.Writeback

	if boolOption(b.Writeback, "comment", true) {
		items = append(items, &domain.Writeback{
			ID:       "wb_" + ulid.Make().String(),
			RunID:    run.ID,
			Provider: b.Provider, ConnectionID: b.ConnectionID, AccountID: accountID,
			Action:   "comment",
			Work:     run.Work,
			Markdown: commentMarkdown(run, runURL, b.Writeback),
			NextAt:   now,
		})
	}

	if boolOption(b.Writeback, "attach", false) && hasCapability(ad, integration.CapAttach) {
		items = append(items, &domain.Writeback{
			ID:       "wb_" + ulid.Make().String(),
			RunID:    run.ID,
			Provider: b.Provider, ConnectionID: b.ConnectionID, AccountID: accountID,
			Action: "attach",
			Work:   run.Work,
			URL:    runURL,
			Kind:   "run",
			NextAt: now,
		})
	}

	if s.cfg.WritebackState {
		if target, _ := b.Writeback["transition"].(string); target != "" &&
			run.Status == domain.RunSucceeded && hasCapability(ad, integration.CapTransition) {
			items = append(items, &domain.Writeback{
				ID:       "wb_" + ulid.Make().String(),
				RunID:    run.ID,
				Provider: b.Provider, ConnectionID: b.ConnectionID, AccountID: accountID,
				Action: "transition",
				Work:   run.Work,
				URL:    target,
				Kind:   "transition",
				NextAt: now,
			})
		}
	}

	for _, item := range items {
		if err := s.store.EnqueueWriteback(ctx, item); err != nil {
			s.log.Warn("enqueue writeback", "run", run.ID, "action", item.Action, "err", err)
		}
	}
}

func summaryMarkdown(run *domain.Run, runURL string) string {
	markdown := fmt.Sprintf(
		"Tandemesh run `%s` finished with status **%s**.\n\nRepository: `%s`  \nBranch: `%s`  \n[Open run](%s)",
		run.ID, run.Status, run.Repo, run.Branch, runURL,
	)
	if run.ErrorCode != "" {
		markdown += fmt.Sprintf("\n\nError: `%s` — %s", run.ErrorCode, run.ErrorMessage)
	}
	return markdown
}

func boolOption(options map[string]any, key string, fallback bool) bool {
	if options == nil {
		return fallback
	}
	if v, ok := options[key].(bool); ok {
		return v
	}
	return fallback
}

func hasCapability(ad *integration.Integration, want integration.Capability) bool {
	for _, cap := range ad.Manifest.Capabilities {
		if cap == want {
			return true
		}
	}
	return false
}

func commentMarkdown(run *domain.Run, runURL string, options map[string]any) string {
	if options != nil {
		if tmpl, ok := options["template"].(string); ok && strings.TrimSpace(tmpl) != "" {
			replacer := strings.NewReplacer(
				"{{run_id}}", run.ID,
				"{{status}}", string(run.Status),
				"{{repo}}", run.Repo,
				"{{branch}}", run.Branch,
				"{{url}}", runURL,
				"{{agent}}", run.AgentID,
				"{{title}}", run.Snapshot.Title,
				"{{error}}", run.ErrorCode,
			)
			return replacer.Replace(tmpl) + "\n\n<!-- orca:run=" + run.ID + " -->"
		}
	}
	return summaryMarkdown(run, runURL) + "\n\n<!-- orca:run=" + run.ID + " -->"
}
