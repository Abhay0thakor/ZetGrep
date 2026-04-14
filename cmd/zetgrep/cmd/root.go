package cmd

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Abhay0thakor/ZetGrep/pkg/models"
	"github.com/Abhay0thakor/ZetGrep/pkg/scanner"
	"github.com/Abhay0thakor/ZetGrep/pkg/utils"
	"github.com/logrusorgru/aurora"
	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var (
	version = "v0.5.0"
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

	// Global / Scan flags
	verbose        bool
	silent         bool
	noColor        bool
	configFiles    []string
	patternsDir    string
	toolsDir       string
	inputConfigs   []string
	listFile       string
	stdin          bool
	inputMode      string
	toolFiles      []string
	allMode        bool
	uniqueMode     bool
	smartMode      bool
	entropyMode    bool
	tags           []string
	jsonMode       bool
	reportMode     bool
	outputFile     string
	outputTemplate string
	toolIDs        string
	resumeFile     string
	processFile    string
	concurrency    int
	dryRun         bool
	format         string
	targetField    string
	targetFields   []string
	csvSeparator   string
	csvNoHeader    bool
	csvIDIndex     int
	csvTargetIndex []int
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version number of ZetGrep",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("ZetGrep version: %s\n", version)
	},
}

var rootCmd = &cobra.Command{
	Use:   "zetgrep [pattern] [targets...]",
	Short: "ZetGrep - A grep-like tool with superpowers",
	Long:  banner + "\nZetGrep is a pattern matching tool that supports multiple input formats (JSONL, CSV, Text) and workflow tool chaining.",
	Args:  cobra.ArbitraryArgs,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		if noColor {
			au = aurora.NewAurora(false)
		}
		utils.InitLogger(verbose, silent)
		if !silent {
			showBanner()
		}
	},
	Run: func(cmd *cobra.Command, args []string) {
		// If no pattern provided and not in special modes, show help
		if len(args) == 0 && !allMode && processFile == "" {
			cmd.Help()
			return
		}

		if len(configFiles) == 0 {
			def := utils.ExpandPath("~/.config/gf/config.yaml")
			if _, err := os.Stat(def); err == nil {
				configFiles = append(configFiles, def)
			}
		}

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

		if targetField != "" {
			finalCfg.Input.Target = targetField
		}
		if len(targetFields) > 0 {
			finalCfg.Input.Targets = append(finalCfg.Input.Targets, targetFields...)
		}

		if csvSeparator != "" {
			finalCfg.Input.CSVConfig.Separator = csvSeparator
		}
		if csvNoHeader {
			finalCfg.Input.CSVConfig.HasHeader = false
		} else {
			finalCfg.Input.CSVConfig.HasHeader = true
		}
		if csvIDIndex != 0 {
			finalCfg.Input.CSVConfig.IDIndex = csvIDIndex
		}
		if len(csvTargetIndex) > 0 {
			finalCfg.Input.CSVConfig.TargetIdx = csvTargetIndex
		}

		for _, ic := range inputConfigs {
			ic = utils.ExpandPath(ic)
			var inc models.InputConfig
			b, err := os.ReadFile(ic)
			if err != nil {
				slog.Error("Error reading input config", "path", ic, "error", err)
				continue
			}
			if err := yaml.Unmarshal(b, &inc); err != nil {
				slog.Error("Error parsing input config", "path", ic, "error", err)
				continue
			}
			mergeInputConfigs(&finalCfg.Input, inc)
		}

		if inputMode != "" {
			finalCfg.Input.Format = inputMode
		} else if len(args) > 1 {
			ext := strings.ToLower(filepath.Ext(args[1]))
			if ext == ".jsonl" || ext == ".json" {
				finalCfg.Input.Format = "jsonl"
			} else if ext == ".csv" {
				finalCfg.Input.Format = "csv"
			} else {
				finalCfg.Input.Format = "text"
			}
		}

		svc, err := scanner.NewScannerService(finalCfg)
		if err != nil {
			slog.Error("Service initialization error", "error", err)
			os.Exit(1)
		}
		for _, tf := range toolFiles {
			if t, err := scanner.LoadToolFromFile(tf); err == nil {
				svc.Tools = append(svc.Tools, t)
			} else {
				slog.Error("Error loading tool", "path", tf, "error", err)
			}
		}

		var targets []string
		if stdin {
			targets = []string{"stdin"}
		} else if listFile != "" {
			f, err := os.Open(utils.ExpandPath(listFile))
			if err != nil {
				slog.Error("Error opening list file", "path", listFile, "error", err)
				os.Exit(1)
			}
			s := bufio.NewScanner(f)
			for s.Scan() {
				targets = append(targets, utils.ExpandPath(s.Text()))
			}
			f.Close()
		} else if len(args) > 1 {
			for _, arg := range args[1:] {
				targets = append(targets, utils.ExpandPath(arg))
			}
		}
		if len(targets) == 0 {
			targets = []string{"."}
		}

		if resumeFile != "" {
			if err := svc.LoadResumeState(resumeFile); err != nil {
				slog.Error("Error loading resume state", "path", resumeFile, "error", err)
			} else {
				slog.Info("Resuming", "file_index", svc.Resume.FileIndex, "line_index", svc.Resume.LineIndex)
			}
		}

		var runPats []string
		if allMode {
			var err error
			runPats, err = scanner.GetPatterns(svc.Config.PatternsDir)
			if err != nil {
				slog.Error("Error getting patterns", "error", err)
				os.Exit(1)
			}
		} else if len(tags) > 0 {
			runPats = svc.FilterPatternsByTag(tags)
		} else if len(args) > 0 {
			runPats = []string{args[0]}
		}

		var activeToolIDs []string
		if toolIDs != "" {
			activeToolIDs = strings.Split(toolIDs, ",")
		}

		if dryRun {
			slog.Info("Dry-run mode enabled. Scanning would proceed with:",
				"targets", targets,
				"patterns", runPats,
				"tools", activeToolIDs,
				"concurrency", concurrency)
			return
		}

		ctx := context.Background()
		var resultChan <-chan *models.Result
		var scanErr error

		if processFile != "" {
			resultChan, scanErr = svc.ProcessResults(ctx, processFile, activeToolIDs)
		} else {
			resultChan, scanErr = svc.RunScan(ctx, scanner.ScannerOptions{
				TargetPaths: targets, Patterns: runPats, Tags: tags, ToolIDs: activeToolIDs,
				SmartMode: smartMode, EntropyMode: entropyMode, Unique: uniqueMode, ResumeFile: resumeFile, Silent: silent,
				Concurrency: concurrency,
			})
		}

		if scanErr != nil {
			slog.Error("Initialization error", "error", scanErr)
			os.Exit(1)
		}

		outputResults(resultChan)
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
	// Global flags
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "verbose mode")
	rootCmd.PersistentFlags().BoolVar(&silent, "silent", false, "silent mode")
	rootCmd.PersistentFlags().BoolVar(&noColor, "no-color", false, "disable color")
	rootCmd.PersistentFlags().StringSliceVar(&configFiles, "config-file", nil, "path to global config")
	rootCmd.PersistentFlags().StringVar(&patternsDir, "pd", "", "patterns directory")
	rootCmd.PersistentFlags().StringVar(&toolsDir, "td", "", "tools directory")

	// Scan flags (now Persistent so they work on root and subcommands)
	rootCmd.PersistentFlags().StringSliceVar(&inputConfigs, "input-config", nil, "path to input config file (YAML)")
	rootCmd.PersistentFlags().StringVarP(&listFile, "list-file", "l", "", "file containing list of targets")
	rootCmd.PersistentFlags().BoolVar(&stdin, "stdin", false, "read targets from stdin")
	rootCmd.PersistentFlags().StringVar(&inputMode, "im", "", "input mode (jsonl, csv, text)")
	rootCmd.PersistentFlags().StringSliceVar(&toolFiles, "tool", nil, "path to tool YAML")
	rootCmd.PersistentFlags().BoolVar(&allMode, "all", false, "run all patterns")
	rootCmd.PersistentFlags().BoolVarP(&uniqueMode, "unique", "u", false, "deduplicate matches")
	rootCmd.PersistentFlags().BoolVar(&smartMode, "smart", false, "AI interest filtering")
	rootCmd.PersistentFlags().BoolVar(&entropyMode, "entropy", false, "high-entropy filtering")
	rootCmd.PersistentFlags().StringSliceVar(&tags, "tags", nil, "filter by tag")
	rootCmd.PersistentFlags().BoolVar(&jsonMode, "json", false, "output in JSON")
	rootCmd.PersistentFlags().BoolVar(&reportMode, "report", false, "generate markdown report")
	rootCmd.PersistentFlags().StringVarP(&outputFile, "output", "o", "", "output file path")
	rootCmd.PersistentFlags().StringVarP(&outputTemplate, "template", "t", "", "output template")
	rootCmd.PersistentFlags().StringVarP(&toolIDs, "workflow", "w", "", "workflow tool IDs")
	rootCmd.PersistentFlags().StringVar(&resumeFile, "resume", "", "resume scan state")
	rootCmd.PersistentFlags().StringVar(&processFile, "process", "", "process a previously saved JSON results file")
	rootCmd.PersistentFlags().IntVarP(&concurrency, "concurrency", "c", 0, "number of concurrent workers (default: CPU * 2)")
	rootCmd.PersistentFlags().BoolVar(&dryRun, "dry-run", false, "show what would be done without executing")
	rootCmd.PersistentFlags().StringVarP(&format, "format", "f", "text", "output format (text, json, table)")
	rootCmd.PersistentFlags().StringVar(&targetField, "target", "", "target field to scan (JSONL)")
	rootCmd.PersistentFlags().StringSliceVar(&targetFields, "targets", nil, "target fields to scan (JSONL)")
	rootCmd.PersistentFlags().StringVar(&csvSeparator, "csv-sep", ",", "CSV separator")
	rootCmd.PersistentFlags().BoolVar(&csvNoHeader, "csv-no-header", false, "CSV has no header")
	rootCmd.PersistentFlags().IntVar(&csvIDIndex, "csv-id", 0, "CSV ID column index")
	rootCmd.PersistentFlags().IntSliceVar(&csvTargetIndex, "csv-targets", nil, "CSV target column indices")
}

func showBanner() {
	fmt.Fprintf(os.Stderr, "%s\n", au.Bold(au.Cyan(banner)))
	fmt.Fprintf(os.Stderr, "\t\t%s\n\n", au.Faint(version))
}

// Helpers
func mergeInputConfigs(dest *models.InputConfig, src models.InputConfig) {
	if src.Format != "" {
		dest.Format = src.Format
	}
	if src.PreProcess != "" {
		dest.PreProcess = src.PreProcess
	}
	if src.Target != "" {
		dest.Target = src.Target
	}
	if len(src.Targets) > 0 {
		dest.Targets = src.Targets
	}
	if src.ID != "" {
		dest.ID = src.ID
	}
	if src.Decode {
		dest.Decode = true
	}
	if len(src.Filters) > 0 {
		if dest.Filters == nil {
			dest.Filters = make(map[string]string)
		}
		for k, v := range src.Filters {
			dest.Filters[k] = v
		}
	}
	if src.CSVConfig.Separator != "" {
		dest.CSVConfig = src.CSVConfig
	}
	if len(src.PostProcess) > 0 {
		if dest.PostProcess == nil {
			dest.PostProcess = make(map[string]string)
		}
		for k, v := range src.PostProcess {
			dest.PostProcess[k] = v
		}
	}
}

func outputResults(resultChan <-chan *models.Result) {
	if jsonMode || format == "json" {
		fmt.Print("[")
	}

	var table *tablewriter.Table
	if format == "table" {
		table = tablewriter.NewWriter(os.Stdout)
		table.Header("Pattern", "File", "Line", "Content")
	}

	first := true
	var reportFile *os.File
	if reportMode {
		name := fmt.Sprintf("zetgrep_report_%d.md", time.Now().Unix())
		if outputFile != "" && strings.HasSuffix(outputFile, ".md") {
			name = outputFile
		}
		var err error
		reportFile, err = os.Create(name)
		if err != nil {
			slog.Error("Error creating report file", "path", name, "error", err)
		} else {
			fmt.Fprintln(reportFile, "# ZetGrep Intelligence Report")
		}
	}

	var saveFile *os.File
	if outputFile != "" && !reportMode {
		var err error
		saveFile, err = os.Create(outputFile)
		if err != nil {
			slog.Error("Error creating output file", "path", outputFile, "error", err)
		} else {
			if jsonMode {
				fmt.Fprint(saveFile, "[")
			}
		}
	}

	hitCount := 0
	for res := range resultChan {
		hitCount++
		if reportFile != nil {
			fmt.Fprintf(reportFile, "### [%s] %s\n- Content: `%s`\n", res.Pattern, res.File, res.Content)
		}

		formatted := ""
		if jsonMode || format == "json" {
			b, err := json.Marshal(res)
			if err != nil {
				slog.Error("Error marshaling result", "error", err)
				continue
			}
			formatted = string(b)
			if !first {
				fmt.Print(",")
				if saveFile != nil {
					fmt.Fprint(saveFile, ",")
				}
			}
			fmt.Print(formatted)
			if saveFile != nil {
				fmt.Fprint(saveFile, formatted)
			}
		} else if format == "table" {
			table.Append(res.Pattern, res.File, fmt.Sprintf("%d", res.Line), res.Content)
		} else if outputTemplate != "" {
			formatted = formatResult(outputTemplate, res)
			fmt.Println(formatted)
			if saveFile != nil {
				fmt.Fprintln(saveFile, formatted)
			}
		} else if !silent {
			fmt.Printf("[%s] %s:%d: %s\n", au.Yellow(res.Pattern), au.Cyan(res.File), res.Line, res.Content)
			for _, td := range res.ToolData {
				fmt.Printf("   ↳ %s: %s\n", au.Magenta(td.Label), td.Value)
			}
			if saveFile != nil {
				fmt.Fprintf(saveFile, "[%s] %s:%d: %s\n", res.Pattern, res.File, res.Line, res.Content)
				for _, td := range res.ToolData {
					fmt.Fprintf(saveFile, "   ↳ %s: %s\n", td.Label, td.Value)
				}
			}
		} else {
			fmt.Println(res.Content)
			if saveFile != nil {
				fmt.Fprintln(saveFile, res.Content)
			}
		}
		first = false
		scanner.PutResult(res)
	}

	if format == "table" {
		table.Render()
	}

	if jsonMode || format == "json" {
		fmt.Println("]")
		if saveFile != nil {
			fmt.Fprintln(saveFile, "]")
		}
	}

	if saveFile != nil {
		saveFile.Close()
	}
	if reportFile != nil {
		reportFile.Close()
	}
	if !silent {
		slog.Info("Finished", "total_hits", hitCount)
	}
}

func formatResult(tmpl string, res *models.Result) string {
	out := tmpl
	out = strings.ReplaceAll(out, "{{pattern}}", res.Pattern)
	out = strings.ReplaceAll(out, "{{file}}", res.File)
	out = strings.ReplaceAll(out, "{{line}}", fmt.Sprintf("%d", res.Line))
	out = strings.ReplaceAll(out, "{{content}}", res.Content)
	out = strings.ReplaceAll(out, "{{ext}}", res.Ext)
	out = strings.ReplaceAll(out, "{{entropy}}", fmt.Sprintf("%.3f", res.Entropy))
	mainMatch := res.Content
	if len(res.Matches) > 0 {
		mainMatch = res.Matches[0]
	}
	out = strings.ReplaceAll(out, "{{match}}", mainMatch)
	for i, m := range res.Matches {
		out = strings.ReplaceAll(out, fmt.Sprintf("{{match[%d]}}", i), m)
	}
	for _, td := range res.ToolData {
		out = strings.ReplaceAll(out, fmt.Sprintf("{{tool:%s}}", td.ToolID), td.Value)
		out = strings.ReplaceAll(out, fmt.Sprintf("{{tool:%s}}", td.Label), td.Value)
	}
	return out
}
