// Package connections keeps provider connections healthy.
package connections

import (
	"context"
	"log/slog"
	"time"

	"github.com/OpenSlash/tandemesh/internal/credentials"
	"github.com/OpenSlash/tandemesh/internal/domain"
	"github.com/OpenSlash/tandemesh/internal/integration"
	"github.com/OpenSlash/tandemesh/internal/store"
)

// Processor periodically probes stored connections.
type Processor struct {
	Store    *store.Store
	Registry *integration.Registry
	Vault    *credentials.Vault
	Interval time.Duration
	Logger   *slog.Logger
}

// New builds a health processor.
func New(st *store.Store, reg *integration.Registry, vault *credentials.Vault, log *slog.Logger) *Processor {
	if log == nil {
		log = slog.Default()
	}
	return &Processor{Store: st, Registry: reg, Vault: vault, Interval: 5 * time.Minute, Logger: log}
}

// Run probes connections until the context ends.
func (p *Processor) Run(ctx context.Context) {
	interval := p.Interval
	if interval <= 0 {
		interval = 5 * time.Minute
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			p.CheckOnce(ctx)
		}
	}
}

// CheckOnce probes every enabled connection and reports how many are healthy.
func (p *Processor) CheckOnce(ctx context.Context) int {
	if p.Vault == nil {
		return 0
	}
	items, err := p.Store.ListAllConnections(ctx)
	if err != nil {
		p.Logger.Warn("connection health list", "err", err)
		return 0
	}
	healthy := 0
	for _, conn := range items {
		if !conn.Enabled || conn.Status == domain.ConnectionRevoked {
			continue
		}
		ad, ok := p.Registry.Get(conn.Provider)
		if !ok || ad.Probe == nil {
			continue
		}
		subject := conn.ID
		if subject == "" {
			subject = "default"
		}
		cred, stored, err := p.Vault.Ensure(ctx, conn.Provider, subject)
		if err != nil || !stored || cred.AccessToken == "" {
			continue
		}
		if !cred.ExpiresAt.IsZero() && cred.ExpiresAt.Before(time.Now()) {
			_ = p.Store.SetConnectionStatus(ctx, conn.ID, domain.ConnectionNeedsReauth)
			continue
		}
		if err := ad.Probe.Probe(ctx, cred); err != nil {
			if integration.IsAuthError(err) {
				_ = p.Store.SetConnectionStatus(ctx, conn.ID, domain.ConnectionNeedsReauth)
			}
			continue
		}
		healthy++
		if conn.Status != domain.ConnectionConnected {
			_ = p.Store.SetConnectionStatus(ctx, conn.ID, domain.ConnectionConnected)
		}
	}
	return healthy
}
