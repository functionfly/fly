//go:build integration

package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/functionfly/fly/internal/manifest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInitCommandCreatesJSONCFile(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "functionfly-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Change to temp directory
	oldWd, _ := os.Getwd()
	defer os.Chdir(oldWd)
	os.Chdir(tempDir)

	// Run init command
	rootCmd.SetArgs([]string{"init", "test-function"})
	err = rootCmd.Execute()
	require.NoError(t, err)

	// Check that functionfly.jsonc was created (not functionfly.json)
	jsoncPath := filepath.Join(tempDir, "functionfly.jsonc")
	jsonPath := filepath.Join(tempDir, "functionfly.json")

	assert.FileExists(t, jsoncPath, "functionfly.jsonc should be created")
	assert.NoFileExists(t, jsonPath, "functionfly.json should not be created")

	// Verify the content contains the function name
	content, err := os.ReadFile(jsoncPath)
	require.NoError(t, err)
	contentStr := string(content)
	assert.Contains(t, contentStr, `"name": "test-function"`)
	assert.Contains(t, contentStr, `"$schema"`)
}

func TestManifestAutoDetection(t *testing.T) {
	tests := []struct {
		name        string
		createFile  string
		content     string
		expectError bool
	}{
		{
			name:       "prefers jsonc over json",
			createFile: "functionfly.jsonc",
			content: `{
				"name": "test-jsonc",
				"version": "1.0.0",
				"runtime": "node18"
			}`,
			expectError: false,
		},
		{
			name:       "falls back to json when jsonc missing",
			createFile: "functionfly.json",
			content: `{
				"name": "test-json",
				"version": "1.0.0",
				"runtime": "node18"
			}`,
			expectError: false,
		},
		{
			name:        "fails when neither file exists",
			createFile:  "",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a temporary directory for testing
			tempDir, err := os.MkdirTemp("", "functionfly-manifest-test-*")
			require.NoError(t, err)
			defer os.RemoveAll(tempDir)

			// Change to temp directory
			oldWd, _ := os.Getwd()
			defer os.Chdir(oldWd)
			os.Chdir(tempDir)

			// Create the manifest file if specified
			if tt.createFile != "" {
				err := os.WriteFile(tt.createFile, []byte(tt.content), 0644)
				require.NoError(t, err)
			}

			// Test manifest loading with auto-detection
			manifest, err := manifest.Load("")
			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, manifest)
			} else {
				assert.NoError(t, err)
				require.NotNil(t, manifest)
				assert.Equal(t, "test-"+strings.TrimPrefix(tt.createFile, "functionfly."), manifest.Name)
			}
		})
	}
}

func TestJSONCCommentsSupport(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "functionfly-jsonc-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Change to temp directory
	oldWd, _ := os.Getwd()
	defer os.Chdir(oldWd)
	os.Chdir(tempDir)

	// Create a JSONC file with comments
	jsoncContent := `{
		// Function metadata
		"name": "jsonc-test",
		"version": "1.0.0",
		"runtime": "node18",
		/* Performance settings */
		"cache_ttl": 3600,
		"timeout_ms": 5000
	}`

	err = os.WriteFile("functionfly.jsonc", []byte(jsoncContent), 0644)
	require.NoError(t, err)

	// Test loading JSONC with comments
	manifest, err := manifest.Load("")
	assert.NoError(t, err)
	require.NotNil(t, manifest)
	assert.Equal(t, "jsonc-test", manifest.Name)
	assert.Equal(t, "1.0.0", manifest.Version)
	assert.Equal(t, "node18", manifest.Runtime)
	assert.Equal(t, 3600, *manifest.CacheTTL)
	assert.Equal(t, 5000, *manifest.TimeoutMS)
}