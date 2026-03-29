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
	"golang.org/x/crypto/bcrypt"
)

var (
	adminEmail    string
	adminPassword string
	adminRole     string
)

// newAdminCreateUserCmd creates the admin create-user command
func newAdminCreateUserCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create-user",
		Short: "Create a new admin user",
		Long: `Create a new admin user in the database.

This command creates a new user with admin privileges.`,
		Example: `  # Create an admin user
  fly admin create-user --email admin@example.com --password secret123

  # Create a user with specific role
  fly admin create-user --email user@example.com --password pass123 --role user`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runAdminCreateUser(cmd)
		},
	}

	cmd.Flags().StringVar(&adminEmail, "email", "", "User email (required)")
	cmd.Flags().StringVar(&adminPassword, "password", "", "User password (required)")
	cmd.Flags().StringVar(&adminRole, "role", "admin", "User role (admin, user)")

	cmd.MarkFlagRequired("email")
	cmd.MarkFlagRequired("password")

	return cmd
}

func runAdminCreateUser(cmd *cobra.Command) error {
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

	// Hash password with bcrypt
	hashedPassword, err := hashPassword(adminPassword)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}

	// Insert user
	var userID string
	err = db.QueryRowContext(cmd.Context(), `
		INSERT INTO users (email, password_hash, tenant_id, role, created_at, updated_at)
		VALUES ($1, $2, (SELECT id FROM tenants LIMIT 1), $3, NOW(), NOW())
		RETURNING id
	`, adminEmail, hashedPassword, adminRole).Scan(&userID)

	if err != nil {
		return fmt.Errorf("failed to create user: %w", err)
	}

	fmt.Printf("✅ Successfully created user: %s\n", adminEmail)
	fmt.Printf("   User ID: %s\n", userID)
	fmt.Printf("   Role: %s\n", adminRole)

	return nil
}

// hashPassword hashes the password using bcrypt (same as rest of codebase).
func hashPassword(password string) (string, error) {
	hashed, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hashed), nil
}
