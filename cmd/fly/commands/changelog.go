package commands

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

// ChangelogEntry represents a single changelog entry.
type ChangelogEntry struct {
	Version string   `json:"version"`
	Date    string   `json:"date,omitempty"`
	Changes []Change `json:"changes"`
}

type Change struct {
	Category string `json:"category"` // "added", "changed", "fixed", "removed", "deprecated"
	Summary  string `json:"summary"`
}

func NewChangelogCmd() *cobra.Command {
	var asJSON bool
	cmd := &cobra.Command{
		Use:   "changelog",
		Short: "Show the CLI changelog",
		Long: `Display the release changelog for the FunctionFly CLI.

Shows a structured list of changes grouped by version, including
new features, bug fixes, breaking changes, and deprecations.`,
		Example: "  ffly changelog\n  ffly changelog --json",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runChangelog(asJSON)
		},
	}
	cmd.Flags().BoolVar(&asJSON, "json", false, "Output as JSON")
	return cmd
}

func runChangelog(asJSON bool) error {
	entries := getChangelogEntries()

	if asJSON || WantJSON() {
		printJSON(entries)
		return nil
	}

	for _, entry := range entries {
		fmt.Printf("## %s", entry.Version)
		if entry.Date != "" {
			fmt.Printf(" (%s)", entry.Date)
		}
		fmt.Println()

		grouped := groupByCategory(entry.Changes)
		for _, cat := range []string{"added", "changed", "fixed", "removed", "deprecated"} {
			changes, ok := grouped[cat]
			if !ok {
				continue
			}
			fmt.Printf("\n### %s\n", capitalize(cat))
			for _, c := range changes {
				fmt.Printf("  - %s\n", c.Summary)
			}
		}
		fmt.Println()
	}

	return nil
}

func groupByCategory(changes []Change) map[string][]Change {
	grouped := make(map[string][]Change)
	for _, c := range changes {
		grouped[c.Category] = append(grouped[c.Category], c)
	}
	return grouped
}

func capitalize(s string) string {
	if len(s) == 0 {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

func getChangelogEntries() []ChangelogEntry {
	return []ChangelogEntry{
		{
			Version: "1.2.0",
			Date:    "2026-04-12",
			Changes: []Change{
				{Category: "added", Summary: "`ffly config set KEY=VALUE` to set config values via CLI"},
				{Category: "added", Summary: "`ffly env apply` to set environment variables from a .env file"},
				{Category: "added", Summary: "`--dry-run` flag on `ffly env set`, `ffly env unset`, and `ffly env apply`"},
				{Category: "added", Summary: "`-y/--yes` global flag to skip all confirmation prompts (CI-safe)"},
				{Category: "added", Summary: "Persistent `--yes/-y` flag to auto-confirm deploy, publish, logout, and env operations"},
				{Category: "fixed", Summary: "Spinners now fall back gracefully in headless/non-TTY environments"},
				{Category: "fixed", Summary: "`--dry-run` for env commands no longer requires credentials"},
				{Category: "fixed", Summary: "Bash completion now works (fixed duplicate -o shorthand conflict)"},
				{Category: "changed", Summary: "Renamed `--format` shorthand from `-o` to `-m` to avoid conflicts with subcommands"},
			},
		},
		{
			Version: "1.1.0",
			Date:    "2026-04-01",
			Changes: []Change{
				{Category: "added", Summary: "`ffly doctor` command for environment diagnostics"},
				{Category: "added", Summary: "Automatic retry with exponential backoff for transient API errors (429, 5xx)"},
				{Category: "added", Summary: "`--json` flag on `ffly version` for machine-readable output"},
				{Category: "added", Summary: "`ffly changelog` command to view release notes from the CLI"},
				{Category: "changed", Summary: "API client now retries failed requests up to 3 times with backoff"},
				{Category: "changed", Summary: "Improved error messages for server-side failures"},
			},
		},
		{
			Version: "1.0.0",
			Date:    "2026-03-15",
			Changes: []Change{
				{Category: "added", Summary: "Initial public release of the FunctionFly CLI"},
				{Category: "added", Summary: "`ffly login` with OAuth and dev-mode authentication"},
				{Category: "added", Summary: "`ffly init` with templates (hello-world, http-api, cron-job, webhook)"},
				{Category: "added", Summary: "`ffly dev` local development server with file watching"},
				{Category: "added", Summary: "`ffly publish` to publish functions to the registry"},
				{Category: "added", Summary: "`ffly deploy` with environment aliasing and canary deployments"},
				{Category: "added", Summary: "`ffly canary` for gradual traffic rollout management"},
				{Category: "added", Summary: "`ffly schedule` for cron-based function execution"},
				{Category: "added", Summary: "`ffly test`, `ffly stats`, `ffly logs`, `ffly health`"},
				{Category: "added", Summary: "`ffly rollback` to revert to a previous function version"},
				{Category: "added", Summary: "`ffly env` and `ffly secrets` management"},
				{Category: "added", Summary: "`ffly completion` for bash, zsh, fish, and powershell"},
				{Category: "added", Summary: "OS keychain credential storage with file fallback"},
			},
		},
	}
}
