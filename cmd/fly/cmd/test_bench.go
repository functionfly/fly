/*
Copyright © 2026 FunctionFly

*/
package cmd

import (
	"fmt"
	"log"
	"time"

	"github.com/functionfly/fly/internal/bundler"
	"github.com/functionfly/fly/internal/manifest"
	"github.com/functionfly/fly/internal/testing"
	"github.com/spf13/cobra"
)

// testBenchCmd represents the test bench command
var testBenchCmd = &cobra.Command{
	Use:   "bench",
	Short: "Performance benchmarking and load testing",
	Long: `Runs performance benchmarking and load testing for functions.
Measures latency, throughput, memory usage, and other performance metrics.

Examples:
  fly test bench
  fly test bench --input="Hello World" --concurrency=10 --iterations=1000
  fly test bench --memory-profile --duration=60s`,
	Run: testBenchRun,
}

var testBenchFlags struct {
	input          string
	concurrency    int
	iterations     int
	duration       time.Duration
	memoryProfile  bool
	loadPattern    string
	jsonOutput     bool
}

func init() {
	testCmd.AddCommand(testBenchCmd)

	// Local flags
	testBenchCmd.Flags().StringVarP(&testBenchFlags.input, "input", "i", "Hello World", "Input data to send to function")
	testBenchCmd.Flags().IntVarP(&testBenchFlags.concurrency, "concurrency", "c", 10, "Number of concurrent requests")
	testBenchCmd.Flags().IntVarP(&testBenchFlags.iterations, "iterations", "n", 100, "Number of test iterations (0 for unlimited)")
	testBenchCmd.Flags().DurationVarP(&testBenchFlags.duration, "duration", "d", 30*time.Second, "Test duration (used when iterations=0)")
	testBenchCmd.Flags().BoolVarP(&testBenchFlags.memoryProfile, "memory-profile", "m", false, "Enable memory profiling")
	testBenchCmd.Flags().StringVarP(&testBenchFlags.loadPattern, "load-pattern", "p", "constant", "Load pattern (constant, ramp-up, spike)")
	testBenchCmd.Flags().BoolVarP(&testBenchFlags.jsonOutput, "json", "j", false, "Output results in JSON format")
}

// testBenchRun implements the test bench command
func testBenchRun(cmd *cobra.Command, args []string) {
	fmt.Println("Running performance benchmark...")

	// 1. Load manifest
	m, err := manifest.Load("")
	if err != nil {
		log.Fatalf("No functionfly.json found. Run 'fly init' first: %v", err)
	}

	if err := m.Validate(); err != nil {
		log.Fatalf("Manifest validation failed: %v", err)
	}

	// 2. Bundle function code
	fmt.Println("✓ Bundling function code...")
	bundle, err := bundler.BundleWithWorkingDirectory(m, "")
	if err != nil {
		log.Fatalf("Bundling failed: %v", err)
	}

	// 3. Create benchmark runner
	runner := testing.NewLocalTestRunner(bundle, m)

	// 4. Configure benchmark
	config := testing.BenchmarkConfig{
		Input:         testBenchFlags.input,
		Concurrency:   testBenchFlags.concurrency,
		Iterations:    testBenchFlags.iterations,
		Duration:      testBenchFlags.duration,
		MemoryProfile: testBenchFlags.memoryProfile,
		LoadPattern:   testBenchFlags.loadPattern,
	}

	// 5. Run benchmark
	fmt.Printf("✓ Starting benchmark (%d concurrency, %d iterations)...\n", config.Concurrency, config.Iterations)
	result, err := runner.Benchmark(config)
	if err != nil {
		log.Fatalf("Benchmark failed: %v", err)
	}

	// 6. Output results
	if testBenchFlags.jsonOutput {
		outputBenchJSON(result)
	} else {
		outputBenchHuman(result)
	}
}

// outputBenchHuman prints benchmark results in human-readable format
func outputBenchHuman(result *testing.BenchmarkResult) {
	fmt.Println("\nBenchmark Results:")
	fmt.Println("==================")

	fmt.Printf("Total Requests:     %d\n", result.TotalRequests)
	fmt.Printf("Successful:         %d\n", result.SuccessfulRequests)
	fmt.Printf("Failed:            %d\n", result.FailedRequests)
	fmt.Printf("Duration:          %v\n\n", result.Duration)

	fmt.Println("Latency Statistics:")
	fmt.Printf("  Average:         %.2fms\n", result.AvgLatencyMs)
	fmt.Printf("  Minimum:         %.2fms\n", result.MinLatencyMs)
	fmt.Printf("  Maximum:         %.2fms\n", result.MaxLatencyMs)
	fmt.Printf("  P50:             %.2fms\n", result.P50LatencyMs)
	fmt.Printf("  P95:             %.2fms\n", result.P95LatencyMs)
	fmt.Printf("  P99:             %.2fms\n\n", result.P99LatencyMs)

	fmt.Println("Throughput:")
	fmt.Printf("  Requests/sec:    %.2f\n", result.RequestsPerSec)
	fmt.Printf("  Data transferred: %.2f MB\n\n", result.DataTransferredMB)

	if result.MemoryProfile != nil {
		fmt.Println("Memory Usage:")
		fmt.Printf("  Peak Memory:     %.2f MB\n", result.MemoryProfile.PeakMemoryMB)
		fmt.Printf("  Average Memory:  %.2f MB\n", result.MemoryProfile.AvgMemoryMB)
		fmt.Printf("  Memory Leaks:    %t\n\n", result.MemoryProfile.HasLeaks)
	}

	fmt.Println("Error Analysis:")
	if len(result.Errors) > 0 {
		for errorType, count := range result.Errors {
			fmt.Printf("  %s: %d\n", errorType, count)
		}
	} else {
		fmt.Println("  No errors detected")
	}
}

// outputBenchJSON prints benchmark results in JSON format
func outputBenchJSON(result *testing.BenchmarkResult) {
	fmt.Printf(`{
  "total_requests": %d,
  "successful_requests": %d,
  "failed_requests": %d,
  "duration_ms": %d,
  "latency": {
    "avg_ms": %.2f,
    "min_ms": %.2f,
    "max_ms": %.2f,
    "p50_ms": %.2f,
    "p95_ms": %.2f,
    "p99_ms": %.2f
  },
  "throughput": {
    "requests_per_sec": %.2f,
    "data_transferred_mb": %.2f
  }`,
		result.TotalRequests,
		result.SuccessfulRequests,
		result.FailedRequests,
		result.Duration.Milliseconds(),
		result.AvgLatencyMs,
		result.MinLatencyMs,
		result.MaxLatencyMs,
		result.P50LatencyMs,
		result.P95LatencyMs,
		result.P99LatencyMs,
		result.RequestsPerSec,
		result.DataTransferredMB)

	if result.MemoryProfile != nil {
		fmt.Printf(`,
  "memory": {
    "peak_mb": %.2f,
    "avg_mb": %.2f,
    "has_leaks": %t
  }`, result.MemoryProfile.PeakMemoryMB, result.MemoryProfile.AvgMemoryMB, result.MemoryProfile.HasLeaks)
	}

	if len(result.Errors) > 0 {
		fmt.Printf(`,
  "errors": {`)
		i := 0
		for errorType, count := range result.Errors {
			if i > 0 {
				fmt.Printf(",")
			}
			fmt.Printf("\n    %q: %d", errorType, count)
			i++
		}
		fmt.Printf("\n  }")
	}

	fmt.Println("\n}")
}