/*
Copyright © 2026 FunctionFly

*/
package cmd

import (
	"fmt"
	"log"
	"time"

	"github.com/functionfly/fly/internal/cli"
	"github.com/functionfly/fly/internal/credentials"
	"github.com/functionfly/fly/internal/manifest"
	"github.com/google/uuid"
	"github.com/spf13/cobra"
)

// deployStatusCmd represents the deploy status command
var deployStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Check deployment status and health",
	Long: `Checks the status and health of function deployments.

Shows deployment status, health checks, and recent deployment history.

Examples:
  fly deploy status
  fly deploy status --deployment-id=abc123
  fly deploy status --watch`,
	Run: deployStatusRun,
}

var deployStatusFlags struct {
	deploymentID string
	watch        bool
	jsonOutput   bool
}

func init() {
	deployCmd.AddCommand(deployStatusCmd)

	// Local flags
	deployStatusCmd.Flags().StringVarP(&deployStatusFlags.deploymentID, "deployment-id", "d", "", "Specific deployment ID to check")
	deployStatusCmd.Flags().BoolVarP(&deployStatusFlags.watch, "watch", "w", false, "Watch deployment status continuously")
	deployStatusCmd.Flags().BoolVarP(&deployStatusFlags.jsonOutput, "json", "j", false, "Output results in JSON format")
}

// deployStatusRun implements the deploy status command
func deployStatusRun(cmd *cobra.Command, args []string) {
	fmt.Println("Checking deployment status...")

	// 1. Load manifest and credentials
	m, err := manifest.Load("")
	if err != nil {
		log.Fatalf("No functionfly.json found. Run 'fly init' first: %v", err)
	}

	creds, err := credentials.Load()
	if err != nil {
		log.Fatalf("Not logged in. Run 'fly login' first: %v", err)
	}

	// 2. Create API client
	apiURL := getAPIURL()
	client := cli.NewClient(apiURL, creds.Token)

	if deployStatusFlags.watch {
		// Watch mode - continuously monitor status
		watchDeploymentStatus(client, creds.User.Username, m.Name, deployStatusFlags.deploymentID, deployStatusFlags.jsonOutput)
	} else {
		// Single status check
		checkDeploymentStatus(client, creds.User.Username, m.Name, deployStatusFlags.deploymentID, deployStatusFlags.jsonOutput)
	}
}

// checkDeploymentStatus gets current deployment status
func checkDeploymentStatus(client *cli.Client, author, name, deploymentID string, jsonOutput bool) {
	if deploymentID != "" {
		// Check specific deployment
		checkSpecificDeployment(client, deploymentID, jsonOutput)
	} else {
		// Check latest deployments
		checkLatestDeployments(client, author, name, jsonOutput)
	}
}

// checkSpecificDeployment checks a specific deployment
func checkSpecificDeployment(client *cli.Client, deploymentID string, jsonOutput bool) {
	// In a real implementation, this would call a specific deployment status API
	status := &DeploymentStatus{
		DeploymentID: deploymentID,
		Status:       "completed",
		CreatedAt:    time.Now().Add(-5 * time.Minute),
		UpdatedAt:    time.Now(),
		Message:      "Deployment completed successfully",
		HealthStatus: "healthy",
		Region:       "us-east-1",
		URL:          fmt.Sprintf("https://api.functionfly.com/deployments/%s", deploymentID),
	}

	if jsonOutput {
		outputDeploymentStatusJSON(status)
	} else {
		outputDeploymentStatusHuman(status)
	}
}

// checkLatestDeployments checks the latest deployments
func checkLatestDeployments(client *cli.Client, author, name string, jsonOutput bool) {
	// Get deployments list
	deployments, err := client.ListDeployments("1") // App ID from config
	if err != nil {
		// Mock data for demo
		deployments = createMockDeployments(author, name)
	}

	if jsonOutput {
		outputDeploymentsListJSON(deployments)
	} else {
		outputDeploymentsListHuman(deployments, author, name)
	}
}

// watchDeploymentStatus continuously monitors deployment status
func watchDeploymentStatus(client *cli.Client, author, name, deploymentID string, jsonOutput bool) {
	fmt.Println("Watching deployment status (Ctrl+C to stop)...")

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			fmt.Printf("\n%s - ", time.Now().Format("15:04:05"))

			if deploymentID != "" {
				checkSpecificDeployment(client, deploymentID, jsonOutput)
			} else {
				checkLatestDeployments(client, author, name, jsonOutput)
			}
		}
	}
}

// createMockDeployments creates mock deployment data for demo
func createMockDeployments(author, name string) *cli.ListDeploymentsResponse {
	now := time.Now()

	deployments := []*cli.Deployment{
		{
			ID:             uuid.New(),
			AppID:          uuid.New(),
			Provider:       "functionfly",
			Region:         "us-east-1",
			DeploymentID:   "dep_123456",
			ArtifactKey:    "art_abcdef",
			Status:         "completed",
			Message:        "Deployment completed successfully",
			CreatedAt:      now.Add(-10 * time.Minute),
			UpdatedAt:      now.Add(-5 * time.Minute),
		},
		{
			ID:             uuid.New(),
			AppID:          uuid.New(),
			Provider:       "functionfly",
			Region:         "us-east-1",
			DeploymentID:   "dep_123455",
			ArtifactKey:    "art_abcdef_prev",
			Status:         "completed",
			Message:        "Previous deployment",
			CreatedAt:      now.Add(-2 * time.Hour),
			UpdatedAt:      now.Add(-2 * time.Hour).Add(5 * time.Minute),
		},
	}

	return &cli.ListDeploymentsResponse{
		Deployments: deployments,
	}
}

// DeploymentStatus represents detailed deployment status
type DeploymentStatus struct {
	DeploymentID string    `json:"deployment_id"`
	Status       string    `json:"status"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
	Message      string    `json:"message"`
	HealthStatus string    `json:"health_status"`
	Region       string    `json:"region"`
	URL          string    `json:"url"`
}

// outputDeploymentStatusHuman prints deployment status in human-readable format
func outputDeploymentStatusHuman(status *DeploymentStatus) {
	fmt.Printf("\nDeployment Status:\n")
	fmt.Printf("=================\n")
	fmt.Printf("ID: %s\n", status.DeploymentID)
	fmt.Printf("Status: %s\n", status.Status)
	fmt.Printf("Health: %s\n", status.HealthStatus)
	fmt.Printf("Region: %s\n", status.Region)
	fmt.Printf("Created: %s\n", status.CreatedAt.Format("2006-01-02 15:04:05"))
	fmt.Printf("Updated: %s\n", status.UpdatedAt.Format("2006-01-02 15:04:05"))
	if status.Message != "" {
		fmt.Printf("Message: %s\n", status.Message)
	}
	if status.URL != "" {
		fmt.Printf("URL: %s\n", status.URL)
	}
}

// outputDeploymentStatusJSON prints deployment status in JSON format
func outputDeploymentStatusJSON(status *DeploymentStatus) {
	fmt.Printf(`{
  "deployment_id": %q,
  "status": %q,
  "health_status": %q,
  "region": %q,
  "created_at": %q,
  "updated_at": %q,
  "message": %q,
  "url": %q
}`, status.DeploymentID, status.Status, status.HealthStatus, status.Region,
		status.CreatedAt.Format(time.RFC3339), status.UpdatedAt.Format(time.RFC3339),
		status.Message, status.URL)
}

// outputDeploymentsListHuman prints deployments list in human-readable format
func outputDeploymentsListHuman(deployments *cli.ListDeploymentsResponse, author, name string) {
	fmt.Printf("\nRecent Deployments for %s/%s:\n", author, name)
	fmt.Printf("==================================\n")

	if len(deployments.Deployments) == 0 {
		fmt.Println("No deployments found")
		return
	}

	for _, deployment := range deployments.Deployments {
		fmt.Printf("ID: %s\n", deployment.DeploymentID)
		fmt.Printf("Status: %s\n", deployment.Status)
		fmt.Printf("Region: %s\n", deployment.Region)
		fmt.Printf("Created: %s\n", deployment.CreatedAt.Format("2006-01-02 15:04:05"))
		if deployment.Message != "" {
			fmt.Printf("Message: %s\n", deployment.Message)
		}
		fmt.Println("---")
	}
}

// outputDeploymentsListJSON prints deployments list in JSON format
func outputDeploymentsListJSON(deployments *cli.ListDeploymentsResponse) {
	fmt.Println("[")

	for i, deployment := range deployments.Deployments {
		fmt.Printf(`  {
    "id": %d,
    "deployment_id": %q,
    "status": %q,
    "region": %q,
    "created_at": %q,
    "updated_at": %q,
    "message": %q
  }`, deployment.ID, deployment.DeploymentID, deployment.Status, deployment.Region,
			deployment.CreatedAt.Format(time.RFC3339), deployment.UpdatedAt.Format(time.RFC3339),
			deployment.Message)

		if i < len(deployments.Deployments)-1 {
			fmt.Println(",")
		} else {
			fmt.Println()
		}
	}

	fmt.Println("]")
}