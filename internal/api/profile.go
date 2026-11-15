package api

import (
	"crypto/subtle"
	"net"
	"net/http"
	"strings"
)

func (s *Server) isPublicProfile() bool {
	return strings.EqualFold(s.cfg.Profile, "public")
}

func (s *Server) publicBaseURL() string {
	if value := strings.TrimRight(s.cfg.PublicURL, "/"); value != "" {
		return value
	}
	return strings.TrimRight(s.cfg.BaseURL, "/")
}

func (s *Server) oauthBaseURL() string {
	mode := strings.ToLower(s.cfg.OAuthRedirect)
	if mode == "public" || (mode != "loopback" && s.isPublicProfile()) {
		return s.publicBaseURL()
	}
	host, port, err := net.SplitHostPort(s.cfg.Listen)
	if err != nil {
		return "http://127.0.0.1:8080"
	}
	if host == "" || host == "0.0.0.0" || host == "::" {
		host = "127.0.0.1"
	}
	return "http://" + net.JoinHostPort(host, port)
}

func (s *Server) publicAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !s.isPublicProfile() || !strings.EqualFold(s.cfg.AuthMode, "basic") {
			next.ServeHTTP(w, r)
			return
		}
		path := r.URL.Path
		if strings.HasPrefix(path, "/webhooks/") || strings.HasPrefix(path, "/api/v1/integrations/") && strings.HasSuffix(path, "/oauth/callback") ||
			path == "/healthz" || path == "/readyz" || path == "/metrics" || path == "/api/v1/capabilities" {
			next.ServeHTTP(w, r)
			return
		}
		user, password, ok := r.BasicAuth()
		if !ok || user != "admin" || subtle.ConstantTimeCompare([]byte(password), []byte(s.cfg.AdminPassword)) != 1 {
			w.Header().Set("WWW-Authenticate", `Basic realm="Tandemesh"`)
			http.Error(w, "authentication required", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (s *Server) capabilities(w http.ResponseWriter, r *http.Request) {
	mode := strings.ToLower(s.cfg.OAuthRedirect)
	oauthRedirect := s.oauthBaseURL()
	writeJSON(w, http.StatusOK, map[string]any{
		"profile":       s.cfg.Profile,
		"authMode":      s.cfg.AuthMode,
		"publicUrl":     s.publicBaseURL(),
		"oauthRedirect": oauthRedirect,
		"loopbackOAuth": !s.isPublicProfile() || mode == "loopback",
		"publicOAuth":   s.isPublicProfile() || mode == "public",
		"webhook":       s.isPublicProfile() && strings.HasPrefix(s.publicBaseURL(), "https://"),
		"polling":       true,
	})
}
