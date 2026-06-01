package cmd

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/Abhay0thakor/ZetGrep/pkg/models"
	"github.com/spf13/cobra"
)

var deltaCmd = &cobra.Command{
	Use:   "delta [old.json] [new.json]",
	Short: "Compare two scan result files and show only NEW findings",
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		oldPath := args[0]
		newPath := args[1]

		oldResults, err := loadResults(oldPath)
		if err != nil {
			slog.Error("Error loading old results", "path", oldPath, "error", err)
			os.Exit(1)
		}

		newResults, err := loadResults(newPath)
		if err != nil {
			slog.Error("Error loading new results", "path", newPath, "error", err)
			os.Exit(1)
		}

		// Create a map for fast lookup
		seen := make(map[string]bool)
		for _, res := range oldResults {
			key := fmt.Sprintf("%s:%s", res.Pattern, res.Content)
			seen[key] = true
		}

		newCount := 0
		fmt.Printf("\n%s Finding deltas between %s and %s...\n", au.Cyan("[*]"), oldPath, newPath)
		fmt.Println(strings.Repeat("─", 80))

		for _, res := range newResults {
			key := fmt.Sprintf("%s:%s", res.Pattern, res.Content)
			if !seen[key] {
				newCount++
				matchPrefix := fmt.Sprintf("[%s] %s:%d", au.Bold(au.Green("NEW:"+res.Pattern)), au.Cyan(res.File), res.Line)
				fmt.Printf("%s\n  ➜ %s\n", matchPrefix, au.White(res.Content))
				for _, td := range res.ToolData {
					fmt.Printf("    %s %s: %s\n", au.Gray(15, "└"), au.Magenta(td.Label), au.White(td.Value))
				}
			}
		}

		fmt.Println(strings.Repeat("─", 80))
		fmt.Printf("✔ Summary: %s new findings identified.\n\n", au.Bold(fmt.Sprintf("%d", newCount)))
	},
}

func loadResults(path string) ([]*models.Result, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var results []*models.Result
	if err := json.Unmarshal(b, &results); err != nil {
		return nil, err
	}
	return results, nil
}

func init() {
	rootCmd.AddCommand(deltaCmd)
}
