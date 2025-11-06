package gemini

// ModelInfo describes a supported model for discovery and validation.
type ModelInfo struct {
	Name        string
	DisplayName string
	Description string
}

// SupportedModels is the canonical list of supported model identifiers.
var SupportedModels = []ModelInfo{
	{Name: "gemini-2.5-flash", DisplayName: "Gemini 2.5 Flash", Description: "Fast multimodal generation"},
	{Name: "gemini-2.5-pro", DisplayName: "Gemini 2.5 Pro", Description: "Accurate multimodal generation"},
	{Name: "gemini-2.5-pro-preview-06-05", DisplayName: "Gemini 2.5 Pro Preview (06-05)", Description: "Accurate multimodal generation"},
	{Name: "gemini-2.5-pro-preview-05-06", DisplayName: "Gemini 2.5 Pro Preview (05-06)", Description: "Accurate multimodal generation"},
	{Name: "gemini-3-pro-preview-11-2025", DisplayName: "Gemini 3.0 Pro Preview (06-11)", Description: "NEW TEST"},
}

// IsSupportedModel reports whether the given model name is supported.
func IsSupportedModel(name string) bool {
	for _, m := range SupportedModels {
		if m.Name == name {
			return true
		}
	}
	return false
}
