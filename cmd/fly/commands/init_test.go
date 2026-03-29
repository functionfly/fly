package commands

import (
	"strings"
	"testing"
)

// Test function name validation via runInit (isValidFunctionName is unexported).
func TestInit_ValidFunctionName(t *testing.T) {
	// Valid: lowercase, digits, hyphens; 1-63 chars; no leading/trailing hyphen
	valid := []string{"a", "hello", "my-function", "v2", "a1b2", strings.Repeat("a", 63)}
	for _, name := range valid {
		if !isValidFunctionName(name) {
			t.Errorf("isValidFunctionName(%q) = false, want true", name)
		}
	}
}

func TestInit_InvalidFunctionName(t *testing.T) {
	invalid := []string{
		"",
		strings.Repeat("a", 64),
		"Uppercase",
		"under_score",
		"-leading",
		"trailing-",
		"space in name",
		"dot.name",
	}
	for _, name := range invalid {
		if isValidFunctionName(name) {
			t.Errorf("isValidFunctionName(%q) = true, want false", name)
		}
	}
}
