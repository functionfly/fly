/*
Copyright © 2026 FunctionFly

*/
package cmd

import (
	"os"

	"github.com/spf13/cobra"
)



// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "fly",
	Short: "FunctionFly CLI - Go from idea to global API in under 60 seconds",
	Long: `The fly CLI enables developers to go from idea → global API in under 60 seconds
with zero infrastructure configuration.

Publishing must feel easier than writing a README. No Docker, no infra config,
no account dashboard required. If developers think about infrastructure → we failed.

Core Commands:
  login     Authenticate with GitHub or Google OAuth
  init      Create a runnable function instantly with templates
  dev       Run local execution environment identical to production
  publish   Publish function to global registry with automatic infrastructure
  deploy    Unified deployment with advanced features and environments
  test      Enhanced testing with local validation and benchmarking
  update    Safely bump version without overwriting
  stats     Provides immediate feedback on function usage
  compile   Compile functions to WebAssembly (python, rust)

Enhanced Commands:
  test local    Run comprehensive local function tests
  test validate Validate function configuration and dependencies
  test bench    Performance benchmarking and load testing
  deploy status Check deployment status and health
  deploy logs   View deployment logs and events
  logs          View function execution logs with filtering
  metrics       Detailed performance metrics and analytics
  health        Check system and function health status

Example:
  fly login
  fly init slugify
  fly dev
  fly compile rust -i ./Cargo.toml -o ./dist
  fly publish
  fly test
  fly stats`,
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.

	// Add compile command
	rootCmd.AddCommand(compileCmd)

	// rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.fly.yaml)")
}


