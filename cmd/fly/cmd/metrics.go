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

// metricsCmd represents the metrics command
var metricsCmd = &cobra.Command{
	Use:   "metrics",
	Short: "Detailed performance metrics and analytics",
	Long: `Shows detailed performance metrics and analytics for deployed functions.

Displays latency percentiles, error rates, throughput, and other performance indicators.

Examples:
  fly metrics
  fly metrics --period=24h
  fly metrics --format=chart`,
	Run: metricsRun,
}

var metricsFlags struct {
	period     string
	format     string
	granularity string
	jsonOutput bool
}

func init() {
	rootCmd.AddCommand(metricsCmd)

	// Local flags
	metricsCmd.Flags().StringVarP(&metricsFlags.period, "period", "p", "24h", "Time period (1h, 24h, 7d, 30d)")
	metricsCmd.Flags().StringVarP(&metricsFlags.format, "format", "f", "table", "Output format (table, chart, json)")
	metricsCmd.Flags().StringVarP(&metricsFlags.granularity, "granularity", "g", "1h", "Data granularity (1m, 5m, 1h, 1d)")
	metricsCmd.Flags().BoolVarP(&metricsFlags.jsonOutput, "json", "j", false, "Output metrics in JSON format")
}

// metricsRun implements the metrics command
func metricsRun(cmd *cobra.Command, args []string) {
	fmt.Println("Fetching performance metrics...")

	// 1. Load manifest and credentials
	m, err := manifest.Load("")
	if err != nil {
		log.Fatalf("No functionfly.json found. Run 'fly init' first: %v", err)
	}

	creds, err := credentials.Load()
	if err != nil {
		log.Fatalf("Not logged in. Run 'fly login' first: %v", err)
	}

	// 2. Validate period
	validPeriods := map[string]bool{"1h": true, "24h": true, "7d": true, "30d": true}
	if !validPeriods[metricsFlags.period] {
		log.Fatalf("Invalid period '%s'. Use: 1h, 24h, 7d, or 30d", metricsFlags.period)
	}

	// 3. Create API client
	apiURL := getAPIURL()
	client := cli.NewClient(apiURL, creds.Token)

	// 4. Get detailed metrics
	metrics, err := getDetailedMetrics(client, creds.User.Username, m.Name, metricsFlags.period)
	if err != nil {
		log.Fatalf("Failed to get metrics: %v", err)
	}

	// 5. Output metrics
	if metricsFlags.jsonOutput || metricsFlags.format == "json" {
		outputMetricsJSON(metrics)
	} else if metricsFlags.format == "chart" {
		outputMetricsChart(metrics)
	} else {
		outputMetricsTable(metrics, creds.User.Username, m.Name)
	}
}

// getDetailedMetrics retrieves detailed performance metrics
func getDetailedMetrics(client *cli.Client, author, name, period string) (*DetailedMetrics, error) {
	// In a real implementation, this would call the metrics API
	// For now, return mock data
	return createMockDetailedMetrics(author, name, period), nil
}

// DetailedMetrics represents comprehensive performance metrics
type DetailedMetrics struct {
	FunctionID       string               `json:"function_id"`
	Name             string               `json:"name"`
	Author           string               `json:"author"`
	Period           string               `json:"period"`
	TotalRequests    int64                `json:"total_requests"`
	SuccessfulReqs   int64                `json:"successful_requests"`
	FailedReqs       int64                `json:"failed_requests"`
	ErrorRate        float64              `json:"error_rate"`
	AvgLatencyMs     float64              `json:"avg_latency_ms"`
	P50LatencyMs     float64              `json:"p50_latency_ms"`
	P95LatencyMs     float64              `json:"p95_latency_ms"`
	P99LatencyMs     float64              `json:"p99_latency_ms"`
	MinLatencyMs     float64              `json:"min_latency_ms"`
	MaxLatencyMs     float64              `json:"max_latency_ms"`
	RequestsPerSec   float64              `json:"requests_per_sec"`
	DataTransferred  float64              `json:"data_transferred_mb"`
	TopErrors        []ErrorCount         `json:"top_errors"`
	StatusCodes      map[int]int64        `json:"status_codes"`
	RegionalStats    []RegionalStats      `json:"regional_stats"`
	TimeSeries       []TimeSeriesPoint    `json:"time_series"`
}

// ErrorCount represents error frequency
type ErrorCount struct {
	Error   string `json:"error"`
	Count   int64  `json:"count"`
	Percent float64 `json:"percent"`
}

// RegionalStats represents per-region statistics
type RegionalStats struct {
	Region          string  `json:"region"`
	Requests        int64   `json:"requests"`
	AvgLatencyMs    float64 `json:"avg_latency_ms"`
	ErrorRate       float64 `json:"error_rate"`
}

// TimeSeriesPoint represents a data point in time series
type TimeSeriesPoint struct {
	Timestamp   time.Time `json:"timestamp"`
	Requests    int64     `json:"requests"`
	AvgLatency  float64   `json:"avg_latency_ms"`
	ErrorRate   float64   `json:"error_rate"`
}

// createMockDetailedMetrics creates mock detailed metrics
func createMockDetailedMetrics(author, name, period string) *DetailedMetrics {
	var totalRequests int64
	var duration time.Duration

	switch period {
	case "1h":
		totalRequests = 3600
		duration = 1 * time.Hour
	case "24h":
		totalRequests = 86400
		duration = 24 * time.Hour
	case "7d":
		totalRequests = 604800
		duration = 7 * 24 * time.Hour
	case "30d":
		totalRequests = 2592000
		duration = 30 * 24 * time.Hour
	}

	successfulReqs := int64(float64(totalRequests) * 0.997)
	failedReqs := totalRequests - successfulReqs
	errorRate := float64(failedReqs) / float64(totalRequests) * 100

	return &DetailedMetrics{
		FunctionID:     fmt.Sprintf("fn_%s_%s", author, name),
		Name:           name,
		Author:         author,
		Period:         period,
		TotalRequests:  totalRequests,
		SuccessfulReqs: successfulReqs,
		FailedReqs:     failedReqs,
		ErrorRate:      errorRate,
		AvgLatencyMs:   42.5,
		P50LatencyMs:   35.0,
		P95LatencyMs:   120.0,
		P99LatencyMs:   250.0,
		MinLatencyMs:   12.0,
		MaxLatencyMs:   500.0,
		RequestsPerSec: float64(totalRequests) / duration.Seconds(),
		DataTransferred: float64(totalRequests) * 1.5, // 1.5MB per request avg
		TopErrors: []ErrorCount{
			{Error: "timeout", Count: 150, Percent: 0.15},
			{Error: "internal_error", Count: 85, Percent: 0.08},
			{Error: "bad_request", Count: 45, Percent: 0.04},
		},
		StatusCodes: map[int]int64{
			200: successfulReqs,
			400: 45,
			500: 235,
		},
		RegionalStats: []RegionalStats{
			{Region: "us-east-1", Requests: totalRequests * 60 / 100, AvgLatencyMs: 40.0, ErrorRate: 0.2},
			{Region: "us-west-2", Requests: totalRequests * 25 / 100, AvgLatencyMs: 45.0, ErrorRate: 0.3},
			{Region: "eu-west-1", Requests: totalRequests * 15 / 100, AvgLatencyMs: 80.0, ErrorRate: 0.25},
		},
		TimeSeries: createMockTimeSeries(period),
	}
}

// createMockTimeSeries creates mock time series data
func createMockTimeSeries(period string) []TimeSeriesPoint {
	var points []TimeSeriesPoint
	now := time.Now()
	var interval time.Duration
	var count int

	switch period {
	case "1h":
		interval = 5 * time.Minute
		count = 12
	case "24h":
		interval = 1 * time.Hour
		count = 24
	case "7d":
		interval = 24 * time.Hour
		count = 7
	case "30d":
		interval = 24 * time.Hour
		count = 30
	}

	for i := count - 1; i >= 0; i-- {
		timestamp := now.Add(-time.Duration(i) * interval)
		requests := int64(100 + (i * 10)) // Trending upward
		avgLatency := 40.0 + float64(i)   // Slight increase
		errorRate := 0.2 + float64(i)*0.01 // Slight increase

		points = append(points, TimeSeriesPoint{
			Timestamp:  timestamp,
			Requests:   requests,
			AvgLatency: avgLatency,
			ErrorRate:  errorRate,
		})
	}

	return points
}

// outputMetricsTable prints metrics in table format
func outputMetricsTable(metrics *DetailedMetrics, author, name string) {
	fmt.Printf("\nPerformance Metrics for %s/%s (%s):\n", author, name, metrics.Period)
	fmt.Println("=====================================")

	fmt.Printf("Requests:     %s total\n", formatNumber(metrics.TotalRequests))
	fmt.Printf("Success:      %s (%.2f%%)\n", formatNumber(metrics.SuccessfulReqs), 100-metrics.ErrorRate)
	fmt.Printf("Errors:       %s (%.2f%%)\n", formatNumber(metrics.FailedReqs), metrics.ErrorRate)
	fmt.Printf("Throughput:   %.2f req/sec\n\n", metrics.RequestsPerSec)

	fmt.Println("Latency (ms):")
	fmt.Printf("  Average:    %.2f\n", metrics.AvgLatencyMs)
	fmt.Printf("  P50:        %.2f\n", metrics.P50LatencyMs)
	fmt.Printf("  P95:        %.2f\n", metrics.P95LatencyMs)
	fmt.Printf("  P99:        %.2f\n", metrics.P99LatencyMs)
	fmt.Printf("  Min:        %.2f\n", metrics.MinLatencyMs)
	fmt.Printf("  Max:        %.2f\n\n", metrics.MaxLatencyMs)

	fmt.Printf("Data Transfer: %.2f MB\n\n", metrics.DataTransferred)

	if len(metrics.TopErrors) > 0 {
		fmt.Println("Top Errors:")
		for _, err := range metrics.TopErrors {
			fmt.Printf("  %s: %s (%.2f%%)\n", err.Error, formatNumber(err.Count), err.Percent)
		}
		fmt.Println()
	}

	if len(metrics.RegionalStats) > 0 {
		fmt.Println("Regional Performance:")
		for _, region := range metrics.RegionalStats {
			fmt.Printf("  %s: %s req, %.1fms avg, %.2f%% errors\n",
				region.Region, formatNumber(region.Requests), region.AvgLatencyMs, region.ErrorRate)
		}
	}
}

// outputMetricsChart prints metrics in chart format
func outputMetricsChart(metrics *DetailedMetrics) {
	fmt.Printf("\nPerformance Chart for %s (%s):\n", metrics.Name, metrics.Period)
	fmt.Println("================================")

	// Simple ASCII chart for time series
	if len(metrics.TimeSeries) > 0 {
		fmt.Println("\nRequests Over Time:")
		fmt.Println("Time                Requests    Latency(ms)    Error%")
		fmt.Println("----------------------------------------------------")

		for _, point := range metrics.TimeSeries {
			timeStr := point.Timestamp.Format("15:04")
			fmt.Printf("%s           %8s      %8.1f       %5.2f%%\n",
				timeStr, formatNumber(point.Requests), point.AvgLatency, point.ErrorRate)
		}
	}

	// Latency distribution chart
	fmt.Println("\nLatency Distribution:")
	fmt.Printf("  P99: %s\n", createLatencyBar(metrics.P99LatencyMs, metrics.MaxLatencyMs))
	fmt.Printf("  P95: %s\n", createLatencyBar(metrics.P95LatencyMs, metrics.MaxLatencyMs))
	fmt.Printf("  P50: %s\n", createLatencyBar(metrics.P50LatencyMs, metrics.MaxLatencyMs))
	fmt.Printf("  Avg: %s\n", createLatencyBar(metrics.AvgLatencyMs, metrics.MaxLatencyMs))
}

// createLatencyBar creates a simple ASCII bar for latency visualization
func createLatencyBar(latency, maxLatency float64) string {
	barWidth := int((latency / maxLatency) * 20)
	bar := ""
	for i := 0; i < barWidth; i++ {
		bar += "█"
	}
	return fmt.Sprintf("%6.1fms %s", latency, bar)
}

// outputMetricsJSON prints metrics in JSON format
func outputMetricsJSON(metrics *DetailedMetrics) {
	fmt.Printf(`{
  "function_id": %q,
  "name": %q,
  "author": %q,
  "period": %q,
  "requests": {
    "total": %d,
    "successful": %d,
    "failed": %d,
    "error_rate": %.4f,
    "per_second": %.2f
  },
  "latency_ms": {
    "avg": %.2f,
    "p50": %.2f,
    "p95": %.2f,
    "p99": %.2f,
    "min": %.2f,
    "max": %.2f
  },
  "data_transferred_mb": %.2f,
  "top_errors": [`, metrics.FunctionID, metrics.Name, metrics.Author, metrics.Period,
		metrics.TotalRequests, metrics.SuccessfulReqs, metrics.FailedReqs,
		metrics.ErrorRate, metrics.RequestsPerSec,
		metrics.AvgLatencyMs, metrics.P50LatencyMs, metrics.P95LatencyMs, metrics.P99LatencyMs,
		metrics.MinLatencyMs, metrics.MaxLatencyMs, metrics.DataTransferred)

	for i, err := range metrics.TopErrors {
		fmt.Printf(`
    {
      "error": %q,
      "count": %d,
      "percent": %.4f
    }`, err.Error, err.Count, err.Percent)

		if i < len(metrics.TopErrors)-1 {
			fmt.Printf(",")
		}
	}

	fmt.Printf(`
  ],
  "regional_stats": [`)

	for i, region := range metrics.RegionalStats {
		fmt.Printf(`
    {
      "region": %q,
      "requests": %d,
      "avg_latency_ms": %.2f,
      "error_rate": %.2f
    }`, region.Region, region.Requests, region.AvgLatencyMs, region.ErrorRate)

		if i < len(metrics.RegionalStats)-1 {
			fmt.Printf(",")
		}
	}

	fmt.Printf(`
  ]
}`)
}