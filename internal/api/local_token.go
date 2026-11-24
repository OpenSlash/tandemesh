package api

import (
	"crypto/subtle"
	"net/http"
	"strings"
)

// localTokenAuth protects browser/API traffic when a desktop launcher supplies
// a per-install token. Agentd endpoints keep their existing join-token flow.
func (s *Server) localTokenAuth(next http.Handler) http.Handler {
	token := strings.TrimSpace(s.cfg.LocalToken)
	if token == "" {
		return next
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/healthz", r.URL.Path == "/readyz", r.URL.Path == "/metrics":
			next.ServeHTTP(w, r)
			return
		case strings.HasPrefix(r.URL.Path, "/api/v1/agentd/"):
			next.ServeHTTP(w, r)
			return
		}
		header := strings.TrimSpace(r.Header.Get("Authorization"))
		presented := strings.TrimPrefix(header, "Bearer ")
		if cookie, err := r.Cookie("tandemesh_local_token"); err == nil {
			presented = cookie.Value
		}
		if query := strings.TrimSpace(r.URL.Query().Get("token")); query != "" {
			presented = query
		}
		if len(presented) == len(token) && subtle.ConstantTimeCompare([]byte(presented), []byte(token)) == 1 {
			if r.URL.Query().Get("token") != "" {
				http.SetCookie(w, &http.Cookie{Name: "tandemesh_local_token", Value: token, Path: "/", HttpOnly: true, SameSite: http.SameSiteLaxMode})
			}
			next.ServeHTTP(w, r)
			return
		}
		http.Error(w, "unauthorized", http.StatusUnauthorized)
	})
}
