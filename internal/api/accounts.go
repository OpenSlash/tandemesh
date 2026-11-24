package api

import (
	"context"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/oklog/ulid/v2"

	"github.com/OpenSlash/tandemesh/internal/domain"
	"github.com/OpenSlash/tandemesh/internal/integration"
)

func accountKind(auth integration.AuthKind) string {
	if auth == integration.AuthBot {
		return "bot"
	}
	return "user"
}

type accountRequest struct {
	Token      string `json:"token"`
	Kind       string `json:"kind"`
	MemberID   string `json:"memberId"`
	Login      string `json:"login"`
	Email      string `json:"email"`
	ExternalID string `json:"externalId"`
	IsDefault  *bool  `json:"isDefault"`
}

func (s *Server) listConnectionAccounts(w http.ResponseWriter, r *http.Request) {
	conn, err := s.store.GetConnection(r.Context(), chi.URLParam(r, "id"))
	if err != nil || conn == nil {
		writeErr(w, http.StatusNotFound, "E_NOT_FOUND", "connection not found")
		return
	}
	if !s.requireWorkspace(w, r, conn.WorkspaceID) {
		return
	}
	items, err := s.store.ListConnectionAccounts(r.Context(), conn.ID)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "E_DB", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (s *Server) createConnectionAccount(w http.ResponseWriter, r *http.Request) {
	conn, err := s.store.GetConnection(r.Context(), chi.URLParam(r, "id"))
	if err != nil || conn == nil {
		writeErr(w, http.StatusNotFound, "E_NOT_FOUND", "connection not found")
		return
	}
	if !s.requireWorkspace(w, r, conn.WorkspaceID) {
		return
	}
	var req accountRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeErr(w, http.StatusBadRequest, "E_BODY", err.Error())
		return
	}
	token := strings.TrimSpace(req.Token)
	if token == "" {
		writeErr(w, http.StatusBadRequest, "E_FIELDS", "token is required")
		return
	}
	account := &domain.ConnectionAccount{
		ID: "acct_" + ulid.Make().String(), WorkspaceID: conn.WorkspaceID,
		ConnectionID: conn.ID, Provider: conn.Provider, Kind: req.Kind,
		MemberID: req.MemberID, Login: req.Login, Email: req.Email, ExternalID: req.ExternalID,
		CredentialSubject: "acct_" + ulid.Make().String(),
	}
	if account.Kind == "" {
		account.Kind = "user"
	}
	if account.Login == "" {
		account.Login = account.Email
	}
	existing, _ := s.store.ListConnectionAccounts(r.Context(), conn.ID)
	account.IsDefault = len(existing) == 0
	if req.IsDefault != nil {
		account.IsDefault = *req.IsDefault
	}
	cred := integration.Credential{
		Provider: conn.Provider, Subject: account.CredentialSubject,
		AccessToken: token, Account: account.Email,
		Fields: map[string]string{"login": account.Login, "kind": account.Kind},
	}
	if ad, ok := s.integrations.Get(conn.Provider); ok && ad.Probe != nil {
		if err := ad.Probe.Probe(r.Context(), cred); err != nil {
			writeErr(w, http.StatusBadGateway, "E_PROBE", err.Error())
			return
		}
	}
	if s.vault != nil {
		if err := s.vault.Put(r.Context(), cred); err != nil {
			writeErr(w, http.StatusInternalServerError, "E_VAULT", err.Error())
			return
		}
	}
	if err := s.store.UpsertConnectionAccount(r.Context(), account); err != nil {
		writeErr(w, http.StatusInternalServerError, "E_DB", err.Error())
		return
	}
	if account.IsDefault {
		s.copyAccountToConnection(r, conn, account, cred)
	}
	writeJSON(w, http.StatusCreated, map[string]any{"account": account})
}

func (s *Server) patchConnectionAccount(w http.ResponseWriter, r *http.Request) {
	account, err := s.store.GetConnectionAccount(r.Context(), chi.URLParam(r, "id"))
	if err != nil || account == nil {
		writeErr(w, http.StatusNotFound, "E_NOT_FOUND", "account not found")
		return
	}
	if !s.requireWorkspace(w, r, account.WorkspaceID) {
		return
	}
	var req accountRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeErr(w, http.StatusBadRequest, "E_BODY", err.Error())
		return
	}
	if req.Kind != "" {
		account.Kind = req.Kind
	}
	if req.MemberID != "" {
		account.MemberID = req.MemberID
	}
	if req.Login != "" {
		account.Login = req.Login
	}
	if req.Email != "" {
		account.Email = req.Email
	}
	if req.IsDefault != nil && *req.IsDefault {
		account.IsDefault = true
	}
	if err := s.store.UpsertConnectionAccount(r.Context(), account); err != nil {
		writeErr(w, http.StatusInternalServerError, "E_DB", err.Error())
		return
	}
	if account.IsDefault {
		conn, _ := s.store.GetConnection(r.Context(), account.ConnectionID)
		if s.vault != nil {
			if cred, ok, err := s.vault.Get(r.Context(), account.Provider, account.CredentialSubject); err == nil && ok {
				s.copyAccountToConnection(r, conn, account, cred)
			}
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{"account": account})
}

func (s *Server) copyAccountToConnection(r *http.Request, conn *domain.Connection, account *domain.ConnectionAccount, cred integration.Credential) {
	if conn == nil || s.vault == nil {
		return
	}
	cred.Subject = conn.ID
	if err := s.vault.Put(r.Context(), cred); err != nil {
		s.log.Warn("copy default account credential", "connection", conn.ID, "err", err)
		return
	}
	conn.AccountLogin = account.Login
	if err := s.store.UpsertConnection(r.Context(), conn); err != nil {
		s.log.Warn("update connection account login", "connection", conn.ID, "err", err)
	}
}

func (s *Server) deleteConnectionAccount(w http.ResponseWriter, r *http.Request) {
	account, err := s.store.GetConnectionAccount(r.Context(), chi.URLParam(r, "id"))
	if err != nil || account == nil {
		writeErr(w, http.StatusNotFound, "E_NOT_FOUND", "account not found")
		return
	}
	if !s.requireWorkspace(w, r, account.WorkspaceID) {
		return
	}
	if account.IsDefault {
		writeErr(w, http.StatusBadRequest, "E_DEFAULT", "choose another default account first")
		return
	}
	if _, err := s.store.DeleteConnectionAccount(r.Context(), account.ID); err != nil {
		writeErr(w, http.StatusInternalServerError, "E_DB", err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) accountCredential(ctx context.Context, provider, accountID string, fallback integration.Credential) integration.Credential {
	if accountID == "" {
		return fallback
	}
	account, err := s.store.GetConnectionAccount(ctx, accountID)
	if err != nil || account == nil || account.Provider != provider {
		return fallback
	}
	if s.vault != nil {
		if cred, ok, err := s.vault.Ensure(ctx, provider, account.CredentialSubject); err == nil && ok && cred.AccessToken != "" {
			return cred
		}
	}
	return fallback
}
