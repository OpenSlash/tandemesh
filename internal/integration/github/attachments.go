package github

import (
	"context"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
	"path"
	"regexp"
	"strings"

	"github.com/OpenSlash/tandemesh/internal/integration"
)

var attachmentPattern = regexp.MustCompile(`https://(?:github\.com/user-attachments/assets/|user-images\.githubusercontent\.com/|private-user-images\.githubusercontent\.com/)[^\s<>"')]+`)

// attachmentsFromText extracts GitHub-hosted attachment links from markdown.
func attachmentsFromText(text string) []integration.Attachment {
	seen := map[string]bool{}
	out := make([]integration.Attachment, 0)
	for _, match := range attachmentPattern.FindAllString(text, -1) {
		match = strings.TrimRight(match, ".,;:!?")
		if seen[match] {
			continue
		}
		seen[match] = true
		parsed, err := url.Parse(match)
		if err != nil {
			continue
		}
		name := path.Base(parsed.Path)
		if name == "" || name == "." || name == "/" {
			name = "attachment"
		}
		out = append(out, integration.Attachment{
			ID: match, Name: name, URL: match, Mime: mime.TypeByExtension(path.Ext(name)),
		})
	}
	return out
}

// OpenAttachment implements integration.AttachmentReader.
func (a *Adapter) OpenAttachment(ctx context.Context, cred integration.Credential, attachment integration.Attachment) (io.ReadCloser, string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, attachment.URL, nil)
	if err != nil {
		return nil, "", err
	}
	if cred.AccessToken != "" {
		req.Header.Set("Authorization", "Bearer "+cred.AccessToken)
	}
	req.Header.Set("Accept", "application/octet-stream")
	resp, err := a.client().Do(req)
	if err != nil {
		return nil, "", err
	}
	if resp.StatusCode >= 300 {
		resp.Body.Close()
		return nil, "", fmt.Errorf("github attachment: status %d", resp.StatusCode)
	}
	contentType := resp.Header.Get("Content-Type")
	if contentType == "" {
		contentType = attachment.Mime
	}
	return resp.Body, contentType, nil
}
