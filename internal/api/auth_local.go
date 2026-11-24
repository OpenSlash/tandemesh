package api

import (
	"context"
	"net/http"
)

func headerWithLegacy(r *http.Request, name, legacy string) string {
	if value := r.Header.Get(name); value != "" {
		return value
	}
	return r.Header.Get(legacy)
}

func (s *Server) identityUser(r *http.Request) string {
	if user := headerWithLegacy(r, "X-Tandemesh-User", "X-Orca-User"); user != "" {
		return user
	}
	if s.cfg.User != "" {
		return s.cfg.User
	}
	return "me"
}

func userOfContext(ctx context.Context) string {
	user, _ := ctx.Value(userContextKey{}).(string)
	return user
}

func (s *Server) authMe(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"name": userOf(r), "local": true})
}

func (s *Server) authLogout(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{Name: "orca_session", Value: "", Path: "/", MaxAge: -1, HttpOnly: true})
	w.WriteHeader(http.StatusNoContent)
}
