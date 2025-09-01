package httpx

import (
	"context"
	"math/rand"
	"net"
	"net/http"
	"net/url"
	"time"

	socks5proxy "golang.org/x/net/proxy"
	"golang.org/x/oauth2"
)

// NewOAuthHTTPClient creates an *http.Client with OAuth2 transport.
// If proxyURL is non-nil, it is used as the upstream proxy. Supported schemes: http, socks5.
func NewOAuthHTTPClient(ts oauth2.TokenSource, proxyURL *url.URL) *http.Client {
	dialer := &net.Dialer{Timeout: 10 * time.Second, KeepAlive: 30 * time.Second}
	tr := &http.Transport{
		Proxy:                 http.ProxyFromEnvironment,
		DialContext:           dialer.DialContext,
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          100,
		MaxIdleConnsPerHost:   10,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}

	if proxyURL != nil {
		switch proxyURL.Scheme {
		case "http":
			tr.Proxy = http.ProxyURL(proxyURL)
		case "socks5":
			// For SOCKS5, configure a custom dialer
			d, err := socks5proxy.FromURL(proxyURL, dialer)
			if err == nil && d != nil {
				// Use Dial for compatibility; http.Transport prefers DialContext when set.
				tr.DialContext = nil
				tr.Dial = d.Dial
				tr.Proxy = nil
			}
		}
	}

	return &http.Client{
		Transport: &oauth2.Transport{Source: ts, Base: tr},
		Timeout:   0, // rely on per-request contexts
	}
}

// WithRetries runs fn with exponential backoff w/ jitter.
func WithRetries(ctx context.Context, max int, baseDelay time.Duration, fn func(attempt int) error) error {
	var err error
	for attempt := 0; attempt <= max; attempt++ {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		err = fn(attempt)
		if err == nil {
			return nil
		}
		if attempt == max {
			break
		}
		// jitter: 1.0 to 1.2 multiplier
		jitter := 1.0 + rand.Float64()*0.2
		factor := 1 << uint(attempt)
		delay := time.Duration(float64(baseDelay) * jitter * float64(factor))
		t := time.NewTimer(delay)
		select {
		case <-ctx.Done():
			t.Stop()
			return ctx.Err()
		case <-t.C:
		}
	}
	return err
}
