package gemini

import (
	"encoding/json"
	"testing"
)

func TestNormalizeRoles_DefaultsOnly(t *testing.T) {
	req := GeminiRequest{
		Contents: []GeminiContent{
			{Role: "", Parts: []GeminiPart{{Text: "hi"}}},
			{Role: "user", Parts: []GeminiPart{{Text: "yo"}}},
		},
	}
	got := NormalizeGeminiRequest(req)
	if got.Contents[0].Role != "user" {
		t.Fatalf("expected default role 'user', got %q", got.Contents[0].Role)
	}
}

// System prompt from file feature removed; no tests required.

func TestGenerationConfig_passthrough(t *testing.T) {
	req := GeminiRequest{GenerationConfig: &GenerationConfig{Temperature: 0.4, MaxOutputTokens: 123, TopP: 0.9, StopSequences: []string{"STOP"}}}
	got := NormalizeGeminiRequest(req)
	if got.GenerationConfig == nil || got.GenerationConfig.MaxOutputTokens != 123 || got.GenerationConfig.TopP != 0.9 || got.GenerationConfig.Temperature != 0.4 {
		t.Fatalf("generation config altered: %+v", got.GenerationConfig)
	}
}

func TestGeminiRequest_UnknownFields(t *testing.T) {
	// Test JSON with unknown fields like safetySettings
	jsonData := `{
		"contents": [
			{
				"role": "user",
				"parts": [{"text": "Hello"}]
			}
		],
		"safetySettings": [
			{
				"category": "HARM_CATEGORY_HARASSMENT",
				"threshold": "BLOCK_MEDIUM_AND_ABOVE"
			}
		],
		"customField": "customValue",
		"generationConfig": {
			"temperature": 0.7
		}
	}`

	// Test unmarshaling
	var req GeminiRequest
	err := json.Unmarshal([]byte(jsonData), &req)
	if err != nil {
		t.Fatalf("Failed to unmarshal JSON with unknown fields: %v", err)
	}

	// Verify known fields are properly set
	if len(req.Contents) != 1 {
		t.Fatalf("Expected 1 content, got %d", len(req.Contents))
	}
	if req.Contents[0].Role != "user" {
		t.Fatalf("Expected role 'user', got %q", req.Contents[0].Role)
	}
	if req.GenerationConfig == nil || req.GenerationConfig.Temperature != 0.7 {
		t.Fatalf("GenerationConfig not properly set: %+v", req.GenerationConfig)
	}

	// Verify unknown fields are captured
	if req.UnknownFields == nil {
		t.Fatal("UnknownFields should not be nil")
	}
	if _, exists := req.UnknownFields["safetySettings"]; !exists {
		t.Fatal("safetySettings should be captured in UnknownFields")
	}
	if _, exists := req.UnknownFields["customField"]; !exists {
		t.Fatal("customField should be captured in UnknownFields")
	}

	// Test marshaling back to JSON
	marshaledData, err := json.Marshal(&req)
	if err != nil {
		t.Fatalf("Failed to marshal GeminiRequest: %v", err)
	}

	// Verify the marshaled JSON contains unknown fields
	var result map[string]interface{}
	err = json.Unmarshal(marshaledData, &result)
	if err != nil {
		t.Fatalf("Failed to unmarshal result: %v", err)
	}

	if _, exists := result["safetySettings"]; !exists {
		t.Fatal("safetySettings should be present in marshaled JSON")
	}
	if _, exists := result["customField"]; !exists {
		t.Fatal("customField should be present in marshaled JSON")
	}
	if result["customField"] != "customValue" {
		t.Fatalf("Expected customField to be 'customValue', got %v", result["customField"])
	}

	t.Logf("Successfully handled unknown fields: %s", string(marshaledData))
}
