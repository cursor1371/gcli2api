package server

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"gcli2api/internal/config"
	"gcli2api/internal/gemini"
)

type fakeCA struct {
	stream []gemini.GeminiAPIResponse
}

func (f *fakeCA) GenerateContent(ctx context.Context, model, project string, req gemini.GeminiRequest) (*gemini.GeminiAPIResponse, error) {
	if len(f.stream) > 0 {
		return &f.stream[0], nil
	}
	return &gemini.GeminiAPIResponse{}, nil
}

func (f *fakeCA) GenerateContentStream(ctx context.Context, model, project string, req gemini.GeminiRequest) (<-chan gemini.GeminiAPIResponse, <-chan error) {
	out := make(chan gemini.GeminiAPIResponse, len(f.stream))
	errs := make(chan error, 1)
	go func() {
		defer close(out)
		defer close(errs)
		for _, g := range f.stream {
			out <- g
			time.Sleep(5 * time.Millisecond)
		}
	}()
	return out, errs
}

func TestListModels_shape(t *testing.T) {
	v := listModels()
	b, _ := json.Marshal(v)
	if !bytes.Contains(b, []byte("models/gemini-2.5-flash")) {
		t.Fatalf("missing flash model: %s", string(b))
	}
}

type flushRecorder struct {
	*httptest.ResponseRecorder
	flushed int
}

func (f *flushRecorder) Flush() { f.flushed++ }

func TestHandler_FlushBehavior(t *testing.T) {
	s := NewWithCAClient(config.Config{}, &fakeCA{stream: []gemini.GeminiAPIResponse{
		{Candidates: []gemini.Candidate{{Content: struct {
			Parts []gemini.GeminiPart `json:"parts"`
		}{Parts: []gemini.GeminiPart{{Text: "a"}}}}}},
		{Candidates: []gemini.Candidate{{Content: struct {
			Parts []gemini.GeminiPart `json:"parts"`
		}{Parts: []gemini.GeminiPart{{Text: "b"}}}}}},
	}})
	rr := &flushRecorder{ResponseRecorder: httptest.NewRecorder()}
	req := httptest.NewRequest(http.MethodPost, "/v1beta/models/gemini-2.5-flash:streamGenerateContent", bytes.NewBufferString(`{"contents":[{"role":"user","parts":[{"text":"hi"}]}]}`))
	s.handleModel(rr, req)

	body := rr.Body.Bytes()
	if !bytes.Contains(body, []byte("data: ")) || rr.flushed == 0 {
		t.Fatalf("expected SSE writes and flushes, flushed=%d body=%s", rr.flushed, string(body))
	}
}
