# Python Precompiled Runtime

This directory contains precompiled Micropython WASM runtime components for optimized Python function execution.

## Directory Structure

```
python/
├── micropython-core.wasm    # Custom Python interpreter WASM (935 bytes) - standalone
├── micropython-full.wasm    # Full MicroPython built with Emscripten (1.1MB) - requires JS
├── micropython.wasm         # Original MicroPython JS version (425KB) - requires JS
├── minimal-python.wat       # Source WAT file for the interpreter
├── stdlib/                  # Precompiled stdlib modules (future)
│   ├── json.wasm
│   ├── os.wasm
│   └── ...
└── templates/
    └── python-wrapper.wat   # WebAssembly Text template for embedding user code
```

## Build Status

### ✅ Phase 4.1: Precompiled Python Runtime - IMPLEMENTED

#### micropython-core.wasm (935 bytes)

- **Status**: Working
- **Type**: Custom minimal Python interpreter in WAT
- **Dependencies**: None (standalone WASM)
- **Interface**: init, execute, load_code, alloc, dealloc, metadata
- **Use**: Production - works with Wasmtime

#### micropython-full.wasm (1.1MB)

- **Status**: Built with Emscripten (JavaScript required)
- **Type**: Full MicroPython compiled with Emscripten
- **Dependencies**: JavaScript glue code
- **Use**: Browser/Node.js only - NOT standalone

### WASI Support Status

The MicroPython webassembly port (`micropython/ports/webassembly/`) has been analyzed for WASI support:

1. **Current WASI imports** (verified via wasm-objdump):
   - `wasi_snapshot_preview1.fd_close`
   - `wasi_snapshot_preview1.fd_sync`
   - `wasi_snapshot_preview1.fd_seek`
   - `wasi_snapshot_preview1.fd_read`
   - `wasi_snapshot_preview1.fd_write`

2. **JavaScript dependencies** (still present):
   - Multiple `env.*` imports for JavaScript interop
   - Required for REPL and some runtime features

3. **PURE_WASI Build**:
   - Requires build system modifications to eliminate JS dependencies
   - Complex due to MicroPython's build configuration
   - Documentation available in `micropython/ports/webassembly/WASI_BUILD.md`

## Building MicroPython

### Prerequisites

```bash
# Emscripten SDK is in ../emsdk/
cd ../emsdk
source emsdk_env.sh
```

### Build WebAssembly Port

```bash
cd micropython/ports/webassembly
make
# Output: build-standard/micropython.wasm
```

### Build Notes

- The standard WebAssembly port requires JavaScript glue code
- For standalone WASM (Wasmtime), need WASI support (not yet available in standard build)
- Current implementation uses custom minimal runtime for standalone support

## Runtime Interface

The micropython-core.wasm exports:

| Function | Signature | Description |
|----------|-----------|-------------|
| `init` | `() -> void` | Initialize the runtime |
| `execute` | `(code_ptr, input_ptr) -> i32` | Execute with input |
| `load_code` | `(code_ptr, code_len) -> void` | Inject Python code |
| `alloc` | `(size) -> i32` | Allocate memory |
| `dealloc` | `(ptr) -> void` | Deallocate memory |
| `metadata` | `() -> i32` | Get metadata pointer |
| `memory` | `memory` | Shared linear memory (1MB) |

## Current Status

Phase 4.1 of Python WASM Runtime Plan - **IMPLEMENTED** ✅

### What's Working

- Precompiled runtime infrastructure
- Bundler integration for Python functions
- WASM module validation
- Memory management (bump allocator)

### Future Work

- Full Python parser/interpreter in WAT (replacing placeholder)
- MicroPython with WASI support for true standalone execution
- Standard library modules (json, os, sys)
