package api

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/oklog/ulid/v2"

	"github.com/OpenSlash/tandemesh/internal/domain"
)

const (
	mobilePairingTTL  = 5 * time.Minute
	mobileTokenPrefix = "tms_"
)

type mobileUserContextKey struct{}

func nowMS() int64 { return time.Now().UnixMilli() }

func randomHex(n int) (string, error) {
	buf := make([]byte, n)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}

func hashCredential(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}

func normalizePairingCode(value string) string {
	replacer := strings.NewReplacer("-", "", " ", "")
	return strings.ToUpper(replacer.Replace(strings.TrimSpace(value)))
}

func newPairingCode() (display, canonical string, err error) {
	raw, err := randomHex(4)
	if err != nil {
		return "", "", err
	}
	code := strings.ToUpper(raw)
	return code[:4] + "-" + code[4:], code, nil
}

func newMobileToken() (string, error) {
	raw, err := randomHex(32)
	if err != nil {
		return "", err
	}
	return mobileTokenPrefix + raw, nil
}

func bearerCredential(r *http.Request) string {
	header := strings.TrimSpace(r.Header.Get("Authorization"))
	if !strings.HasPrefix(header, "Bearer ") {
		return ""
	}
	return strings.TrimSpace(strings.TrimPrefix(header, "Bearer "))
}

// mobileUser resolves a mobile bearer token. It returns false for unknown or
// revoked tokens.
func (s *Server) mobileUser(ctx context.Context, credential string) (string, bool) {
	if !strings.HasPrefix(credential, mobileTokenPrefix) {
		return "", false
	}
	token, err := s.store.FindMobileToken(ctx, hashCredential(credential), nowMS())
	if err != nil {
		return "", false
	}
	return token.UserID, true
}

func mobileUserOf(r *http.Request) (string, bool) {
	user, ok := r.Context().Value(mobileUserContextKey{}).(string)
	return user, ok
}

type createPairingRequest struct {
	WorkspaceID string `json:"workspaceId"`
}

func (s *Server) createMobilePairing(w http.ResponseWriter, r *http.Request) {
	var req createPairingRequest
	if r.ContentLength > 0 {
		if err := decodeJSON(w, r, &req); err != nil {
			writeErr(w, http.StatusBadRequest, "E_BAD_REQUEST", "invalid JSON body")
			return
		}
	}
	display, canonical, err := newPairingCode()
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "E_INTERNAL", "generate pairing code")
		return
	}
	now := nowMS()
	pairing := &domain.MobilePairing{
		ID:          "pair_" + ulid.Make().String(),
		CodeHash:    hashCredential(canonical),
		UserID:      userOf(r),
		WorkspaceID: strings.TrimSpace(req.WorkspaceID),
		ExpiresAtMS: now + mobilePairingTTL.Milliseconds(),
		CreatedAtMS: now,
	}
	if err := s.store.CreateMobilePairing(r.Context(), pairing); err != nil {
		writeErr(w, http.StatusInternalServerError, "E_INTERNAL", "create pairing")
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{
		"id":        pairing.ID,
		"code":      display,
		"expiresAt": pairing.ExpiresAtMS,
	})
}

func (s *Server) getMobilePairing(w http.ResponseWriter, r *http.Request) {
	pairing, err := s.store.GetMobilePairing(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		writeErr(w, http.StatusNotFound, "E_NOT_FOUND", "pairing not found")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"id":        pairing.ID,
		"expiresAt": pairing.ExpiresAtMS,
		"used":      pairing.UsedAtMS > 0,
	})
}

type exchangePairingRequest struct {
	Code       string `json:"code"`
	DeviceName string `json:"deviceName"`
	Platform   string `json:"platform"`
}

func (s *Server) exchangeMobilePairing(w http.ResponseWriter, r *http.Request) {
	var req exchangePairingRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeErr(w, http.StatusBadRequest, "E_BAD_REQUEST", "invalid JSON body")
		return
	}
	code := normalizePairingCode(req.Code)
	if code == "" {
		writeErr(w, http.StatusBadRequest, "E_BAD_REQUEST", "pairing code required")
		return
	}
	deviceName := strings.TrimSpace(req.DeviceName)
	if deviceName == "" {
		deviceName = "Mobile device"
	}
	rawToken, err := newMobileToken()
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "E_INTERNAL", "generate token")
		return
	}
	token, err := s.store.ConsumeMobilePairing(r.Context(), hashCredential(code),
		"mtk_"+ulid.Make().String(), hashCredential(rawToken), deviceName,
		strings.TrimSpace(req.Platform), nowMS())
	if err != nil {
		writeErr(w, http.StatusUnauthorized, "E_PAIRING_INVALID", "pairing code is invalid or expired")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"token": rawToken,
		"device": map[string]any{
			"id":        token.ID,
			"name":      token.DeviceName,
			"platform":  token.Platform,
			"createdAt": token.CreatedAtMS,
		},
		"user": token.UserID,
	})
}

func (s *Server) listMobileDevices(w http.ResponseWriter, r *http.Request) {
	user := userOf(r)
	if mobileUser, ok := mobileUserOf(r); ok {
		user = mobileUser
	}
	tokens, err := s.store.ListMobileTokens(r.Context(), user)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "E_INTERNAL", "list devices")
		return
	}
	items := make([]map[string]any, 0, len(tokens))
	for _, token := range tokens {
		items = append(items, map[string]any{
			"id":         token.ID,
			"name":       token.DeviceName,
			"platform":   token.Platform,
			"createdAt":  token.CreatedAtMS,
			"lastUsedAt": token.LastUsedMS,
			"revokedAt":  token.RevokedAtMS,
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (s *Server) revokeMobileDevice(w http.ResponseWriter, r *http.Request) {
	user := userOf(r)
	if mobileUser, ok := mobileUserOf(r); ok {
		user = mobileUser
	}
	if err := s.store.RevokeMobileToken(r.Context(), chi.URLParam(r, "id"), user, nowMS()); err != nil {
		writeErr(w, http.StatusNotFound, "E_NOT_FOUND", "device not found")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// constantTimeEqual is used for local-token comparisons.
func constantTimeEqual(a, b string) bool {
	return len(a) == len(b) && subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}
