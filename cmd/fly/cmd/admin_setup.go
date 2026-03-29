/*
Copyright © 2026 FunctionFly
*/
package cmd

import (
	"database/sql"
	"fmt"
	"os"

	_ "github.com/lib/pq"
	"github.com/spf13/cobra"
)

// newAdminSetupCmd creates the admin setup command
func newAdminSetupCmd() *cobra.Command {
	var password string
	cmd := &cobra.Command{
		Use:   "setup",
		Short: "Initialize system setup",
		Long: `Initialize the FunctionFly system with default tenant and admin user.

This command sets up the initial system configuration including
creating a default tenant and admin user.

The admin password must be provided via --password or ADMIN_CREATE_PASSWORD env var.`,
		Example: `  # Initialize system setup
  fly admin setup --password <secure-password>

  # Or via environment variable
  ADMIN_CREATE_PASSWORD=<secure-password> fly admin setup`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runAdminSetup(cmd, password)
		},
	}
	cmd.Flags().StringVar(&password, "password", "", "Admin user password (or set ADMIN_CREATE_PASSWORD)")
	return cmd
}

func runAdminSetup(cmd *cobra.Command, password string) error {
	// Get database URL from environment
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		return fmt.Errorf("DATABASE_URL environment variable is required")
	}

	// Resolve password: flag > env var
	if password == "" {
		password = os.Getenv("ADMIN_CREATE_PASSWORD")
	}
	if password == "" {
		return fmt.Errorf("admin password is required (use --password flag or ADMIN_CREATE_PASSWORD env var)")
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

	// Create default tenant
	var tenantID string
	err = db.QueryRow(`
		INSERT INTO tenants (id, name, created_at, updated_at)
		VALUES (gen_random_uuid(), 'Default Tenant', NOW(), NOW())
		ON CONFLICT DO NOTHING
		RETURNING id
	`).Scan(&tenantID)

	if err != nil && err != sql.ErrNoRows {
		return fmt.Errorf("failed to create tenant: %w", err)
	}

	if tenantID == "" {
		// Tenant already exists, get it
		err = db.QueryRow(`SELECT id FROM tenants LIMIT 1`).Scan(&tenantID)
		if err != nil {
			return fmt.Errorf("failed to get tenant: %w", err)
		}
		fmt.Println("Default tenant already exists")
	} else {
		fmt.Println("Created default tenant")
	}

	// Create default admin user if not exists
	var userID string
	hashedPassword, err := hashPassword(password)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}

	err = db.QueryRow(`
		INSERT INTO users (email, password_hash, tenant_id, role, created_at, updated_at)
		VALUES ('admin@example.com', $1, $2, 'admin', NOW(), NOW())
		ON CONFLICT (email) DO NOTHING
		RETURNING id
	`, hashedPassword, tenantID).Scan(&userID)

	if err != nil && err != sql.ErrNoRows {
		return fmt.Errorf("failed to create admin user: %w", err)
	}

	if userID == "" {
		fmt.Println("Admin user already exists")
	} else {
		fmt.Println("Created admin user")
	}

	fmt.Println()
	fmt.Println("Setup complete!")
	fmt.Println("  Admin email: admin@example.com")
	fmt.Println()
	fmt.Println("You can now login with:")
	fmt.Println("  fly login")

	return nil
}
