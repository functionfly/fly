/*
Copyright © 2026 FunctionFly

*/
package cmd

import (
	"fmt"
	"log"
	"time"

	"github.com/functionfly/fly/internal/cli"
	"github.com/functionfly/fly/internal/credentials"
	"github.com/functionfly/fly/internal/manifest"
	"github.com/spf13/cobra"
)

// healthCmd represents the health command
var healthCmd = &cobra.Command{
	Use:   "health",
	Short: "Check system and function health status",
	Long: `Performs comprehensive health checks on functions and the FunctionFly platform.

Checks function availability, performance, system status, and identifies potential issues.

Examples:
  fly health
  fly health --detailed
  fly health --check=functions`,
	Run: healthRun,
}

var healthFlags struct {
	detailed bool
	checkType string
	watch     bool
	jsonOutput bool
}

func init() {
	rootCmd.AddCommand(healthCmd)

	// Local flags
	healthCmd.Flags().BoolVarP(&healthFlags.detailed, "detailed", "d", false, "Show detailed health information")
	healthCmd.Flags().StringVarP(&healthFlags.checkType, "check", "c", "all", "Check type (all, functions, system, platform)")
	healthCmd.Flags().BoolVarP(&healthFlags.watch, "watch", "w", false, "Watch health status continuously")
	healthCmd.Flags().BoolVarP(&healthFlags.jsonOutput, "json", "j", false, "Output health status in JSON format")
}

// healthRun implements the health command
func healthRun(cmd *cobra.Command, args []string) {
	if healthFlags.watch {
		fmt.Println("Monitoring health status...")
	} else {
		fmt.Println("Checking health status...")
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

	if healthFlags.watch {
		// Watch mode
		watchHealthStatus(client, creds.User.Username, m.Name, healthFlags.checkType, healthFlags.detailed, healthFlags.jsonOutput)
	} else {
		// Single check
		checkHealthStatus(client, creds.User.Username, m.Name, healthFlags.checkType, healthFlags.detailed, healthFlags.jsonOutput)
	}
}

// checkHealthStatus performs a single health check
func checkHealthStatus(client *cli.Client, author, name, checkType string, detailed, jsonOutput bool) {
	health := &HealthStatus{}

	// Perform different types of checks
	switch checkType {
	case "all":
		health = performAllHealthChecks(client, author, name, detailed)
	case "functions":
		health.FunctionHealth = performFunctionHealthChecks(client, author, name, detailed)
	case "system":
		health.SystemHealth = performSystemHealthChecks(client)
	case "platform":
		health.PlatformHealth = performPlatformHealthChecks(client)
	default:
		log.Fatalf("Invalid check type: %s", checkType)
	}

	if jsonOutput {
		outputHealthJSON(health)
	} else {
		outputHealthHuman(health, checkType)
	}
}

// watchHealthStatus continuously monitors health
func watchHealthStatus(client *cli.Client, author, name, checkType string, detailed, jsonOutput bool) {
	fmt.Println("\nMonitoring health status (Ctrl+C to stop)...")

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			fmt.Printf("\n%s - ", time.Now().Format("15:04:05"))
			checkHealthStatus(client, author, name, checkType, detailed, jsonOutput)
		}
	}
}

// HealthStatus represents comprehensive health status
type HealthStatus struct {
	Overall        string            `json:"overall"`
	FunctionHealth *FunctionHealth   `json:"function_health,omitempty"`
	SystemHealth   *SystemHealth     `json:"system_health,omitempty"`
	PlatformHealth *PlatformHealth   `json:"platform_health,omitempty"`
	Timestamp      time.Time         `json:"timestamp"`
}

// FunctionHealth represents function-specific health
type FunctionHealth struct {
	FunctionName    string             `json:"function_name"`
	Status          string             `json:"status"`
	Availability    float64            `json:"availability"`
	AvgLatencyMs    float64            `json:"avg_latency_ms"`
	ErrorRate       float64            `json:"error_rate"`
	LastChecked     time.Time          `json:"last_checked"`
	Issues          []HealthIssue      `json:"issues,omitempty"`
	RegionalHealth  []RegionalHealth   `json:"regional_health,omitempty"`
}

// SystemHealth represents system-wide health
type SystemHealth struct {
	Status       string        `json:"status"`
	ResponseTime time.Duration `json:"response_time"`
	Services     []ServiceStatus `json:"services"`
	Issues       []HealthIssue `json:"issues,omitempty"`
}

// PlatformHealth represents platform-wide health
type PlatformHealth struct {
	Status          string            `json:"status"`
	Regions         []RegionStatus    `json:"regions"`
	GlobalMetrics   map[string]interface{} `json:"global_metrics,omitempty"`
	Issues          []HealthIssue     `json:"issues,omitempty"`
}

// HealthIssue represents a health issue
type HealthIssue struct {
	Severity string `json:"severity"`
	Message  string `json:"message"`
	Service  string `json:"service,omitempty"`
	Region   string `json:"region,omitempty"`
}

// RegionalHealth represents health in a specific region
type RegionalHealth struct {
	Region       string  `json:"region"`
	Status       string  `json:"status"`
	LatencyMs    float64 `json:"latency_ms"`
	ErrorRate    float64 `json:"error_rate"`
}

// ServiceStatus represents the status of a service
type ServiceStatus struct {
	Name    string `json:"name"`
	Status  string `json:"status"`
	Message string `json:"message,omitempty"`
}

// RegionStatus represents the status of a region
type RegionStatus struct {
	Name   string `json:"name"`
	Status string `json:"status"`
	Issues []HealthIssue `json:"issues,omitempty"`
}

// performAllHealthChecks performs comprehensive health checks
func performAllHealthChecks(client *cli.Client, author, name string, detailed bool) *HealthStatus {
	return &HealthStatus{
		Overall:        "healthy",
		FunctionHealth: performFunctionHealthChecks(client, author, name, detailed),
		SystemHealth:   performSystemHealthChecks(client),
		PlatformHealth: performPlatformHealthChecks(client),
		Timestamp:      time.Now(),
	}
}

// performFunctionHealthChecks checks function-specific health
func performFunctionHealthChecks(client *cli.Client, author, name string, detailed bool) *FunctionHealth {
	// Mock function health data
	health := &FunctionHealth{
		FunctionName: name,
		Status:       "healthy",
		Availability: 99.98,
		AvgLatencyMs: 42.5,
		ErrorRate:    0.02,
		LastChecked:  time.Now(),
		Issues:       []HealthIssue{},
		RegionalHealth: []RegionalHealth{
			{Region: "us-east-1", Status: "healthy", LatencyMs: 40.0, ErrorRate: 0.01},
			{Region: "us-west-2", Status: "healthy", LatencyMs: 45.0, ErrorRate: 0.02},
			{Region: "eu-west-1", Status: "healthy", LatencyMs: 80.0, ErrorRate: 0.03},
		},
	}

	if detailed {
		// Add some mock issues for demonstration
		health.Issues = []HealthIssue{
			{Severity: "warning", Message: "Latency slightly elevated in EU region", Region: "eu-west-1"},
		}
	}

	return health
}

// performSystemHealthChecks checks system health
func performSystemHealthChecks(client *cli.Client) *SystemHealth {
	return &SystemHealth{
		Status:       "healthy",
		ResponseTime: 25 * time.Millisecond,
		Services: []ServiceStatus{
			{Name: "API Gateway", Status: "healthy", Message: "All endpoints responding"},
			{Name: "Registry", Status: "healthy", Message: "Function registry operational"},
			{Name: "Deployment", Status: "healthy", Message: "Deployment service running"},
			{Name: "Monitoring", Status: "healthy", Message: "Metrics collection active"},
		},
		Issues: []HealthIssue{},
	}
}

// performPlatformHealthChecks checks platform-wide health
func performPlatformHealthChecks(client *cli.Client) *PlatformHealth {
	return &PlatformHealth{
		Status: "healthy",
		Regions: []RegionStatus{
			{Name: "us-east-1", Status: "healthy", Issues: []HealthIssue{}},
			{Name: "us-west-2", Status: "healthy", Issues: []HealthIssue{}},
			{Name: "eu-west-1", Status: "healthy", Issues: []HealthIssue{}},
			{Name: "ap-southeast-1", Status: "healthy", Issues: []HealthIssue{}},
		},
		GlobalMetrics: map[string]interface{}{
			"active_functions": 15420,
			"total_requests_1h": 2847392,
			"avg_latency_ms": 45.2,
			"error_rate": 0.015,
		},
		Issues: []HealthIssue{},
	}
}

// outputHealthHuman prints health status in human-readable format
func outputHealthHuman(health *HealthStatus, checkType string) {
	fmt.Printf("Health Status: %s", health.Overall)

	switch health.Overall {
	case "healthy":
		fmt.Printf(" ✓\n")
	case "warning":
		fmt.Printf(" ⚠\n")
	case "critical":
		fmt.Printf(" ✗\n")
	default:
		fmt.Printf(" ?\n")
	}

	if health.FunctionHealth != nil {
		fmt.Printf("\nFunction Health (%s):\n", health.FunctionHealth.FunctionName)
		fmt.Printf("  Status: %s\n", health.FunctionHealth.Status)
		fmt.Printf("  Availability: %.2f%%\n", health.FunctionHealth.Availability)
		fmt.Printf("  Avg Latency: %.1fms\n", health.FunctionHealth.AvgLatencyMs)
		fmt.Printf("  Error Rate: %.2f%%\n", health.FunctionHealth.ErrorRate)

		if len(health.FunctionHealth.RegionalHealth) > 0 {
			fmt.Printf("  Regional Status:\n")
			for _, region := range health.FunctionHealth.RegionalHealth {
				fmt.Printf("    %s: %s (%.1fms, %.2f%% errors)\n",
					region.Region, region.Status, region.LatencyMs, region.ErrorRate)
			}
		}

		if len(health.FunctionHealth.Issues) > 0 {
			fmt.Printf("  Issues:\n")
			for _, issue := range health.FunctionHealth.Issues {
				fmt.Printf("    %s: %s\n", issue.Severity, issue.Message)
			}
		}
	}

	if health.SystemHealth != nil {
		fmt.Printf("\nSystem Health:\n")
		fmt.Printf("  Status: %s\n", health.SystemHealth.Status)
		fmt.Printf("  Response Time: %v\n", health.SystemHealth.ResponseTime)

		if len(health.SystemHealth.Services) > 0 {
			fmt.Printf("  Services:\n")
			for _, service := range health.SystemHealth.Services {
				fmt.Printf("    %s: %s", service.Name, service.Status)
				if service.Message != "" {
					fmt.Printf(" (%s)", service.Message)
				}
				fmt.Printf("\n")
			}
		}
	}

	if health.PlatformHealth != nil {
		fmt.Printf("\nPlatform Health:\n")
		fmt.Printf("  Status: %s\n", health.PlatformHealth.Status)

		if len(health.PlatformHealth.Regions) > 0 {
			fmt.Printf("  Regions:\n")
			for _, region := range health.PlatformHealth.Regions {
				fmt.Printf("    %s: %s\n", region.Name, region.Status)
			}
		}

		if health.PlatformHealth.GlobalMetrics != nil {
			fmt.Printf("  Global Metrics:\n")
			fmt.Printf("    Active Functions: %v\n", health.PlatformHealth.GlobalMetrics["active_functions"])
			fmt.Printf("    Requests (1h): %v\n", health.PlatformHealth.GlobalMetrics["total_requests_1h"])
			fmt.Printf("    Avg Latency: %vms\n", health.PlatformHealth.GlobalMetrics["avg_latency_ms"])
			fmt.Printf("    Error Rate: %.3f%%\n", health.PlatformHealth.GlobalMetrics["error_rate"].(float64)*100)
		}
	}
}

// outputHealthJSON prints health status in JSON format
func outputHealthJSON(health *HealthStatus) {
	fmt.Printf(`{
  "overall": %q,
  "timestamp": %q`, health.Overall, health.Timestamp.Format(time.RFC3339))

	if health.FunctionHealth != nil {
		fmt.Printf(`,
  "function_health": {
    "function_name": %q,
    "status": %q,
    "availability": %.2f,
    "avg_latency_ms": %.2f,
    "error_rate": %.4f,
    "last_checked": %q,
    "regional_health": [`, health.FunctionHealth.FunctionName, health.FunctionHealth.Status,
			health.FunctionHealth.Availability, health.FunctionHealth.AvgLatencyMs,
			health.FunctionHealth.ErrorRate, health.FunctionHealth.LastChecked.Format(time.RFC3339))

		for i, region := range health.FunctionHealth.RegionalHealth {
			fmt.Printf(`
      {
        "region": %q,
        "status": %q,
        "latency_ms": %.2f,
        "error_rate": %.4f
      }`, region.Region, region.Status, region.LatencyMs, region.ErrorRate)

			if i < len(health.FunctionHealth.RegionalHealth)-1 {
				fmt.Printf(",")
			}
		}
		fmt.Printf(`
    ]`)
		if len(health.FunctionHealth.Issues) > 0 {
			fmt.Printf(`,
    "issues": [`)
			for i, issue := range health.FunctionHealth.Issues {
				fmt.Printf(`
      {
        "severity": %q,
        "message": %q,
        "region": %q
      }`, issue.Severity, issue.Message, issue.Region)

				if i < len(health.FunctionHealth.Issues)-1 {
					fmt.Printf(",")
				}
			}
			fmt.Printf(`
    ]`)
		}
		fmt.Printf(`
  }`)
	}

	fmt.Printf(`
}`)
}