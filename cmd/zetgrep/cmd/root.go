package cmd

import (
	"fmt"
	"os"

	"github.com/Abhay0thakor/ZetGrep/pkg/utils"
	"github.com/logrusorgru/aurora"
	"github.com/spf13/cobra"
)

var (
	version = "v0.4.6"
	banner  = `
  ______     _   _____                 
 |___  /    | | |  __ \                
    / /  ___| |_| |  \/_ __ ___ _ __  
   / /  / _ \ __| | __| '__/ _ \ '_ \ 
 ./ /__|  __/ |_| |_\ \ | |  __/ |_) |
 \_____/\___|\__|\____/_|  \___| .__/ 
                               | |    
                               |_|    
`
	au = aurora.NewAurora(true)

	// Global flags
	verbose     bool
	silent      bool
	noColor     bool
	configFiles []string
	patternsDir string
	toolsDir    string
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version number of ZetGrep",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("ZetGrep version: %s\n", version)
	},
}

var rootCmd = &cobra.Command{
	Use:   "zetgrep",
	Short: "ZetGrep - A grep-like tool with superpowers",
	Long:  banner + "\nZetGrep is a pattern matching tool that supports multiple input formats (JSONL, CSV, Text) and workflow tool chaining.",
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		if noColor {
			au = aurora.NewAurora(false)
		}
		utils.InitLogger(verbose, silent)
		if !silent {
			showBanner()
		}
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.AddCommand(versionCmd)
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "verbose mode")
	rootCmd.PersistentFlags().BoolVar(&silent, "silent", false, "silent mode")
	rootCmd.PersistentFlags().BoolVar(&noColor, "no-color", false, "disable color")
	rootCmd.PersistentFlags().StringSliceVar(&configFiles, "config-file", nil, "path to global config")
	rootCmd.PersistentFlags().StringVar(&patternsDir, "pd", "", "patterns directory")
	rootCmd.PersistentFlags().StringVar(&toolsDir, "td", "", "tools directory")
}

func showBanner() {
	fmt.Fprintf(os.Stderr, "%s\n", au.Bold(au.Cyan(banner)))
	fmt.Fprintf(os.Stderr, "\t\t%s\n\n", au.Faint(version))
}
