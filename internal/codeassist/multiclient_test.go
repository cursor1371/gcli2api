package codeassist

import (
	"context"
	"net/http"
	"sync/atomic"
	"testing"
	"time"

	"gcli2api/internal/auth"
	"gcli2api/internal/gemini"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

// Focused test: rotation behavior on 401 vs 500.
func TestMultiClient_RotationPolicy_Unary(t *testing.T) {
	oauthCfg := oauth2.Config{ClientID: "test", ClientSecret: "s", Scopes: []string{"s"}, Endpoint: google.Endpoint}
	sources := []CredSource{
		{Path: "a.json", Raw: auth.RawToken{AccessToken: "xa", RefreshToken: "ra"}, Persist: false},
		{Path: "b.json", Raw: auth.RawToken{AccessToken: "xb", RefreshToken: "rb"}, Persist: false},
	}
	mc, err := NewMultiClient(oauthCfg, sources, 0, 1*time.Millisecond, nil, nil)
	if err != nil {
		t.Fatalf("init multiclient: %v", err)
	}

	// Subtest: rotates on 401 to next credential and succeeds
	t.Run("rotate on 401", func(t *testing.T) {
		// entry[0] returns 401; entry[1] returns 200
		attempts := []int{0, 0}
		mc.entries[0].ca = NewClient(mkClient(rtFunc(func(r *http.Request) (*http.Response, error) {
			attempts[0]++
			return resp(401, "unauthorized", "text/plain"), nil
		})), 0, 1*time.Millisecond)
		mc.entries[1].ca = NewClient(mkClient(rtFunc(func(r *http.Request) (*http.Response, error) {
			attempts[1]++
			return resp(200, `{"response": {"candidates":[{"content":{"parts":[{"text":"ok"}]}}]}}`, "application/json"), nil
		})), 0, 1*time.Millisecond)

		g, err := mc.GenerateContent(context.Background(), "gemini-2.5-flash", "proj", gemini.GeminiRequest{Contents: []gemini.GeminiContent{{Role: "user", Parts: []gemini.GeminiPart{{Text: "hi"}}}}})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(g.Candidates) == 0 || len(g.Candidates[0].Content.Parts) == 0 || g.Candidates[0].Content.Parts[0].Text != "ok" {
			t.Fatalf("bad response: %+v", g)
		}
		if attempts[0] != 1 || attempts[1] != 1 {
			t.Fatalf("expected attempts [1,1], got %v", attempts)
		}
	})

	// Subtest: does not rotate on 500; returns immediately
	t.Run("no rotate on 500", func(t *testing.T) {
		// Reset round-robin so we start from idx=0
		atomic.StoreUint64(&mc.rr, 0)
		attempts := []int{0, 0}
		mc.entries[0].ca = NewClient(mkClient(rtFunc(func(r *http.Request) (*http.Response, error) {
			attempts[0]++
			return resp(500, "boom", "text/plain"), nil
		})), 0, 1*time.Millisecond)
		mc.entries[1].ca = NewClient(mkClient(rtFunc(func(r *http.Request) (*http.Response, error) {
			attempts[1]++
			// Would succeed if tried, but should not be invoked
			return resp(200, `{"response": {"candidates":[{"content":{"parts":[{"text":"ok"}]}}]}}`, "application/json"), nil
		})), 0, 1*time.Millisecond)

		_, err := mc.GenerateContent(context.Background(), "gemini-2.5-flash", "proj", gemini.GeminiRequest{Contents: []gemini.GeminiContent{{Role: "user", Parts: []gemini.GeminiPart{{Text: "hi"}}}}})
		if err == nil {
			t.Fatalf("expected error on 500")
		}
		if attempts[0] != 1 || attempts[1] != 0 {
			t.Fatalf("expected attempts [1,0], got %v", attempts)
		}
	})
}
