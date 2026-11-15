package github

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/OpenSlash/tandemesh/internal/domain"
	"github.com/OpenSlash/tandemesh/internal/integration"
)

func TestListCommentsAttachments(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`[{"id":1,"body":"see https://github.com/user-attachments/assets/abc","created_at":"2026-11-15T00:00:00Z","user":{"login":"alice"}}]`))
	}))
	defer srv.Close()
	a := &Adapter{HTTPClient: srv.Client(), BaseURL: srv.URL}
	comments, err := a.ListComments(t.Context(), integration.Credential{AccessToken: "tok"}, domain.WorkRef{Instance: "acme/api", ID: "1"})
	if err != nil {
		t.Fatal(err)
	}
	if len(comments) != 1 || len(comments[0].Attachments) != 1 {
		t.Fatalf("comments = %+v", comments)
	}
}
