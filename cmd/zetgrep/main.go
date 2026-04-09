package main

import (
	"bufio"
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
	"github.com/Abhay0thakor/ZetGrep/pkg/utils"
	"gopkg.in/yaml.v3"
)

const (
	version = "v0.4.4"
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
)

var (
	au = aurora.NewAurora(true)
)

type multiFlag []string

func (m *multiFlag) String() string { return strings.Join(*m, ",") }
func (m *multiFlag) Set(v string) error { *m = append(*m, v); return nil }

func showBanner() {
	fmt.Fprintf(os.Stderr, "%s\n", au.Bold(au.Cyan(banner)))
	fmt.Fprintf(os.Stderr, "\t\t%s\n\n", au.Faint(version))
}

func main() {
	var (
		inputConfigs multiFlag
		listFile     string
		stdin        bool
		inputMode    string
		configFiles multiFlag
		toolFiles   multiFlag
		patternsDir string
		toolsDir    string
		allMode     bool
		smartMode   bool
		entropyMode bool
		diagnose    string
		tags        multiFlag
		jsonMode       bool
		reportMode     bool
		outputTemplate string
		silent         bool
		verbose        bool
		noColor        bool
		webMode     string
		toolIDs     string
		processFile string
		resumeFile  string
		rewriteMode bool
		updateMode  bool
		versionMode bool
		listObjects bool
		healthCheck bool
	)

	flag.Var(&inputConfigs, "input-config", "path to input config file (YAML)")
	flag.StringVar(&listFile, "l", "", "file containing list of targets")
	flag.BoolVar(&stdin, "stdin", false, "read targets from stdin")
	flag.StringVar(&inputMode, "im", "", "input mode (jsonl, csv, text)")
	flag.Var(&configFiles, "config-file", "path to global config")
	flag.Var(&toolFiles, "tool", "path to tool YAML")
	flag.StringVar(&patternsDir, "pd", "", "patterns directory")
	flag.StringVar(&toolsDir, "td", "", "tools directory")
	flag.BoolVar(&allMode, "all", false, "run all patterns")
	flag.BoolVar(&smartMode, "smart", false, "AI interest filtering")
	flag.BoolVar(&entropyMode, "entropy", false, "high-entropy filtering")
	flag.StringVar(&diagnose, "diagnose", "", "diagnose a single line")
	flag.StringVar(&diagnose, "dignose", "", "alias for -diagnose")
	flag.Var(&tags, "tags", "filter by tag")
	flag.BoolVar(&jsonMode, "json", false, "output in JSON")
	flag.BoolVar(&reportMode, "report", false, "generate markdown report")
	flag.StringVar(&outputTemplate, "o", "", "output template")
	flag.BoolVar(&silent, "silent", false, "silent mode")
	flag.BoolVar(&verbose, "verbose", false, "verbose mode")
	flag.BoolVar(&noColor, "no-color", false, "disable color")
	flag.StringVar(&webMode, "web", "", "start web dashboard")
	flag.StringVar(&toolIDs, "w", "", "workflow tool IDs")
	flag.StringVar(&toolIDs, "workflow", "", "workflow tool IDs")
	flag.StringVar(&processFile, "process", "", "re-process results.json")
	flag.StringVar(&resumeFile, "resume", "", "resume scan state")
	flag.BoolVar(&rewriteMode, "rewrite", false, "permanent post-processing")
	flag.BoolVar(&updateMode, "update", false, "self-update")
	flag.BoolVar(&versionMode, "version", false, "show version")
	flag.BoolVar(&listObjects, "list", false, "list patterns/tools")
	flag.BoolVar(&healthCheck, "health-check", false, "verify environment")

	flag.Usage = func() {
		showBanner()
		fmt.Fprintf(os.Stderr, "Usage: zetgrep [flags] [pattern] [targets...]\n\n")
		groups := map[string][]string{
			"INPUT": {"input-config", "im", "l", "stdin"},
			"CONFIG": {"config-file", "tool", "pd", "td"},
			"FILTER": {"all", "smart", "entropy", "diagnose", "dignose", "tags"},
			"OUTPUT": {"json", "report", "o", "silent", "verbose", "no-color"},
			"LOGIC": {"web", "w", "workflow", "process", "resume", "rewrite", "update", "version", "list", "health-check"},
		}
		for _, name := range []string{"INPUT", "CONFIG", "FILTER", "OUTPUT", "LOGIC"} {
			fmt.Fprintf(os.Stderr, "%s:\n", name)
			for _, fname := range groups[name] {
				f := flag.Lookup(fname); if f == nil { continue }
				fmt.Fprintf(os.Stderr, "   -%-15s %s\n", f.Name, f.Usage)
			}
			fmt.Fprintln(os.Stderr)
		}
	}
	flag.Parse()

	// Check for stray flags (Strict Validation)
	for _, arg := range flag.Args() {
		if strings.HasPrefix(arg, "-") {
			fmt.Fprintf(os.Stderr, "%s Unknown or misplaced flag: %s\n", au.Red("[ERROR]"), arg)
			flag.Usage()
			os.Exit(1)
		}
	}

	if noColor { au = aurora.NewAurora(false) }
	if !silent { showBanner() }
	if versionMode { fmt.Printf("ZetGrep version: %s\n", version); return }

	if updateMode {
		fmt.Printf("%s Updating...\n", au.Cyan("[*]"))
		cmd := exec.Command("go", "install", "-v", "github.com/Abhay0thakor/ZetGrep/cmd/zetgrep@latest")
		cmd.Env = append(os.Environ(), "GOPROXY=direct")
		cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
		if err := cmd.Run(); err == nil { fmt.Println(au.Green("[+] Success!")) }
		return
	}

	var finalCfg models.Config
	if len(configFiles) == 0 {
		def := utils.ExpandPath("~/.config/gf/config.yaml")
		if _, err := os.Stat(def); err == nil { configFiles = append(configFiles, def) }
	}
	for _, cf := range configFiles {
		var cfg models.Config; b, _ := os.ReadFile(cf)
		if strings.HasSuffix(cf, ".json") { json.Unmarshal(b, &cfg) } else { yaml.Unmarshal(b, &cfg) }
		mergeConfigs(&finalCfg, cfg, cf)
	}
	if patternsDir != "" { finalCfg.PatternsDir = utils.ExpandPath(patternsDir) }
	if toolsDir != "" { finalCfg.ToolsDir = utils.ExpandPath(toolsDir) }

	for _, ic := range inputConfigs {
		ic = utils.ExpandPath(ic); var inc models.InputConfig; b, _ := os.ReadFile(ic); yaml.Unmarshal(b, &inc)
		mergeInputConfigs(&finalCfg.Input, inc)
	}
	// Final Format Determination (Command-line and Auto-Detection take priority over config)
	if inputMode != "" {
		finalCfg.Input.Format = inputMode
	} else if flag.NArg() > 0 {
		ext := strings.ToLower(filepath.Ext(flag.Arg(0)))
		if ext == ".jsonl" || ext == ".json" {
			finalCfg.Input.Format = "jsonl"
		} else if ext == ".csv" {
			finalCfg.Input.Format = "csv"
		} else {
			finalCfg.Input.Format = "text"
		}
	} else if finalCfg.Input.Format == "" {
		finalCfg.Input.Format = "text"
	}

	svc, err := scanner.NewScannerService(finalCfg)
	if err != nil { fmt.Fprintf(os.Stderr, "[!] Error: %v\n", err); os.Exit(1) }
	for _, tf := range toolFiles { if t, err := scanner.LoadToolFromFile(tf); err == nil { svc.Tools = append(svc.Tools, t) } }

	if listObjects {
		pats, _ := scanner.GetPatterns(svc.Config.PatternsDir)
		fmt.Printf("%s Patterns: %s\n", au.Bold("Available"), strings.Join(pats, ", "))
		fmt.Printf("%s Tools:\n", au.Bold("Available"))
		for _, t := range svc.Tools { fmt.Printf("  - %s: %s\n", au.Cyan(t.ID), t.Description) }
		return
	}

	if healthCheck {
		fmt.Printf("%s Health Check:\nPatterns: %s\nTools: %s\nFormat: %s\n", au.Cyan("[*]"), svc.Config.PatternsDir, svc.Config.ToolsDir, svc.Config.Input.Format)
		return
	}

	var targets []string
	if stdin { targets = []string{"stdin"}
	} else if listFile != "" {
		f, _ := os.Open(utils.ExpandPath(listFile)); s := bufio.NewScanner(f)
		for s.Scan() { targets = append(targets, utils.ExpandPath(s.Text())) }; f.Close()
	} else {
		for _, arg := range flag.Args() { targets = append(targets, utils.ExpandPath(arg)) }
		if len(targets) > 1 && !allMode { targets = targets[1:] }
	}
	if len(targets) == 0 { targets = []string{"."} }

	if rewriteMode {
		ctx := context.Background()
		for _, path := range targets { if err := svc.RewriteFile(ctx, path); err != nil { fmt.Fprintf(os.Stderr, "[!] Error rewriting %s: %v\n", path, err) } }
		return
	}

	if webMode != "" { srv := api.NewServer(webMode, svc); srv.Start(); return }

	if resumeFile != "" {
		if err := svc.LoadResumeState(resumeFile); err == nil {
			if !silent { fmt.Printf("[*] Resuming from file %d, line %d\n", svc.Resume.FileIndex, svc.Resume.LineIndex) }
		}
	}

	if diagnose != "" {
		pats := flag.Args(); if allMode { pats, _ = scanner.GetPatterns(svc.Config.PatternsDir) }
		for _, l := range svc.DiagnoseLine(diagnose, pats) { fmt.Println(l) }
		return
	}

	var runPats []string
	if allMode { runPats, _ = scanner.GetPatterns(svc.Config.PatternsDir)
	} else if len(tags) > 0 { runPats = svc.FilterPatternsByTag(tags)
	} else if flag.NArg() > 0 { runPats = []string{flag.Arg(0)} }

	var activeToolIDs []string
	if toolIDs != "" { activeToolIDs = strings.Split(toolIDs, ",") }

	ctx := context.Background(); var resultChan <-chan *models.Result
	if processFile != "" { resultChan, _ = svc.ProcessResults(ctx, processFile, activeToolIDs)
	} else {
		resultChan, _ = svc.RunScan(ctx, scanner.ScannerOptions{
			TargetPaths: targets, Patterns: runPats, Tags: tags, ToolIDs: activeToolIDs,
			SmartMode: smartMode, EntropyMode: entropyMode, ResumeFile: resumeFile, Silent: silent,
		})
	}

	if jsonMode { fmt.Print("[") }
	first := true; var reportFile *os.File
	if reportMode {
		name := fmt.Sprintf("zetgrep_report_%d.md", time.Now().Unix()); reportFile, _ = os.Create(name)
		fmt.Fprintln(reportFile, "# ZetGrep Intelligence Report\n")
	}

	hitCount := 0
	for res := range resultChan {
		hitCount++
		if reportFile != nil { fmt.Fprintf(reportFile, "### [%s] %s\n- Content: `%s`\n", res.Pattern, res.File, res.Content) }
		if jsonMode {
			if !first { fmt.Print(",") }; b, _ := json.Marshal(res); fmt.Print(string(b)); first = false
		} else if outputTemplate != "" { fmt.Println(formatResult(outputTemplate, res))
		} else if !silent {
			fmt.Printf("[%s] %s:%d: %s\n", au.Yellow(res.Pattern), au.Cyan(res.File), res.Line, res.Content)
			for _, td := range res.ToolData { fmt.Printf("   ↳ %s: %s\n", au.Magenta(td.Label), td.Value) }
		} else { fmt.Println(res.Content) }
		scanner.PutResult(res)
	}
	if jsonMode { fmt.Println("]") }
	if reportFile != nil { reportFile.Close() }
	if !silent { fmt.Printf("\n[+] Finished. Total Hits: %d\n", hitCount) }
}

func mergeConfigs(dest *models.Config, src models.Config, configPath string) {
	configDir := filepath.Dir(configPath)
	resolve := func(path string) string {
		if path == "" || filepath.IsAbs(path) { return path }
		if strings.HasPrefix(path, "~") { return utils.ExpandPath(path) }
		return filepath.Join(configDir, path)
	}
	if src.PatternsDir != "" { dest.PatternsDir = resolve(src.PatternsDir) }
	if src.ToolsDir != "" { dest.ToolsDir = resolve(src.ToolsDir) }
	dest.Globals.IgnoreExtensions = append(dest.Globals.IgnoreExtensions, src.Globals.IgnoreExtensions...)
	dest.Globals.IgnoreFiles = append(dest.Globals.IgnoreFiles, src.Globals.IgnoreFiles...)
	mergeInputConfigs(&dest.Input, src.Input)
}

func mergeInputConfigs(dest *models.InputConfig, src models.InputConfig) {
	if src.Format != "" { dest.Format = src.Format }
	if src.PreProcess != "" { dest.PreProcess = src.PreProcess }
	if src.Target != "" { dest.Target = src.Target }
	if len(src.Targets) > 0 { dest.Targets = src.Targets }
	if src.ID != "" { dest.ID = src.ID }
	if src.Decode { dest.Decode = true }
	if len(src.Filters) > 0 {
		if dest.Filters == nil { dest.Filters = make(map[string]string) }
		for k, v := range src.Filters { dest.Filters[k] = v }
	}
	if src.CSVConfig.Separator != "" { dest.CSVConfig = src.CSVConfig }
	if len(src.PostProcess) > 0 {
		if dest.PostProcess == nil { dest.PostProcess = make(map[string]string) }
		for k, v := range src.PostProcess { dest.PostProcess[k] = v }
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
	mainMatch := res.Content; if len(res.Matches) > 0 { mainMatch = res.Matches[0] }
	out = strings.ReplaceAll(out, "{{match}}", mainMatch)
	for i, m := range res.Matches { out = strings.ReplaceAll(out, fmt.Sprintf("{{match[%d]}}", i), m) }
	for _, td := range res.ToolData {
		out = strings.ReplaceAll(out, fmt.Sprintf("{{tool:%s}}", td.ToolID), td.Value)
		out = strings.ReplaceAll(out, fmt.Sprintf("{{tool:%s}}", td.Label), td.Value)
	}
	return out
}
