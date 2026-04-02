package commands

import (
	"fmt"

	"github.com/spf13/cobra"
)

func NewLogoutCmd() *cobra.Command {
	var force bool
	cmd := &cobra.Command{
		Use:     "logout",
		Short:   "Clear stored credentials and log out",
		Example: "  ffly logout\n  ffly logout --force",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runLogout(force)
		},
	}
	cmd.Flags().BoolVar(&force, "force", false, "Skip confirmation prompt")
	return cmd
}

func runLogout(force bool) error {
	creds, err := LoadCredentials()
	if err != nil {
		fmt.Println("You are not currently logged in.")
		return nil
	}
	if !force && IsInteractive() {
		confirmed := PromptConfirm(fmt.Sprintf("Log out %s?", creds.User.Username), false)
		if !confirmed {
			fmt.Println("Logout cancelled.")
			return nil
		}
	}
	if err := DeleteCredentials(); err != nil {
		return fmt.Errorf("could not remove credentials: %w", err)
	}
	fmt.Printf("✅ Logged out %s\n", creds.User.Username)
	fmt.Println("   Run 'ffly login' to authenticate again.")
	return nil
}
