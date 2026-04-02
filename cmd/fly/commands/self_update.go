package commands

import (
	"fmt"

	"github.com/spf13/cobra"
)

// NewSelfUpdateCmd returns the self-update command (upgrade instructions for the ffly CLI).
func NewSelfUpdateCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "self-update",
		Short: "Show how to upgrade the ffly CLI",
		Long: `Show instructions to upgrade the ffly CLI to the latest version.

The ffly CLI does not replace its own binary. Use one of these methods:

  Install script (Linux/macOS):
    curl -fsSL https://raw.githubusercontent.com/functionfly/fly/main/scripts/install.sh | bash

  Homebrew (macOS/Linux, when tap is configured):
    brew upgrade ffly

  Scoop (Windows):
    scoop update ffly

  Chocolatey (Windows):
    choco upgrade ffly

  Manual: Download the latest release from
  https://github.com/functionfly/fly/releases
  and replace your ffly binary.`,
		RunE: runSelfUpdate,
	}
}

func runSelfUpdate(cmd *cobra.Command, args []string) error {
	fmt.Println("Upgrade the ffly CLI using one of these methods:")
	fmt.Println()
	fmt.Println("  Install script (Linux/macOS):")
	fmt.Println("    curl -fsSL https://raw.githubusercontent.com/functionfly/fly/main/scripts/install.sh | bash")
	fmt.Println()
	fmt.Println("  Homebrew (when tap is configured):")
	fmt.Println("    brew upgrade ffly")
	fmt.Println()
	fmt.Println("  Windows: Scoop (scoop update ffly) or Chocolatey (choco upgrade ffly)")
	fmt.Println()
	fmt.Println("  Releases: https://github.com/functionfly/fly/releases")
	return nil
}
