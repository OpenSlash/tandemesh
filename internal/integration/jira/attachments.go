package jira

import (
	"context"
	"fmt"
	"io"
	"net/http"

	"github.com/OpenSlash/tandemesh/internal/integration"
)

// OpenAttachment implements integration.AttachmentReader.
func (a *Adapter) OpenAttachment(ctx context.Context, cred integration.Credential, attachment integration.Attachment) (io.ReadCloser, string, error) {
	if attachment.URL == "" {
		return nil, "", fmt.Errorf("jira: attachment url is required")
	}
	resp, err := a.do(ctx, http.MethodGet, attachment.URL, cred, nil)
	if err != nil {
		return nil, "", err
	}
	if resp.StatusCode >= 300 {
		resp.Body.Close()
		return nil, "", fmt.Errorf("jira attachment: status %d", resp.StatusCode)
	}
	contentType := resp.Header.Get("Content-Type")
	if contentType == "" {
		contentType = attachment.Mime
	}
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	return resp.Body, contentType, nil
}
