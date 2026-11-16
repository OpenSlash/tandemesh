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

type connectRequest struct {
	Provider          string            `json:"provider"`
	Token             string            `json:"token"`
	BaseURL           string            `json:"baseUrl"`
	Instance          string            `json:"instance"`
	AgentID           string            `json:"agentId"`
	Name              string            `json:"name"`
	WorkspaceID       string            `json:"workspaceId"`
	Comment           *bool             `json:"comment"`
	Transition        string            `json:"transition"`
	CommentTemplate   string            `json:"commentTemplate"`
	WebhookSecret     string            `json:"webhookSecret"`
	ReuseConnectionID string            `json:"reuseConnectionId"`
	Fields            map[string]string `json:"fields,omitempty"`
}

// connectProvider validates a credential, stores it and creates a binding in
// one step, mirroring the HiFox connection flow.
func (s *Server) connectProvider(w http.ResponseWriter, r *http.Request) {
	var req connectRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeErr(w, http.StatusBadRequest, "E_BODY", err.Error())
		return
	}
	if req.Provider == "" {
		writeErr(w, http.StatusBadRequest, "E_FIELDS", "provider is required")
		return
	}
	ad, ok := s.integrations.Get(req.Provider)
	if !ok {
		writeErr(w, http.StatusBadRequest, "E_PROVIDER", "provider is not registered")
		return
	}
	if req.WorkspaceID == "" {
		req.WorkspaceID = "ws_default"
	}
	if !s.requireWorkspace(w, r, req.WorkspaceID) {
		return
	}
	fields := connectFields(ad.Manifest, &req)
	var reuseSource *domain.Connection
	if reuse := strings.TrimSpace(req.ReuseConnectionID); reuse != "" && fields["token"] == "" {
		source, err := s.store.GetConnection(r.Context(), reuse)
		if err != nil || source == nil || source.Provider != req.Provider || s.vault == nil {
			writeErr(w, http.StatusBadRequest, "E_REUSE", "source connection is not available")
			return
		}
		stored, ok, err := s.vault.Get(r.Context(), source.Provider, source.ID)
		if err != nil || !ok || stored.AccessToken == "" {
			writeErr(w, http.StatusBadRequest, "E_REUSE", "source credential is not available")
			return
		}
		reuseSource = source
		fields["token"] = stored.AccessToken
		if fields["baseUrl"] == "" {
			fields["baseUrl"] = source.BaseURL
		}
	}
	for _, field := range ad.Manifest.ConfigFields {
		if field.Required && fields[field.Key] == "" {
			writeErr(w, http.StatusBadRequest, "E_FIELDS", "missing required field: "+field.Key)
			return
		}
	}
	token := fields["token"]
	if token == "" && ad.Manifest.Auth != integration.AuthNone {
		writeErr(w, http.StatusBadRequest, "E_FIELDS", "token is required")
		return
	}
	instance := fields["instance"]
	baseURL := fields["baseUrl"]
	if req.Provider == "jira" && baseURL == "" {
		baseURL = instance
	}
	cred := integration.Credential{
		Provider: req.Provider, AccessToken: token, BaseURL: baseURL,
		Account: fields["email"], Fields: fields,
	}
	existing, _ := s.store.ListConnections(r.Context(), req.WorkspaceID, req.Provider)
	for _, item := range existing {
		if item.Instance == instance && strings.TrimRight(item.BaseURL, "/") == strings.TrimRight(baseURL, "/") {
			writeErr(w, http.StatusConflict, "E_EXISTS", "connection already exists")
			return
		}
	}
	if ad.Probe != nil {
		if err := ad.Probe.Probe(r.Context(), cred); err != nil {
			writeErr(w, http.StatusBadGateway, "E_PROBE", err.Error())
			return
		}
	}
	if reuseSource != nil {
		if instance == "" {
			writeErr(w, http.StatusBadRequest, "E_FIELDS", "instance is required")
			return
		}
		bindings, _ := s.store.ListBindings(r.Context(), reuseSource.WorkspaceID, reuseSource.ID)
		for _, item := range bindings {
			if item.ExternalID == instance {
				writeErr(w, http.StatusConflict, "E_EXISTS", "binding already exists")
				return
			}
		}
		binding := &domain.ResourceBinding{
			ID: "bind_" + ulid.Make().String(), WorkspaceID: req.WorkspaceID,
			ConnectionID: reuseSource.ID, Provider: reuseSource.Provider, ExternalID: instance,
			Name: bindingDisplayName(req.Name, instance), Kind: "resource", Direction: "read_write", Enabled: true,
		}
		if err := s.store.UpsertBinding(r.Context(), binding); err != nil {
			writeErr(w, http.StatusInternalServerError, "E_DB", err.Error())
			return
		}
		if ad.Manifest.Kind == integration.KindTask {
			automation := &domain.Automation{
				ID: "auto_" + ulid.Make().String(), WorkspaceID: req.WorkspaceID,
				Name: instance, Provider: reuseSource.Provider, Instance: instance,
				TriggerKind: domain.TriggerCreated, ConnectionID: reuseSource.ID,
				Writeback: map[string]any{"comment": true}, Enabled: true, Base: "main",
			}
			if err := s.store.UpsertAutomation(r.Context(), automation); err == nil {
				if binding.Settings == nil {
					binding.Settings = map[string]string{}
				}
				binding.Settings["automationId"] = automation.ID
				_ = s.store.UpsertBinding(r.Context(), binding)
			}
		}
		writeJSON(w, http.StatusCreated, map[string]any{"provider": req.Provider, "connection": reuseSource, "binding": binding, "probe": "ok"})
		return
	}
	name := req.Name
	if name == "" {
		name = req.Provider + " connection"
	}
	direction := fields["syncDirection"]
	if direction != "read" {
		direction = "read_write"
	}
	conn := &domain.Connection{
		ID: "conn_" + ulid.Make().String(), WorkspaceID: req.WorkspaceID,
		Provider: req.Provider, Name: name, Status: domain.ConnectionConnected,
		AuthMethod: string(ad.Manifest.Auth), SyncDirection: direction, Instance: instance, BaseURL: baseURL,
		AccountLogin: fields["email"], ExecutionMode: "cloud", Enabled: true,
	}
	cred.Subject = conn.ID
	if s.vault != nil {
		if err := s.vault.Put(r.Context(), cred); err != nil {
			writeErr(w, http.StatusInternalServerError, "E_VAULT", err.Error())
			return
		}
		if secret := fields["webhookSecret"]; secret != "" {
			if err := s.vault.Put(r.Context(), integration.Credential{
				Provider: req.Provider, Subject: "webhook:" + conn.ID, AccessToken: secret,
			}); err != nil {
				writeErr(w, http.StatusInternalServerError, "E_VAULT", err.Error())
				return
			}
		}
		if ad.Manifest.Kind == integration.KindChat {
			defaultCred := cred
			defaultCred.Subject = "default"
			if err := s.vault.Put(r.Context(), defaultCred); err != nil {
				writeErr(w, http.StatusInternalServerError, "E_VAULT", err.Error())
				return
			}
		}
	}
	if err := s.store.UpsertConnection(r.Context(), conn); err != nil {
		writeErr(w, http.StatusInternalServerError, "E_DB", err.Error())
		return
	}
	var automation *domain.Automation
	if ad.Manifest.Kind == integration.KindTask {
		comment := direction != "read"
		if req.Comment != nil && direction != "read" {
			comment = *req.Comment
		}
		automation = &domain.Automation{
			ID: "auto_" + ulid.Make().String(), WorkspaceID: req.WorkspaceID,
			Name: name, Provider: req.Provider, Instance: instance,
			AgentID: req.AgentID, TriggerKind: domain.TriggerCreated, ConnectionID: conn.ID,
			Writeback: map[string]any{"comment": comment}, Enabled: true, Base: "main",
		}
		if req.Transition != "" {
			automation.Writeback["transition"] = req.Transition
		}
		if req.CommentTemplate != "" {
			automation.Writeback["template"] = req.CommentTemplate
		}
		if err := s.store.UpsertAutomation(r.Context(), automation); err != nil {
			writeErr(w, http.StatusInternalServerError, "E_DB", err.Error())
			return
		}
		conn.AutomationID = automation.ID
		if err := s.store.UpsertConnection(r.Context(), conn); err != nil {
			writeErr(w, http.StatusInternalServerError, "E_DB", err.Error())
			return
		}
	}
	account := &domain.ConnectionAccount{
		ID: "acct_" + ulid.Make().String(), WorkspaceID: req.WorkspaceID,
		ConnectionID: conn.ID, Provider: req.Provider, Kind: accountKind(ad.Manifest.Auth),
		Login: fields["email"], Email: fields["email"], ExternalID: instance,
		CredentialSubject: conn.ID, IsDefault: true,
	}
	if account.Login == "" {
		account.Login = instance
	}
	if err := s.store.UpsertConnectionAccount(r.Context(), account); err != nil {
		writeErr(w, http.StatusInternalServerError, "E_DB", err.Error())
		return
	}
	if ad.Manifest.Kind == integration.KindTask && instance != "" {
		binding := &domain.ResourceBinding{
			ID: "bind_" + ulid.Make().String(), WorkspaceID: req.WorkspaceID,
			ConnectionID: conn.ID, Provider: req.Provider, ExternalID: instance,
			Name: bindingDisplayName(req.Name, instance), Kind: bindingKind(req.Provider), Direction: direction,
			AccountID: account.ID, Enabled: true,
		}
		if err := s.store.UpsertBinding(r.Context(), binding); err != nil {
			writeErr(w, http.StatusInternalServerError, "E_DB", err.Error())
			return
		}
	}
	webhookID := ""
	if automation != nil && ad.Registrar != nil {
		secret := fields["webhookSecret"]
		url := strings.TrimRight(s.cfg.BaseURL, "/") + "/webhooks/" + req.Provider + "/" + automation.ID
		if id, err := ad.Registrar.RegisterWebhook(r.Context(), cred, instance, url, secret); err != nil {
			s.log.Warn("auto-register webhook", "provider", req.Provider, "err", err)
		} else {
			webhookID = id
		}
	}
	response := map[string]any{
		"provider": req.Provider, "connection": conn, "probe": "ok", "webhookId": webhookID,
	}
	if automation != nil {
		response["automation"] = automationView(automation)
	}
	writeJSON(w, http.StatusCreated, response)
}

// connectFields maps both legacy top-level fields and manifest-driven fields
// into one normalized map.
func bindingDisplayName(name, instance string) string {
	if value := strings.TrimSpace(name); value != "" {
		return value
	}
	return instance
}

func connectFields(manifest integration.Manifest, req *connectRequest) map[string]string {
	fields := map[string]string{}
	for key, value := range req.Fields {
		fields[key] = strings.TrimSpace(value)
	}
	legacy := map[string]string{
		"token": req.Token, "instance": req.Instance, "baseUrl": req.BaseURL,
		"webhookSecret": req.WebhookSecret,
	}
	for key, value := range legacy {
		if fields[key] == "" {
			fields[key] = strings.TrimSpace(value)
		}
	}
	for _, field := range manifest.ConfigFields {
		if field.Key == "token" && fields[field.Key] == "" {
			fields[field.Key] = strings.TrimSpace(req.Token)
		}
	}
	return fields
}

type reconnectRequest struct {
	Token   string `json:"token"`
	BaseURL string `json:"baseUrl"`
}

// reconnectConnection replaces a connection token and revalidates it.
func (s *Server) reconnectConnection(w http.ResponseWriter, r *http.Request) {
	conn, err := s.store.GetConnection(r.Context(), chi.URLParam(r, "id"))
	if err != nil || conn == nil {
		writeErr(w, http.StatusNotFound, "E_CONNECTION", "connection not found")
		return
	}
	if !s.requireWorkspace(w, r, conn.WorkspaceID) {
		return
	}
	var req reconnectRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeErr(w, http.StatusBadRequest, "E_BODY", err.Error())
		return
	}
	if req.Token == "" {
		writeErr(w, http.StatusBadRequest, "E_FIELDS", "token is required")
		return
	}
	base := conn.BaseURL
	if req.BaseURL != "" {
		base = strings.TrimSpace(req.BaseURL)
	}
	cred := integration.Credential{Provider: conn.Provider, Subject: conn.ID, AccessToken: req.Token, BaseURL: base}
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
	conn.BaseURL = base
	conn.Status = domain.ConnectionConnected
	if err := s.store.UpsertConnection(r.Context(), conn); err != nil {
		writeErr(w, http.StatusInternalServerError, "E_DB", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"connection": conn})
}

// listConnections returns workspace-scoped provider connections.
func (s *Server) listConnections(w http.ResponseWriter, r *http.Request) {
	workspaceID := r.URL.Query().Get("workspace")
	if workspaceID == "" {
		workspaceID = "ws_default"
	}
	if !s.requireWorkspace(w, r, workspaceID) {
		return
	}
	all, err := s.store.ListAllConnections(r.Context())
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "E_DB", err.Error())
		return
	}
	providerFilter := r.URL.Query().Get("provider")
	items := make([]*domain.Connection, 0, len(all))
	for _, item := range all {
		if providerFilter != "" && item.Provider != providerFilter {
			continue
		}
		items = append(items, item)
	}
	automations, _ := s.store.ListAutomations(r.Context(), workspaceID)
	byConnection := map[string]*domain.Automation{}
	for _, automation := range automations {
		if automation.ConnectionID != "" {
			byConnection[automation.ConnectionID] = automation
		}
	}
	for _, c := range items {
		if automation := byConnection[c.ID]; automation != nil {
			c.AutomationID = automation.ID
			c.AgentID = automation.AgentID
			if transition, ok := automation.Writeback["transition"].(string); ok {
				c.Transition = transition
			}
		}
		c.Bindings, _ = s.store.ListBindings(r.Context(), workspaceID, c.ID)
		if all, err := s.store.ListAllBindings(r.Context()); err == nil {
			for _, binding := range all {
				if binding.ConnectionID == c.ID {
					c.AllBindings = append(c.AllBindings, binding)
				}
			}
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

// deleteConnectionByID removes one connection, its credential and bindings.
func (s *Server) deleteConnectionByID(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	conn, err := s.store.GetConnection(r.Context(), id)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "E_DB", err.Error())
		return
	}
	if conn == nil {
		writeErr(w, http.StatusNotFound, "E_CONNECTION", "connection not found")
		return
	}
	if !s.requireWorkspace(w, r, conn.WorkspaceID) {
		return
	}
	if ad, ok := s.integrations.Get(conn.Provider); ok {
		if unreg, ok := ad.Work.(integration.WebhookUnregistrar); ok {
			cred := s.providerCredential(r.Context(), conn.Provider, conn.ID)
			automations, _ := s.store.ListAutomations(r.Context(), conn.WorkspaceID)
			for _, automation := range automations {
				if automation.ConnectionID != conn.ID {
					continue
				}
				hookURL := strings.TrimRight(s.cfg.BaseURL, "/") + "/webhooks/" + conn.Provider + "/" + automation.ID
				if err := unreg.UnregisterWebhook(r.Context(), cred, conn.Instance, hookURL); err != nil {
					s.log.Warn("unregister webhook", "connection", conn.ID, "err", err)
				}
			}
		}
	}
	if s.vault != nil {
		_, _ = s.vault.Delete(r.Context(), conn.Provider, conn.ID)
		_, _ = s.vault.Delete(r.Context(), conn.Provider, "webhook:"+conn.ID)
	}
	automations, _ := s.store.ListAutomations(r.Context(), conn.WorkspaceID)
	for _, automation := range automations {
		if automation.ConnectionID == conn.ID {
			_, _ = s.store.DeleteAutomation(r.Context(), automation.ID)
		}
	}
	if _, err := s.store.DeleteConnection(r.Context(), conn.ID); err != nil {
		writeErr(w, http.StatusInternalServerError, "E_DB", err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// deleteConnection removes the credential and the matching workspace bindings.
func (s *Server) deleteConnection(w http.ResponseWriter, r *http.Request) {
	provider := chi.URLParam(r, "provider")
	workspaceID := r.URL.Query().Get("workspace")
	if workspaceID == "" {
		workspaceID = "ws_default"
	}
	if !s.requireWorkspace(w, r, workspaceID) {
		return
	}
	if s.vault != nil {
		_, _ = s.vault.Delete(r.Context(), provider, "default")
		_, _ = s.vault.Delete(r.Context(), provider, "webhook")
	}
	automations, err := s.store.ListAutomations(r.Context(), workspaceID)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "E_DB", err.Error())
		return
	}
	for _, automation := range automations {
		if automation.Provider == provider {
			_, _ = s.store.DeleteAutomation(r.Context(), automation.ID)
		}
	}
	w.WriteHeader(http.StatusNoContent)
}

type statesRequest struct {
	Provider string `json:"provider"`
	Token    string `json:"token"`
	BaseURL  string `json:"baseUrl"`
	Instance string `json:"instance"`
}

func (s *Server) integrationStates(w http.ResponseWriter, r *http.Request) {
	var req statesRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeErr(w, http.StatusBadRequest, "E_BODY", err.Error())
		return
	}
	ad, ok := s.integrations.Get(req.Provider)
	if !ok || ad.States == nil {
		writeErr(w, http.StatusBadRequest, "E_PROVIDER", "provider does not expose states")
		return
	}
	states, err := ad.States.ListStates(r.Context(), integration.Credential{
		Provider: req.Provider, AccessToken: req.Token, BaseURL: req.BaseURL,
	}, req.Instance)
	if err != nil {
		writeErr(w, http.StatusBadGateway, "E_PROBE", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": states})
}

func (s *Server) verifyWebhook(ctx context.Context, provider, connectionID string, wh integration.Webhook, r *http.Request, body []byte) error {
	if s.vault != nil {
		subjects := []string{"webhook"}
		if connectionID != "" {
			subjects = append([]string{"webhook:" + connectionID}, subjects...)
		}
		for _, subject := range subjects {
			if cred, ok, err := s.vault.Get(ctx, provider, subject); err == nil && ok && cred.AccessToken != "" {
				if verifier, ok := wh.(integration.SecretVerifier); ok {
					return verifier.VerifyWithSecret(r, body, cred.AccessToken)
				}
			}
		}
	}
	return wh.Verify(r, body)
}

type patchConnectionRequest struct {
	Name       *string           `json:"name"`
	BaseURL    *string           `json:"baseUrl"`
	Instance   *string           `json:"instance"`
	AgentID    *string           `json:"agentId"`
	Enabled    *bool             `json:"enabled"`
	Transition *string           `json:"transition"`
	StateMap   map[string]string `json:"stateMap"`
}

// patchConnection updates a connection and its binding.
func (s *Server) patchConnection(w http.ResponseWriter, r *http.Request) {
	conn, err := s.store.GetConnection(r.Context(), chi.URLParam(r, "id"))
	if err != nil || conn == nil {
		writeErr(w, http.StatusNotFound, "E_CONNECTION", "connection not found")
		return
	}
	if !s.requireWorkspace(w, r, conn.WorkspaceID) {
		return
	}
	var req patchConnectionRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeErr(w, http.StatusBadRequest, "E_BODY", err.Error())
		return
	}
	if req.Name != nil {
		conn.Name = *req.Name
	}
	if req.BaseURL != nil {
		conn.BaseURL = strings.TrimSpace(*req.BaseURL)
	}
	if req.Instance != nil {
		conn.Instance = strings.TrimSpace(*req.Instance)
	}
	if req.Enabled != nil {
		conn.Enabled = *req.Enabled
	}
	if err := s.store.UpsertConnection(r.Context(), conn); err != nil {
		writeErr(w, http.StatusInternalServerError, "E_DB", err.Error())
		return
	}
	automations, _ := s.store.ListAutomations(r.Context(), conn.WorkspaceID)
	for _, automation := range automations {
		if automation.ConnectionID != conn.ID {
			continue
		}
		if req.Name != nil {
			automation.Name = *req.Name
		}
		if req.Instance != nil {
			automation.Instance = conn.Instance
		}
		if req.AgentID != nil {
			automation.AgentID = *req.AgentID
		}
		if req.StateMap != nil {
			if automation.Writeback == nil {
				automation.Writeback = map[string]any{}
			}
			keys := map[string]string{"open": "stateOpen", "in_progress": "stateInProgress", "done": "stateDone"}
			for local, target := range req.StateMap {
				if key, ok := keys[local]; ok && target != "" {
					automation.Writeback[key] = target
				}
			}
		}
		if req.Transition != nil {
			if automation.Writeback == nil {
				automation.Writeback = map[string]any{}
			}
			if *req.Transition == "" {
				delete(automation.Writeback, "transition")
			} else {
				automation.Writeback["transition"] = *req.Transition
			}
		}
		if req.Name == nil && req.Instance == nil && req.AgentID == nil && req.Transition == nil {
			continue
		}
		_ = s.store.UpsertAutomation(r.Context(), automation)
	}
	writeJSON(w, http.StatusOK, map[string]any{"connection": conn})
}

// connectionStates lists the states a connection can transition to.
func (s *Server) connectionStates(w http.ResponseWriter, r *http.Request) {
	conn, err := s.store.GetConnection(r.Context(), chi.URLParam(r, "id"))
	if err != nil || conn == nil {
		writeErr(w, http.StatusNotFound, "E_CONNECTION", "connection not found")
		return
	}
	if !s.requireWorkspace(w, r, conn.WorkspaceID) {
		return
	}
	ad, ok := s.integrations.Get(conn.Provider)
	if !ok || ad.States == nil {
		writeErr(w, http.StatusBadRequest, "E_PROVIDER", "provider does not expose states")
		return
	}
	cred := s.providerCredential(r.Context(), conn.Provider, conn.ID)
	states, err := ad.States.ListStates(r.Context(), cred, conn.Instance)
	if err != nil {
		writeErr(w, http.StatusBadGateway, "E_STATES", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": states})
}

func (s *Server) connectionUsers(w http.ResponseWriter, r *http.Request) {
	conn, err := s.store.GetConnection(r.Context(), chi.URLParam(r, "id"))
	if err != nil || conn == nil {
		writeErr(w, http.StatusNotFound, "E_CONNECTION", "connection not found")
		return
	}
	if !s.requireWorkspace(w, r, conn.WorkspaceID) {
		return
	}
	ad, ok := s.integrations.Get(conn.Provider)
	if !ok {
		writeErr(w, http.StatusBadRequest, "E_PROVIDER", "provider not registered")
		return
	}
	searcher, ok := ad.Reader.(integration.UserSearcher)
	if !ok {
		writeErr(w, http.StatusBadRequest, "E_UNSUPPORTED", "provider does not expose users")
		return
	}
	cred := s.providerCredential(r.Context(), conn.Provider, conn.ID)
	if cred.BaseURL == "" {
		cred.BaseURL = conn.BaseURL
	}
	if cred.BaseURL == "" {
		cred.BaseURL = conn.Instance
	}
	items, err := searcher.SearchUsers(r.Context(), cred, r.URL.Query().Get("q"), 20)
	if err != nil {
		writeErr(w, http.StatusBadGateway, "E_USERS", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}
