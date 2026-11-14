// Package credentials stores provider secrets encrypted at rest and returns
// them only to the adapters that need them.
package credentials

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/OpenSlash/tandemesh/internal/integration"
	"github.com/OpenSlash/tandemesh/internal/store"
)

// Vault encrypts credentials with AES-256-GCM.
type Vault struct {
	store *store.Store
	aead  cipher.AEAD
	log   *slog.Logger

	// HTTPClient is used for OAuth refresh; nil means a default client.
	HTTPClient *http.Client
	// OAuth configures refresh clients per provider.
	OAuth OAuthConfig
}

// New derives a 256-bit key from the configured secret.
func New(st *store.Store, secret string, log *slog.Logger) (*Vault, error) {
	if secret == "" {
		secret = "orca-development-secret"
	}
	if log == nil {
		log = slog.Default()
	}
	sum := sha256.Sum256([]byte(secret))
	block, err := aes.NewCipher(sum[:])
	if err != nil {
		return nil, err
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	return &Vault{store: st, aead: aead, log: log}, nil
}

func subjectOf(subject string) string {
	if strings.TrimSpace(subject) == "" {
		return "default"
	}
	return subject
}

// Put encrypts and stores a credential.
func (v *Vault) Put(ctx context.Context, cred integration.Credential) error {
	if cred.Provider == "" {
		return errors.New("credential provider is required")
	}
	cred.Subject = subjectOf(cred.Subject)
	raw, err := json.Marshal(cred)
	if err != nil {
		return err
	}
	nonce := make([]byte, v.aead.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return err
	}
	sealed := v.aead.Seal(nonce, nonce, raw, nil)
	ciphertext := "v1:" + base64.StdEncoding.EncodeToString(sealed)

	kind := "token"
	if cred.RefreshToken != "" {
		kind = "oauth2"
	}
	var expires *time.Time
	if !cred.ExpiresAt.IsZero() {
		t := cred.ExpiresAt
		expires = &t
	}
	return v.store.PutCredential(ctx, cred.Provider, cred.Subject, kind, ciphertext, map[string]any{
		"scopes": cred.Scopes,
	}, expires)
}

// Get decrypts a credential. It reports false when none is stored.
func (v *Vault) Get(ctx context.Context, provider, subject string) (integration.Credential, bool, error) {
	ciphertext, _, _, _, ok, err := v.store.GetCredential(ctx, provider, subjectOf(subject))
	if err != nil || !ok {
		return integration.Credential{}, false, err
	}
	raw, err := decode(ciphertext)
	if err != nil {
		return integration.Credential{}, false, err
	}
	plain, err := v.aead.Open(nil, raw[:v.aead.NonceSize()], raw[v.aead.NonceSize():], nil)
	if err != nil {
		return integration.Credential{}, false, fmt.Errorf("decrypt credential: %w", err)
	}
	var cred integration.Credential
	if err := json.Unmarshal(plain, &cred); err != nil {
		return integration.Credential{}, false, err
	}
	return cred, true, nil
}

// List returns metadata only.
func (v *Vault) List(ctx context.Context) ([]store.CredentialMeta, error) {
	return v.store.ListCredentialMeta(ctx)
}

// Delete removes a credential.
func (v *Vault) Delete(ctx context.Context, provider, subject string) (bool, error) {
	return v.store.DeleteCredential(ctx, provider, subjectOf(subject))
}

func decode(value string) ([]byte, error) {
	if strings.HasPrefix(value, "v1:") {
		return base64.StdEncoding.DecodeString(strings.TrimPrefix(value, "v1:"))
	}
	return base64.StdEncoding.DecodeString(value)
}
