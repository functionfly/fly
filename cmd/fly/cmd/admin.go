/*
Copyright © 2026 FunctionFly
*/
package cmd

import (
	"github.com/spf13/cobra"
)

// adminCmd represents the admin command
var adminCmd = &cobra.Command{
	Use:   "admin",
	Short: "Administrative operations",
	Long: `Administrative operations for system management.

These commands are typically used by administrators to manage
users, system configuration, and database operations.

Examples:
  # Create an admin user
  fly admin create-user --email admin@example.com

  # Initialize system setup
  fly admin setup

  # Database operations
  fly admin db clean-functions`,
	SilenceUsage: true,
}

func init() {
	adminCmd.AddCommand(newAdminCreateUserCmd())
	adminCmd.AddCommand(newAdminSetupCmd())
	adminCmd.AddCommand(newAdminDBCmd())
}

// AdminCmd returns the admin command for attachment to the live root (e.g. from main).
func AdminCmd() *cobra.Command {
	return adminCmd
}
