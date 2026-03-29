/*
Copyright © 2026 FunctionFly

*/
package cmd

import (
	"fmt"
	"log"

	"github.com/functionfly/fly/internal/manifest"
	"github.com/functionfly/fly/internal/testing"
	"github.com/spf13/cobra"
)

// testValidateCmd represents the test validate command
var testValidateCmd = &cobra.Command{
	Use:   "validate",
	Short: "Validate function configuration and dependencies",
	Long: `Validates function configuration, manifest, dependencies, and runtime compatibility.
Performs comprehensive checks to ensure the function is ready for deployment.

Examples:
  fly test validate
  fly test validate --strict
  fly test validate --runtime-check`,
	Run: testValidateRun,
}

var testValidateFlags struct {
	strict       bool
	runtimeCheck bool
	depsCheck    bool
	jsonOutput   bool
}

func init() {
	testCmd.AddCommand(testValidateCmd)

	// Local flags
	testValidateCmd.Flags().BoolVarP(&testValidateFlags.strict, "strict", "s", false, "Run strict validation with additional checks")
	testValidateCmd.Flags().BoolVarP(&testValidateFlags.runtimeCheck, "runtime-check", "r", true, "Validate runtime compatibility")
	testValidateCmd.Flags().BoolVarP(&testValidateFlags.depsCheck, "deps-check", "d", true, "Check dependencies and imports")
	testValidateCmd.Flags().BoolVarP(&testValidateFlags.jsonOutput, "json", "j", false, "Output results in JSON format")
}

// testValidateRun implements the test validate command
func testValidateRun(cmd *cobra.Command, args []string) {
	fmt.Println("Validating function configuration...")

	// 1. Load manifest
	m, err := manifest.Load("")
	if err != nil {
		log.Fatalf("No functionfly.json found. Run 'fly init' first: %v", err)
	}

	// 2. Create validator
	validator := testing.NewValidator(m)

	// 3. Run validation checks
	results := []testing.ValidationResult{}

	// Manifest validation
	fmt.Println("✓ Validating manifest...")
	manifestResult := validator.ValidateManifest()
	results = append(results, manifestResult)

	// Runtime validation
	if testValidateFlags.runtimeCheck {
		fmt.Println("✓ Validating runtime compatibility...")
		runtimeResult := validator.ValidateRuntime()
		results = append(results, runtimeResult)
	}

	// Dependencies validation
	if testValidateFlags.depsCheck {
		fmt.Println("✓ Checking dependencies...")
		depsResult := validator.ValidateDependencies()
		results = append(results, depsResult)
	}

	// Strict validation
	if testValidateFlags.strict {
		fmt.Println("✓ Running strict validation...")
		strictResults := validator.ValidateStrict()
		results = append(results, strictResults...)
	}

	// 4. Output results
	if testValidateFlags.jsonOutput {
		outputValidateJSON(results)
	} else {
		outputValidateHuman(results)
	}

	// 5. Exit with error if any validation failed
	for _, result := range results {
		if !result.Passed {
			log.Fatalf("Validation failed: %s", result.Message)
		}
	}

	fmt.Println("\n✓ All validations passed!")
}

// outputValidateHuman prints validation results in human-readable format
func outputValidateHuman(results []testing.ValidationResult) {
	fmt.Println("\nValidation Results:")
	fmt.Println("==================")

	passed := 0
	total := len(results)

	for _, result := range results {
		if result.Passed {
			fmt.Printf("✓ %s\n", result.Check)
			passed++
		} else {
			fmt.Printf("✗ %s: %s\n", result.Check, result.Message)
		}
	}

	fmt.Printf("\nPassed: %d/%d\n", passed, total)
}

// outputValidateJSON prints validation results in JSON format
func outputValidateJSON(results []testing.ValidationResult) {
	fmt.Println("[")

	for i, result := range results {
		fmt.Printf(`  {
    "check": %q,
    "passed": %t,
    "message": %q
  }`, result.Check, result.Passed, result.Message)

		if i < len(results)-1 {
			fmt.Println(",")
		} else {
			fmt.Println()
		}
	}

	fmt.Println("]")
}