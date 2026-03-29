/*
Copyright © 2026 FunctionFly

*/
package cmd

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/functionfly/fly/internal/manifest"
	"github.com/spf13/cobra"
)

// deployLogsCmd represents the deploy logs command
var deployLogsCmd = &cobra.Command{
	Use:   "logs",
	Short: "View deployment logs and events",
	Long: `Views logs and events for deployments.

Shows deployment logs, build output, and system events with filtering and search capabilities.

Examples:
  fly deploy logs
  fly deploy logs --deployment-id=abc123
  fly deploy logs --tail --filter=error`,
	Run: deployLogsRun,
}

var deployLogsFlags struct {
	deploymentID string
	tail         bool
	filter       string
	since        time.Duration
	lines        int
	jsonOutput   bool
}

func init() {
	deployCmd.AddCommand(deployLogsCmd)

	// Local flags
	deployLogsCmd.Flags().StringVarP(&deployLogsFlags.deploymentID, "deployment-id", "d", "", "Specific deployment ID to view logs for")
	deployLogsCmd.Flags().BoolVarP(&deployLogsFlags.tail, "tail", "t", false, "Tail logs continuously")
	deployLogsCmd.Flags().StringVarP(&deployLogsFlags.filter, "filter", "f", "", "Filter logs by level (info, warn, error) or content")
	deployLogsCmd.Flags().DurationVarP(&deployLogsFlags.since, "since", "s", 1*time.Hour, "Show logs since duration")
	deployLogsCmd.Flags().IntVarP(&deployLogsFlags.lines, "lines", "n", 100, "Number of lines to show")
	deployLogsCmd.Flags().BoolVarP(&deployLogsFlags.jsonOutput, "json", "j", false, "Output logs in JSON format")
}

// deployLogsRun implements the deploy logs command
func deployLogsRun(cmd *cobra.Command, args []string) {
	if deployLogsFlags.tail {
		fmt.Println("Tailing deployment logs...")
	} else {
		fmt.Println("Fetching deployment logs...")
	}

	// 1. Load manifest
	m, err := manifest.Load("")
	if err != nil {
		log.Fatalf("No functionfly.json found. Run 'fly init' first: %v", err)
	}

	// 2. Get deployment logs
	logs, err := getDeploymentLogs(m.Name, deployLogsFlags.deploymentID, deployLogsFlags.filter, deployLogsFlags.since, deployLogsFlags.lines)
	if err != nil {
		log.Fatalf("Failed to get deployment logs: %v", err)
	}

	// 3. Output logs
	if deployLogsFlags.jsonOutput {
		outputLogsJSON(logs)
	} else {
		outputLogsHuman(logs)
	}

	// 4. Tail mode
	if deployLogsFlags.tail {
		tailDeploymentLogs(m.Name, deployLogsFlags.deploymentID, deployLogsFlags.filter)
	}
}

// getDeploymentLogs retrieves deployment logs
func getDeploymentLogs(functionName, deploymentID, filter string, since time.Duration, lines int) ([]*LogEntry, error) {
	// Mock logs for demonstration
	logs := createMockDeploymentLogs(functionName, deploymentID, lines)

	// Apply filtering
	if filter != "" {
		filtered := []*LogEntry{}
		for _, log := range logs {
			if matchesFilter(log, filter) {
				filtered = append(filtered, log)
			}
		}
		logs = filtered
	}

	// Apply time filtering
	if since > 0 {
		cutoff := time.Now().Add(-since)
		filtered := []*LogEntry{}
		for _, log := range logs {
			if log.Timestamp.After(cutoff) {
				filtered = append(filtered, log)
			}
		}
		logs = filtered
	}

	return logs, nil
}

// tailDeploymentLogs continuously tails logs
func tailDeploymentLogs(functionName, deploymentID, filter string) {
	fmt.Println("\nTailing logs (Ctrl+C to stop)...")

	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	lastLogTime := time.Now()

	for {
		select {
		case <-ticker.C:
			// Get new logs since last check
			logs, err := getDeploymentLogs(functionName, deploymentID, filter, time.Since(lastLogTime), 50)
			if err == nil && len(logs) > 0 {
				for _, log := range logs {
					outputLogEntryHuman(log)
				}
				lastLogTime = time.Now()
			}
		}
	}
}

// LogEntry represents a log entry
type LogEntry struct {
	Timestamp time.Time `json:"timestamp"`
	Level     string    `json:"level"`
	Message   string    `json:"message"`
	Source    string    `json:"source"`
	DeploymentID string `json:"deployment_id,omitempty"`
}

// matchesFilter checks if a log entry matches the filter
func matchesFilter(log *LogEntry, filter string) bool {
	switch filter {
	case "info", "warn", "error":
		return log.Level == filter
	default:
		// Check if filter string appears in message
		return strings.Contains(fmt.Sprintf("%s %s %s", log.Level, log.Source, log.Message), filter)
	}
}

// createMockDeploymentLogs creates mock deployment logs
func createMockDeploymentLogs(functionName, deploymentID string, count int) []*LogEntry {
	logs := []*LogEntry{}
	now := time.Now()

	// Generate mock log entries
	entries := []struct {
		level  string
		source string
		message string
		offset  time.Duration
	}{
		{"info", "deployer", "Starting deployment", 10 * time.Minute},
		{"info", "bundler", "Bundling function code", 9 * time.Minute},
		{"info", "bundler", "Code bundled successfully (2.1KB)", 9 * time.Minute},
		{"info", "deployer", "Uploading artifact to registry", 8 * time.Minute},
		{"info", "deployer", "Artifact uploaded successfully", 7 * time.Minute},
		{"info", "deployer", "Deploying to edge network", 6 * time.Minute},
		{"info", "deployer", "Deployment completed successfully", 5 * time.Minute},
		{"info", "health-check", "Health check passed", 4 * time.Minute},
		{"info", "monitor", "Function is responding correctly", 3 * time.Minute},
		{"warn", "monitor", "High latency detected: 150ms", 2 * time.Minute},
		{"info", "monitor", "Function performance is normal", 1 * time.Minute},
	}

	// Limit to requested count
	if count < len(entries) {
		entries = entries[len(entries)-count:]
	}

	for _, entry := range entries {
		logs = append(logs, &LogEntry{
			Timestamp:    now.Add(-entry.offset),
			Level:        entry.level,
			Message:      entry.message,
			Source:       entry.source,
			DeploymentID: deploymentID,
		})
	}

	return logs
}

// outputLogsHuman prints logs in human-readable format
func outputLogsHuman(logs []*LogEntry) {
	if len(logs) == 0 {
		fmt.Println("No logs found")
		return
	}

	fmt.Println("\nDeployment Logs:")
	fmt.Println("================")

	for _, log := range logs {
		outputLogEntryHuman(log)
	}
}

// outputLogEntryHuman prints a single log entry
func outputLogEntryHuman(log *LogEntry) {
	timestamp := log.Timestamp.Format("2006-01-02 15:04:05")
	level := fmt.Sprintf("[%s]", log.Level)

	// Colorize based on level
	switch log.Level {
	case "error":
		fmt.Printf("\033[31m%s %s %s: %s\033[0m\n", timestamp, level, log.Source, log.Message)
	case "warn":
		fmt.Printf("\033[33m%s %s %s: %s\033[0m\n", timestamp, level, log.Source, log.Message)
	case "info":
		fmt.Printf("\033[36m%s %s %s: %s\033[0m\n", timestamp, level, log.Source, log.Message)
	default:
		fmt.Printf("%s %s %s: %s\n", timestamp, level, log.Source, log.Message)
	}
}

// outputLogsJSON prints logs in JSON format
func outputLogsJSON(logs []*LogEntry) {
	fmt.Println("[")

	for i, log := range logs {
		fmt.Printf(`  {
    "timestamp": %q,
    "level": %q,
    "source": %q,
    "message": %q,
    "deployment_id": %q
  }`, log.Timestamp.Format(time.RFC3339), log.Level, log.Source, log.Message, log.DeploymentID)

		if i < len(logs)-1 {
			fmt.Println(",")
		} else {
			fmt.Println()
		}
	}

	fmt.Println("]")
}