package github

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/OpenSlash/tandemesh/internal/integration"
)

func TestAttachmentsFromText(t *testing.T) {
	text := "see https://github.com/user-attachments/assets/abc-123 and " +
		"https://user-images.githubusercontent.com/1/2/photo.png. again " +
		"https://github.com/user-attachments/assets/abc-123"
	got := attachmentsFromText(text)
	if len(got) != 2 {
		t.Fatalf("attachments = %+v", got)
	}
	if got[0].ID != "https://github.com/user-attachments/assets/abc-123" || got[1].Name != "photo.png" {
		t.Fatalf("attachments = %+v", got)
	}
}

func TestOpenAttachment(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer tok" {
			t.Errorf("auth = %q", r.Header.Get("Authorization"))
		}
		_, _ = w.Write([]byte("payload"))
	}))
	defer srv.Close()
	a := &Adapter{HTTPClient: srv.Client()}
	body, contentType, err := a.OpenAttachment(t.Context(), integration.Credential{AccessToken: "tok"}, integration.Attachment{URL: srv.URL + "/file.png", Name: "file.png", Mime: "image/png"})
	if err != nil {
		t.Fatal(err)
	}
	defer body.Close()
	raw, _ := io.ReadAll(body)
	if string(raw) != "payload" || contentType == "" {
		t.Fatalf("body=%q type=%q", raw, contentType)
	}
}
