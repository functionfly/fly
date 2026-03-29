package commands

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

func NewEnvCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "env",
		Short:   "Manage environment variables",
		Example: "  fly env list\n  fly env set KEY=value\n  fly env get KEY\n  fly env unset KEY",
	}
	cmd.AddCommand(newEnvListCmd(), newEnvSetCmd(), newEnvGetCmd(), newEnvUnsetCmd())
	return cmd
}

func newEnvListCmd() *cobra.Command {
	var asJSON bool
	cmd := &cobra.Command{
		Use: "list", Aliases: []string{"ls"}, Short: "List all environment variables",
		RunE: func(cmd *cobra.Command, args []string) error { return runEnvList(asJSON) },
	}
	cmd.Flags().BoolVar(&asJSON, "json", false, "Output as JSON")
	return cmd
}

func newEnvSetCmd() *cobra.Command {
	return &cobra.Command{
		Use: "set KEY=value [KEY=value ...]", Short: "Set one or more environment variables",
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error { return runEnvSet(args) },
	}
}

func newEnvGetCmd() *cobra.Command {
	return &cobra.Command{
		Use: "get KEY", Short: "Get the value of an environment variable",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error { return runEnvGet(args[0]) },
	}
}

func newEnvUnsetCmd() *cobra.Command {
	return &cobra.Command{
		Use: "unset KEY [KEY ...]", Aliases: []string{"delete", "rm"}, Short: "Remove one or more environment variables",
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error { return runEnvUnset(args) },
	}
}

func runEnvList(asJSON bool) error {
	manifest, err := LoadManifest("")
	if err != nil {
		return err
	}
	creds, err := LoadCredentials()
	if err != nil {
		return err
	}
	client, err := NewAPIClient()
	if err != nil {
		return err
	}
	var envVars map[string]string
	path := fmt.Sprintf("/v1/registry/%s/%s/env", creds.User.Username, manifest.Name)
	if err := client.Get(path, &envVars); err != nil {
		return fmt.Errorf("could not fetch environment variables: %w", err)
	}
	if asJSON {
		printJSON(envVars)
		return nil
	}
	if len(envVars) == 0 {
		fmt.Println("No environment variables set.")
		fmt.Println("   → Use: fly env set KEY=value")
		return nil
	}
	fmt.Printf("Environment variables for %s/%s:\n\n", creds.User.Username, manifest.Name)
	for k := range envVars {
		fmt.Printf("  %s\n", k)
	}
	fmt.Printf("\n%d variable(s) — values hidden; use 'fly env get KEY' to view a specific value\n", len(envVars))
	return nil
}

func runEnvSet(pairs []string) error {
	manifest, err := LoadManifest("")
	if err != nil {
		return err
	}
	creds, err := LoadCredentials()
	if err != nil {
		return err
	}
	envVars := map[string]string{}
	for _, pair := range pairs {
		parts := strings.SplitN(pair, "=", 2)
		if len(parts) != 2 {
			return fmt.Errorf("invalid format %q — expected KEY=value", pair)
		}
		key := strings.TrimSpace(parts[0])
		if key == "" {
			return fmt.Errorf("empty key in %q", pair)
		}
		envVars[key] = parts[1]
	}
	client, err := NewAPIClient()
	if err != nil {
		return err
	}
	path := fmt.Sprintf("/v1/registry/%s/%s/env", creds.User.Username, manifest.Name)
	if err := client.Put(path, envVars, nil); err != nil {
		return fmt.Errorf("could not set environment variables: %w", err)
	}
	for k := range envVars {
		fmt.Printf("  %s (set)\n", k)
	}
	return nil
}

func runEnvGet(key string) error {
	manifest, err := LoadManifest("")
	if err != nil {
		return err
	}
	creds, err := LoadCredentials()
	if err != nil {
		return err
	}
	client, err := NewAPIClient()
	if err != nil {
		return err
	}
	var envVars map[string]string
	path := fmt.Sprintf("/v1/registry/%s/%s/env", creds.User.Username, manifest.Name)
	if err := client.Get(path, &envVars); err != nil {
		return fmt.Errorf("could not fetch environment variables: %w", err)
	}
	value, ok := envVars[key]
	if !ok {
		return fmt.Errorf("environment variable %q not found\n   → Use 'fly env list' to see all variables", key)
	}
	fmt.Println(value)
	return nil
}

func runEnvUnset(keys []string) error {
	manifest, err := LoadManifest("")
	if err != nil {
		return err
	}
	creds, err := LoadCredentials()
	if err != nil {
		return err
	}
	client, err := NewAPIClient()
	if err != nil {
		return err
	}
	for _, key := range keys {
		path := fmt.Sprintf("/v1/registry/%s/%s/env/%s", creds.User.Username, manifest.Name, key)
		if err := client.Delete(path, nil); err != nil {
			return fmt.Errorf("could not unset %s: %w", key, err)
		}
		fmt.Printf("✅ Unset %s\n", key)
	}
	return nil
}
