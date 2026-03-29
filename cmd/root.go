package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "havok-go",
	Short: "Havok physics engine – Go tooling",
	Long: `havok-go provides two sub-commands:

  convert  – parse HavokPhysics.d.ts and generate Go binding scaffolding
  example  – load HavokPhysics.wasm and run a short physics demo

The 'havok' package (./havok/) uses wazero to host the emscripten-compiled
Havok WASM without any system-level native libraries.`,
}

// Execute runs the root command.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
