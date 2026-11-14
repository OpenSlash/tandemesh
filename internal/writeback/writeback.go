// Package writeback publishes run results back to the source system.
//
// Rows are persisted in the outbox before delivery, so a crash cannot lose a
// result and a retry cannot double-post (unique run/action index).
package writeback

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/OpenSlash/tandemesh/internal/domain"
	"github.com/OpenSlash/tandemesh/internal/integration"
	"github.com/OpenSlash/tandemesh/internal/store"
)

const maxAttempts = 5

var backoff = []time.Duration{
	1 * time.Minute,
	5 * time.Minute,
	30 * time.Minute,
	2 * time.Hour,
	6 * time.Hour,
}

// Credentials resolves the token for a provider.
type Credentials func(provider, connectionID, accountID string) integration.Credential

// Processor drains the outbox.
type Processor struct {
	Store       *store.Store
	Registry    *integration.Registry
	Credentials Credentials
	Policy      func(context.Context, *domain.Writeback) bool
	Logger      *slog.Logger
}

// New creates a processor.
func New(st *store.Store, reg *integration.Registry, creds Credentials, log *slog.Logger) *Processor {
	if log == nil {
		log = slog.Default()
	}
	return &Processor{Store: st, Registry: reg, Credentials: creds, Logger: log}
}

// Run drains the outbox until the context ends.
func (p *Processor) Run(ctx context.Context) {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if n, err := p.ProcessOnce(ctx); err != nil {
				p.Logger.Warn("writeback pass", "err", err)
			} else if n > 0 {
				p.Logger.Info("writeback delivered", "count", n)
			}
		}
	}
}

// ProcessOnce delivers all due rows and reports how many succeeded.
func (p *Processor) ProcessOnce(ctx context.Context) (int, error) {
	items, err := p.Store.DueWritebacks(ctx, time.Now().UTC(), 20)
	if err != nil {
		return 0, err
	}
	delivered := 0
	for _, item := range items {
		if err := p.process(ctx, item); err != nil {
			if item.ConnectionID != "" && integration.IsAuthError(err) {
				_ = p.Store.SetConnectionStatus(ctx, item.ConnectionID, domain.ConnectionNeedsReauth)
			}
			p.Logger.Warn("writeback failed", "run", item.RunID, "action", item.Action, "err", err)
			continue
		}
		delivered++
	}
	return delivered, nil
}

func (p *Processor) process(ctx context.Context, w *domain.Writeback) error {
	if p.Policy != nil && !p.Policy(ctx, w) {
		if err := p.Store.CompleteWriteback(ctx, w.ID); err != nil {
			return err
		}
		_, _ = p.Store.AppendEvent(ctx, w.RunID, "writeback.skipped", map[string]any{
			"provider": w.Provider, "action": w.Action, "reason": "read-only binding",
		})
		return nil
	}
	ad, ok := p.Registry.Get(w.Provider)
	if !ok || ad.Work == nil {
		return p.fail(ctx, w, integration.ErrUnsupported, true)
	}
	cred := p.Credentials(w.Provider, w.ConnectionID, w.AccountID)
	if cred.AccessToken == "" {
		return p.fail(ctx, w, errors.New("missing credential"), true)
	}

	var err error
	switch w.Action {
	case "comment":
		_, err = ad.Work.Comment(ctx, w.Work, cred, w.Markdown)
	case "attach":
		if !hasCapability(ad, integration.CapAttach) {
			return p.fail(ctx, w, integration.ErrUnsupported, true)
		}
		err = ad.Work.Attach(ctx, w.Work, cred, w.Kind, w.URL)
	case "transition":
		if !hasCapability(ad, integration.CapTransition) {
			return p.fail(ctx, w, integration.ErrUnsupported, true)
		}
		err = ad.Work.Transition(ctx, w.Work, cred, w.URL)
	default:
		err = integration.ErrUnsupported
	}
	if err != nil {
		return p.fail(ctx, w, err, false)
	}

	if err := p.Store.CompleteWriteback(ctx, w.ID); err != nil {
		return err
	}
	_, _ = p.Store.AppendEvent(ctx, w.RunID, "writeback.done", map[string]any{
		"provider": w.Provider,
		"action":   w.Action,
	})
	return nil
}

func (p *Processor) fail(ctx context.Context, w *domain.Writeback, cause error, dead bool) error {
	attempts := w.Attempts + 1
	if attempts >= maxAttempts {
		dead = true
	}
	next := time.Now().UTC()
	if !dead {
		idx := attempts - 1
		if idx >= len(backoff) {
			idx = len(backoff) - 1
		}
		next = next.Add(backoff[idx])
	}
	if err := p.Store.FailWriteback(ctx, w.ID, attempts, next, cause.Error(), dead); err != nil {
		return err
	}
	_, _ = p.Store.AppendEvent(ctx, w.RunID, "writeback.failed", map[string]any{
		"provider": w.Provider,
		"action":   w.Action,
		"error":    cause.Error(),
		"dead":     dead,
	})
	return cause
}

func hasCapability(ad *integration.Integration, want integration.Capability) bool {
	for _, cap := range ad.Manifest.Capabilities {
		if cap == want {
			return true
		}
	}
	return false
}
