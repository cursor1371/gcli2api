package auth

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"golang.org/x/oauth2"
)

func TestTokenLoadAndRefreshShape(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "creds.json")
	rt := RawToken{AccessToken: "a1", RefreshToken: "r", TokenType: "Bearer", ExpiryDateMS: time.Now().Add(1 * time.Hour).UnixMilli()}
	b, _ := json.Marshal(rt)
	if err := os.WriteFile(p, b, 0o600); err != nil {
		t.Fatal(err)
	}
	got, xp, err := LoadRawTokenFromFile(p)
	if err != nil || xp == "" {
		t.Fatalf("load failed: %v %v", xp, err)
	}
	tok := got.ToOAuth2Token()
	if tok.AccessToken != "a1" || tok.TokenType != "Bearer" || tok.RefreshToken != "r" {
		t.Fatalf("bad mapping: %+v", tok)
	}

	// Refresh and persistence
	fake := &fakeTS{toks: []*oauth2.Token{{AccessToken: "a1", Expiry: time.Now().Add(-1 * time.Minute), RefreshToken: "r"}, {AccessToken: "a2", Expiry: time.Now().Add(1 * time.Hour), RefreshToken: "r"}}}
	pts := NewPersistingTokenSource(fake, got, p, true)
	if _, err := pts.Token(); err != nil { // first expired, returns expired, but next call refreshes
		t.Fatal(err)
	}
	if _, err := pts.Token(); err != nil {
		t.Fatal(err)
	}
	// Verify file updated
	nb, _ := os.ReadFile(p)
	var nrt RawToken
	_ = json.Unmarshal(nb, &nrt)
	if nrt.AccessToken != "a2" {
		t.Fatalf("expected persisted access token a2, got %q", nrt.AccessToken)
	}
}

type fakeTS struct{ toks []*oauth2.Token }

func (f *fakeTS) Token() (*oauth2.Token, error) {
	if len(f.toks) == 0 {
		return &oauth2.Token{AccessToken: "final", Expiry: time.Now().Add(1 * time.Hour)}, nil
	}
	t := f.toks[0]
	f.toks = f.toks[1:]
	return t, nil
}
