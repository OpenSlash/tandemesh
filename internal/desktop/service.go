// Package desktop exposes application operations as plain Go methods so a
// Wails v3 shell can bind them without depending on the HTTP API.
//
// The package deliberately imports no Wails types: the web build, the CLI and
// the desktop shell share one core, and Wails is only a presentation layer.
package desktop

import (
	"context"
	"errors"
	"time"

	"github.com/oklog/ulid/v2"

	"github.com/OpenSlash/tandemesh/internal/domain"
	"github.com/OpenSlash/tandemesh/internal/integration"
	"github.com/OpenSlash/tandemesh/internal/store"
)

// Service is the bindable surface for the desktop shell.
type Service struct {
	store        *store.Store
	integrations *integration.Registry
	version      string
}

// New creates a service.
func New(st *store.Store, reg *integration.Registry, version string) *Service {
	return &Service{store: st, integrations: reg, version: version}
}

// Version returns the build version.
func (s *Service) Version() string { return s.version }

// ListRuns returns the latest runs.
func (s *Service) ListRuns(ctx context.Context, status string) ([]*domain.Run, error) {
	runs, err := s.store.ListRuns(ctx, "", status, 100)
	if runs == nil {
		runs = []*domain.Run{}
	}
	return runs, err
}

// GetRun returns one run.
func (s *Service) GetRun(ctx context.Context, id string) (*domain.Run, error) {
	run, err := s.store.GetRun(ctx, id)
	if err != nil {
		return nil, err
	}
	if run == nil {
		return nil, errors.New("run not found")
	}
	return run, nil
}

// CreateRunInput is the desktop request shape.
type CreateRunInput struct {
	Repo    string `json:"repo"`
	Prompt  string `json:"prompt"`
	AgentID string `json:"agentId"`
}

// CreateRun starts a manual run.
func (s *Service) CreateRun(ctx context.Context, in CreateRunInput) (*domain.Run, error) {
	if in.AgentID == "" {
		in.AgentID = "claude"
	}
	id := "run_" + ulid.Make().String()
	run := &domain.Run{
		ID:        id,
		Work:      domain.WorkRef{Provider: "manual", Instance: "desktop", Type: "task", ID: id},
		Status:    domain.RunQueued,
		Repo:      in.Repo,
		Base:      "main",
		Branch:    "orca/" + id[len(id)-8:],
		Prompt:    in.Prompt,
		AgentID:   in.AgentID,
		CreatedAt: time.Now().UTC(),
	}
	if err := s.store.CreateRun(ctx, run); err != nil {
		return nil, err
	}
	_, _ = s.store.AppendEvent(ctx, run.ID, domain.EventRunCreated, map[string]any{
		"provider": "manual",
		"source":   "desktop",
	})
	return run, nil
}

// CancelRun cancels a run.
func (s *Service) CancelRun(ctx context.Context, id string) (bool, error) {
	ok, err := s.store.CancelRun(ctx, id)
	if ok {
		_, _ = s.store.AppendEvent(ctx, id, "run.cancelled", map[string]any{"by": "desktop"})
	}
	return ok, err
}

// ListApprovals returns pending approvals.
func (s *Service) ListApprovals(ctx context.Context) ([]*domain.Approval, error) {
	items, err := s.store.ListApprovals(ctx, domain.ApprovalPending, "")
	if items == nil {
		items = []*domain.Approval{}
	}
	return items, err
}

// DecideApproval records an approval decision.
func (s *Service) DecideApproval(ctx context.Context, id, decision, by string) (bool, error) {
	if decision != domain.ApprovalApproved && decision != domain.ApprovalDenied {
		return false, errors.New("decision must be approved or denied")
	}
	if by == "" {
		by = "desktop"
	}
	return s.store.DecideApproval(ctx, id, decision, by)
}

// Providers lists integration manifests for the shell.
func (s *Service) Providers() []integration.Manifest {
	return s.integrations.List()
}

// ListMachines returns registered computers.
func (s *Service) ListMachines(ctx context.Context) ([]*domain.Machine, error) {
	machines, err := s.store.ListMachines(ctx)
	if machines == nil {
		machines = []*domain.Machine{}
	}
	return machines, err
}
