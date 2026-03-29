package commands

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"

	"github.com/spf13/cobra"
)

const configHelpLong = `Manage global fly CLI configuration.

Configuration precedence (highest first):
  1. Environment variables (FFLY_*)
  2. Global config file (~/.functionfly/config.yaml)
  3. Defaults

Environment variables (override config file):
  FFLY_API_URL       API base URL (e.g. https://api.functionfly.com or http://localhost:8080)
  FFLY_API_TIMEOUT  Request timeout (e.g. 30s)
  FFLY_DEV_EMAIL    Email for dev login (fly login --dev)
  FFLY_DEV_PASSWORD Password for dev login
  FFLY_DEV_LOGIN=1  Force dev email/password login
  FFLY_TOKEN        Bearer token (overrides stored credentials)
  FFLY_TELEMETRY    Set to 0, false, or no to disable telemetry
  FFLY_CONFIG       Path to config file (overrides ~/.functionfly/config.yaml)

Use "fly config" or "fly config view" to show current config and path.
Use "fly config reset" to restore defaults (removes or overwrites config file).`

// NewConfigCmd returns the config command and its subcommands (view, reset).
func NewConfigCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "View or reset global CLI configuration",
		Long:  configHelpLong,
		Example: `  fly config
  fly config view
  fly config reset`,
		RunE: runConfigView, // "fly config" with no subcommand runs view
	}
	cmd.AddCommand(newConfigViewCmd(), newConfigResetCmd())
	return cmd
}

func newConfigViewCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "view",
		Short: "Show config file path and current configuration",
		Long:  "Prints the path to the global config file and its contents (or 'using defaults' if the file does not exist). Environment overrides are applied when the CLI runs; this shows the merged view from file + env.",
		RunE:  runConfigView,
	}
}

func runConfigView(cmd *cobra.Command, args []string) error {
	path, err := ConfigPath()
	if err != nil {
		return NewCLIError(err, ExitCodeConfigError, fmt.Sprintf("could not determine config path: %v", err))
	}
	cfg, err := LoadConfig()
	if err != nil {
		return err
	}
	fmt.Println("Config path:", path)
	if _, err := os.Stat(path); os.IsNotExist(err) {
		fmt.Println("(file does not exist — using defaults)")
	}
	fmt.Println()
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}
	fmt.Println(string(data))
	return nil
}

func newConfigResetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "reset",
		Short: "Reset config to defaults",
		Long:  "Writes default configuration to the config file (or removes it so defaults are used). Prints the config path.",
		RunE:  runConfigReset,
	}
}

func runConfigReset(cmd *cobra.Command, args []string) error {
	path, err := ConfigPath()
	if err != nil {
		return NewCLIError(err, ExitCodeConfigError, fmt.Sprintf("could not determine config path: %v", err))
	}
	if err := SaveConfig(DefaultConfig()); err != nil {
		return NewCLIError(err, ExitCodeConfigError, fmt.Sprintf("could not reset config: %v\n   → Check permissions or FFLY_CONFIG", err))
	}
	fmt.Printf("Config reset to defaults.\nConfig file: %s\n", path)
	return nil
}
