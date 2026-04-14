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

var diagLine string

var diagCmd = &cobra.Command{
	Use:   "diagnose",
	Short: "Diagnose a single line against patterns",
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

		pats := args
		if len(pats) == 0 {
			pats, _ = scanner.GetPatterns(svc.Config.PatternsDir)
		}

		for _, l := range svc.DiagnoseLine(diagLine, pats) {
			fmt.Println(l)
		}
	},
}

func init() {
	rootCmd.AddCommand(diagCmd)
	diagCmd.Flags().StringVarP(&diagLine, "line", "l", "", "line to diagnose")
	diagCmd.MarkFlagRequired("line")
}
