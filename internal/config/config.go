package config

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"

	"github.com/sirupsen/logrus"
)

type Config struct {
	Host                   string   `json:"host"`
	ServerPort             int      `json:"port"`
	AuthKey                string   `json:"authKey"`
	GeminiCredsFilePaths   []string `json:"geminiOauthCredsFiles"`
	RequestMaxRetries      int      `json:"requestMaxRetries"`
	RequestBaseDelayMillis int      `json:"requestBaseDelay"`
	SQLitePath             string   `json:"sqlitePath"`
	// Proxy is an optional upstream proxy URL. Must be http or socks5.
	// Example: "http://127.0.0.1:8080" or "socks5://127.0.0.1:1080"
	Proxy string `json:"proxy"`
	// RequestMaxBodyBytes limits incoming request size to mitigate DoS via large payloads.
	// If zero, a safe default is applied.
	RequestMaxBodyBytes int64 `json:"requestMaxBodyBytes"`
	// MaxConcurrentRequests limits concurrent in-flight requests for lightweight backpressure.
	// If zero, a default value is applied.
	MaxConcurrentRequests int `json:"maxConcurrentRequests"`
}

func LoadConfig(path string) (Config, error) {
	var cfg Config
	b, err := os.ReadFile(path)
	logrus.Infof("loading config from %s", path)
	if err != nil {
		return cfg, fmt.Errorf("read config: %w", err)
	}
	dec := json.NewDecoder(bytes.NewReader(b))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&cfg); err != nil {
		// Try to extract unknown field name from the error and surface just the key
		// Typical error: "json: unknown field \"foo\""
		var se *json.SyntaxError
		if !errors.As(err, &se) {
			msg := err.Error()
			const p = "json: unknown field \""
			if i := bytes.Index([]byte(msg), []byte(p)); i >= 0 {
				// Extract between quotes
				start := i + len(p)
				rest := msg[start:]
				if j := bytes.IndexByte([]byte(rest), '"'); j >= 0 {
					unknown := rest[:j]
					return cfg, fmt.Errorf("unknown config key: %s", unknown)
				}
			}
		}
		return cfg, fmt.Errorf("parse config: %w", err)
	}
	// Defaults
	if cfg.Host == "" {
		cfg.Host = "127.0.0.1"
	}
	if cfg.ServerPort == 0 {
		cfg.ServerPort = 8085
	}
	if cfg.RequestMaxRetries == 0 {
		cfg.RequestMaxRetries = 3
	}
	if cfg.RequestBaseDelayMillis == 0 {
		cfg.RequestBaseDelayMillis = 1000
	}
	if cfg.SQLitePath == "" {
		cfg.SQLitePath = "./data/state.db"
	}
	if cfg.RequestMaxBodyBytes == 0 {
		// 16 MiB by default
		cfg.RequestMaxBodyBytes = 16 * 1024 * 1024
	}
	if cfg.MaxConcurrentRequests == 0 {
		cfg.MaxConcurrentRequests = 64
	}
	return cfg, nil
}

func (c Config) Validate(cfgPath string) error {
	if c.AuthKey == "" {
		return fmt.Errorf("authKey must be set in config file %s", cfgPath)
	}
	// Fail when authKey equals the default placeholder from example file.
	if c.AuthKey == "UNSAFE-KEY-REPLACE" {
		return fmt.Errorf("authKey must be changed from default placeholder")
	}
	// Validate proxy scheme if provided
	if c.Proxy != "" {
		u, err := url.Parse(c.Proxy)
		if err != nil {
			return fmt.Errorf("invalid proxy URL: %w", err)
		}
		switch u.Scheme {
		case "http", "socks5":
			// ok
		default:
			return fmt.Errorf("proxy scheme must be http or socks5")
		}
		if u.Host == "" {
			return fmt.Errorf("proxy URL must include host:port")
		}
	}
	return nil
}
