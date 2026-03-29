/*
Copyright © 2026 FunctionFly

*/
package cmd

import (
	"fmt"
	"log"
	"time"

	"github.com/functionfly/fly/internal/bundler"
	"github.com/functionfly/fly/internal/credentials"
	"github.com/functionfly/fly/internal/manifest"
	"github.com/functionfly/fly/internal/testing"
	"github.com/spf13/cobra"
)

// testLocalCmd represents the test local command
var testLocalCmd = &cobra.Command{
	Use:   "local",
	Short: "Run comprehensive local function tests with validation",
	Long: `Runs comprehensive local function tests including validation, performance benchmarking,
and automated test suites. Tests are executed in an environment identical to production.

Examples:
  fly test local
  fly test local --input="Hello World"
  fly test local --concurrency=10 --iterations=100`,
	Run: testLocalRun,
}

var testLocalFlags struct {
	input       string
	concurrency int
	iterations  int
	timeout     time.Duration
	validate    bool
	benchmark   bool
	jsonOutput  bool
}

func init() {
	testCmd.AddCommand(testLocalCmd)

	// Local flags
	testLocalCmd.Flags().StringVarP(&testLocalFlags.input, "input", "i", "Hello World", "Input data to send to function")
	testLocalCmd.Flags().IntVarP(&testLocalFlags.concurrency, "concurrency", "c", 1, "Number of concurrent requests")
	testLocalCmd.Flags().IntVarP(&testLocalFlags.iterations, "iterations", "n", 1, "Number of test iterations")
	testLocalCmd.Flags().DurationVarP(&testLocalFlags.timeout, "timeout", "t", 30*time.Second, "Test timeout duration")
	testLocalCmd.Flags().BoolVarP(&testLocalFlags.validate, "validate", "v", true, "Run input/output validation")
	testLocalCmd.Flags().BoolVarP(&testLocalFlags.benchmark, "benchmark", "b", false, "Run performance benchmarking")
	testLocalCmd.Flags().BoolVarP(&testLocalFlags.jsonOutput, "json", "j", false, "Output results in JSON format")
}

// testLocalRun implements the test local command
func testLocalRun(cmd *cobra.Command, args []string) {
	fmt.Println("Running local function tests...")

	// 1. Load and validate manifest
	m, err := manifest.Load("")
	if err != nil {
		log.Fatalf("No functionfly.json found. Run 'fly init' first: %v", err)
	}

	if err := m.Validate(); err != nil {
		log.Fatalf("Manifest validation failed: %v", err)
	}

	// 2. Load credentials
	creds, err := credentials.Load()
	if err != nil {
		log.Fatalf("Not logged in. Run 'fly login' first: %v", err)
	}

	// 3. Bundle function code for local execution
	fmt.Println("✓ Bundling function code...")
	bundle, err := bundler.BundleWithWorkingDirectory(m, "")
	if err != nil {
		log.Fatalf("Bundling failed: %v", err)
	}

	// 4. Create local test runner
	runner := testing.NewLocalTestRunner(bundle, m)

	// 5. Run validation if requested
	if testLocalFlags.validate {
		fmt.Println("✓ Running input/output validation...")
		if err := runner.Validate(testLocalFlags.input); err != nil {
			log.Fatalf("Validation failed: %v", err)
		}
		fmt.Println("✓ Validation passed")
	}

	// 6. Run performance tests if requested
	var benchResult *testing.BenchmarkResult
	if testLocalFlags.benchmark {
		fmt.Println("✓ Running performance benchmark...")
		benchResult, err = runner.Benchmark(testing.BenchmarkConfig{
			Input:       testLocalFlags.input,
			Concurrency: testLocalFlags.concurrency,
			Iterations:  testLocalFlags.iterations,
			Duration:    testLocalFlags.timeout,
		})
		if err != nil {
			log.Fatalf("Benchmark failed: %v", err)
		}
		fmt.Printf("✓ Benchmark completed: %.2fms avg latency\n", benchResult.AvgLatencyMs)
	}

	// 7. Run functional tests
	fmt.Println("✓ Running functional tests...")
	testResult, err := runner.Test(testing.TestConfig{
		Input:       testLocalFlags.input,
		Concurrency: testLocalFlags.concurrency,
		Iterations:  testLocalFlags.iterations,
		Timeout:     testLocalFlags.timeout,
	})
	if err != nil {
		log.Fatalf("Tests failed: %v", err)
	}

	// 8. Output results
	if testLocalFlags.jsonOutput {
		outputTestLocalJSON(testResult, benchResult)
	} else {
		outputTestLocalHuman(testResult, benchResult, creds.User.Username, m.Name)
	}
}

// outputTestLocalHuman prints local test results in human-readable format
func outputTestLocalHuman(result *testing.TestResult, benchResult *testing.BenchmarkResult, author, name string) {
	fmt.Printf("\nFunction: %s/%s\n", author, name)
	fmt.Printf("Input: %s\n\n", result.Input)

	fmt.Printf("Functional Test Results:\n")
	fmt.Printf("✓ Status: %d %s\n", result.Status, getStatusText(result.Status))
	fmt.Printf("✓ Latency: %dms\n", result.LatencyMs)
	fmt.Printf("✓ Output: %s\n\n", result.Output)

	if result.Status >= 200 && result.Status < 300 {
		fmt.Printf("✓ Functional test passed\n")
	} else {
		fmt.Printf("✗ Functional test failed\n")
		log.Fatalf("Function returned error status: %d", result.Status)
	}

	if benchResult != nil {
		fmt.Printf("Performance Benchmark Results:\n")
		fmt.Printf("✓ Total Requests: %d\n", benchResult.TotalRequests)
		fmt.Printf("✓ Successful Requests: %d\n", benchResult.SuccessfulRequests)
		fmt.Printf("✓ Failed Requests: %d\n", benchResult.FailedRequests)
		fmt.Printf("✓ Average Latency: %.2fms\n", benchResult.AvgLatencyMs)
		fmt.Printf("✓ Min Latency: %.2fms\n", benchResult.MinLatencyMs)
		fmt.Printf("✓ Max Latency: %.2fms\n", benchResult.MaxLatencyMs)
		fmt.Printf("✓ P95 Latency: %.2fms\n", benchResult.P95LatencyMs)
		fmt.Printf("✓ Requests/sec: %.2f\n", benchResult.RequestsPerSec)
		if benchResult.MemoryProfile != nil {
			fmt.Printf("✓ Memory Usage: %.2fMB\n\n", benchResult.MemoryProfile.AvgMemoryMB)
		}
	}
}

// outputTestLocalJSON prints local test results in JSON format
func outputTestLocalJSON(result *testing.TestResult, benchResult *testing.BenchmarkResult) {
	// Simple JSON output for automation
	fmt.Printf(`{
  "functional_test": {
    "status": %d,
    "output": %q,
    "latency_ms": %d,
    "success": %t
  }`, result.Status, result.Output, result.LatencyMs, result.Status >= 200 && result.Status < 300)

	if benchResult != nil {
		fmt.Printf(`,
  "benchmark": {
    "total_requests": %d,
    "successful_requests": %d,
    "failed_requests": %d,
    "avg_latency_ms": %.2f,
    "min_latency_ms": %.2f,
    "max_latency_ms": %.2f,
    "p95_latency_ms": %.2f,
    "requests_per_sec": %.2f,
    "memory_usage_mb": %.2f
  }`, benchResult.TotalRequests, benchResult.SuccessfulRequests, benchResult.FailedRequests,
			benchResult.AvgLatencyMs, benchResult.MinLatencyMs, benchResult.MaxLatencyMs,
			benchResult.P95LatencyMs, benchResult.RequestsPerSec, benchResult.MemoryProfile.AvgMemoryMB)
	}

	fmt.Println("\n}")
}

// getStatusText returns human-readable status text
func getStatusText(status int) string {
	switch status {
	case 200:
		return "OK"
	case 400:
		return "Bad Request"
	case 401:
		return "Unauthorized"
	case 403:
		return "Forbidden"
	case 404:
		return "Not Found"
	case 500:
		return "Internal Server Error"
	default:
		return "Unknown"
	}
}