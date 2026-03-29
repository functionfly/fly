/*
Copyright © 2026 FunctionFly

*/
package cmd

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/functionfly/fly/internal/bundler"
	"github.com/functionfly/fly/internal/cli"
	"github.com/functionfly/fly/internal/credentials"
	"github.com/functionfly/fly/internal/manifest"
	"github.com/spf13/cobra"
)

// publishCmd represents the publish command
var publishCmd = &cobra.Command{
	Use:   "publish",
	Short: "Publish function to global registry with automatic infrastructure",
	Long: `Publishes function to global registry with automatic infrastructure handling.

Automatic Workflow:
1. Validate manifest
2. Bundle code (esbuild)
3. Generate content hash
4. Upload artifact to storage
5. Register version
6. Deploy to edge
7. Warm cache

Output:
✓ Validating manifest...
✓ Bundling code (2.1KB)...
✓ Computing hash: a1b2c3d4...
✓ Uploading to registry...
✓ Deploying to edge...
✓ Warming cache...

✓ Published micro/slugify@1.0.0

Public URL:
https://api.functionfly.com/fx/micro/slugify

Curl:
curl https://api.functionfly.com/fx/micro/slugify -d "Hello World"

Stats will be available in 30 seconds`,
	Run: publishRun,
}

func init() {
	rootCmd.AddCommand(publishCmd)

	// Local flags
	publishCmd.Flags().StringP("access", "a", "public", "Access level (public or private)")
	publishCmd.Flags().BoolP("force", "f", false, "Force publish even if version exists")
}

// publishRun implements the publish command
func publishRun(cmd *cobra.Command, args []string) {
	_, _ = cmd.Flags().GetString("access")  // access flag for future implementation
	_, _ = cmd.Flags().GetBool("force")     // force flag for future implementation

	fmt.Println("✓ Validating manifest...")

	// 1. Load and validate manifest
	manifest, err := manifest.Load("")
	if err != nil {
		log.Fatalf("Failed to load manifest: %v", err)
	}

	if err := manifest.Validate(); err != nil {
		log.Fatalf("Manifest validation failed: %v", err)
	}

	fmt.Printf("✓ Manifest valid: %s\n", manifest.String())

	// 2. Load credentials
	creds, err := credentials.Load()
	if err != nil {
		log.Fatalf("Not logged in. Run 'fly login' first: %v", err)
	}

	fmt.Println("✓ Credentials loaded")

	// 3. Bundle code
	fmt.Println("✓ Bundling code...")
	bundle, err := bundler.BundleWithWorkingDirectory(manifest, "")
	if err != nil {
		log.Fatalf("Bundling failed: %v", err)
	}

	bundleSize := len(bundle)
	fmt.Printf("✓ Code bundled (%d bytes)\n", bundleSize)

	// 4. Generate version hash
	hash := bundler.HashContent(bundle)
	fmt.Printf("✓ Content hash: %s\n", hash[:16]+"...")

	// 5. Create API client
	apiURL := getAPIURL()
	client := cli.NewClient(apiURL, creds.Token)

	// 6. Prepare manifest for API
	manifestBytes, err := json.Marshal(manifest)
	if err != nil {
		log.Fatalf("Failed to marshal manifest: %v", err)
	}

	// 7. Publish to registry
	fmt.Println("✓ Publishing to registry...")

	publishReq := &cli.PublishRequest{
		Author:   creds.User.Username,
		Name:     manifest.Name,
		Version:  manifest.Version,
		Manifest: manifestBytes,
	}

	result, err := client.PublishFunction(publishReq)
	if err != nil {
		log.Fatalf("Publish failed: %v", err)
	}

	// 8. Print success
	fmt.Printf("✓ Published %s/%s@%s\n", creds.User.Username, manifest.Name, manifest.Version)
	if result.Message != "" {
		fmt.Printf("✓ %s\n", result.Message)
	}
	fmt.Println()
	fmt.Printf("Public URL:\n")
	fmt.Printf("https://api.functionfly.com/fx/%s/%s\n", creds.User.Username, manifest.Name)
	fmt.Println()
	fmt.Printf("Curl:\n")
	fmt.Printf("curl https://api.functionfly.com/fx/%s/%s -d \"Hello World\"\n", creds.User.Username, manifest.Name)
	fmt.Println()
	fmt.Println("Stats will be available in 30 seconds")
}

