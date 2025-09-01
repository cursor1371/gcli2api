package gemini

import (
	"encoding/json"
	"fmt"
	// "gcli2api/internal/utils"
	// "github.com/sirupsen/logrus"
)

type InlineData struct {
	MimeType string `json:"mimeType"`
	Data     string `json:"data"`
}

type FileData struct {
	MimeType string `json:"mimeType"`
	FileURI  string `json:"fileUri"`
}

type FunctionCall struct {
	Name string      `json:"name"`
	Args interface{} `json:"args"`
}

type FunctionResponse struct {
	Name     string      `json:"name"`
	Response interface{} `json:"response"`
}

type GeminiPart struct {
	Text         string            `json:"text,omitempty"`
	Thought      bool              `json:"thought,omitempty"`
	InlineData   *InlineData       `json:"inlineData,omitempty"`
	FileData     *FileData         `json:"fileData,omitempty"`
	FunctionCall *FunctionCall     `json:"functionCall,omitempty"`
	FunctionResp *FunctionResponse `json:"functionResponse,omitempty"`
}

type GeminiContent struct {
	Role  string       `json:"role"`
	Parts []GeminiPart `json:"parts"`
}

type FunctionDeclaration struct {
	Name        string      `json:"name"`
	Description string      `json:"description,omitempty"`
	Parameters  interface{} `json:"parameters,omitempty"`
}

type GenerationConfig struct {
	Temperature     float64  `json:"temperature,omitempty"`
	MaxOutputTokens int      `json:"maxOutputTokens,omitempty"`
	TopP            float64  `json:"topP,omitempty"`
	StopSequences   []string `json:"stopSequences,omitempty"`
	// ThinkingConfig carries optional reasoning/thinking settings passed through to upstream APIs.
	ThinkingConfig interface{} `json:"thinkingConfig,omitempty"`
}

type GeminiRequest struct {
	SystemInstruction *GeminiContent    `json:"systemInstruction,omitempty"`
	Contents          []GeminiContent   `json:"contents"`
	GenerationConfig  *GenerationConfig `json:"generationConfig,omitempty"`
	// UnknownFields captures any additional fields not explicitly defined
	UnknownFields map[string]interface{} `json:"-"`
}

type UsageMetadata struct {
	PromptTokenCount     int `json:"promptTokenCount,omitempty"`
	CandidatesTokenCount int `json:"candidatesTokenCount,omitempty"`
	TotalTokenCount      int `json:"totalTokenCount,omitempty"`
}

type Candidate struct {
	Content struct {
		Parts []GeminiPart `json:"parts"`
	} `json:"content"`
}

type GeminiAPIResponse struct {
	Candidates             []Candidate    `json:"candidates"`
	UsageMetadata          *UsageMetadata `json:"usageMetadata,omitempty"`
	PromptFeedback         interface{}    `json:"promptFeedback,omitempty"`
	AutomaticFunctionCalls interface{}    `json:"automaticFunctionCallingHistory,omitempty"`
}

// UnmarshalJSON implements custom JSON unmarshaling for GeminiRequest
// to capture unknown fields while preserving known ones
func (gr *GeminiRequest) UnmarshalJSON(data []byte) error {
	// First unmarshal into a map to capture all fields
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	// logrus.Infof("raw request: %s", utils.TruncateLongStringInObject(raw, 100))

	// Create a temporary struct with the same fields but without custom unmarshaling
	type TempGeminiRequest struct {
		SystemInstruction *GeminiContent    `json:"systemInstruction,omitempty"`
		Contents          []GeminiContent   `json:"contents"`
		GenerationConfig  *GenerationConfig `json:"generationConfig,omitempty"`
	}

	// Unmarshal known fields
	var temp TempGeminiRequest
	if err := json.Unmarshal(data, &temp); err != nil {
		return err
	}

	// Copy known fields to the actual struct
	gr.SystemInstruction = temp.SystemInstruction
	gr.Contents = temp.Contents
	gr.GenerationConfig = temp.GenerationConfig

	// Initialize UnknownFields map
	gr.UnknownFields = make(map[string]interface{})

	// Define known field names
	knownFields := map[string]bool{
		"systemInstruction": true,
		"contents":          true,
		"generationConfig":  true,
	}

	// Store unknown fields
	for key, rawValue := range raw {
		if !knownFields[key] {
			var value interface{}
			if err := json.Unmarshal(rawValue, &value); err != nil {
				return fmt.Errorf("failed to unmarshal unknown field %s: %v", key, err)
			}
			gr.UnknownFields[key] = value
		}
	}

	return nil
}

// MarshalJSON implements custom JSON marshaling for GeminiRequest
// to include unknown fields in the output
func (gr *GeminiRequest) MarshalJSON() ([]byte, error) {
	// Create a temporary struct with the same fields but without custom marshaling
	type TempGeminiRequest struct {
		SystemInstruction *GeminiContent    `json:"systemInstruction,omitempty"`
		Contents          []GeminiContent   `json:"contents"`
		GenerationConfig  *GenerationConfig `json:"generationConfig,omitempty"`
	}

	// Marshal known fields
	temp := TempGeminiRequest{
		SystemInstruction: gr.SystemInstruction,
		Contents:          gr.Contents,
		GenerationConfig:  gr.GenerationConfig,
	}

	// Marshal to map for manipulation
	var result map[string]interface{}
	tempData, err := json.Marshal(temp)
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(tempData, &result); err != nil {
		return nil, err
	}

	// Add unknown fields
	for key, value := range gr.UnknownFields {
		result[key] = value
	}

	// Marshal final result
	return json.Marshal(result)
}
