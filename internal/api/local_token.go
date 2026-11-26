package api

import (
	"context"
	"net/http"
	"strings"
)

// localTokenAuth protects browser/API traffic with either the per-install
// desktop token or a paired mobile bearer token. Pairing exchange and agentd
// endpoints use their own credential flows.
func (s *Server) localTokenAuth(next http.Handler) http.Handler {
	localToken := strings.TrimSpace(s.cfg.LocalToken)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/healthz", r.URL.Path == "/readyz", r.URL.Path == "/metrics":
			next.ServeHTTP(w, r)
			return
		case strings.HasPrefix(r.URL.Path, "/api/v1/agentd/"):
			next.ServeHTTP(w, r)
			return
		case r.URL.Path == "/ws":
			// The daemon socket authenticates with the join token in its own
			// `token` query parameter, which would otherwise be mistaken for
			// a local token and rejected.
			next.ServeHTTP(w, r)
			return
		case r.URL.Path == "/api/v1/mobile/pairings/exchange":
			next.ServeHTTP(w, r)
			return
		case strings.HasPrefix(r.URL.Path, "/api/v1/acp/icons/"):
			// Public registry icons are loaded via CSS mask/img, which cannot
			// attach an Authorization header. They contain no private data.
			next.ServeHTTP(w, r)
			return
		}

		presented := bearerCredential(r)
		if presented == "" {
			if cookie, err := r.Cookie("tandemesh_local_token"); err == nil {
				presented = cookie.Value
			}
		}
		if query := strings.TrimSpace(r.URL.Query().Get("token")); query != "" {
			presented = query
		}

		if presented != "" {
			if localToken != "" && constantTimeEqual(presented, localToken) {
				if r.URL.Query().Get("token") != "" {
					http.SetCookie(w, &http.Cookie{Name: "tandemesh_local_token", Value: localToken, Path: "/", HttpOnly: true, SameSite: http.SameSiteLaxMode})
				}
				next.ServeHTTP(w, r)
				return
			}
			if user, ok := s.mobileUser(r.Context(), presented); ok {
				next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), mobileUserContextKey{}, user)))
				return
			}
			if localToken != "" || strings.HasPrefix(presented, mobileTokenPrefix) {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}
		}
		if localToken == "" {
			next.ServeHTTP(w, r)
			return
		}
		http.Error(w, "unauthorized", http.StatusUnauthorized)
	})
}
