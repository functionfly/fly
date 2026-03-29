package bundler

import (
	"bytes"
	"testing"
)

func TestValidateWASM_ValidModule(t *testing.T) {
	// Create a minimal valid WASM module
	wasm := createMinimalValidWASM()

	err := validateWasmModule(wasm)
	if err != nil {
		t.Errorf("Expected valid WASM to pass, got: %v", err)
	}
}

func TestValidateWASM_TooSmall(t *testing.T) {
	wasm := []byte{0x00, 0x61, 0x73, 0x6D} // Only magic bytes, no version

	err := validateWasmModule(wasm)
	if err == nil {
		t.Error("Expected error for WASM too small")
	}
}

func TestValidateWASM_InvalidMagic(t *testing.T) {
	wasm := []byte{
		0x01, 0x02, 0x03, 0x04, // Invalid magic
		0x01, 0x00, 0x00, 0x00, // Version 1
	}

	err := validateWasmModule(wasm)
	if err == nil {
		t.Error("Expected error for invalid magic bytes")
	}
}

func TestValidateWASM_UnsupportedVersion(t *testing.T) {
	wasm := []byte{
		0x00, 0x61, 0x73, 0x6D, // Magic
		0x00, 0x00, 0x00, 0x03, // Version 3 (unsupported)
	}

	err := validateWasmModule(wasm)
	if err == nil {
		t.Error("Expected error for unsupported WASM version")
	}
}

func TestValidateWASM_TooLarge(t *testing.T) {
	config := &WASMValidationConfig{
		MaxBinarySize: 100,
	}

	// Create WASM that's too large
	wasm := make([]byte, 200)
	wasm[0] = 0x00
	wasm[1] = 0x61
	wasm[2] = 0x73
	wasm[3] = 0x6D
	wasm[4] = 0x01
	wasm[5] = 0x00
	wasm[6] = 0x00
	wasm[7] = 0x00

	err := ValidateWASM(wasm, config)
	if err == nil {
		t.Error("Expected error for WASM exceeding max size")
	}
}

func TestValidateWASM_TooManyImports(t *testing.T) {
	config := &WASMValidationConfig{
		MaxImports: 5,
	}

	// Create WASM with too many imports
	wasm := createWASMWithImports(10)

	err := ValidateWASM(wasm, config)
	if err == nil {
		t.Error("Expected error for too many imports")
	}
}

func TestValidateWASM_BlockedImport(t *testing.T) {
	config := &WASMValidationConfig{
		BlockedImports: []string{"env:blocked_function"},
	}

	// Create WASM with a blocked import
	wasm := createWASMWithImport("env", "blocked_function", 0)

	err := ValidateWASM(wasm, config)
	if err == nil {
		t.Error("Expected error for blocked import")
	}
}

func TestValidateWASM_BlockedImportModule(t *testing.T) {
	config := &WASMValidationConfig{
		BlockedImports: []string{"env:*"},
	}

	// Create WASM with blocked import module
	wasm := createWASMWithImport("env", "any_function", 0)

	err := ValidateWASM(wasm, config)
	if err == nil {
		t.Error("Expected error for blocked import module")
	}
}

func TestValidateWASM_MemoryLimits(t *testing.T) {
	config := &WASMValidationConfig{
		MaxMemoryPages: 10, // 640KB
	}

	// Create WASM with memory exceeding limit
	wasm := createWASMWithMemory(20) // 1.25MB

	err := ValidateWASM(wasm, config)
	if err == nil {
		t.Error("Expected error for memory exceeding limit")
	}
}

func TestDefaultWASMValidationConfig(t *testing.T) {
	config := DefaultWASMValidationConfig()

	if config.MaxBinarySize != 10*1024*1024 {
		t.Errorf("Expected MaxBinarySize=10MB, got %d", config.MaxBinarySize)
	}

	if config.MaxImports != 100 {
		t.Errorf("Expected MaxImports=100, got %d", config.MaxImports)
	}

	if config.MaxExports != 200 {
		t.Errorf("Expected MaxExports=200, got %d", config.MaxExports)
	}

	if config.MaxFunctions != 500 {
		t.Errorf("Expected MaxFunctions=500, got %d", config.MaxFunctions)
	}

	if config.MaxMemoryPages != 1024 {
		t.Errorf("Expected MaxMemoryPages=1024, got %d", config.MaxMemoryPages)
	}
}

// Helper: create minimal valid WASM module
func createMinimalValidWASM() []byte {
	// WASM binary format:
	// - Magic: \0asm
	// - Version: 1
	// - Type section (id=1): empty (just count=0)
	// - Function section (id=3): empty (just count=0)
	// - Export section (id=7): empty (just count=0)
	// - Code section (id=10): empty (just count=0)

	var wasm bytes.Buffer

	// Magic and version
	wasm.Write([]byte{0x00, 0x61, 0x73, 0x6D}) // \0asm
	wasm.Write([]byte{0x01, 0x00, 0x00, 0x00}) // Version 1

	// Type section (id=1), size=1, count=0
	wasm.Write([]byte{0x01, 0x01, 0x00})

	// Function section (id=3), size=1, count=0
	wasm.Write([]byte{0x03, 0x01, 0x00})

	// Export section (id=7), size=1, count=0
	wasm.Write([]byte{0x07, 0x01, 0x00})

	// Code section (id=10), size=1, count=0
	wasm.Write([]byte{0x0A, 0x01, 0x00})

	return wasm.Bytes()
}

// Helper: create WASM with specified number of empty imports
func createWASMWithImports(count int) []byte {
	var wasm bytes.Buffer

	// Magic and version
	wasm.Write([]byte{0x00, 0x61, 0x73, 0x6D})
	wasm.Write([]byte{0x01, 0x00, 0x00, 0x00})

	// Type section (id=1)
	wasm.Write([]byte{0x01, 0x01, 0x00})

	// Calculate import section size
	// For each import: module_len(1) + module + field_len(1) + field + kind(1) + type_index(varint)
	importSize := uint64(1) // count
	for i := 0; i < count; i++ {
		importSize += uint64(5 + 1 + 4 + 1 + 1) // "env" + "func" + kind + type_idx
	}

	// Import section (id=2)
	wasm.Write([]byte{0x02})
	writeVarUint32(&wasm, importSize)
	writeVarUint32(&wasm, uint64(count))

	for i := 0; i < count; i++ {
		writeVarUint32(&wasm, 3) // "env" length
		wasm.Write([]byte{'e', 'n', 'v'})
		writeVarUint32(&wasm, 4) // "func" length
		wasm.Write([]byte{'f', 'u', 'n', 'c'})
		wasm.Write([]byte{0x00}) // kind = function
		writeVarUint32(&wasm, 0) // type index
	}

	// Function section (id=3)
	wasm.Write([]byte{0x03, 0x01, 0x00})

	// Export section (id=7)
	wasm.Write([]byte{0x07, 0x01, 0x00})

	// Code section (id=10)
	wasm.Write([]byte{0x0A, 0x01, 0x00})

	return wasm.Bytes()
}

// Helper: create WASM with a specific import
func createWASMWithImport(module, field string, kind byte) []byte {
	var wasm bytes.Buffer

	// Magic and version
	wasm.Write([]byte{0x00, 0x61, 0x73, 0x6D})
	wasm.Write([]byte{0x01, 0x00, 0x00, 0x00})

	// Type section (id=1)
	wasm.Write([]byte{0x01, 0x01, 0x00})

	// Import section (id=2)
	importSize := uint64(1 + len(module) + 1 + len(field) + 1 + 1)
	wasm.Write([]byte{0x02})
	writeVarUint32(&wasm, importSize)
	writeVarUint32(&wasm, 1) // count = 1

	writeVarUint32(&wasm, uint64(len(module)))
	wasm.Write([]byte(module))
	writeVarUint32(&wasm, uint64(len(field)))
	wasm.Write([]byte(field))
	wasm.Write([]byte{kind})
	writeVarUint32(&wasm, 0) // type index

	// Function section (id=3)
	wasm.Write([]byte{0x03, 0x01, 0x00})

	// Export section (id=7)
	wasm.Write([]byte{0x07, 0x01, 0x00})

	// Code section (id=10)
	wasm.Write([]byte{0x0A, 0x01, 0x00})

	return wasm.Bytes()
}

// Helper: create WASM with memory specification
func createWASMWithMemory(initialPages uint32) []byte {
	var wasm bytes.Buffer

	// Magic and version
	wasm.Write([]byte{0x00, 0x61, 0x73, 0x6D})
	wasm.Write([]byte{0x01, 0x00, 0x00, 0x00})

	// Type section (id=1)
	wasm.Write([]byte{0x01, 0x01, 0x00})

	// Function section (id=3)
	wasm.Write([]byte{0x03, 0x01, 0x00})

	// Memory section (id=5)
	// Size = 1 (flags) + encoding of initial pages
	memorySize := uint64(1 + varUint32Size(initialPages))
	if initialPages > 65536 {
		memorySize += uint64(varUint32Size(initialPages))
	}

	wasm.Write([]byte{0x05})
	writeVarUint32(&wasm, memorySize)
	wasm.Write([]byte{0x00}) // flags = 0 (no max)
	writeVarUint32(&wasm, uint64(initialPages))

	// Export section (id=7)
	wasm.Write([]byte{0x07, 0x01, 0x00})

	// Code section (id=10)
	wasm.Write([]byte{0x0A, 0x01, 0x00})

	return wasm.Bytes()
}

// Helper: write varint to buffer
func writeVarUint32(buf *bytes.Buffer, val uint64) {
	for {
		b := byte(val & 0x7F)
		val >>= 7
		if val != 0 {
			b |= 0x80
		}
		buf.WriteByte(b)
		if val == 0 {
			break
		}
	}
}

// Helper: calculate size of varint encoding
func varUint32Size(val uint32) int {
	size := 1
	for val > 0x7F {
		size++
		val >>= 7
	}
	return size
}
