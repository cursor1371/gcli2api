package config

import "testing"

func TestConfig_ProjectIds_UnknownKey_Fails(t *testing.T) {
	cfg := Config{
		AuthKey:              "k",
		GeminiCredsFilePaths: []string{"a.json"},
		ProjectIds: map[string][]string{
			"b.json": {"p1"},
		},
	}
	if err := cfg.Validate("test.json"); err == nil {
		t.Fatalf("expected validation to fail for unknown projectIds key")
	}
}
