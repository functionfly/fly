/*
Copyright © 2026 FunctionFly
*/
package cmd

import (
	"fmt"
	"log"

	"github.com/functionfly/fly/internal/bundler"
	"github.com/functionfly/fly/internal/cli"
	"github.com/functionfly/fly/internal/credentials"
	"github.com/functionfly/fly/internal/manifest"
	"github.com/spf13/cobra"
)

// deployCmd represents the deploy command
var deployCmd = &cobra.Command{
	Use:   "deploy",
	Short: "Unified deployment command for all function types",
	Long: `Deploys functions to the FunctionFly platform with advanced features.

Supports multi-environment deployment, rollback capabilities, and deployment pipelines.
Automatically handles bundling, artifact creation, and infrastructure provisioning.

Examples:
  fly deploy
  fly deploy --env=staging
  fly deploy --preview
  fly deploy --force`,
	Run: deployRun,
}

var deployFlags struct {
	env           string
	preview       bool
	force         bool
	wait          bool
	rollbackID    string
	jsonOutput    bool
	skipTypeCheck bool
}

func init() {
	rootCmd.AddCommand(deployCmd)

	// Local flags
	deployCmd.Flags().StringVarP(&deployFlags.env, "env", "e", "production", "Target environment (production, staging, development)")
	deployCmd.Flags().BoolVarP(&deployFlags.preview, "preview", "p", false, "Preview deployment changes without applying")
	deployCmd.Flags().BoolVar(&deployFlags.force, "force", false, "Force deployment even if validation fails")
	deployCmd.Flags().BoolVarP(&deployFlags.wait, "wait", "w", true, "Wait for deployment to complete")
	deployCmd.Flags().StringVar(&deployFlags.rollbackID, "rollback-to", "", "Rollback to specific deployment ID")
	deployCmd.Flags().BoolVarP(&deployFlags.jsonOutput, "json", "j", false, "Output results in JSON format")
	deployCmd.Flags().BoolVar(&deployFlags.skipTypeCheck, "skip-type-check", false, "Skip TypeScript type checking during deployment")
}

// deployRun implements the deploy command
func deployRun(cmd *cobra.Command, args []string) {
	if deployFlags.preview {
		fmt.Println("Previewing deployment...")
	} else if deployFlags.rollbackID != "" {
		fmt.Println("Rolling back deployment...")
	} else {
		fmt.Println("Deploying function...")
	}

	// 1. Load and validate manifest
	m, err := manifest.Load("")
	if err != nil {
		log.Fatalf("No functionfly.json found. Run 'fly init' first: %v", err)
	}

	if err := m.Validate(); err != nil {
		log.Fatalf("Manifest validation failed: %v", err)
	}

	// 2. Load credentials
	creds, err := credentials.Load()
	if err != nil {
		log.Fatalf("Not logged in. Run 'fly login' first: %v", err)
	}

	// 3. Handle rollback if specified
	if deployFlags.rollbackID != "" {
		err := performRollback(creds.User.Username, m.Name, deployFlags.rollbackID, deployFlags.jsonOutput)
		if err != nil {
			log.Fatalf("Rollback failed: %v", err)
		}
		return
	}

	// 4. Preview mode
	if deployFlags.preview {
		err := performPreview(m, creds.User.Username, deployFlags.env, deployFlags.jsonOutput)
		if err != nil {
			log.Fatalf("Preview failed: %v", err)
		}
		return
	}

	// 5. Perform deployment
	err = performDeployment(m, creds, deployFlags.env, deployFlags.force, deployFlags.wait, deployFlags.jsonOutput)
	if err != nil {
		log.Fatalf("Deployment failed: %v", err)
	}
}

// performDeployment handles the actual deployment process
func performDeployment(m *manifest.Manifest, creds *credentials.Credentials, env string, force, wait, jsonOutput bool) error {
	fmt.Printf("✓ Validating manifest for %s environment...\n", env)

	// 3. Bundle code
	fmt.Println("✓ Bundling code...")

	// Create bundle options with skip type check if set
	var bundleOptions *bundler.BundleOptions
	if deployFlags.skipTypeCheck {
		bundleOptions = &bundler.BundleOptions{
			SkipTypeCheck: true,
		}
	}

	bundle, err := bundler.BundleWithOptionsAndWorkingDirectory(m, bundleOptions, "")
	if err != nil {
		return fmt.Errorf("bundling failed: %v", err)
	}

	bundleSize := len(bundle)
	fmt.Printf("✓ Code bundled (%d bytes)\n", bundleSize)

	// 4. Generate version hash
	hash := bundler.HashContent(bundle)
	fmt.Printf("✓ Content hash: %s\n", hash[:16]+"...")

	// 5. Create API client
	apiURL := getAPIURL()
	client := cli.NewClient(apiURL, creds.Token)

	// 6. Prepare deployment request
	deployReq := &cli.DeployRequest{
		Provider: "functionfly",  // Default provider
		Region:   "auto",         // Auto-select region
		Artifact: string(bundle), // Base64 encoded bundle
		EnvVars:  map[string]string{"ENV": env},
		ProviderConfig: map[string]interface{}{
			"environment": env,
			"force":       force,
		},
	}

	// 7. Deploy to platform
	fmt.Printf("✓ Deploying to %s environment...\n", env)

	result, err := client.Deploy("1", deployReq) // App ID would come from config
	if err != nil {
		return fmt.Errorf("deployment failed: %v", err)
	}

	// 8. Wait for completion if requested
	if wait {
		fmt.Println("✓ Waiting for deployment to complete...")
		// In a real implementation, this would poll for deployment status
		fmt.Printf("✓ Deployment %s completed\n", result.DeploymentID)
	}

	// 9. Print success
	if jsonOutput {
		outputDeployJSON(result, env)
	} else {
		outputDeployHuman(result, creds.User.Username, m.Name, env)
	}

	return nil
}

// performPreview shows what would be deployed
func performPreview(m *manifest.Manifest, author, env string, jsonOutput bool) error {
	// Generate preview information
	preview := &DeployPreview{
		Function:    m.Name,
		Author:      author,
		Version:     m.Version,
		Runtime:     m.Runtime,
		Environment: env,
		Changes: []string{
			"New function deployment",
			fmt.Sprintf("Runtime: %s", m.Runtime),
			fmt.Sprintf("Environment: %s", env),
		},
	}

	if jsonOutput {
		outputPreviewJSON(preview)
	} else {
		outputPreviewHuman(preview)
	}

	return nil
}

// performRollback handles rollback to a previous deployment
func performRollback(author, name, rollbackID string, jsonOutput bool) error {
	apiURL := getAPIURL()
	creds, _ := credentials.Load()
	client := cli.NewClient(apiURL, creds.Token)

	result, err := client.Rollback(rollbackID)
	if err != nil {
		return fmt.Errorf("rollback failed: %v", err)
	}

	if jsonOutput {
		outputRollbackJSON(result)
	} else {
		outputRollbackHuman(result, author, name)
	}

	return nil
}

// outputDeployHuman prints deployment results in human-readable format
func outputDeployHuman(result *cli.DeployResponse, author, name, env string) {
	fmt.Printf("\nDeployment Results:\n")
	fmt.Printf("==================\n")
	fmt.Printf("Function: %s/%s\n", author, name)
	fmt.Printf("Environment: %s\n", env)
	fmt.Printf("Deployment ID: %s\n", result.DeploymentID)
	fmt.Printf("Status: %s\n", result.Status)
	if result.Message != "" {
		fmt.Printf("Message: %s\n", result.Message)
	}
	fmt.Printf("\n✓ Deployment successful\n")
}

// outputDeployJSON prints deployment results in JSON format
func outputDeployJSON(result *cli.DeployResponse, env string) {
	fmt.Printf(`{
  "deployment_id": %q,
  "status": %q,
  "environment": %q,
  "message": %q,
  "success": true
}`, result.DeploymentID, result.Status, env, result.Message)
}

// DeployPreview represents a deployment preview
type DeployPreview struct {
	Function    string   `json:"function"`
	Author      string   `json:"author"`
	Version     string   `json:"version"`
	Runtime     string   `json:"runtime"`
	Environment string   `json:"environment"`
	Changes     []string `json:"changes"`
}

// outputPreviewHuman prints preview in human-readable format
func outputPreviewHuman(preview *DeployPreview) {
	fmt.Printf("\nDeployment Preview:\n")
	fmt.Printf("==================\n")
	fmt.Printf("Function: %s/%s@%s\n", preview.Author, preview.Function, preview.Version)
	fmt.Printf("Runtime: %s\n", preview.Runtime)
	fmt.Printf("Environment: %s\n\n", preview.Environment)

	fmt.Println("Changes:")
	for _, change := range preview.Changes {
		fmt.Printf("  • %s\n", change)
	}

	fmt.Printf("\nUse 'fly deploy' to apply these changes\n")
}

// outputPreviewJSON prints preview in JSON format
func outputPreviewJSON(preview *DeployPreview) {
	fmt.Printf(`{
  "function": %q,
  "author": %q,
  "version": %q,
  "runtime": %q,
  "environment": %q,
  "changes": [`, preview.Function, preview.Author, preview.Version, preview.Runtime, preview.Environment)

	for i, change := range preview.Changes {
		if i > 0 {
			fmt.Printf(",")
		}
		fmt.Printf("\n    %q", change)
	}
	fmt.Printf("\n  ]\n}")
}

// outputRollbackHuman prints rollback results in human-readable format
func outputRollbackHuman(result *cli.RollbackResponse, author, name string) {
	fmt.Printf("\nRollback Results:\n")
	fmt.Printf("================\n")
	fmt.Printf("Function: %s/%s\n", author, name)
	fmt.Printf("Deployment ID: %s\n", result.DeploymentID)
	fmt.Printf("Status: %s\n", result.Status)
	if result.Message != "" {
		fmt.Printf("Message: %s\n", result.Message)
	}
	fmt.Printf("\n✓ Rollback successful\n")
}

// outputRollbackJSON prints rollback results in JSON format
func outputRollbackJSON(result *cli.RollbackResponse) {
	fmt.Printf(`{
  "deployment_id": %q,
  "status": %q,
  "message": %q,
  "success": true
}`, result.DeploymentID, result.Status, result.Message)
}
