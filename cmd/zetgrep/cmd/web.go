package cmd

import (
	"log/slog"
	"os"

	"github.com/Abhay0thakor/ZetGrep/pkg/api"
	"github.com/Abhay0thakor/ZetGrep/pkg/models"
	"github.com/Abhay0thakor/ZetGrep/pkg/scanner"
	"github.com/spf13/cobra"
)

var webAddr string

var webCmd = &cobra.Command{
	Use:   "web",
	Short: "Start the web dashboard",
	Run: func(cmd *cobra.Command, args []string) {
		finalCfg, err := models.LoadConfig(configFiles)
		if err != nil {
			slog.Error("Error loading configuration", "error", err)
			os.Exit(1)
		}
		svc, err := scanner.NewScannerService(finalCfg)
		if err != nil {
			slog.Error("Service initialization error", "error", err)
			os.Exit(1)
		}
		srv := api.NewServer(webAddr, svc)
		if err := srv.Start(); err != nil {
			slog.Error("Web Dashboard Error", "error", err)
			os.Exit(1)
		}
	},
}

func init() {
	rootCmd.AddCommand(webCmd)
	webCmd.Flags().StringVarP(&webAddr, "listen", "l", ":8080", "address to listen on")
}
