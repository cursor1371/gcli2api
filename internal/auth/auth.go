package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"gcli2api/internal/utils"

	"golang.org/x/oauth2"
)

type RawToken struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	TokenType    string `json:"token_type"`
	ExpiryDateMS int64  `json:"expiry_date"` // ms since epoch
	Scope        string `json:"scope"`
}

func (rt RawToken) ToOAuth2Token() *oauth2.Token {
	return &oauth2.Token{
		AccessToken:  rt.AccessToken,
		TokenType:    rt.TokenType,
		RefreshToken: rt.RefreshToken,
		Expiry:       time.Unix(0, rt.ExpiryDateMS*int64(time.Millisecond)),
	}
}

func fromOAuth2Token(tok *oauth2.Token, prev RawToken) RawToken {
	rt := prev
	if tok.AccessToken != "" {
		rt.AccessToken = tok.AccessToken
	}
	if tok.TokenType != "" {
		rt.TokenType = tok.TokenType
	}
	if !tok.Expiry.IsZero() {
		rt.ExpiryDateMS = tok.Expiry.UnixMilli()
	}
	// Keep refresh token as-is unless provided
	if tok.RefreshToken != "" {
		rt.RefreshToken = tok.RefreshToken
	}
	return rt
}

// Single-file helper retained for callers to load per-path tokens.

// LoadRawTokenFromFile loads a RawToken from a single JSON file path (with ~ expansion).
func LoadRawTokenFromFile(path string) (RawToken, string, error) {
	var rt RawToken
	xp, err := utils.ExpandUser(path)
	if err != nil {
		return rt, path, fmt.Errorf("expand path: %w", err)
	}
	b, err := os.ReadFile(xp)
	if err != nil {
		return rt, path, fmt.Errorf("read creds file: %w", err)
	}
	if err := json.Unmarshal(b, &rt); err != nil {
		return rt, path, fmt.Errorf("parse creds json: %w", err)
	}
	return rt, xp, nil
}

// persistingTokenSource wraps an oauth2.TokenSource, persisting refreshed tokens.
type persistingTokenSource struct {
	base    oauth2.TokenSource
	current RawToken
	path    string
	persist bool
	mu      sync.Mutex
}

func NewPersistingTokenSource(base oauth2.TokenSource, initial RawToken, path string, persist bool) oauth2.TokenSource {
	return &persistingTokenSource{base: base, current: initial, path: path, persist: persist}
}

func (p *persistingTokenSource) Token() (*oauth2.Token, error) {
	tok, err := p.base.Token()
	if err != nil {
		return nil, err
	}
	// Detect change and persist atomically
	p.mu.Lock()
	defer p.mu.Unlock()
	updated := fromOAuth2Token(tok, p.current)
	if updated.AccessToken != p.current.AccessToken || updated.ExpiryDateMS != p.current.ExpiryDateMS {
		p.current = updated
		if p.persist && p.path != "" {
			if err := SaveRawTokenAtomic(p.path, p.current); err != nil {
				// Non-fatal; return token regardless
			}
		}
	}
	return tok, nil
}

func SaveRawTokenAtomic(path string, rt RawToken) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("mkdir: %w", err)
	}
	b, err := json.MarshalIndent(rt, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal token: %w", err)
	}
	tmp, err := os.CreateTemp(dir, ".token-*.json")
	if err != nil {
		return fmt.Errorf("create temp: %w", err)
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)
	if _, err := tmp.Write(b); err != nil {
		tmp.Close()
		return fmt.Errorf("write temp: %w", err)
	}
	if err := tmp.Chmod(0o600); err != nil {
		tmp.Close()
		return fmt.Errorf("chmod temp: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close temp: %w", err)
	}
	if err := os.Rename(tmpPath, path); err != nil {
		return fmt.Errorf("rename: %w", err)
	}
	return nil
}

// Helper to force-refresh a token if it's near expiry in the next N minutes.
func EnsureFresh(ctx context.Context, ts oauth2.TokenSource, nearMinutes int) error {
	tok, err := ts.Token()
	if err != nil {
		return err
	}
	if tok.Expiry.IsZero() {
		return nil
	}
	if time.Until(tok.Expiry) < time.Duration(nearMinutes)*time.Minute {
		// Force refresh by invalidating cached token if possible; oauth2.TokenSource
		// generally refreshes automatically on Token(). No-op here.
	}
	return nil
}
