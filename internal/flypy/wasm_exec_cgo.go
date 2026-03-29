//go:build cgo

package flypy

import (
	"encoding/json"
	"fmt"

	"github.com/bytecodealliance/wasmtime-go/v19"
)

// RunWasm executes the WASM module with the given JSON input and returns JSON output.
// Requires cgo (wasmtime). When cgo is disabled, use the stub in wasm_exec_nocgo.go.
func RunWasm(wasmBytes []byte, inputJSON []byte) ([]byte, error) {
	if len(wasmBytes) == 0 {
		return nil, fmt.Errorf("empty WASM module")
	}

	engine := wasmtime.NewEngine()
	module, err := wasmtime.NewModule(engine, wasmBytes)
	if err != nil {
		return nil, fmt.Errorf("compile WASM module: %w", err)
	}

	store := wasmtime.NewStore(engine)
	wasiConfig := wasmtime.NewWasiConfig()
	wasiConfig.InheritStdout()
	wasiConfig.InheritStderr()
	store.SetWasi(wasiConfig)

	linker := wasmtime.NewLinker(engine)
	if err := linker.DefineWasi(); err != nil {
		return nil, fmt.Errorf("define WASI: %w", err)
	}

	instance, err := linker.Instantiate(store, module)
	if err != nil {
		return nil, fmt.Errorf("instantiate module: %w", err)
	}

	// Try handler(input_ptr, input_len) -> result_ptr (common for Python/embedder WASM)
	handlerExport := instance.GetExport(store, "handler")
	if handlerExport != nil && handlerExport.Func() != nil {
		memExport := instance.GetExport(store, "memory")
		if memExport == nil || memExport.Memory() == nil {
			return nil, fmt.Errorf("module exports handler but not memory")
		}
		memory := memExport.Memory()

		allocExport := instance.GetExport(store, "alloc")
		if allocExport == nil || allocExport.Func() == nil {
			return nil, fmt.Errorf("module exports handler but not alloc")
		}
		allocFunc := allocExport.Func()

		// Allocate and write input
		allocResult, err := allocFunc.Call(store, len(inputJSON))
		if err != nil {
			return nil, fmt.Errorf("alloc: %w", err)
		}
		inputPtr, ok := allocResult.(int32)
		if !ok || inputPtr == 0 {
			return nil, fmt.Errorf("alloc returned invalid pointer")
		}

		data := memory.UnsafeData(store)
		if int(inputPtr)+len(inputJSON) > len(data) {
			return nil, fmt.Errorf("memory write out of bounds")
		}
		copy(data[inputPtr:], inputJSON)

		handlerFunc := handlerExport.Func()
		result, err := handlerFunc.Call(store, inputPtr, len(inputJSON))
		if err != nil {
			return nil, fmt.Errorf("handler call: %w", err)
		}

		resultPtr, ok := result.(int32)
		if !ok {
			return nil, fmt.Errorf("handler returned invalid type")
		}
		if resultPtr <= 0 {
			return nil, fmt.Errorf("handler returned error indicator: %d", resultPtr)
		}

		// Read null-terminated or bounded string from result pointer
		const maxOutput = 1024 * 1024
		start := int(resultPtr)
		if start >= len(data) {
			return nil, fmt.Errorf("result pointer out of bounds")
		}
		end := start
		for end < len(data) && end < start+maxOutput && data[end] != 0 {
			end++
		}
		out := make([]byte, end-start)
		copy(out, data[start:end])

		// If output looks like JSON, return as-is; otherwise wrap in {"result": "<output>"}
		if len(out) > 0 && (out[0] == '{' || out[0] == '[') {
			return out, nil
		}
		return json.Marshal(map[string]interface{}{"result": string(out)})
	}

	// Fallback: try _start() (WASI entry) and rely on stdout - not captured here, so return placeholder
	startExport := instance.GetExport(store, "_start")
	if startExport != nil && startExport.Func() != nil {
		_, err := startExport.Func().Call(store)
		if err != nil {
			return nil, fmt.Errorf("_start: %w", err)
		}
		return json.Marshal(map[string]interface{}{"result": "executed (_start)", "stdout": "not captured"})
	}

	return nil, fmt.Errorf("WASM module exports neither handler nor _start")
}
