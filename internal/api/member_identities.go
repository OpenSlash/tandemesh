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

type memberIdentityRequest struct {
	MemberID   string `json:"memberId"`
	Provider   string `json:"provider"`
	ExternalID string `json:"externalId"`
	Login      string `json:"login"`
	Email      string `json:"email"`
	Token      string `json:"token"`
}

func (s *Server) listMemberIdentities(w http.ResponseWriter, r *http.Request) {
	workspaceID := chi.URLParam(r, "workspace")
	if workspaceID == "" {
		workspaceID = "ws_default"
	}
	if !s.requireWorkspace(w, r, workspaceID) {
		return
	}
	items, err := s.store.ListMemberIdentities(r.Context(), workspaceID, r.URL.Query().Get("memberId"))
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "E_DB", err.Error())
		return
	}
	if items == nil {
		items = []*domain.MemberIdentity{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (s *Server) createMemberIdentity(w http.ResponseWriter, r *http.Request) {
	workspaceID := chi.URLParam(r, "workspace")
	if workspaceID == "" {
		workspaceID = "ws_default"
	}
	if !s.requireWorkspace(w, r, workspaceID) {
		return
	}
	var req memberIdentityRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeErr(w, http.StatusBadRequest, "E_BODY", err.Error())
		return
	}
	req.MemberID = strings.TrimSpace(req.MemberID)
	req.Provider = strings.TrimSpace(req.Provider)
	req.ExternalID = strings.TrimSpace(req.ExternalID)
	if req.MemberID == "" || req.Provider == "" || req.ExternalID == "" {
		writeErr(w, http.StatusBadRequest, "E_FIELDS", "memberId, provider and externalId are required")
		return
	}
	identity := &domain.MemberIdentity{
		ID: "mid_" + ulid.Make().String(), WorkspaceID: workspaceID,
		MemberID: req.MemberID, Provider: req.Provider, ExternalID: req.ExternalID,
		Login: req.Login, Email: req.Email,
	}
	if identity.Login == "" {
		identity.Login = identity.ExternalID
	}
	identity.CredentialSubject = "mid:" + identity.ID
	if token := strings.TrimSpace(req.Token); token != "" {
		cred := integration.Credential{Provider: req.Provider, Subject: identity.CredentialSubject, AccessToken: token, Account: req.Email}
		if ad, ok := s.integrations.Get(req.Provider); ok && ad.Probe != nil {
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
	}
	if err := s.store.UpsertMemberIdentity(r.Context(), identity); err != nil {
		writeErr(w, http.StatusInternalServerError, "E_DB", err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"identity": identity})
}

func (s *Server) deleteMemberIdentity(w http.ResponseWriter, r *http.Request) {
	identity, err := s.store.GetMemberIdentity(r.Context(), chi.URLParam(r, "id"))
	if err != nil || identity == nil {
		writeErr(w, http.StatusNotFound, "E_NOT_FOUND", "member identity not found")
		return
	}
	if !s.requireWorkspace(w, r, identity.WorkspaceID) {
		return
	}
	if _, err := s.store.DeleteMemberIdentity(r.Context(), identity.ID); err != nil {
		writeErr(w, http.StatusInternalServerError, "E_DB", err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// memberCredential resolves the acting member's personal provider credential.
func (s *Server) memberCredential(ctx context.Context, workspaceID, provider string) (integration.Credential, bool) {
	memberID, _ := ctx.Value(userContextKey{}).(string)
	return s.memberCredentialFor(ctx, workspaceID, provider, memberID)
}

// memberCredentialFor resolves one member's personal provider credential.
func (s *Server) memberCredentialFor(ctx context.Context, workspaceID, provider, memberID string) (integration.Credential, bool) {
	if memberID == "" {
		return integration.Credential{}, false
	}
	items, err := s.store.ListMemberIdentities(ctx, workspaceID, memberID)
	if err != nil {
		return integration.Credential{}, false
	}
	for _, identity := range items {
		if identity.Provider != provider || identity.CredentialSubject == "" || s.vault == nil {
			continue
		}
		if cred, ok, err := s.vault.Get(ctx, provider, identity.CredentialSubject); err == nil && ok && cred.AccessToken != "" {
			return cred, true
		}
	}
	return integration.Credential{}, false
}
