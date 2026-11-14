package api

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"strings"
	"time"

	"github.com/oklog/ulid/v2"
)

const machineJoinTTL = 30 * time.Minute

func (s *Server) createMachineJoinToken(w http.ResponseWriter, r *http.Request) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		writeErr(w, http.StatusInternalServerError, "E_RANDOM", err.Error())
		return
	}
	token := "mjt_" + hex.EncodeToString(raw)
	hash := sha256.Sum256([]byte(token))
	expires := time.Now().UTC().Add(machineJoinTTL)
	creator := userOf(r)
	if creator == "" {
		creator = s.cfg.User
	}
	if err := s.store.CreateMachineJoinToken(r.Context(),
		"mjt_"+ulid.Make().String(), hex.EncodeToString(hash[:]), creator, expires); err != nil {
		writeErr(w, http.StatusInternalServerError, "E_DB", err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{
		"token":     token,
		"expiresAt": expires,
	})
}

func (s *Server) authorizeMachineToken(ctx context.Context, token string) bool {
	if token == "" {
		return s.cfg.JoinToken == ""
	}
	if token == s.cfg.JoinToken {
		return true
	}
	hash := sha256.Sum256([]byte(token))
	ok, err := s.store.ConsumeMachineJoinToken(ctx, hex.EncodeToString(hash[:]))
	return err == nil && ok
}

func (s *Server) systemInfo(w http.ResponseWriter, r *http.Request) {
	base := strings.TrimRight(s.cfg.BaseURL, "/")
	if base == "" || strings.Contains(base, "localhost") || strings.Contains(base, "127.0.0.1") {
		scheme := "http"
		if r.TLS != nil {
			scheme = "https"
		}
		base = scheme + "://" + r.Host
	}
	writeJSON(w, http.StatusOK, map[string]any{"baseUrl": base})
}
