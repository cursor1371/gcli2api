package utils

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

// TruncateLongStringInObject recursively truncates long strings in a JSON-like object.
// It mimics the Python function truncate_long_strings_in_object.
func truncateLongStringInImpl(obj interface{}, n int) interface{} {
	switch v := obj.(type) {
	case map[string]interface{}:
		result := make(map[string]interface{})
		for k, val := range v {
			result[k] = truncateLongStringInImpl(val, n)
		}
		return result
	case []interface{}:
		result := make([]interface{}, len(v))
		for i, val := range v {
			result[i] = truncateLongStringInImpl(val, n)
		}
		return result
	case string:
		if len(v) > n {
			return v[:n]
		}
		return v
	default:
		return v
	}
}

func TruncateLongStringInObject(v any, n int) string {
	vBytes, _ := json.Marshal(v)
	var vInterface interface{}
	json.Unmarshal(vBytes, &vInterface)

	// Apply truncation to the unmarshaled object
	truncated := truncateLongStringInImpl(vInterface, n)

	// Marshal with indentation for pretty JSON
	vString, _ := json.MarshalIndent(truncated, "", "  ")
	return string(vString)
}

func ExpandUser(path string) (string, error) {
	if strings.HasPrefix(path, "~/") || path == "~" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		if path == "~" {
			return home, nil
		}
		return filepath.Join(home, path[2:]), nil
	}
	return path, nil
}
