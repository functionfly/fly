/*
Copyright © 2026 FunctionFly
*/
package cmd

import (
	"context"
	"database/sql"
	"fmt"
	"os"

	_ "github.com/lib/pq"
	"github.com/spf13/cobra"
)

var (
	dbForce  bool
	dbDryRun bool
)

// newAdminDBCmd creates the admin db command
func newAdminDBCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "db",
		Short: "Database management operations",
		Long: `Database management operations for administrators.

Subcommands:
  clean-functions    Delete all functions from the database`,
		SilenceUsage: true,
	}

	cmd.AddCommand(newAdminDBCleanFunctionsCmd())

	return cmd
}

// newAdminDBCleanFunctionsCmd creates the admin db clean-functions command
func newAdminDBCleanFunctionsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "clean-functions",
		Short: "Delete all functions from the database",
		Long: `WARNING: This will permanently delete ALL functions from the database!

This includes:
- User-created functions
- Function deployments
- Function logs
- Registry functions and versions
- Function executions and ratings`,
		Example: `  # Dry run to see what would be deleted
  fly admin db clean-functions --dry-run

  # Actually delete all functions
  fly admin db clean-functions --force`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runAdminDBCleanFunctions(cmd)
		},
	}

	cmd.Flags().BoolVar(&dbForce, "force", false, "Skip confirmation prompt")
	cmd.Flags().BoolVar(&dbDryRun, "dry-run", false, "Show what would be deleted without actually deleting")

	return cmd
}

func runAdminDBCleanFunctions(cmd *cobra.Command) error {
	// Get database URL from environment
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		return fmt.Errorf("DATABASE_URL environment variable is required")
	}

	// Connect to database
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}
	defer db.Close()

	// Test connection
	if err := db.Ping(); err != nil {
		return fmt.Errorf("failed to ping database: %w", err)
	}

	// Confirm the operation
	if !dbForce && !dbDryRun {
		fmt.Println("WARNING: This will permanently delete ALL functions from the database!")
		fmt.Println("This includes:")
		fmt.Println("- User-created functions")
		fmt.Println("- Function deployments")
		fmt.Println("- Function logs")
		fmt.Println("- Registry functions and versions")
		fmt.Println("- Function executions and ratings")
		fmt.Println("")
		fmt.Print("Are you sure you want to continue? Type 'yes' to confirm: ")

		var response string
		fmt.Scanln(&response)
		if response != "yes" {
			fmt.Println("Operation cancelled.")
			return nil
		}
	}

	ctx := context.Background()

	// Tables to delete from (in order to respect foreign key constraints)
	tables := []string{
		"registry_function_approval_comments",
		"registry_function_approvals",
		"registry_function_malware_scans",
		"registry_function_signatures",
		"registry_function_ratings",
		"registry_executions_public",
		"registry_function_executions",
		"registry_function_versions",
		"registry_functions",
		"function_logs",
		"function_deployments",
		"functions",
	}

	if dbDryRun {
		fmt.Println("Dry run mode - the following would be deleted:")
		for _, table := range tables {
			fmt.Printf("  - All rows from %s\n", table)
		}
		fmt.Println()
		fmt.Println("Run without --dry-run to actually delete.")
		return nil
	}

	fmt.Println("Deleting all functions...")

	for _, table := range tables {
		fmt.Printf("Deleting from %s...\n", table)
		query := fmt.Sprintf("DELETE FROM %q", table)
		result, err := db.ExecContext(ctx, query)
		if err != nil {
			return fmt.Errorf("failed to delete from %s: %w", table, err)
		}
		rowsAffected, _ := result.RowsAffected()
		fmt.Printf("  Deleted %d rows from %s\n", rowsAffected, table)
	}

	// Reset sequences
	fmt.Println("Resetting sequences...")
	sequences := []string{
		"functions_id_seq",
		"function_deployments_id_seq",
		"function_logs_id_seq",
		"registry_functions_id_seq",
		"registry_function_versions_id_seq",
		"registry_function_executions_id_seq",
		"registry_executions_public_id_seq",
		"registry_function_ratings_id_seq",
		"registry_function_signatures_id_seq",
		"registry_function_malware_scans_id_seq",
		"registry_function_approvals_id_seq",
		"registry_function_approval_comments_id_seq",
	}

	for _, seq := range sequences {
		query := fmt.Sprintf("ALTER SEQUENCE %q RESTART WITH 1", seq)
		_, err := db.ExecContext(ctx, query)
		if err != nil {
			fmt.Printf("  Warning: Failed to reset %s: %v\n", seq, err)
		}
	}

	fmt.Println()
	fmt.Println("✅ Successfully deleted all functions and reset sequences")

	return nil
}
