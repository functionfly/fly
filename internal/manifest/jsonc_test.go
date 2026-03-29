package manifest

import (
	"strings"
	"testing"
)

func TestStripComments(t *testing.T) {
	tests := []struct {
		name  string
		input string
		// Expected output after trimming whitespace
		expected string
	}{
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "simple JSON",
			input:    `{"name": "test"}`,
			expected: `{"name": "test"}`,
		},
		{
			name: "single line comment",
			input: `// comment
{"name": "test"}`,
			expected: `{"name": "test"}`,
		},
		{
			name:     "multi line comment",
			input:    `/* comment */{"name": "test"}`,
			expected: `{"name": "test"}`,
		},
		{
			name: "multiple comments",
			input: `// name field
"name": "test",
// version
"version": "1.0.0"`,
			expected: `"name": "test", "version": "1.0.0"`,
		},
		{
			name:     "comment inside string should be preserved",
			input:    `{"name": "test // not a comment"}`,
			expected: `{"name": "test // not a comment"}`,
		},
		{
			name:     "url in string should be preserved",
			input:    `{"url": "https://example.com/path"}`,
			expected: `{"url": "https://example.com/path"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := StripComments(tt.input)
			// Trim whitespace for comparison since we add spaces
			result = strings.TrimSpace(result)
			if result != tt.expected {
				t.Errorf("StripComments() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestIsValidJSON(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{
			name:     "valid JSON object",
			input:    `{"name": "test"}`,
			expected: true,
		},
		{
			name:     "valid JSON array",
			input:    `[1, 2, 3]`,
			expected: true,
		},
		{
			name: "JSONC with comment",
			input: `// comment
{"name": "test"}`,
			expected: true,
		},
		{
			name:     "JSONC with multi-line comment",
			input:    `/* comment */{"name": "test"}`,
			expected: true,
		},
		{
			name:     "empty string",
			input:    "",
			expected: false,
		},
		{
			name:     "whitespace only",
			input:    "   ",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsValidJSON(tt.input)
			if result != tt.expected {
				t.Errorf("IsValidJSON() = %v, want %v", result, tt.expected)
			}
		})
	}
}
