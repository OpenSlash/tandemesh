package plane

import (
	"context"
	"encoding/json"
	"fmt"
	"html"
	"net/url"
	"strings"

	"github.com/OpenSlash/tandemesh/internal/domain"
	"github.com/OpenSlash/tandemesh/internal/integration"
)

// Comment implements integration.WorkWriter.
func (a *Adapter) Comment(ctx context.Context, work domain.WorkRef, cred integration.Credential, markdown string) (string, error) {
	slug, project, err := splitProjectInstance(work.Instance)
	if err != nil {
		return "", err
	}
	if !strings.Contains(markdown, integration.OutboundCommentMarker) {
		markdown += "\n\n" + integration.OutboundCommentMarker
	}
	body := map[string]any{
		"comment_html":     "<p>" + strings.ReplaceAll(html.EscapeString(markdown), "\n", "<br/>") + "</p>",
		"comment_stripped": markdown,
	}
	paths := []string{
		fmt.Sprintf("/workspaces/%s/projects/%s/work-items/%s/comments/", url.PathEscape(slug), url.PathEscape(project), url.PathEscape(work.ID)),
		fmt.Sprintf("/workspaces/%s/projects/%s/issues/%s/comments/", url.PathEscape(slug), url.PathEscape(project), url.PathEscape(work.ID)),
	}
	for _, path := range paths {
		raw, status, err := a.do(ctx, "POST", a.apiAt(cred, path), cred, body)
		if err != nil {
			if status == 404 {
				continue
			}
			return "", err
		}
		var out struct {
			ID json.RawMessage `json:"id"`
		}
		_ = json.Unmarshal(raw, &out)
		var id string
		if json.Unmarshal(out.ID, &id) != nil {
			var n int
			if json.Unmarshal(out.ID, &n) == nil {
				id = fmt.Sprintf("%d", n)
			}
		}
		if id != "" {
			return id, nil
		}
		return "comment", nil
	}
	return "", fmt.Errorf("plane: comment endpoint not found")
}

// Transition implements integration.WorkWriter by patching the work item state.
func (a *Adapter) Transition(ctx context.Context, work domain.WorkRef, cred integration.Credential, to string) error {
	slug, project, err := splitProjectInstance(work.Instance)
	if err != nil {
		return err
	}
	body := map[string]any{"state": to}
	paths := []string{
		fmt.Sprintf("/workspaces/%s/projects/%s/work-items/%s/", url.PathEscape(slug), url.PathEscape(project), url.PathEscape(work.ID)),
		fmt.Sprintf("/workspaces/%s/projects/%s/issues/%s/", url.PathEscape(slug), url.PathEscape(project), url.PathEscape(work.ID)),
	}
	for _, path := range paths {
		_, status, err := a.do(ctx, "PATCH", a.apiAt(cred, path), cred, body)
		if err != nil {
			if status == 404 {
				continue
			}
			return err
		}
		return nil
	}
	return fmt.Errorf("plane: transition endpoint not found")
}

// Attach implements integration.WorkWriter. Plane attachments need an upload
// handshake that is not implemented yet.
func (a *Adapter) Attach(context.Context, domain.WorkRef, integration.Credential, string, string) error {
	return integration.ErrUnsupported
}

// Assign implements integration.AssigneeWriter.
func (a *Adapter) Assign(ctx context.Context, work domain.WorkRef, cred integration.Credential, assignee string) error {
	slug, project, err := splitProjectInstance(work.Instance)
	if err != nil {
		return err
	}
	body := map[string]any{"assignees": []string{assignee}}
	paths := []string{
		fmt.Sprintf("/workspaces/%s/projects/%s/work-items/%s/", url.PathEscape(slug), url.PathEscape(project), url.PathEscape(work.ID)),
		fmt.Sprintf("/workspaces/%s/projects/%s/issues/%s/", url.PathEscape(slug), url.PathEscape(project), url.PathEscape(work.ID)),
	}
	for _, path := range paths {
		_, status, err := a.do(ctx, "PATCH", a.apiAt(cred, path), cred, body)
		if err != nil {
			if status == 404 {
				continue
			}
			return err
		}
		return nil
	}
	return fmt.Errorf("plane: assign endpoint not found")
}
