# FlyPy Compiler

FlyPy is a Python-to-WebAssembly compiler that enables running Python functions as deterministic, edge-deployable WebAssembly modules.

## Features

- **Python to WASM Compilation**: Compiles Python functions to WebAssembly for edge deployment
- **Deterministic Execution**: Guarantees reproducible results for the same inputs
- **Multiple Execution Modes**: Supports deterministic, complex, and compatible modes
- **Type Inference**: Automatically infers types from Python expressions
- **Comprehensive Error Messages**: Provides helpful suggestions for fixing issues

## Supported Python Features

### Core Features (All Modes)

| Feature | Support | Notes |
|---------|---------|-------|
| Function definitions | ✅ | Single handler function |
| Variables | ✅ | Type inference supported |
| Arithmetic operations | ✅ | +, -, *, /, %, ** |
| Comparison operators | ✅ | ==, !=, <, <=, >, >= |
| Boolean operators | ✅ | and, or, not |
| If/else statements | ✅ | Full support |
| Lists | ✅ | Creation and manipulation |
| Dictionaries | ✅ | Creation and access |
| String operations | ✅ | Common methods supported |
| Subscript access | ✅ | arr[i], dict[key] |
| Slice operations | ✅ | arr[1:3], arr[:5], arr[2:] |

### Complex Mode Features

| Feature | Support | Notes |
|---------|---------|-------|
| For loops | ✅ | With break/continue |
| While loops | ✅ | With runtime limits |
| List comprehensions | ✅ | With filter conditions |
| Augmented assignment | ✅ | +=, -=, *=, /= |
| Try/except blocks | ✅ | Exception handling |
| Raise statements | ✅ | Limited support |

### Standard Library Modules

#### Deterministic Mode
- `json` - JSON parsing and serialization
- `math` - Mathematical functions
- `typing` - Type hints
- `collections` - Collection utilities

#### Complex Mode (extends Deterministic)
- `csv` - CSV parsing and writing
- `io` - StringIO and BytesIO
- `re` - Regular expressions
- `datetime` - Date/time operations
- `hashlib` - Deterministic hashing (MD5, SHA256)
- `base64` - Base64 encoding/decoding
- `itertools` - Iterator utilities
- `functools` - Functional programming
- `operator` - Operator functions
- `string` - String utilities
- `textwrap` - Text wrapping
- `uuid` - UUID5 (deterministic only)
- `decimal` - Decimal arithmetic
- `fractions` - Rational numbers

## Execution Modes

### Deterministic Mode
- Pure functional execution
- No side effects
- Fully reproducible
- Best for caching and CDN deployment

### Complex Mode
- Extended stdlib support
- Loops and comprehensions
- Exception handling
- Still deterministic

### Compatible Mode
- Full Python compatibility via MicroPython
- May have non-deterministic behavior
- Use for migration and testing

## Usage

```go
package main

import (
    "context"
    "fmt"
    
    "github.com/functionfly/functionfly/internal/flypy"
)

func main() {
    // Create compiler with complex mode
    compiler := flypy.NewCompiler(&flypy.Config{
        Mode:      flypy.ComplexMode,
        OutputDir: "./dist",
        Verbose:   true,
    })
    
    // Python source code
    source := `
def handler(event):
    items = event.get("items", [])
    result = [x * 2 for x in items if x > 0]
    return {"doubled": result}
`
    
    // Compile
    result, err := compiler.Compile(context.Background(), source, "my_function")
    if err != nil {
        panic(err)
    }
    
    fmt.Printf("Compiled WASM: %d bytes\n", len(result.Artifact.WasmModule))
}
```

## Error Messages

FlyPy provides detailed error messages with suggestions:

```
❌ Compilation Error: FORBIDDEN_BUILTIN
   Line 5: builtin function 'print' is not allowed in deterministic mode
   💡 Tip: Return data as part of your function output instead of printing
   📖 Docs: https://docs.functionfly.dev/restrictions/builtins
```

## Type Inference

FlyPy automatically infers types from expressions:

```python
def handler(event):
    x = 42              # inferred as int
    name = "hello"      # inferred as string
    items = [1, 2, 3]   # inferred as list
    data = {"a": 1}     # inferred as dict
    flag = True         # inferred as bool
    result = x + 10     # inferred as int
    total = sum(items)  # inferred as int
    return {"result": result}
```

## List Comprehensions

```python
# Simple comprehension
doubled = [x * 2 for x in items]

# With filter condition
evens = [x for x in range(10) if x % 2 == 0]

# Nested comprehension (limited support)
matrix = [[i * j for j in range(5)] for i in range(5)]
```

## Slice Operations

```python
# Basic slices
first_three = items[:3]
last_two = items[-2:]
middle = items[1:4]
every_other = items[::2]

# In list comprehensions
processed = [x.strip() for x in lines[1:-1]]
```

## Try/Except Blocks

```python
def handler(event):
    try:
        value = event["key"]
        return {"found": value}
    except KeyError:
        return {"error": "key not found"}
    finally:
        # cleanup code
        pass
```

## Restrictions

### Forbidden Builtins
- I/O: `print`, `input`, `open`
- Dynamic code: `eval`, `exec`, `compile`
- System: `exit`, `quit`, `system`
- Random: `random`, `randint`, `choice`, `shuffle`
- Time: `time`, `sleep`
- Environment: `getenv`, `setenv`
- Reflection: `vars`, `dir`, `globals`, `locals`
- File ops: `remove`, `rename`, `mkdir`, `rmdir`, `chdir`
- Network: `socket`, `urllib`, `requests`

### Forbidden Patterns
- `__import__` - Dynamic imports
- `getattr`, `setattr`, `delattr`, `hasattr` - Reflection

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                    FlyPy Compiler Pipeline                   │
├─────────────────────────────────────────────────────────────┤
│                                                              │
│  ┌──────────┐    ┌──────────┐    ┌──────────┐              │
│  │  Python  │───▶│   AST    │───▶│    IR    │              │
│  │  Source  │    │  Parser  │    │ Generator│              │
│  └──────────┘    └──────────┘    └──────────┘              │
│                                        │                     │
│                                        ▼                     │
│  ┌──────────┐    ┌──────────┐    ┌──────────┐              │
│  │  WASM    │◀───│   Rust   │◀───│Verifier  │              │
│  │  Output  │    │  Emitter │    │(Det/SE)  │              │
│  └──────────┘    └──────────┘    └──────────┘              │
│                                        │                     │
│                                        ▼                     │
│                                 ┌──────────┐                │
│                                 │ Artifact │                │
│                                 │  Bundle  │                │
│                                 └──────────┘                │
└─────────────────────────────────────────────────────────────┘
```

## Testing

Run the test suite:

```bash
# Unit tests
go test ./internal/flypy/...

# Integration tests (requires Python)
go test ./internal/flypy/... -tags=integration

# Benchmarks
go test ./internal/flypy/... -bench=.
```

## Performance

Typical compilation times:
- Simple function: < 100ms
- Complex function with loops: < 500ms
- Full module with dependencies: < 2s

Generated WASM sizes:
- Simple function: ~50KB
- With stdlib modules: ~200KB
- Complex mode: ~500KB

## Contributing

1. Fork the repository
2. Create a feature branch
3. Add tests for new features
4. Submit a pull request

## License

MIT License - see LICENSE file for details.
