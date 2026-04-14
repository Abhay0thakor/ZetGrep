package cmd

import (
	"github.com/spf13/cobra"
)

// scanCmd is now just an alias for the root command's logic.
// Since all flags are now on the root command, we just need to delegate.
var scanCmd = &cobra.Command{
	Use:   "scan [pattern] [targets...]",
	Short: "Run a scan (alias for default behavior)",
	Run: func(cmd *cobra.Command, args []string) {
		rootCmd.Run(rootCmd, args)
	},
}

func init() {
	rootCmd.AddCommand(scanCmd)
}
