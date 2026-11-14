package plane

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"time"

	"github.com/OpenSlash/tandemesh/internal/domain"
	"github.com/OpenSlash/tandemesh/internal/integration"
)

type commentRow struct {
	ID              string `json:"id"`
	CommentStripped string `json:"comment_stripped"`
	CommentHTML     string `json:"comment_html"`
	CreatedAt       string `json:"created_at"`
	CreatedBy       string `json:"created_by"`
	Actor           string `json:"actor"`
}

type commentPage struct {
	Results []commentRow `json:"results"`
}

func (a *Adapter) ListComments(ctx context.Context, cred integration.Credential, ref domain.WorkRef) ([]integration.Comment, error) {
	slug, project, err := splitInstance(ref.Instance)
	if err != nil {
		return nil, err
	}
	path := fmt.Sprintf("/workspaces/%s/projects/%s/issues/%s/comments/?per_page=100", url.PathEscape(slug), url.PathEscape(project), url.PathEscape(ref.ID))
	raw, status, err := a.do(ctx, "GET", a.apiAt(cred, path), cred, nil)
	if err != nil {
		if status == 404 {
			return nil, nil
		}
		return nil, err
	}
	var list []commentRow
	if json.Unmarshal(raw, &list) != nil {
		var page commentPage
		if err := json.Unmarshal(raw, &page); err != nil {
			return nil, err
		}
		list = page.Results
	}
	out := make([]integration.Comment, 0, len(list))
	members := a.memberIndex(ctx, cred, slug)
	for _, row := range list {
		body := row.CommentStripped
		if body == "" {
			body = stripHTML(row.CommentHTML)
		}
		created, _ := time.Parse(time.RFC3339, row.CreatedAt)
		author := row.Actor
		if author == "" {
			author = row.CreatedBy
		}
		if name, ok := members[author]; ok && name != "" {
			author = name
		}
		out = append(out, integration.Comment{ID: row.ID, Author: author, Body: body, CreatedAt: created})
	}
	return out, nil
}
