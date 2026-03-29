package testing

import (
	"context"
	"fmt"
	"time"

	"github.com/functionfly/fly/internal/manifest"
)

// RuntimeResult represents the result of a runtime execution
type RuntimeResult struct {
	Status    int
	Output    string
	Error     string
	LatencyMs int
}

// Runtime defines the interface for function runtimes
type Runtime interface {
	Initialize(ctx context.Context, bundle []byte, manifest *manifest.Manifest) error
	Execute(ctx context.Context, input string) (*RuntimeResult, error)
	Cleanup() error
}

// RuntimeRegistry holds available runtimes
var runtimeRegistry = make(map[string]Runtime)

// RegisterRuntime registers a runtime implementation
func RegisterRuntime(name string, runtime Runtime) {
	runtimeRegistry[name] = runtime
}

// GetRuntime returns the runtime for the given name
func GetRuntime(name string) Runtime {
	runtime, exists := runtimeRegistry[name]
	if !exists {
		// Return a generic runtime as fallback
		return &GenericRuntime{}
	}
	return runtime
}

// GenericRuntime provides a basic runtime implementation
type GenericRuntime struct{}

func (r *GenericRuntime) Initialize(ctx context.Context, bundle []byte, manifest *manifest.Manifest) error {
	// Basic initialization - in a real implementation this would set up
	// the runtime environment (Node.js, Python, etc.)
	return nil
}

func (r *GenericRuntime) Execute(ctx context.Context, input string) (*RuntimeResult, error) {
	start := time.Now()

	// Simulate function execution
	// In a real implementation, this would invoke the actual function
	output := fmt.Sprintf("Processed: %s", input)

	latency := time.Since(start)

	return &RuntimeResult{
		Status:    200,
		Output:    output,
		LatencyMs: int(latency.Milliseconds()),
	}, nil
}

func (r *GenericRuntime) Cleanup() error {
	// Clean up runtime resources
	return nil
}

// Initialize runtimes
func init() {
	RegisterRuntime("node", &GenericRuntime{})
	RegisterRuntime("python", &GenericRuntime{})
	RegisterRuntime("go", &GenericRuntime{})
	RegisterRuntime("generic", &GenericRuntime{})
}