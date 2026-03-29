package commands

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

// NewManifestCmd returns the manifest command.
func NewManifestCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "manifest",
		Short: "Manifest utilities",
	}
	cmd.AddCommand(NewManifestEnsureDescriptionsCmd())
	return cmd
}

// NewManifestEnsureDescriptionsCmd returns the ensure-descriptions subcommand.
func NewManifestEnsureDescriptionsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "ensure-descriptions [directory]",
		Short: "Add description to each functionfly.jsonc when missing (humanized from name)",
		Long:  `Reads each functionfly.jsonc under the directory, and when "description" is missing or empty, sets it from the function name (e.g. text-truncate → "Text truncate"). Writes the file back with pretty-printed JSON.`,
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			dir := "."
			if len(args) > 0 {
				dir = args[0]
			}
			return runManifestEnsureDescriptions(dir)
		},
	}
}

func runManifestEnsureDescriptions(baseDir string) error {
	dirs, err := findFunctionDirs(baseDir, "")
	if err != nil {
		return fmt.Errorf("find manifests: %w", err)
	}
	if len(dirs) == 0 {
		fmt.Printf("No functionfly.jsonc found under %s\n", baseDir)
		return nil
	}
	updated := 0
	for _, d := range dirs {
		manifestPath := filepath.Join(d, "functionfly.jsonc")
		data, err := os.ReadFile(manifestPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "  skip %s: %v\n", manifestPath, err)
			continue
		}
		cleaned := stripJSONCComments(data)
		var m map[string]interface{}
		if err := json.Unmarshal(cleaned, &m); err != nil {
			fmt.Fprintf(os.Stderr, "  skip %s: invalid JSON: %v\n", manifestPath, err)
			continue
		}
		name, _ := m["name"].(string)
		if name == "" {
			fmt.Fprintf(os.Stderr, "  skip %s: no name\n", manifestPath)
			continue
		}
		if desc, ok := m["description"].(string); ok && desc != "" {
			continue
		}
		m["description"] = humanizeFunctionName(name)
		out, err := json.MarshalIndent(m, "", "  ")
		if err != nil {
			fmt.Fprintf(os.Stderr, "  skip %s: %v\n", manifestPath, err)
			continue
		}
		if err := os.WriteFile(manifestPath, append(out, '\n'), 0644); err != nil {
			fmt.Fprintf(os.Stderr, "  fail %s: %v\n", manifestPath, err)
			continue
		}
		fmt.Printf("  ✓ %s → description: %q\n", name, m["description"])
		updated++
	}
	fmt.Printf("Updated %d manifest(s) with description.\n", updated)
	return nil
}
