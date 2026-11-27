package updatesig

import "testing"

func TestSignVerify(t *testing.T) {
	pub, priv, err := GenerateKey()
	if err != nil {
		t.Fatal(err)
	}
	otherPub, _, _ := GenerateKey()
	m := Manifest{Version: "0.2.0", URL: "https://example.com/a.zip", SHA256: "AB12", Notes: "line one\nline two"}
	if err := Sign(&m, priv); err != nil {
		t.Fatal(err)
	}
	if err := Verify(m, pub); err != nil {
		t.Fatalf("valid manifest rejected: %v", err)
	}

	tests := []struct {
		name   string
		mutate func(*Manifest)
		key    string
	}{
		{"other key", func(*Manifest) {}, otherPub},
		{"version", func(m *Manifest) { m.Version = "9.9.9" }, pub},
		{"url", func(m *Manifest) { m.URL = "https://evil.example/a.zip" }, pub},
		{"sha256", func(m *Manifest) { m.SHA256 = "cd34" }, pub},
		{"notes", func(m *Manifest) { m.Notes = "changed" }, pub},
		{"unsigned", func(m *Manifest) { m.Signature = "" }, pub},
		{"newline in url", func(m *Manifest) { m.URL = "https://example.com/a.zip\nAB12" }, pub},
	}
	for _, tt := range tests {
		changed := m
		tt.mutate(&changed)
		if err := Verify(changed, tt.key); err == nil {
			t.Errorf("%s: tampered manifest accepted", tt.name)
		}
	}
}
