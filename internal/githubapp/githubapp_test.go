package githubapp

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestInstallationToken(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	pemKey := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.Header.Get("Authorization"), "Bearer eyJ") {
			t.Errorf("authorization = %s", r.Header.Get("Authorization"))
		}
		if r.URL.Path != "/app/installations/7/access_tokens" || r.Method != http.MethodPost {
			t.Errorf("request = %s %s", r.Method, r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"token":"ghs_test","expires_at":"2030-01-01T00:00:00Z"}`))
	}))
	defer srv.Close()
	client := New("123", "orca", string(pemKey), srv.URL)
	token, expires, err := client.InstallationToken(t.Context(), 7)
	if err != nil {
		t.Fatal(err)
	}
	if token != "ghs_test" || expires.Year() != 2030 {
		t.Fatalf("token=%+v expires=%v", token, expires)
	}
	// Cache should avoid a second request even if the test server is closed.
	srv.Close()
	if token, _, err := client.InstallationToken(t.Context(), 7); err != nil || token != "ghs_test" {
		t.Fatalf("cached token=%q err=%v", token, err)
	}
}
