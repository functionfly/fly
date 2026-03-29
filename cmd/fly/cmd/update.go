/*
Copyright © 2026 FunctionFly

*/
package cmd

import (
	"fmt"
	"log"
	"strconv"
	"strings"

	"github.com/functionfly/fly/internal/manifest"
	"github.com/spf13/cobra"
)

// updateCmd represents the update command
var updateCmd = &cobra.Command{
	Use:   "update [level|version]",
	Short: "Safely bump version without overwriting",
	Long: `Safely bumps version without overwriting existing deployments.

Levels:
  patch    1.0.0 → 1.0.1
  minor    1.0.0 → 1.1.0
  major    1.0.0 → 2.0.0

Or set specific version:
  fly update 1.2.3

Examples:
  fly update patch
  fly update minor
  fly update major
  fly update 2.1.0`,
	Run: updateRun,
}

func init() {
	rootCmd.AddCommand(updateCmd)
}

// updateRun implements the update command
func updateRun(cmd *cobra.Command, args []string) {
	if len(args) == 0 {
		log.Fatalf("Version level or specific version required. Use: patch, minor, major, or x.y.z")
	}

	levelOrVersion := args[0]

	// Load manifest
	m, err := manifest.Load("")
	if err != nil {
		log.Fatalf("No functionfly.json found. Run 'fly init' first: %v", err)
	}

	currentVersion := m.Version
	fmt.Printf("Current: %s\n", currentVersion)

	var newVersion string

	// Check if it's a specific version (contains dots) or a level
	if strings.Contains(levelOrVersion, ".") {
		// Specific version
		if !isValidVersion(levelOrVersion) {
			log.Fatalf("Invalid version format '%s'. Use semantic versioning: major.minor.patch", levelOrVersion)
		}
		newVersion = levelOrVersion
	} else {
		// Version level bump
		newVersion, err = bumpVersion(currentVersion, levelOrVersion)
		if err != nil {
			log.Fatalf("Failed to bump version: %v", err)
		}
	}

	// Update manifest
	m.Version = newVersion
	if err := manifest.Save(m, "functionfly.json"); err != nil {
		log.Fatalf("Failed to save manifest: %v", err)
	}

	fmt.Printf("Updated: %s\n\n", newVersion)
	fmt.Println("Run 'fly publish' to deploy the new version")
}

// isValidVersion checks if version string is valid semver format
func isValidVersion(version string) bool {
	parts := strings.Split(version, ".")
	if len(parts) != 3 {
		return false
	}

	for _, part := range parts {
		if _, err := strconv.Atoi(part); err != nil {
			return false
		}
	}

	return true
}

// bumpVersion increments version based on level
func bumpVersion(current, level string) (string, error) {
	if !isValidVersion(current) {
		return "", fmt.Errorf("current version '%s' is not valid semver", current)
	}

	parts := strings.Split(current, ".")
	major, _ := strconv.Atoi(parts[0])
	minor, _ := strconv.Atoi(parts[1])
	patch, _ := strconv.Atoi(parts[2])

	switch strings.ToLower(level) {
	case "major":
		major++
		minor = 0
		patch = 0
	case "minor":
		minor++
		patch = 0
	case "patch":
		patch++
	default:
		return "", fmt.Errorf("invalid level '%s'. Use: patch, minor, or major", level)
	}

	return fmt.Sprintf("%d.%d.%d", major, minor, patch), nil
}