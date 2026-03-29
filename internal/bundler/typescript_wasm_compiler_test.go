//go:build integration

package bundler

import (
	"testing"
)

// TestCompileTypeScript tests the TypeScript compiler
func TestCompileTypeScript(t *testing.T) {
	// Test source code
	source := `const handler = async (request: any, env: any, context: any) => {
  return {
    status: 200,
    headers: { "Content-Type": "application/json" },
    body: { message: "Hello, World!" }
  };
};
export default handler;`

	// Test compilation
	compiled, err := CompileTypeScript(source, nil)
	if err != nil {
		t.Fatalf("Failed to compile TypeScript: %v", err)
	}

	if compiled == nil {
		t.Fatal("Compiled result is nil")
	}

	if compiled.Binary == nil {
		t.Fatal("Compiled binary is nil")
	}

	if compiled.Metadata == nil {
		t.Fatal("Compiled metadata is nil")
	}
}

// TestExtractMetadata tests metadata extraction from bundled WASM
func TestExtractMetadata(t *testing.T) {
	// Create a test bundle with embedded metadata
	metadata := &WASMMetadata{
		HandlerName:       "testHandler",
		MemoryPages:       128,
		ExportedFunctions: []string{"init", "execute", "alloc"},
		WASITarget:        true,
	}

	// Embed metadata
	wasmBinary := []byte{0, 1, 2, 3, 4, 5} // Dummy WASM bytes
	embedded, err := embedMetadata(wasmBinary, metadata)
	if err != nil {
		t.Fatalf("Failed to embed metadata: %v", err)
	}

	// Extract metadata
	extracted, err := ExtractMetadata(embedded)
	if err != nil {
		t.Fatalf("Failed to extract metadata: %v", err)
	}

	if extracted.HandlerName != metadata.HandlerName {
		t.Errorf("Handler name mismatch: got %s, want %s", extracted.HandlerName, metadata.HandlerName)
	}

	if extracted.MemoryPages != metadata.MemoryPages {
		t.Errorf("Memory pages mismatch: got %d, want %d", extracted.MemoryPages, metadata.MemoryPages)
	}

	if !extracted.WASITarget {
		t.Error("WASI target should be true")
	}
}

// TestGetWASMBinary tests extracting WASM binary from bundle
func TestGetWASMBinary(t *testing.T) {
	// Create a test bundle
	wasmBinary := []byte{0x00, 0x61, 0x73, 0x6d} // WASM magic number
	metadata := &WASMMetadata{
		HandlerName: "handler",
	}

	embedded, err := embedMetadata(wasmBinary, metadata)
	if err != nil {
		t.Fatalf("Failed to embed metadata: %v", err)
	}

	// Extract WASM binary
	extracted, err := GetWASMBinary(embedded)
	if err != nil {
		t.Fatalf("Failed to extract WASM binary: %v", err)
	}

	// Verify it's the original WASM binary
	if len(extracted) != len(wasmBinary) {
		t.Errorf("WASM binary length mismatch: got %d, want %d", len(extracted), len(wasmBinary))
	}

	// Check magic number
	if extracted[0] != 0x00 || extracted[1] != 0x61 || extracted[2] != 0x73 || extracted[3] != 0x6d {
		t.Error("WASM magic number not preserved")
	}
}

// TestIsTypeScriptFile tests TypeScript file detection
func TestIsTypeScriptFile(t *testing.T) {
	tests := []struct {
		path     string
		expected bool
	}{
		{"main.ts", true},
		{"main.tsx", true},
		{"index.js", false},
		{"app.jsx", false},
		{"test.TS", true},
		{"test.TSX", true},
		{"handler", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			result := isTypeScriptFile(tt.path)
			if result != tt.expected {
				t.Errorf("isTypeScriptFile(%q) = %v, want %v", tt.path, result, tt.expected)
			}
		})
	}
}

// TestCompileWithDeps tests compilation with dependencies (resolved via npm registry).
func TestCompileWithDeps(t *testing.T) {
	source := `import isOdd from "is-odd";
const handler = async (request: any, env: any, context: any) => {
  return {
    status: 200,
    headers: { "Content-Type": "application/json" },
    body: { message: "Hello!", odd: isOdd(1) }
  };
};
export default handler;`

	deps := map[string]string{
		"is-odd": "^3.0.0",
	}

	compiled, err := CompileWithDeps(source, deps)
	if err != nil {
		t.Fatalf("Failed to compile TypeScript with deps: %v", err)
	}

	if compiled == nil || compiled.Binary == nil {
		t.Fatal("Expected compiled result with binary")
	}
}

// TestDefaultTypeScriptWASMConfig tests default configuration
func TestDefaultTypeScriptWASMConfig(t *testing.T) {
	config := DefaultTypeScriptWASMConfig()

	if config.Target != "wasip1" {
		t.Errorf("Default target = %q, want %q", config.Target, "wasip1")
	}

	if config.ModuleType != "esm" {
		t.Errorf("Default module type = %q, want %q", config.ModuleType, "esm")
	}

	if config.Minify != false {
		t.Error("Default minify should be false")
	}

	if config.TreeShaking != true {
		t.Error("Default tree shaking should be true")
	}
}
