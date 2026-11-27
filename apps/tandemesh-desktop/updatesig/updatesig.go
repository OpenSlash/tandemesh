// Package updatesig signs and verifies desktop update manifests with
// Ed25519. The release pipeline holds the private key; the app embeds the
// public key, so a mirror or a hijacked download URL cannot push a build
// that the release pipeline did not sign. It needs no Apple or Microsoft
// code-signing certificate.
package updatesig

import (
	"crypto/ed25519"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
)

// Manifest is the JSON document published next to each release package.
type Manifest struct {
	Version   string `json:"version"`
	URL       string `json:"url"`
	SHA256    string `json:"sha256"`
	Notes     string `json:"notes,omitempty"`
	Signature string `json:"signature"`
}

// message is what gets signed. Notes come last because they may contain
// newlines; the other fields may not, so the layout is unambiguous.
func (m Manifest) message() ([]byte, error) {
	for _, field := range []string{m.Version, m.URL, m.SHA256} {
		if field == "" || strings.ContainsAny(field, "\r\n") {
			return nil, errors.New("manifest version, url and sha256 must be single non-empty lines")
		}
	}
	return []byte("tandemesh-update-v1\n" + m.Version + "\n" + m.URL + "\n" + strings.ToLower(m.SHA256) + "\n" + m.Notes), nil
}

// Sign fills in m.Signature with a base64 Ed25519 private key (seed or full key).
func Sign(m *Manifest, privateKey string) error {
	raw, err := base64.StdEncoding.DecodeString(strings.TrimSpace(privateKey))
	if err != nil {
		return fmt.Errorf("decode private key: %w", err)
	}
	var key ed25519.PrivateKey
	switch len(raw) {
	case ed25519.SeedSize:
		key = ed25519.NewKeyFromSeed(raw)
	case ed25519.PrivateKeySize:
		key = ed25519.PrivateKey(raw)
	default:
		return fmt.Errorf("private key is %d bytes, want %d or %d", len(raw), ed25519.SeedSize, ed25519.PrivateKeySize)
	}
	msg, err := m.message()
	if err != nil {
		return err
	}
	m.Signature = base64.StdEncoding.EncodeToString(ed25519.Sign(key, msg))
	return nil
}

// Verify checks m.Signature against a base64 Ed25519 public key.
func Verify(m Manifest, publicKey string) error {
	pub, err := base64.StdEncoding.DecodeString(strings.TrimSpace(publicKey))
	if err != nil || len(pub) != ed25519.PublicKeySize {
		return errors.New("invalid update public key")
	}
	sig, err := base64.StdEncoding.DecodeString(m.Signature)
	if err != nil || len(sig) != ed25519.SignatureSize {
		return errors.New("update manifest is not signed")
	}
	msg, err := m.message()
	if err != nil {
		return err
	}
	if !ed25519.Verify(ed25519.PublicKey(pub), msg, sig) {
		return errors.New("update manifest signature does not match")
	}
	return nil
}

// GenerateKey returns a new base64 public key and private key seed.
func GenerateKey() (publicKey, privateKey string, err error) {
	pub, priv, err := ed25519.GenerateKey(nil)
	if err != nil {
		return "", "", err
	}
	return base64.StdEncoding.EncodeToString(pub), base64.StdEncoding.EncodeToString(priv.Seed()), nil
}
