// Package ingress verifies and normalizes inbound webhooks from external
// task sources. It only extracts "what changed"; the full item is never
// copied into the database.
package ingress

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/OpenSlash/tandemesh/internal/domain"
)

// ErrBadSignature is returned when the HMAC does not match.
var ErrBadSignature = errors.New("bad signature")

// VerifyGitHub checks the X-Hub-Signature-256 header.
func VerifyGitHub(secret string, body []byte, header string) error {
	if secret == "" {
		return nil // development mode; the caller must warn.
	}
	const prefix = "sha256="
	if !strings.HasPrefix(header, prefix) {
		return ErrBadSignature
	}
	want, err := hex.DecodeString(strings.TrimPrefix(header, prefix))
	if err != nil {
		return ErrBadSignature
	}
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	if !hmac.Equal(want, mac.Sum(nil)) {
		return ErrBadSignature
	}
	return nil
}

// GitHubPayload is the subset of a GitHub issue event Orca needs.
type GitHubPayload struct {
	Action string `json:"action"`
	Issue  struct {
		Number    int    `json:"number"`
		Title     string `json:"title"`
		Body      string `json:"body"`
		HTMLURL   string `json:"html_url"`
		UpdatedAt string `json:"updated_at"`
		Labels    []struct {
			Name string `json:"name"`
		} `json:"labels"`
		User struct {
			Login string `json:"login"`
		} `json:"user"`
	} `json:"issue"`
	Repository struct {
		FullName string `json:"full_name"`
		HTMLURL  string `json:"html_url"`
	} `json:"repository"`
	Comment struct {
		Body string `json:"body"`
		User struct {
			Login string `json:"login"`
		} `json:"user"`
	} `json:"comment"`
	Assignee struct {
		Login string `json:"login"`
	} `json:"assignee"`
	Sender struct {
		Login string `json:"login"`
	} `json:"sender"`
}

// ParseGitHub decodes a GitHub issue payload.
func ParseGitHub(body []byte) (*GitHubPayload, error) {
	var p GitHubPayload
	if err := json.Unmarshal(body, &p); err != nil {
		return nil, fmt.Errorf("decode github payload: %w", err)
	}
	if p.Issue.Number == 0 || p.Issue.HTMLURL == "" {
		return nil, errors.New("not an issue event")
	}
	return &p, nil
}

// NormalizeGitHub converts a GitHub issue event into a WorkRef, a snapshot and
// a prompt template. The returned skip reason is non-empty when the event
// should not start a run.
func NormalizeGitHub(p *GitHubPayload) (domain.WorkRef, domain.Snapshot, string, string) {
	labels := make([]string, 0, len(p.Issue.Labels))
	for _, l := range p.Issue.Labels {
		labels = append(labels, l.Name)
	}
	work := domain.WorkRef{
		Provider: "github",
		Instance: p.Repository.FullName,
		Type:     "issue",
		ID:       fmt.Sprintf("%d", p.Issue.Number),
		URL:      p.Issue.HTMLURL,
		Revision: p.Issue.UpdatedAt,
	}
	author := p.Issue.User.Login
	if p.Comment.User.Login != "" {
		author = p.Comment.User.Login
	} else if author == "" {
		author = p.Sender.Login
	}
	snap := domain.Snapshot{
		Title:    p.Issue.Title,
		Body:     p.Issue.Body,
		BodyHash: hashBody(p.Issue.Body),
		Labels:   labels,
		Author:   author,
	}
	// Only opened / labeled / assigned / reopened events and comment commands
	// start work.
	switch p.Action {
	case "opened", "reopened", "labeled", "assigned":
	case "created":
		if p.Comment.Body == "" {
			return work, snap, "", "SKIP_EVENT_ACTION"
		}
	default:
		return work, snap, "", "SKIP_EVENT_ACTION"
	}
	prompt := "Fix the following issue.\n\nTitle: " + p.Issue.Title + "\nURL: " + p.Issue.HTMLURL
	if p.Comment.Body != "" {
		prompt = "The user requested the following on the issue.\n\nCommand: " + p.Comment.Body +
			"\n\nIssue: " + p.Issue.Title + "\nURL: " + p.Issue.HTMLURL
	}
	return work, snap, prompt, ""
}

func hashBody(body string) string {
	sum := sha256.Sum256([]byte(body))
	return hex.EncodeToString(sum[:])
}
