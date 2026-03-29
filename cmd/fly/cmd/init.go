/*
Copyright © 2026 FunctionFly

*/
package cmd

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

// initCmd represents the init command
var initCmd = &cobra.Command{
	Use:   "init [name]",
	Short: "Create a runnable function instantly",
	Long: `Creates a runnable function instantly with zero configuration.

Generates:
  index.js              Function code
  functionfly.jsonc     Manifest configuration
  test.http             Local test requests

Templates:
  javascript  (default) ES modules with async/await
  typescript            TypeScript with type definitions
  python               Python async functions

Examples:
  fly init slugify
  fly init --template=typescript slugify
  fly init --template=python myfunction`,
	Run: initRun,
}

// Template represents a function template
type Template struct {
	File    string
	Content string
	Manifest string
}

// Available templates
var templates = map[string]Template{
	"javascript": {
		File: "index.js",
		Content: `export default async function (input) {
  return input
    .toLowerCase()
    .replace(/[^a-z0-9]+/g, "-")
    .replace(/(^-|-$)/g, "");
}`,
		Manifest: `{
  "$schema": "https://functionfly.com/schemas/functionfly.json",
  "name": "%s",
  "version": "1.0.0",
  "runtime": "node18",
  "public": true,
  "deterministic": true,
  "cache_ttl": 86400,
  "timeout_ms": 5000,
  "memory_mb": 128,
  "description": "Convert string to URL-friendly slug"
}`,
	},
	"typescript": {
		File: "index.ts",
		Content: `export default async function (input: string): Promise<string> {
  return input
    .toLowerCase()
    .replace(/[^a-z0-9]+/g, "-")
    .replace(/(^-|-$)/g, "");
}`,
		Manifest: `{
  "$schema": "https://functionfly.com/schemas/functionfly.json",
  "name": "%s",
  "version": "1.0.0",
  "runtime": "node18",
  "public": true,
  "deterministic": true,
  "cache_ttl": 86400,
  "timeout_ms": 5000,
  "memory_mb": 128,
  "description": "Convert string to URL-friendly slug"
}`,
	},
	"python": {
		File: "main.py",
		Content: `async def handler(input: str) -> str:
    return input.lower().replace(" ", "-")`,
		Manifest: `{
  "$schema": "https://functionfly.com/schemas/functionfly.json",
  "name": "%s",
  "version": "1.0.0",
  "runtime": "python3.11",
  "public": true,
  "deterministic": true,
  "cache_ttl": 86400,
  "timeout_ms": 5000,
  "memory_mb": 128,
  "description": "Convert string to URL-friendly slug"
}`,
	},
}

func init() {
	rootCmd.AddCommand(initCmd)

	// Local flags
	initCmd.Flags().StringP("template", "t", "javascript", "Template to use (javascript, typescript, python)")
}

// initRun implements the init command
func initRun(cmd *cobra.Command, args []string) {
	template, _ := cmd.Flags().GetString("template")

	// Validate template
	if _, exists := templates[template]; !exists {
		log.Fatalf("Invalid template '%s'. Supported templates: javascript, typescript, python", template)
	}

	// Get function name from args or use current directory name
	name := "myfunction"
	if len(args) > 0 {
		name = args[0]
	} else {
		// Use current directory name
		if cwd, err := os.Getwd(); err == nil {
			name = filepath.Base(cwd)
		}
	}

	// Validate name format
	if !isValidFunctionName(name) {
		log.Fatalf("Invalid function name '%s'. Name must contain only lowercase letters, numbers, and hyphens", name)
	}

	fmt.Printf("Creating function '%s' with %s template...\n", name, template)

	// Create function file
	tmpl := templates[template]
	if err := os.WriteFile(tmpl.File, []byte(tmpl.Content), 0644); err != nil {
		log.Fatalf("Failed to create %s: %v", tmpl.File, err)
	}

	// Create manifest file
	manifestContent := fmt.Sprintf(tmpl.Manifest, name)
	if err := os.WriteFile("functionfly.jsonc", []byte(manifestContent), 0644); err != nil {
		log.Fatalf("Failed to create functionfly.jsonc: %v", err)
	}

	// Create test file
	testContent := `POST http://localhost:8787
Content-Type: text/plain

Hello World`
	if err := os.WriteFile("test.http", []byte(testContent), 0644); err != nil {
		log.Fatalf("Failed to create test.http: %v", err)
	}

	fmt.Printf("✓ Created %s\n", tmpl.File)
	fmt.Printf("✓ Created functionfly.jsonc\n")
	fmt.Printf("✓ Created test.http\n")
	fmt.Printf("\nFunction '%s' initialized successfully!\n", name)
	fmt.Printf("Run 'fly dev' to start local development server\n")
}

// isValidFunctionName validates function name format
func isValidFunctionName(name string) bool {
	if len(name) == 0 || len(name) > 64 {
		return false
	}

	for _, r := range name {
		if !((r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-') {
			return false
		}
	}

	return !strings.HasPrefix(name, "-") && !strings.HasSuffix(name, "-")
}
