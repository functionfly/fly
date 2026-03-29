package commands

import (
	"fmt"
	"math"
	"strings"

	"github.com/spf13/cobra"
)

func NewStatsCmd() *cobra.Command {
	var period string
	var asJSON bool
	cmd := &cobra.Command{
		Use:   "stats",
		Short: "View function usage statistics",
		Example: "  fly stats\n  fly stats --period 7d\n  fly stats --json",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runStats(period, asJSON)
		},
	}
	cmd.Flags().StringVar(&period, "period", "24h", "Time period (24h, 7d, 30d)")
	cmd.Flags().BoolVar(&asJSON, "json", false, "Output as JSON")
	return cmd
}

type StatsResponse struct {
	FunctionID   string  `json:"function_id"`
	Name         string  `json:"name"`
	Author       string  `json:"author"`
	TotalCalls   int64   `json:"total_calls"`
	SuccessRate  float64 `json:"success_rate"`
	AvgLatencyMs float64 `json:"avg_latency_ms"`
	Revenue      float64 `json:"revenue"`
	Period       string  `json:"period"`
	DailyCalls   []int64 `json:"daily_calls,omitempty"`
}

func runStats(period string, asJSON bool) error {
	manifest, err := LoadManifest("")
	if err != nil {
		return err
	}
	creds, err := LoadCredentials()
	if err != nil {
		return err
	}
	client, err := NewAPIClient()
	if err != nil {
		return err
	}
	var stats StatsResponse
	path := fmt.Sprintf("/v1/registry/%s/%s/stats?period=%s", creds.User.Username, manifest.Name, period)
	if err := client.Get(path, &stats); err != nil {
		return fmt.Errorf("could not fetch stats: %w", err)
	}
	if asJSON {
		printJSON(stats)
		return nil
	}
	fmt.Printf("📊 %s by %s\n\n", manifest.Name, creds.User.Username)
	fmt.Printf("Period: %s\n\n", period)
	fmt.Printf("Calls:        %s\n", formatNumber(stats.TotalCalls))
	fmt.Printf("Success rate: %.2f%%\n", stats.SuccessRate*100)
	fmt.Printf("Avg latency:  %.0fms\n", stats.AvgLatencyMs)
	if stats.Revenue > 0 {
		fmt.Printf("Revenue:      $%.2f\n", stats.Revenue)
	}
	if len(stats.DailyCalls) > 0 {
		fmt.Printf("\nLast %d days:\n", len(stats.DailyCalls))
		maxCalls := int64(0)
		for _, c := range stats.DailyCalls {
			if c > maxCalls {
				maxCalls = c
			}
		}
		for _, calls := range stats.DailyCalls {
			barLen := 0
			if maxCalls > 0 {
				barLen = int(math.Round(float64(calls) / float64(maxCalls) * 20))
			}
			bar := strings.Repeat("█", barLen) + strings.Repeat("░", 20-barLen)
			fmt.Printf("  %s %s\n", bar, formatNumber(calls))
		}
	}
	return nil
}

func formatNumber(n int64) string {
	if n >= 1_000_000 {
		return fmt.Sprintf("%.1fM", float64(n)/1_000_000)
	}
	if n >= 1_000 {
		return fmt.Sprintf("%.1fK", float64(n)/1_000)
	}
	return fmt.Sprintf("%d", n)
}
