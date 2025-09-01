package gemini

import (
	"errors"
	"io"
	"strings"
)

// NormalizeGeminiRequest ensures roles are present.
func NormalizeGeminiRequest(req GeminiRequest) GeminiRequest {
	for i := range req.Contents {
		if strings.TrimSpace(req.Contents[i].Role) == "" {
			req.Contents[i].Role = "user"
		}
	}
	return req
}

// System prompt injection removed.

// DecodeJSON is a small helper for consistent decoding with size guard.
func DecodeJSON(r io.Reader, dst interface{}) error {
	// For simplicity we rely on json decoder in handlers directly.
	return errors.New("not implemented")
}
