package cmd

import (
	"log/slog"
	"os"
	"os/exec"

	"github.com/spf13/cobra"
)

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update ZetGrep to the latest version",
	Run: func(cmd *cobra.Command, args []string) {
		slog.Info("Updating...")
		c := exec.Command("go", "install", "-v", "github.com/Abhay0thakor/ZetGrep/cmd/zetgrep@latest")
		c.Env = append(os.Environ(), "GOPROXY=direct")
		c.Stdout, c.Stderr = os.Stdout, os.Stderr
		if err := c.Run(); err == nil {
			slog.Info("Success!")
		} else {
			slog.Error("Update failed", "error", err)
			os.Exit(1)
		}
	},
}

func init() {
	rootCmd.AddCommand(updateCmd)
}
