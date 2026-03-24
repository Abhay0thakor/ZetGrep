package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/logrusorgru/aurora"
	"github.com/Abhay0thakor/ZetGrep/pkg/api"
	"github.com/Abhay0thakor/ZetGrep/pkg/models"
	"github.com/Abhay0thakor/ZetGrep/pkg/scanner"
	"gopkg.in/yaml.v3"
)

const version = "v0.1.6"

var (
	au = aurora.NewAurora(true)
)

// multiFlag allows multiple occurrences of a flag
type multiFlag []string

func (m *multiFlag) String() string {
	return strings.Join(*m, ",")
}

func (m *multiFlag) Set(value string) error {
	*m = append(*m, value)
	return nil
}

func main() {
	var (
		listMode, versionMode, smartMode, entropyMode, allMode, jsonMode, configMode, updateMode, reportMode bool
		webMode, toolIDs, outputTemplate, processFile, diagnoseLine string
		configFiles, inputConfigs, toolFiles multiFlag
	)
	flag.BoolVar(&listMode, "list", false, "list patterns and tools")
	flag.BoolVar(&versionMode, "version", false, "show version")
	flag.BoolVar(&updateMode, "update", false, "update zetgrep to the latest version")
	flag.BoolVar(&reportMode, "report", false, "generate a markdown intelligence report")
	flag.BoolVar(&smartMode, "smart", false, "use AI filtering")
	flag.BoolVar(&entropyMode, "entropy", false, "filter by high entropy")
	flag.BoolVar(&allMode, "all", false, "run all patterns")
	flag.BoolVar(&jsonMode, "json", false, "output in JSON")
	flag.BoolVar(&configMode, "config", false, "show configuration paths")
	flag.StringVar(&toolIDs, "tools", "", "comma-separated tool IDs to run")
	flag.StringVar(&webMode, "web", "", "start web ui (e.g. :8080)")
	flag.Var(&configFiles, "config-file", "path to config file (JSON or YAML, multiple allowed)")
	flag.Var(&inputConfigs, "input-config", "path to input config file (YAML, multiple allowed)")
	flag.Var(&toolFiles, "tool", "path to individual tool YAML file (multiple allowed)")
	flag.StringVar(&outputTemplate, "o", "", "output template (e.g. [{{pattern}}] {{file}}:{{match}})")
	flag.StringVar(&processFile, "process", "", "path to results.json to re-process")
	flag.StringVar(&diagnoseLine, "diagnose", "", "test a single line of data against the current config")
	flag.Parse()

	if updateMode {
		fmt.Printf("Updating %s to the latest version...\n", au.Bold(au.Cyan("ZetGrep")))
		cmd := exec.Command("go", "install", "github.com/Abhay0thakor/ZetGrep/cmd/zetgrep@latest")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			fmt.Fprintf(os.Stderr, "Update failed: %v\n", err)
			os.Exit(1)
		}
		fmt.Println(au.Green("ZetGrep updated successfully!"))
		return
	}

	if versionMode {
		fmt.Printf("ZetGrep version %s\n", version)
		return
	}

	// Merge multiple global configs
	if len(configFiles) == 0 {
		home, _ := os.UserHomeDir()
		defaultConfig := filepath.Join(home, ".config", "gf", "config.yaml")
		if _, err := os.Stat(defaultConfig); err == nil {
			configFiles = append(configFiles, defaultConfig)
		}
	}

	var finalCfg models.Config
	for _, cf := range configFiles {
		var cfg models.Config
		b, err := os.ReadFile(cf)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading %s: %v\n", cf, err)
			continue
		}
		if strings.HasSuffix(cf, ".yaml") || strings.HasSuffix(cf, ".yml") {
			yaml.Unmarshal(b, &cfg)
		} else {
			json.Unmarshal(b, &cfg)
		}
		mergeConfigs(&finalCfg, cfg)
	}

	// Merge multiple input configs (last one wins for specific fields)
	for _, ic := range inputConfigs {
		var inc models.InputConfig
		b, err := os.ReadFile(ic)
		if err != nil {
			continue
		}
		yaml.Unmarshal(b, &inc)
		if inc.Format != "" { finalCfg.Input.Format = inc.Format }
		if inc.Target != "" { finalCfg.Input.Target = inc.Target }
		if len(inc.Targets) > 0 { finalCfg.Input.Targets = inc.Targets }
		if inc.ID != "" { finalCfg.Input.ID = inc.ID }
		if len(inc.Filters) > 0 { finalCfg.Input.Filters = inc.Filters }
		if inc.CSVConfig.Separator != "" { finalCfg.Input.CSVConfig = inc.CSVConfig }
		finalCfg.Input.Decode = finalCfg.Input.Decode || inc.Decode
	}

	svc, err := scanner.NewScannerService(finalCfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	// Load individual tool files
	for _, tf := range toolFiles {
		if t, err := scanner.LoadToolFromFile(tf); err == nil {
			svc.Tools = append(svc.Tools, t)
		}
	}

	if configMode {
		fmt.Printf("Patterns: %s\nTools: %s\nJSONL: %+v\n", au.Cyan(svc.Config.PatternsDir), au.Cyan(svc.Config.ToolsDir), svc.Config.Input)
		return
	}

	if webMode != "" {
		srv := api.NewServer(webMode, svc)
		if err := srv.Start(); err != nil {
			os.Exit(1)
		}
		return
	}

	if diagnoseLine != "" {
		fmt.Printf("%s Starting deep diagnostic...\n\n", au.Bold(au.Magenta("[DIAGNOSE]")))
		var patterns []string
		if allMode {
			patterns, _ = scanner.GetPatterns()
		} else if flag.NArg() > 0 {
			patterns = []string{flag.Arg(0)}
		}
		
		logs := svc.DiagnoseLine(diagnoseLine, patterns)
		for _, l := range logs {
			fmt.Println(l)
		}
		return
	}

	if listMode {
		pats, _ := scanner.GetPatterns()
		fmt.Println(au.Bold("Available Patterns:"), strings.Join(pats, ", "))
		fmt.Println(au.Bold("Available Plugins:"))
		for _, t := range svc.Tools {
			fmt.Printf("- %s: %s\n", au.Cyan(t.ID), t.Description)
		}
		return
	}

	var activeToolIDs []string
	if toolIDs != "" {
		activeToolIDs = strings.Split(toolIDs, ",")
	}

	ctx := context.Background()
	var resultChan <-chan *models.Result

	if processFile != "" {
		resultChan, err = svc.ProcessResults(ctx, processFile, activeToolIDs)
	} else {
		var targets []string
		var runPats []string
		
		if allMode {
			targets = flag.Args(); if len(targets) == 0 { targets = []string{"."} }
			pats, _ := filepath.Glob(filepath.Join(svc.Config.PatternsDir, "*.json"))
			for _, f := range pats { runPats = append(runPats, strings.TrimSuffix(filepath.Base(f), ".json")) }
		} else {
			if flag.NArg() > 0 {
				runPats = []string{flag.Arg(0)}
				targets = flag.Args()[1:]; if len(targets) == 0 { targets = []string{"."} }
			} else {
				fmt.Println("Usage: zetgrep [pattern] [path...] OR zetgrep -all [path...] OR zetgrep -process [results.json]")
				return
			}
		}

		opts := scanner.ScannerOptions{
			TargetPaths: targets,
			Patterns:    runPats,
			ToolIDs:     activeToolIDs,
			SmartMode:   smartMode,
			EntropyMode: entropyMode,
		}
		resultChan, err = svc.RunScan(ctx, opts)
	}

	if jsonMode { fmt.Print("[") }
	first := true
	
	var reportFile *os.File
	if reportMode {
		reportName := fmt.Sprintf("report_%d.md", time.Now().Unix())
		reportFile, _ = os.Create(reportName)
		if reportFile != nil {
			fmt.Fprintln(reportFile, "# ZetGrep Intelligence Report")
			fmt.Fprintf(reportFile, "*Generated on: %s*\n\n", time.Now().Format(time.RFC1123))
			fmt.Fprintln(reportFile, "## Detailed Findings")
		}
		fmt.Printf("%s Streaming intelligence to: %s\n", au.Green("[+]"), au.Bold(reportName))
	}

	hitCount := 0
	for res := range resultChan {
		hitCount++
		if reportFile != nil {
			fmt.Fprintf(reportFile, "### Finding %d: [%s]\n", hitCount, res.Pattern)
			fmt.Fprintf(reportFile, "- **Source**: `%s` (Line: %d)\n", res.File, res.Line)
			fmt.Fprintf(reportFile, "- **Match Content**:\n```text\n%s\n```\n", res.Content)
			if len(res.ToolData) > 0 {
				fmt.Fprintln(reportFile, "- **Augmented Intelligence**:")
				for _, td := range res.ToolData {
					fmt.Fprintf(reportFile, "  - **%s**: `%s`\n", td.Label, td.Value)
				}
			}
			fmt.Fprintln(reportFile, "---\n")
		}

		if jsonMode {
			if !first { fmt.Print(",") }
			b, _ := json.Marshal(res); fmt.Print(string(b))
			first = false
		} else if outputTemplate != "" {
			fmt.Println(formatResult(outputTemplate, *res))
		} else {
			fmt.Printf("[%s] %s:%d: %s\n", au.Yellow(res.Pattern), au.Cyan(res.File), res.Line, res.Content)
			for _, td := range res.ToolData { fmt.Printf("   %s %s\n", au.Bold(au.Magenta("↳ "+td.Label+":")), td.Value) }
		}
		scanner.PutResult(res)
	}
	if jsonMode { fmt.Println("]") }

	if reportFile != nil {
		fmt.Fprintf(reportFile, "\n## Executive Summary\n- **Total Intel Hits**: %d\n", hitCount)
		reportFile.Close()
	}
}

func mergeConfigs(dest *models.Config, src models.Config) {
	if src.PatternsDir != "" { dest.PatternsDir = src.PatternsDir }
	if src.ToolsDir != "" { dest.ToolsDir = src.ToolsDir }
	dest.Globals.IgnoreExtensions = append(dest.Globals.IgnoreExtensions, src.Globals.IgnoreExtensions...)
	dest.Globals.IgnoreFiles = append(dest.Globals.IgnoreFiles, src.Globals.IgnoreFiles...)
}

func formatResult(tmpl string, res models.Result) string {
	out := tmpl
	out = strings.ReplaceAll(out, "{{pattern}}", res.Pattern)
	out = strings.ReplaceAll(out, "{{file}}", res.File)
	out = strings.ReplaceAll(out, "{{line}}", fmt.Sprintf("%d", res.Line))
	out = strings.ReplaceAll(out, "{{content}}", res.Content)
	out = strings.ReplaceAll(out, "{{ext}}", res.Ext)
	mainMatch := res.Content; if len(res.Matches) > 0 { mainMatch = res.Matches[0] }
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
