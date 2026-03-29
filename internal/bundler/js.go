// Package bundler provides utilities for compiling function source code to
// WebAssembly.  This file implements the JavaScript/TypeScript → Wasm
// compilation path using Javy (Shopify's QuickJS-to-Wasm compiler).
//
// Compilation pipeline:
//
//	TypeScript:  index.ts → esbuild → index.js → javy → function.wasm
//	JavaScript:  index.js            → javy → function.wasm
//
// The resulting .wasm file uses the standard WASI stdin/stdout I/O model and
// runs inside the existing Wasmtime execution path without any changes to the
// runtime layer.
package bundler

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// JSCompileOptions controls how JavaScript source is compiled to Wasm.
type JSCompileOptions struct {
	// Dynamic enables Javy's dynamic linking mode (smaller output, requires
	// the javy_quickjs_provider.wasm side-car at runtime).
	// When false (default) a self-contained static binary is produced.
	Dynamic bool

	// JavyBinaryPath overrides the path to the javy binary.
	// If empty, "javy" is looked up on $PATH.
	JavyBinaryPath string

	// ExtraArgs are appended verbatim to the javy compile command.
	ExtraArgs []string
}

// TSCompileOptions controls how TypeScript source is compiled to Wasm.
type TSCompileOptions struct {
	JSCompileOptions

	// EsbuildBinaryPath overrides the path to the esbuild binary.
	// If empty, "esbuild" is looked up on $PATH.
	EsbuildBinaryPath string

	// Target is the esbuild --target flag value (default: "es2020").
	Target string
}

// CompileJSToWasm compiles JavaScript source code to a WebAssembly binary
// using Javy.
//
// The function:
//  1. Writes jsSource to a temporary file.
//  2. Runs: javy compile <input.js> -o <output.wasm> [extra args]
//  3. Reads and returns the compiled .wasm bytes.
//
// The caller is responsible for storing the returned bytes in the function
// registry (WasmBinary field).
func CompileJSToWasm(jsSource []byte, opts JSCompileOptions) ([]byte, error) {
	// Resolve javy binary
	javyBin := opts.JavyBinaryPath
	if javyBin == "" {
		var err error
		javyBin, err = exec.LookPath("javy")
		if err != nil {
			return nil, fmt.Errorf(
				"javy binary not found on PATH; install from https://github.com/bytecodealliance/javy/releases: %w",
				err,
			)
		}
	}

	// Create a temporary working directory
	tmpDir, err := os.MkdirTemp("", "functionfly-javy-*")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp dir for javy compilation: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	// Write JS source to a temp file
	inputPath := filepath.Join(tmpDir, "input.js")
	if err := os.WriteFile(inputPath, jsSource, 0600); err != nil {
		return nil, fmt.Errorf("failed to write JS source to temp file: %w", err)
	}

	outputPath := filepath.Join(tmpDir, "output.wasm")

	// Build javy command
	args := []string{"compile", inputPath, "-o", outputPath}
	if opts.Dynamic {
		args = append(args, "--dynamic")
	}
	args = append(args, opts.ExtraArgs...)

	cmd := exec.Command(javyBin, args...)
	cmd.Dir = tmpDir

	if out, err := cmd.CombinedOutput(); err != nil {
		return nil, fmt.Errorf("javy compilation failed: %w\noutput:\n%s", err, string(out))
	}

	// Read compiled Wasm
	wasmBytes, err := os.ReadFile(outputPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read javy output: %w", err)
	}

	return wasmBytes, nil
}

// CompileTSToWasm compiles TypeScript source code to a WebAssembly binary.
//
// The function first transpiles TypeScript to JavaScript using esbuild, then
// compiles the resulting JavaScript to Wasm using Javy.
//
// TypeScript → esbuild → JavaScript → Javy → WebAssembly
func CompileTSToWasm(tsSource []byte, opts TSCompileOptions) ([]byte, error) {
	// Step 1: Transpile TypeScript → JavaScript using esbuild
	jsSource, err := transpileTSWithEsbuild(tsSource, opts)
	if err != nil {
		return nil, fmt.Errorf("TypeScript transpilation failed: %w", err)
	}

	// Step 2: Compile JavaScript → Wasm using Javy
	return CompileJSToWasm(jsSource, opts.JSCompileOptions)
}

// transpileTSWithEsbuild transpiles TypeScript source to JavaScript using
// esbuild.  The output is a single bundled ES module suitable for Javy.
func transpileTSWithEsbuild(tsSource []byte, opts TSCompileOptions) ([]byte, error) {
	// Resolve esbuild binary
	esbuildBin := opts.EsbuildBinaryPath
	if esbuildBin == "" {
		var err error
		esbuildBin, err = exec.LookPath("esbuild")
		if err != nil {
			return nil, fmt.Errorf(
				"esbuild binary not found on PATH; install with: npm install -g esbuild: %w",
				err,
			)
		}
	}

	// Create a temporary working directory
	tmpDir, err := os.MkdirTemp("", "functionfly-esbuild-*")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp dir for esbuild: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	// Write TypeScript source to a temp file
	inputPath := filepath.Join(tmpDir, "input.ts")
	if err := os.WriteFile(inputPath, tsSource, 0600); err != nil {
		return nil, fmt.Errorf("failed to write TS source to temp file: %w", err)
	}

	outputPath := filepath.Join(tmpDir, "output.js")

	// Determine target
	target := opts.Target
	if target == "" {
		target = "es2020"
	}

	// Build esbuild command
	// --bundle: inline all imports into a single file
	// --platform=browser: avoid Node.js-specific APIs
	// --format=esm: output ES module (required by Javy)
	args := []string{
		inputPath,
		"--bundle",
		"--platform=browser",
		"--format=esm",
		fmt.Sprintf("--target=%s", target),
		fmt.Sprintf("--outfile=%s", outputPath),
	}

	cmd := exec.Command(esbuildBin, args...)
	cmd.Dir = tmpDir

	if out, err := cmd.CombinedOutput(); err != nil {
		return nil, fmt.Errorf("esbuild failed: %w\noutput:\n%s", err, string(out))
	}

	// Read transpiled JavaScript
	jsBytes, err := os.ReadFile(outputPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read esbuild output: %w", err)
	}

	return jsBytes, nil
}

// IsJavyAvailable reports whether the javy binary is available on the system.
// This can be used to gate JS/TS compilation at startup.
func IsJavyAvailable() bool {
	_, err := exec.LookPath("javy")
	return err == nil
}

// IsEsbuildAvailable reports whether the esbuild binary is available on the
// system.  This can be used to gate TypeScript compilation at startup.
func IsEsbuildAvailable() bool {
	_, err := exec.LookPath("esbuild")
	return err == nil
}
