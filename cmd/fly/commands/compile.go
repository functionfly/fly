/*
Copyright © 2026 FunctionFly
*/
package commands

import (
	"github.com/spf13/cobra"
)

// compileCmd represents the compile command
var compileCmd = &cobra.Command{
	Use:   "compile",
	Short: "Compile functions to various formats",
	Long: `Compile functions to different output formats.

Supported compilers:
  python    Compile Python functions to WebAssembly (WASM)
  rust      Compile Rust functions to WebAssembly (WASM)`,
	SilenceUsage: true,
}

// CompileCmd returns the compile command for attachment to the fly CLI
func CompileCmd() *cobra.Command {
	return compileCmd
}

func init() {
	// Add python subcommand (wraps flypy-go)
	compileCmd.AddCommand(newCompilePythonCmd())

	// Add rust subcommand
	compileCmd.AddCommand(newCompileRustCmd())
}
