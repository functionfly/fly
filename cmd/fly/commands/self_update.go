package commands

import (
	"fmt"

	"github.com/spf13/cobra"
)

// NewSelfUpdateCmd returns the self-update command (upgrade instructions for the fly CLI).
func NewSelfUpdateCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "self-update",
		Short: "Show how to upgrade the fly CLI",
		Long: `Show instructions to upgrade the fly CLI to the latest version.

The fly CLI does not replace its own binary. Use one of these methods:

  Install script (Linux/macOS):
    curl -fsSL https://raw.githubusercontent.com/functionfly/fly/main/scripts/install.sh | bash

  Homebrew (macOS/Linux, when tap is configured):
    brew upgrade fly

  Scoop (Windows):
    scoop update fly

  Chocolatey (Windows):
    choco upgrade fly

  Manual: Download the latest release from
  https://github.com/functionfly/fly/releases
  and replace your fly binary.`,
		RunE: runSelfUpdate,
	}
}

func runSelfUpdate(cmd *cobra.Command, args []string) error {
	fmt.Println("Upgrade the fly CLI using one of these methods:")
	fmt.Println()
	fmt.Println("  Install script (Linux/macOS):")
	fmt.Println("    curl -fsSL https://raw.githubusercontent.com/functionfly/fly/main/scripts/install.sh | bash")
	fmt.Println()
	fmt.Println("  Homebrew (when tap is configured):")
	fmt.Println("    brew upgrade fly")
	fmt.Println()
	fmt.Println("  Windows: Scoop (scoop update fly) or Chocolatey (choco upgrade fly)")
	fmt.Println()
	fmt.Println("  Releases: https://github.com/functionfly/fly/releases")
	return nil
}
