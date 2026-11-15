// Package integration defines one uniform abstraction for every external
// system Orca talks to.
//
// The insight is that task sources, chat tools, document stores and agent
// backends share the same plumbing: credentials, HTTP, retries, webhooks,
// health checks and audit. They differ in which typed operations they support.
//
// So there is exactly one Integration value, tagged with a Kind, carrying
// optional typed slots (Webhook, Work, Chat, Approver, Docs). The tag drives UI
// defaults and policy; the slots drive behaviour. Nothing is an untyped RPC bag.
package integration

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/OpenSlash/tandemesh/internal/domain"
)

// Kind tags what the integration is used for. It is metadata, not behaviour.
type Kind string

const (
	KindTask  Kind = "task"  // Jira, Linear, GitHub Issues
	KindChat  Kind = "chat"  // Slack, Feishu, Teams
	KindDoc   Kind = "doc"   // Notion, Outline, Confluence
	KindEmail Kind = "email" // Outlook, Gmail
	KindAgent Kind = "agent" // ACP backends, SDK sidecars
)

// Capability is a declared operation the integration supports.
type Capability string

const (
	CapWebhook    Capability = "webhook"
	CapComment    Capability = "comment"
	CapTransition Capability = "transition"
	CapAttach     Capability = "attach"
	CapSend       Capability = "send"
	CapApprove    Capability = "approve"
	CapSearch     Capability = "search"
	CapFetch      Capability = "fetch"
	CapRun        Capability = "run"
)

// AuthKind describes how credentials are obtained.
type AuthKind string

const (
	AuthNone   AuthKind = "none"
	AuthToken  AuthKind = "token"  // personal access token or app key
	AuthOAuth2 AuthKind = "oauth2" // authorization code flow
	AuthBot    AuthKind = "bot"    // bot token issued by the platform
	AuthApp    AuthKind = "app"    // provider app installation
)

// ConfigField describes one manifest-driven setup input. The UI renders these
// fields generically instead of hardcoding provider-specific forms.
type ConfigField struct {
	Key         string   `json:"key"`
	Label       string   `json:"label,omitempty"`
	LabelKey    string   `json:"labelKey,omitempty"`
	Type        string   `json:"type,omitempty"` // text|password|url|textarea|select
	Placeholder string   `json:"placeholder,omitempty"`
	Help        string   `json:"help,omitempty"`
	HelpKey     string   `json:"helpKey,omitempty"`
	Required    bool     `json:"required,omitempty"`
	Secret      bool     `json:"secret,omitempty"`
	Options     []string `json:"options,omitempty"`
}

// Manifest is the machine-readable descriptor exposed to the UI and policy
// engine. It is the only part of an integration the core inspects.
type Manifest struct {
	ID           string          `json:"id"`
	Name         string          `json:"name"`
	Kind         Kind            `json:"kind"`
	Auth         AuthKind        `json:"auth"`
	AuthOptions  []AuthOption    `json:"authOptions,omitempty"`
	Capabilities []Capability    `json:"capabilities"`
	ConfigFields []ConfigField   `json:"configFields,omitempty"`
	ConfigSchema json.RawMessage `json:"configSchema,omitempty"`
}

// Credential is passed to operations explicitly; integrations never read the
// process environment.
type Credential struct {
	Provider     string            `json:"provider"`
	BaseURL      string            `json:"baseUrl,omitempty"`
	Subject      string            `json:"subject"`
	Account      string            `json:"account,omitempty"`
	AccessToken  string            `json:"accessToken"`
	RefreshToken string            `json:"refreshToken,omitempty"`
	ExpiresAt    time.Time         `json:"expiresAt,omitempty"`
	Scopes       []string          `json:"scopes,omitempty"`
	Fields       map[string]string `json:"fields,omitempty"`
}

// Event is a normalized inbound delivery. Kind-specific details that policy
// needs (labels, assignee) are lifted into generic fields so the core never
// parses provider payloads.
type Event struct {
	Delivery    string              `json:"delivery"`
	Kind        string              `json:"kind"`
	Actor       string              `json:"actor"`
	Labels      []string            `json:"labels,omitempty"`
	Assignee    string              `json:"assignee,omitempty"`
	Attachments []Attachment        `json:"attachments,omitempty"`
	Fields      map[string][]string `json:"fields,omitempty"`
	Text        string              `json:"text,omitempty"`
	Raw         json.RawMessage     `json:"raw,omitempty"`
}

// Webhook covers task sources and any provider that pushes events.
type Webhook interface {
	Verify(r *http.Request, body []byte) error
	Parse(body []byte) (Event, error)
	Normalize(Event) (domain.WorkRef, domain.Snapshot, string, string, error)
}

// User is a provider account search result.
type User struct {
	ID    string `json:"id"`
	Login string `json:"login"`
	Name  string `json:"name,omitempty"`
	Email string `json:"email,omitempty"`
}

// UserSearcher finds provider users for identity mapping.
type UserSearcher interface {
	SearchUsers(ctx context.Context, cred Credential, query string, limit int) ([]User, error)
}

// FieldUpdate is a provider-neutral issue field update.
type FieldUpdate struct {
	Priority     string
	Components   []string
	FixVersions  []string
	CustomFields map[string]string
}

// FieldCatalog exposes provider field value candidates.
type FieldCatalog interface {
	ListFieldOptions(ctx context.Context, cred Credential, instance, project string) (map[string][]string, error)
}

// FieldWriter updates structured fields on a work item.
type FieldWriter interface {
	UpdateFields(ctx context.Context, work domain.WorkRef, cred Credential, update FieldUpdate) error
}

// AssigneeWriter assigns a provider user to a work item.
type AssigneeWriter interface {
	Assign(ctx context.Context, work domain.WorkRef, cred Credential, assignee string) error
}

// WorkWriter covers write-back to a task source.
type WorkWriter interface {
	Comment(ctx context.Context, work domain.WorkRef, cred Credential, markdown string) (string, error)
	Transition(ctx context.Context, work domain.WorkRef, cred Credential, to string) error
	Attach(ctx context.Context, work domain.WorkRef, cred Credential, kind, url string) error
}

// ChatMessage is the transport-neutral message shape.
type ChatMessage struct {
	Channel string          `json:"channel"`
	Text    string          `json:"text"`
	Thread  string          `json:"thread,omitempty"`
	Blocks  json.RawMessage `json:"blocks,omitempty"`
}

// ApprovalCard is the transport-neutral approval payload.
type ApprovalCard struct {
	ApprovalID string `json:"approvalId"`
	Channel    string `json:"channel"`
	RunID      string `json:"runId"`
	Tool       string `json:"tool"`
	Command    string `json:"command"`
	Risk       string `json:"risk"`
	ApproveURL string `json:"approveUrl"`
	DenyURL    string `json:"denyUrl"`
}

// Chat covers messaging integrations.
type Chat interface {
	Send(ctx context.Context, cred Credential, msg ChatMessage) (string, error)
	Update(ctx context.Context, cred Credential, ref string, msg ChatMessage) error
}

// Approver covers integrations that can carry an approval card.
type Approver interface {
	RequestApproval(ctx context.Context, cred Credential, card ApprovalCard) (string, error)
}

// DocHit and Document are the transport-neutral document shapes.
type DocHit struct {
	ID      string `json:"id,omitempty"`
	Title   string `json:"title"`
	URL     string `json:"url"`
	Snippet string `json:"snippet"`
}

type Document struct {
	Title     string `json:"title"`
	URL       string `json:"url"`
	Text      string `json:"text"`
	UpdatedAt string `json:"updatedAt,omitempty"`
}

// Docs covers read-only document sources.
type Docs interface {
	Search(ctx context.Context, cred Credential, query string, limit int) ([]DocHit, error)
	Fetch(ctx context.Context, cred Credential, url string) (Document, error)
}

// Integration is the single value every adapter returns. Optional slots decide
// what it can do; Manifest.Kind decides how the product presents it.
// WebhookUnregistrar removes a provider webhook when a connection is deleted.
type WebhookUnregistrar interface {
	UnregisterWebhook(ctx context.Context, cred Credential, instance, url string) error
}

// WebhookRegistrar creates a webhook in the provider when supported.
type WebhookRegistrar interface {
	RegisterWebhook(ctx context.Context, cred Credential, instance, url, secret string) (string, error)
}

type Integration struct {
	Manifest  Manifest
	Webhook   Webhook
	Registrar WebhookRegistrar
	Work      WorkWriter
	Chat      Chat
	Approver  Approver
	Docs      Docs
	Probe     Prober
	Reader    Reader
	States    StateLister
}

// State is a provider state a work item can transition to.
type State struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Group string `json:"group,omitempty"`
}

// StateLister exposes provider states for connection setup.
type StateLister interface {
	ListStates(ctx context.Context, cred Credential, instance string) ([]State, error)
}

// SecretVerifier verifies webhooks with a per-connection secret.
type SecretVerifier interface {
	VerifyWithSecret(r *http.Request, body []byte, secret string) error
}

// Prober checks that a credential actually works against the provider.
type Prober interface {
	Probe(ctx context.Context, cred Credential) error
}

// ListOptions narrows an inbox query.
type ListOptions struct {
	Instance string
	Limit    int
	Since    time.Time
	Fields   map[string]string
}

// WorkItem is a task source row for the inbox.
type WorkItem struct {
	Ref         domain.WorkRef      `json:"ref"`
	Title       string              `json:"title"`
	Body        string              `json:"body,omitempty"`
	Labels      []string            `json:"labels,omitempty"`
	Author      string              `json:"author,omitempty"`
	Assignee    string              `json:"assignee,omitempty"`
	Status      string              `json:"status,omitempty"`
	StatusGroup string              `json:"statusGroup,omitempty"`
	Attachments []Attachment        `json:"attachments,omitempty"`
	Fields      map[string][]string `json:"fields,omitempty"`
	Updated     time.Time           `json:"updated"`
}

// Attachment is a file linked from a work item.
type Attachment struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	URL       string    `json:"url"`
	Mime      string    `json:"mime,omitempty"`
	Size      int64     `json:"size,omitempty"`
	Author    string    `json:"author,omitempty"`
	CreatedAt time.Time `json:"createdAt,omitempty"`
}

// ImportPager searches one page of historical work items.
type ImportPager interface {
	SearchIssuesPage(ctx context.Context, cred Credential, instance, jql string, startAt, limit int) ([]WorkItem, error)
}

// ImportReader searches historical work items for a bulk import.
type ImportReader interface {
	SearchIssues(ctx context.Context, cred Credential, instance, jql string, limit int) ([]WorkItem, error)
}

// AttachmentReader streams a provider attachment with credentials applied.
type AttachmentReader interface {
	OpenAttachment(ctx context.Context, cred Credential, attachment Attachment) (io.ReadCloser, string, error)
}

// Comment is one platform discussion entry.
type Comment struct {
	ID          string       `json:"id"`
	Author      string       `json:"author,omitempty"`
	Body        string       `json:"body"`
	Attachments []Attachment `json:"attachments,omitempty"`
	CreatedAt   time.Time    `json:"createdAt"`
}

// CommentReader optionally exposes platform discussion for a work item.
type CommentReader interface {
	ListComments(ctx context.Context, cred Credential, ref domain.WorkRef) ([]Comment, error)
}

// Reader lists work assigned to the credential owner.
type Reader interface {
	ListAssigned(ctx context.Context, cred Credential, opts ListOptions) ([]WorkItem, error)
}

// PullRequest is a source-control pull request that may reference a task.
type PullRequest struct {
	Number            int
	URL               string
	Title             string
	Body              string
	State             string
	Branch            string
	Author            string
	AuthorAssociation string
}

// Commit is a source-control commit that may reference a task.
type Commit struct {
	SHA     string
	URL     string
	Message string
	Author  string
}

// CommitReader lists commits for a repository.
type CommitReader interface {
	ListCommits(ctx context.Context, cred Credential, repo string, limit int) ([]Commit, error)
}

// CommitReaderForRef lists commits for a branch or ref.
type CommitReaderForRef interface {
	ListCommitsForRef(ctx context.Context, cred Credential, repo, ref string, limit int) ([]Commit, error)
}

// PullRequestReader lists pull requests for a repository.
type PullRequestReader interface {
	ListPullRequests(ctx context.Context, cred Credential, repo string) ([]PullRequest, error)
}

// ErrUnsupported is returned when a slot is nil.
var ErrUnsupported = errors.New("operation not supported by this integration")

// Supported reports whether the integration declares and implements a capability.
func (i *Integration) Supported(c Capability) bool {
	if i == nil {
		return false
	}
	declared := false
	for _, have := range i.Manifest.Capabilities {
		if have == c {
			declared = true
			break
		}
	}
	if !declared {
		return false
	}
	switch c {
	case CapWebhook:
		return i.Webhook != nil
	case CapComment, CapTransition, CapAttach:
		return i.Work != nil
	case CapSend:
		return i.Chat != nil
	case CapApprove:
		return i.Approver != nil && i.Chat != nil
	case CapSearch, CapFetch:
		return i.Docs != nil
	}
	return false
}

// Registry holds the compiled-in integrations. One registry, many kinds.
type Registry struct {
	mu    sync.RWMutex
	items map[string]*Integration
}

// NewRegistry creates an empty registry.
func NewRegistry() *Registry {
	return &Registry{items: map[string]*Integration{}}
}

// Register validates and adds an integration.
func (r *Registry) Register(i *Integration) error {
	if i == nil || i.Manifest.ID == "" {
		return fmt.Errorf("integration manifest id is required")
	}
	switch i.Manifest.Kind {
	case KindTask, KindChat, KindDoc, KindEmail, KindAgent:
	default:
		return fmt.Errorf("integration %q has unknown kind %q", i.Manifest.ID, i.Manifest.Kind)
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.items[i.Manifest.ID]; exists {
		return fmt.Errorf("integration %q already registered", i.Manifest.ID)
	}
	r.items[i.Manifest.ID] = i
	return nil
}

// Get returns an integration by manifest id.
func (r *Registry) Get(id string) (*Integration, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	i, ok := r.items[id]
	return i, ok
}

// List returns manifests sorted by kind then id.
func (r *Registry) List() []Manifest {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]Manifest, 0, len(r.items))
	for _, i := range r.items {
		out = append(out, i.Manifest)
	}
	sort.Slice(out, func(a, b int) bool {
		if out[a].Kind != out[b].Kind {
			return out[a].Kind < out[b].Kind
		}
		return out[a].ID < out[b].ID
	})
	return out
}

// ListByKind filters manifests by kind.
func (r *Registry) ListByKind(kind Kind) []Manifest {
	all := r.List()
	out := all[:0]
	for _, m := range all {
		if m.Kind == kind {
			out = append(out, m)
		}
	}
	return out
}

// IsAuthError reports whether a provider error looks like an expired or revoked credential.
func IsAuthError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	for _, needle := range []string{"401", "403", "unauthorized", "forbidden", "invalid token", "authentication", "reauth"} {
		if strings.Contains(msg, needle) {
			return true
		}
	}
	return false
}

// FieldMetadata describes one editable Jira field.
type FieldMetadata struct {
	ID       string   `json:"id"`
	Name     string   `json:"name"`
	Type     string   `json:"type,omitempty"`
	Required bool     `json:"required"`
	Allowed  []string `json:"allowed,omitempty"`
}

// FieldMetadataReader exposes editable field metadata for a work item.
type FieldMetadataReader interface {
	ListFieldMetadata(ctx context.Context, cred Credential, instance, issue string) ([]FieldMetadata, error)
}

// AuthOption declares one supported authentication method for a provider.
type AuthOption struct {
	Kind     AuthKind      `json:"kind"`
	LabelKey string        `json:"labelKey,omitempty"`
	Default  bool          `json:"default,omitempty"`
	Fields   []ConfigField `json:"fields,omitempty"`
}
