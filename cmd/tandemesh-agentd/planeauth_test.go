package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestPlaneClientAuthorization(t *testing.T) {
	var got string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = r.Header.Get("Authorization")
	}))
	defer srv.Close()

	tests := []struct {
		name  string
		plane string
		token string
		want  string
	}{
		{"plane host gets token", srv.URL, "secret", "Bearer secret"},
		{"other host never sees token", "http://plane.invalid:1", "secret", ""},
		{"no token configured", srv.URL, "", ""},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got = ""
			resp, err := newPlaneClient(tc.plane, tc.token, 5*time.Second).Get(srv.URL)
			if err != nil {
				t.Fatalf("get: %v", err)
			}
			resp.Body.Close()
			if got != tc.want {
				t.Fatalf("Authorization = %q, want %q", got, tc.want)
			}
		})
	}
}
