//go:build !cgo

package flypy

import "fmt"

// RunWasm executes the WASM module. When cgo is disabled, WASM execution is
// unavailable and this returns an error so the caller can fall back to mock mode.
func RunWasm(wasmBytes []byte, inputJSON []byte) ([]byte, error) {
	return nil, fmt.Errorf("WASM execution requires cgo (install wasmtime); build with CGO_ENABLED=1")
}
