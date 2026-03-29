/*
Copyright © 2026 FunctionFly

*/
package cmd

import (
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/functionfly/fly/internal/cli"
	"github.com/functionfly/fly/internal/credentials"
	"github.com/functionfly/fly/internal/manifest"
	"github.com/spf13/cobra"
)

// testCmd represents the test command
var testCmd = &cobra.Command{
	Use:   "test",
	Short: "Run remote execution tests to verify deployment",
	Long: `Runs remote execution tests to verify deployment.

Makes HTTP request to deployed function and shows:
- Response status and body
- Latency measurement
- Cache status
- Serving region

Examples:
  fly test
  fly test --input="Hello World"
  fly test --json`,
	Run: testRun,
}

func init() {
	rootCmd.AddCommand(testCmd)

	// Local flags
	testCmd.Flags().StringP("input", "i", "Hello World", "Input data to send to function")
	testCmd.Flags().BoolP("json", "j", false, "Output results in JSON format")
}

// testRun implements the test command
func testRun(cmd *cobra.Command, args []string) {
	input, _ := cmd.Flags().GetString("input")
	jsonOutput, _ := cmd.Flags().GetBool("json")

	fmt.Println("Testing remote deployment...")

	// 1. Load manifest
	m, err := manifest.Load("")
	if err != nil {
		log.Fatalf("No functionfly.json found. Run 'fly init' first: %v", err)
	}

	// 2. Load credentials
	creds, err := credentials.Load()
	if err != nil {
		log.Fatalf("Not logged in. Run 'fly login' first: %v", err)
	}

	// 3. Create API client and test function
	apiURL := getAPIURL()
	client := cli.NewClient(apiURL, creds.Token)

	if !jsonOutput {
		fmt.Printf("Testing %s/%s...\n\n", creds.User.Username, m.Name)
	}

	// 4. Test function via API
	result, err := client.TestFunction(creds.User.Username, m.Name, input)
	if err != nil {
		log.Fatalf("Test failed: %v", err)
	}

	// 5. Output results
	if jsonOutput {
		outputJSON(result.Status, result.Body, result.LatencyMs, result.Cached, result.Region)
	} else {
		outputHuman(result.Status, result.Body, result.LatencyMs, result.Cached, result.Region)
	}
}

// outputHuman prints results in human-readable format
func outputHuman(status int, body string, latencyMs int, cached bool, region string) {
	fmt.Printf("Response (%d %s):\n", status, http.StatusText(status))
	fmt.Printf("%s\n\n", body)
	fmt.Printf("latency: %dms\n", latencyMs)
	fmt.Printf("cached: %t\n", cached)
	fmt.Printf("region: %s\n\n", region)

	if status >= 200 && status < 300 {
		fmt.Printf("✓ Test passed\n")
	} else {
		fmt.Printf("✗ Test failed\n")
		log.Fatalf("Function returned error status: %d", status)
	}
}

// outputJSON prints results in JSON format
func outputJSON(status int, body string, latencyMs int, cached bool, region string) {
	// Simple JSON output for automation
	fmt.Printf(`{
  "status": %d,
  "body": %q,
  "latency_ms": %d,
  "cached": %t,
  "region": %q,
  "success": %t
}`, status, body, latencyMs, cached, region, status >= 200 && status < 300)
}

// extractRegion extracts region from Cloudflare ray ID
// CF-Ray format: [ray-id]-[region-code]
func extractRegion(cfRay string) string {
	if cfRay == "" {
		return "unknown"
	}

	// Cloudflare ray ID format: [16-char hex]-[3-char region]
	parts := strings.Split(cfRay, "-")
	if len(parts) >= 2 {
		region := parts[len(parts)-1]
		// Map common region codes to names
		regionMap := map[string]string{
			"dfw": "Dallas",
			"iad": "Washington DC",
			"sfo": "San Francisco",
			"lax": "Los Angeles",
			"ord": "Chicago",
			"atl": "Atlanta",
			"sea": "Seattle",
			"den": "Denver",
			"mci": "Kansas City",
			"bos": "Boston",
			"jfk": "New York",
			"ewr": "Newark",
			"phx": "Phoenix",
			"slc": "Salt Lake City",
		}

		if name, exists := regionMap[strings.ToLower(region)]; exists {
			return name
		}
		return region
	}

	return cfRay
}
