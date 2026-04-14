package cmd

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/Abhay0thakor/ZetGrep/pkg/models"
	"github.com/Abhay0thakor/ZetGrep/pkg/scanner"
	"github.com/Abhay0thakor/ZetGrep/pkg/utils"
	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List available patterns and tools",
	Run: func(cmd *cobra.Command, args []string) {
		finalCfg, err := models.LoadConfig(configFiles)
		if err != nil {
			slog.Error("Error loading configuration", "error", err)
			os.Exit(1)
		}

		if patternsDir != "" {
			finalCfg.PatternsDir = utils.ExpandPath(patternsDir)
		}
		if toolsDir != "" {
			finalCfg.ToolsDir = utils.ExpandPath(toolsDir)
		}

		svc, err := scanner.NewScannerService(finalCfg)
		if err != nil {
			slog.Error("Service initialization error", "error", err)
			os.Exit(1)
		}

		pats, err := scanner.GetPatterns(svc.Config.PatternsDir)
		if err != nil {
			slog.Warn("Error getting patterns", "error", err)
		} else {
			fmt.Printf("%s Patterns:\n", au.Bold("Available"))
			if len(pats) == 0 {
				fmt.Println("  (none found)")
			}
			for _, p := range pats {
				fmt.Printf("  - %s\n", p)
			}
		}

		fmt.Printf("\n%s Tools:\n", au.Bold("Available"))
		if len(svc.Tools) == 0 {
			fmt.Println("  (none found)")
		}
		for _, t := range svc.Tools {
			fmt.Printf("  - %s: %s\n", au.Cyan(t.ID), t.Description)
		}
	},
}

func init() {
	rootCmd.AddCommand(listCmd)
}
