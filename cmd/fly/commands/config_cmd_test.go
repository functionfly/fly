package commands

import (
	"bytes"
	"testing"

	"github.com/spf13/cobra"
)

func TestNewConfigCmd_HasShowAlias(t *testing.T) {
	cmd := NewConfigCmd()

	// Check that both "view" and "show" subcommands exist
	viewCmd := findSubcommand(cmd, "view")
	if viewCmd == nil {
		t.Fatal("config should have 'view' subcommand")
	}

	showCmd := findSubcommand(cmd, "show")
	if showCmd == nil {
		t.Fatal("config should have 'show' subcommand (alias)")
	}

	if !showCmd.Hidden {
		t.Error("'show' subcommand should be hidden")
	}
}

func TestConfigView_RunsWithoutError(t *testing.T) {
	cmd := NewConfigCmd()
	out := bytes.NewBuffer(nil)
	cmd.SetOut(out)
	cmd.SetArgs([]string{"view"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("config view returned error: %v", err)
	}
}

func TestConfigShow_RunsWithoutError(t *testing.T) {
	cmd := NewConfigCmd()
	out := bytes.NewBuffer(nil)
	cmd.SetOut(out)
	cmd.SetArgs([]string{"show"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("config show returned error: %v", err)
	}
}

func findSubcommand(parent *cobra.Command, name string) *cobra.Command {
	for _, c := range parent.Commands() {
		if c.Name() == name {
			return c
		}
	}
	return nil
}
