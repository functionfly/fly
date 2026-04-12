package commands

import (
	"fmt"

	"github.com/functionfly/fly/internal/version"
	"github.com/spf13/cobra"
)

func NewRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:   "ffly",
		Short: "FunctionFly CLI — publish functions to the global edge",
		Long: `ffly is the FunctionFly developer CLI.

Go from idea → global API in under 60 seconds.

  ffly login              Authenticate with FunctionFly
  ffly init <name>        Scaffold a new function project
  ffly dev                Run function locally
  ffly publish            Publish function to the registry
  ffly deploy --env       Publish and promote to staging or production
  ffly deploy --canary N  Publish and start a canary at N% traffic
  ffly canary             Manage canary deployments
  ffly test               Test your deployed function
  ffly health             Check deployed function health
  ffly update <bump>      Bump function version
  ffly stats              View usage statistics
  ffly logs               Stream live execution logs
  ffly rollback           Roll back to a previous version
  ffly env                Manage environment variables
  ffly secrets            Manage secrets
  ffly whoami             Show current logged-in user
  ffly logout             Clear stored credentials
  ffly completion         Generate shell completion scripts`,
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	// Add --version flag (Cobra's built-in version support)
	root.Version = version.Short()

	// Add persistent flags for debug/verbose/trace modes
	// These are available to all subcommands
	root.PersistentFlags().BoolVar(&DebugMode, "debug", false, "Enable full debug output")
	root.PersistentFlags().BoolVarP(&VerboseMode, "verbose", "v", false, "Enable verbose API calls")
	root.PersistentFlags().BoolVar(&TraceMode, "trace", false, "Enable HTTP trace with request/response bodies")
	root.PersistentFlags().StringVarP(&OutputFormat, "format", "m", "table", "Output format: table, json")
	root.PersistentFlags().BoolVarP(&YesMode, "yes", "y", false, "Skip all confirmation prompts and answer yes automatically")

	// Set up persistent pre-run to handle debug mode
	root.PersistentPreRun = func(cmd *cobra.Command, args []string) {
		if DebugMode {
			Debug("Debug mode enabled")
			Debug("Version: %s", version.Info())
		}
	}

	// Add version command
	root.AddCommand(NewVersionCmd())

	root.AddCommand(
		NewLoginCmd(),
		NewWhoamiCmd(),
		NewLogoutCmd(),
		NewAuthRefreshCmd(),
		NewConfigCmd(),
		NewSelfUpdateCmd(),
		NewInitCmd(),
		NewDevCmd(),
		NewPublishCmd(),
		NewDeployCmd(),
		NewPublishBatchCmd(),
		NewManifestCmd(),
		NewTestCmd(),
		NewUpdateCmd(),
		NewStatsCmd(),
		NewLogsCmd(),
		NewRollbackCmd(),
		NewHealthCmd(),
		NewCanaryCmd(),
		NewEnvCmd(),
		NewSecretsCmd(),
		NewScheduleCmd(),
		NewDreCmd(),
		NewCompletionCmd(root),
		NewCompletionsAliasCmd(root),
		NewDoctorCmd(),
		NewChangelogCmd(),
		BackendCmd(),
		FlypyCmd(),
		CompileCmd(),
	)

	return root
}

// NewVersionCmd creates the version command.
func NewVersionCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "version",
		Short: "Show the ffly CLI version",
		Long: `Show version information for the ffly CLI.

This displays the semantic version, git commit hash, and build date.
Use this to verify which version of ffly you have installed.`,
		Example: `  ffly version
  ffly version --short  # Show only version number
  ffly version --json   # Output as JSON`,
		Run: func(cmd *cobra.Command, args []string) {
			short, _ := cmd.Flags().GetBool("short")
			asJSON, _ := cmd.Flags().GetBool("json")
			if short {
				fmt.Println(version.Short())
			} else if asJSON {
				printJSON(map[string]interface{}{
					"version": version.Version,
					"commit":  version.Commit,
					"date":    version.Date,
				})
			} else {
				PrintVersion()
			}
		},
	}

	cmd.Flags().Bool("short", false, "Show only version number")
	cmd.Flags().Bool("json", false, "Output as JSON")

	return cmd
}

// GetVersion returns the current CLI version string.
