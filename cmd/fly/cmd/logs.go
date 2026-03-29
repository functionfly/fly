/*
Copyright © 2026 FunctionFly

*/
package cmd

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/functionfly/fly/internal/cli"
	"github.com/functionfly/fly/internal/credentials"
	"github.com/functionfly/fly/internal/manifest"
	"github.com/spf13/cobra"
)

// logsCmd represents the logs command
var logsCmd = &cobra.Command{
	Use:   "logs",
	Short: "View function execution logs with filtering",
	Long: `Views real-time execution logs for deployed functions.

Shows function invocations, errors, performance metrics, and system events with advanced filtering.

Examples:
  fly logs
  fly logs --tail --filter=error
  fly logs --since=1h --level=warn
  fly logs --request-id=abc123`,
	Run: logsRun,
}

var logsFlags struct {
	tail      bool
	filter    string
	level     string
	since     time.Duration
	requestID string
	limit     int
	jsonOutput bool
}

func init() {
	rootCmd.AddCommand(logsCmd)

	// Local flags
	logsCmd.Flags().BoolVarP(&logsFlags.tail, "tail", "t", false, "Tail logs continuously")
	logsCmd.Flags().StringVarP(&logsFlags.filter, "filter", "f", "", "Filter logs by content")
	logsCmd.Flags().StringVarP(&logsFlags.level, "level", "l", "", "Filter by log level (debug, info, warn, error)")
	logsCmd.Flags().DurationVarP(&logsFlags.since, "since", "s", 1*time.Hour, "Show logs since duration")
	logsCmd.Flags().StringVarP(&logsFlags.requestID, "request-id", "r", "", "Show logs for specific request ID")
	logsCmd.Flags().IntVarP(&logsFlags.limit, "limit", "n", 100, "Maximum number of logs to show")
	logsCmd.Flags().BoolVarP(&logsFlags.jsonOutput, "json", "j", false, "Output logs in JSON format")
}

// logsRun implements the logs command
func logsRun(cmd *cobra.Command, args []string) {
	if logsFlags.tail {
		fmt.Println("Tailing function logs...")
	} else {
		fmt.Println("Fetching function logs...")
	}

	// 1. Load manifest and credentials
	m, err := manifest.Load("")
	if err != nil {
		log.Fatalf("No functionfly.json found. Run 'fly init' first: %v", err)
	}

	creds, err := credentials.Load()
	if err != nil {
		log.Fatalf("Not logged in. Run 'fly login' first: %v", err)
	}

	// 2. Create API client
	apiURL := getAPIURL()
	client := cli.NewClient(apiURL, creds.Token)

	// 3. Get function logs
	logs, err := getFunctionLogs(client, creds.User.Username, m.Name, logsFlags.filter, logsFlags.level, logsFlags.since, logsFlags.requestID, logsFlags.limit)
	if err != nil {
		log.Fatalf("Failed to get function logs: %v", err)
	}

	// 4. Output logs
	if logsFlags.jsonOutput {
		outputFunctionLogsJSON(logs)
	} else {
		outputFunctionLogsHuman(logs, creds.User.Username, m.Name)
	}

	// 5. Tail mode
	if logsFlags.tail {
		tailFunctionLogs(client, creds.User.Username, m.Name, logsFlags.filter, logsFlags.level, logsFlags.requestID)
	}
}

// getFunctionLogs retrieves function execution logs
func getFunctionLogs(client *cli.Client, author, name, filter, level string, since time.Duration, requestID string, limit int) ([]*FunctionLogEntry, error) {
	// In a real implementation, this would call the logs API
	// For now, return mock data
	logs := createMockFunctionLogs(author, name, limit)

	// Apply filtering
	filtered := []*FunctionLogEntry{}
	for _, log := range logs {
		if matchesLogFilters(log, filter, level, since, requestID) {
			filtered = append(filtered, log)
		}
	}

	return filtered, nil
}

// tailFunctionLogs continuously tails function logs
func tailFunctionLogs(client *cli.Client, author, name, filter, level, requestID string) {
	fmt.Println("\nTailing logs (Ctrl+C to stop)...")

	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()

	lastLogTime := time.Now()

	for {
		select {
		case <-ticker.C:
			// Get new logs since last check
			logs, err := getFunctionLogs(client, author, name, filter, level, time.Since(lastLogTime), requestID, 50)
			if err == nil && len(logs) > 0 {
				for _, log := range logs {
					outputFunctionLogEntryHuman(log)
				}
				lastLogTime = time.Now()
			}
		}
	}
}

// FunctionLogEntry represents a function execution log entry
type FunctionLogEntry struct {
	Timestamp   time.Time `json:"timestamp"`
	Level       string    `json:"level"`
	Message     string    `json:"message"`
	RequestID   string    `json:"request_id"`
	StatusCode  int       `json:"status_code,omitempty"`
	LatencyMs   int       `json:"latency_ms,omitempty"`
	Region      string    `json:"region,omitempty"`
	UserAgent   string    `json:"user_agent,omitempty"`
	IP          string    `json:"ip,omitempty"`
}

// matchesLogFilters checks if a log entry matches the filters
func matchesLogFilters(log *FunctionLogEntry, filter, level string, since time.Duration, requestID string) bool {
	// Time filter
	if since > 0 && time.Since(log.Timestamp) > since {
		return false
	}

	// Request ID filter
	if requestID != "" && log.RequestID != requestID {
		return false
	}

	// Level filter
	if level != "" && log.Level != level {
		return false
	}

	// Content filter
	if filter != "" {
		content := fmt.Sprintf("%s %s %s", log.Level, log.Message, log.RequestID)
		if !strings.Contains(strings.ToLower(content), strings.ToLower(filter)) {
			return false
		}
	}

	return true
}

// createMockFunctionLogs creates mock function execution logs
func createMockFunctionLogs(author, name string, count int) []*FunctionLogEntry {
	logs := []*FunctionLogEntry{}
	now := time.Now()

	// Generate mock log entries
	entries := []struct {
		level      string
		message    string
		requestID  string
		statusCode int
		latencyMs  int
		region     string
		offset     time.Duration
	}{
		{"info", "Function invoked", "req_123456", 200, 45, "us-east-1", 5 * time.Minute},
		{"info", "Processing input: Hello World", "req_123456", 0, 0, "us-east-1", 5 * time.Minute},
		{"info", "Function completed successfully", "req_123456", 0, 0, "us-east-1", 5 * time.Minute},
		{"info", "Function invoked", "req_123455", 200, 38, "us-east-1", 4 * time.Minute},
		{"info", "Processing input: Test Input", "req_123455", 0, 0, "us-east-1", 4 * time.Minute},
		{"info", "Function completed successfully", "req_123455", 0, 0, "us-east-1", 4 * time.Minute},
		{"warn", "High latency detected: 150ms", "req_123454", 200, 150, "us-east-1", 3 * time.Minute},
		{"info", "Function invoked", "req_123454", 200, 150, "us-east-1", 3 * time.Minute},
		{"error", "Function execution failed: timeout", "req_123453", 500, 30000, "us-east-1", 2 * time.Minute},
		{"info", "Function invoked", "req_123452", 200, 42, "us-east-1", 1 * time.Minute},
		{"info", "Processing input: Another test", "req_123452", 0, 0, "us-east-1", 1 * time.Minute},
		{"info", "Function completed successfully", "req_123452", 0, 0, "us-east-1", 1 * time.Minute},
	}

	// Limit to requested count
	if count < len(entries) {
		entries = entries[len(entries)-count:]
	}

	for _, entry := range entries {
		logs = append(logs, &FunctionLogEntry{
			Timestamp:   now.Add(-entry.offset),
			Level:       entry.level,
			Message:     entry.message,
			RequestID:   entry.requestID,
			StatusCode:  entry.statusCode,
			LatencyMs:   entry.latencyMs,
			Region:      entry.region,
		})
	}

	return logs
}

// outputFunctionLogsHuman prints function logs in human-readable format
func outputFunctionLogsHuman(logs []*FunctionLogEntry, author, name string) {
	fmt.Printf("\nFunction Logs for %s/%s:\n", author, name)
	fmt.Println("========================")

	if len(logs) == 0 {
		fmt.Println("No logs found")
		return
	}

	for _, log := range logs {
		outputFunctionLogEntryHuman(log)
	}
}

// outputFunctionLogEntryHuman prints a single function log entry
func outputFunctionLogEntryHuman(log *FunctionLogEntry) {
	timestamp := log.Timestamp.Format("2006-01-02 15:04:05")
	level := fmt.Sprintf("[%s]", log.Level)

	// Colorize based on level
	switch log.Level {
	case "error":
		fmt.Printf("\033[31m%s %s\033[0m ", timestamp, level)
	case "warn":
		fmt.Printf("\033[33m%s %s\033[0m ", timestamp, level)
	case "info":
		fmt.Printf("\033[36m%s %s\033[0m ", timestamp, level)
	default:
		fmt.Printf("%s %s ", timestamp, level)
	}

	// Show request ID
	if log.RequestID != "" {
		fmt.Printf("[%s] ", log.RequestID[:8])
	}

	// Show message
	fmt.Printf("%s", log.Message)

	// Show additional metadata
	if log.StatusCode > 0 {
		fmt.Printf(" (status: %d)", log.StatusCode)
	}
	if log.LatencyMs > 0 {
		fmt.Printf(" (latency: %dms)", log.LatencyMs)
	}
	if log.Region != "" {
		fmt.Printf(" (region: %s)", log.Region)
	}

	fmt.Println()
}

// outputFunctionLogsJSON prints function logs in JSON format
func outputFunctionLogsJSON(logs []*FunctionLogEntry) {
	fmt.Println("[")

	for i, log := range logs {
		fmt.Printf(`  {
    "timestamp": %q,
    "level": %q,
    "message": %q,
    "request_id": %q,
    "status_code": %d,
    "latency_ms": %d,
    "region": %q
  }`, log.Timestamp.Format(time.RFC3339), log.Level, log.Message, log.RequestID,
			log.StatusCode, log.LatencyMs, log.Region)

		if i < len(logs)-1 {
			fmt.Println(",")
		} else {
			fmt.Println()
		}
	}

	fmt.Println("]")
}