package jira

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/OpenSlash/tandemesh/internal/integration"
)

func TestOpenAttachment(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/secure/attachment/1/spec.pdf" {
			t.Errorf("path = %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/pdf")
		_, _ = w.Write([]byte("pdf-bytes"))
	}))
	defer srv.Close()
	a := &Adapter{HTTPClient: srv.Client()}
	body, contentType, err := a.OpenAttachment(t.Context(), integration.Credential{AccessToken: "tok"},
		integration.Attachment{URL: srv.URL + "/secure/attachment/1/spec.pdf"})
	if err != nil {
		t.Fatal(err)
	}
	defer body.Close()
	raw, _ := io.ReadAll(body)
	if string(raw) != "pdf-bytes" || contentType != "application/pdf" {
		t.Fatalf("body = %q, type = %s", raw, contentType)
	}
}
