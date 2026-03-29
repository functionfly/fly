/*
Copyright © 2026 FunctionFly
*/
package cmd

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/functionfly/fly/cmd/fly/commands"
	"github.com/spf13/cobra"
)

// newBackendListCmd creates the backend list command
func newBackendListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List backends",
		Long: `List all backends for your application.

This command shows all configured execution backends for an application,
including their status, region, and health information.`,
		Example: `  # List backends for an app
  fly backend list --app myapp

  # List backends with JSON output
  fly backend list --app myapp --json`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runBackendList(cmd)
		},
	}

	cmd.Flags().StringVar(&appName, "app", "", "Application name (required)")
	cmd.Flags().BoolVar(&asJSON, "json", false, "Output as JSON")

	cmd.MarkFlagRequired("app")

	return cmd
}

func runBackendList(cmd *cobra.Command) error {
	// Get API client
	client, err := commands.NewAPIClient()
	if err != nil {
		return fmt.Errorf("failed to create API client: %w", err)
	}

	// Get app ID from name
	app, err := client.GetApp(appName)
	if err != nil {
		return fmt.Errorf("failed to get app: %w", err)
	}

	// Get app status (includes backends)
	status, err := client.GetStatus(app.ID)
	if err != nil {
		return fmt.Errorf("failed to get app status: %w", err)
	}

	// Handle JSON output
	if asJSON {
		printJSON(status)
		return nil
	}

	// Print as simple table
	if len(status.Backends) == 0 {
		fmt.Printf("No backends configured for app %s\n", appName)
		return nil
	}

	fmt.Printf("\nBackends for %s\n\n", appName)
	fmt.Printf("%-10s %-12s %-12s %-40s %s\n", "ID", "Provider", "Region", "URL", "Status")
	fmt.Println(strings.Repeat("-", 95))

	for _, backend := range status.Backends {
		statusText := "unknown"
		if backend.CircuitState != nil {
			statusText = backend.CircuitState.State
		}
		if backend.LatestHealthCheck != nil && backend.LatestHealthCheck.OK {
			statusText = "healthy"
		}

		backendID := backend.Backend.ID
		if len(backendID) > 8 {
			backendID = backendID[:8]
		}

		url := backend.Backend.URL
		if len(url) > 38 {
			url = url[:35] + "..."
		}

		fmt.Printf("%-10s %-12s %-12s %-40s %s\n",
			backendID,
			backend.Backend.Provider,
			backend.Backend.Region,
			url,
			statusText,
		)
	}

	fmt.Println()

	return nil
}

func printJSON(v interface{}) {
	data, _ := json.MarshalIndent(v, "", "  ")
	fmt.Println(string(data))
}

// Global variable for JSON flag
var asJSON bool
