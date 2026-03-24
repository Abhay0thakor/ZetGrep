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
	"gopkg.in/yaml.v3"
)

const (
	version = "v0.1.7"
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
	fmt.Fprintf(os.Stderr, "\t\t%s\n\n", au.Faint("v"+version))
}

func main() {
	// Flag Groups
	var (
		// Input
		inputConfigs multiFlag
		listFile     string
		stdin        bool

		// Config
		configFiles multiFlag
		toolFiles   multiFlag
		patternsDir string
		toolsDir    string

		// Filter
		allMode     bool
		smartMode   bool
		entropyMode bool
		diagnose    string

		// Output
		jsonMode       bool
		reportMode     bool
		outputTemplate string
		silent         bool
		verbose        bool
		noColor        bool

		// Logic
		webMode     string
		toolIDs     string
		processFile string
		updateMode  bool
		versionMode bool
		listObjects bool
	)

	// Define Flags
	flag.Var(&inputConfigs, "input-config", "path to input config file (YAML, multiple allowed)")
	flag.StringVar(&listFile, "l", "", "file containing list of targets to scan")
	flag.BoolVar(&stdin, "stdin", false, "read targets from stdin")

	flag.Var(&configFiles, "config-file", "path to global config file (YAML/JSON, multiple allowed)")
	flag.Var(&toolFiles, "tool", "path to individual tool YAML file (multiple allowed)")
	flag.StringVar(&patternsDir, "pd", "", "directory containing patterns")
	flag.StringVar(&toolsDir, "td", "", "directory containing tool definitions")

	flag.BoolVar(&allMode, "all", false, "run all available patterns")
	flag.BoolVar(&smartMode, "smart", false, "use AI-based interest filtering")
	flag.BoolVar(&entropyMode, "entropy", false, "filter by high-entropy content")
	flag.StringVar(&diagnose, "diagnose", "", "diagnose a single line against current config")

	flag.BoolVar(&jsonMode, "json", false, "output in JSON format")
	flag.BoolVar(&reportMode, "report", false, "generate markdown intelligence report")
	flag.StringVar(&outputTemplate, "o", "", "custom output template")
	flag.BoolVar(&silent, "silent", false, "display only results")
	flag.BoolVar(&verbose, "verbose", false, "display verbose information")
	flag.BoolVar(&noColor, "no-color", false, "disable colorized output")

	flag.StringVar(&webMode, "web", "", "start web dashboard (e.g. :8080)")
	flag.StringVar(&toolIDs, "tools", "", "comma-separated tool IDs to execute")
	flag.StringVar(&processFile, "process", "", "path to previous results.json to re-process")
	flag.BoolVar(&updateMode, "update", false, "update zetgrep to the latest version")
	flag.BoolVar(&versionMode, "version", false, "show version")
	flag.BoolVar(&listObjects, "list", false, "list all available patterns and tools")

	flag.Usage = func() {
		showBanner()
		fmt.Fprintf(os.Stderr, "Usage: zetgrep [flags] [pattern] [targets...]\n\n")
		fmt.Fprintf(os.Stderr, "INPUT:\n")
		flag.PrintDefaults()
	}
	flag.Parse()

	if noColor { au = aurora.NewAurora(false) }
	if !silent { showBanner() }

	if versionMode {
		fmt.Printf("ZetGrep version: %s\n", version)
		return
	}

	if updateMode {
		fmt.Printf("%s Updating to latest...\n", au.Cyan("[*]"))
		cmd := exec.Command("go", "install", "-v", "github.com/Abhay0thakor/ZetGrep/cmd/zetgrep@latest")
		cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
		if err := cmd.Run(); err == nil {
			fmt.Println(au.Green("[+] ZetGrep updated successfully!"))
		}
		return
	}

	// 1. Resolve Global Configuration
	var finalCfg models.Config
	if len(configFiles) == 0 {
		home, _ := os.UserHomeDir()
		def := filepath.Join(home, ".config", "gf", "config.yaml")
		if _, err := os.Stat(def); err == nil { configFiles = append(configFiles, def) }
	}
	for _, cf := range configFiles {
		var cfg models.Config
		b, _ := os.ReadFile(cf)
		if strings.HasSuffix(cf, ".json") { json.Unmarshal(b, &cfg) } else { yaml.Unmarshal(b, &cfg) }
		mergeConfigs(&finalCfg, cfg)
	}
	if patternsDir != "" { finalCfg.PatternsDir = patternsDir }
	if toolsDir != "" { finalCfg.ToolsDir = toolsDir }

	// 2. Resolve Input Configuration
	for _, ic := range inputConfigs {
		var inc models.InputConfig
		b, _ := os.ReadFile(ic)
		yaml.Unmarshal(b, &inc)
		if inc.Format != "" { finalCfg.Input.Format = inc.Format }
		if len(inc.Targets) > 0 { finalCfg.Input.Targets = inc.Targets }
		if inc.ID != "" { finalCfg.Input.ID = inc.ID }
		if len(inc.Filters) > 0 { finalCfg.Input.Filters = inc.Filters }
		finalCfg.Input.Decode = finalCfg.Input.Decode || inc.Decode
	}

	svc, err := scanner.NewScannerService(finalCfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[!] Error: %v\n", err)
		os.Exit(1)
	}

	// Load individual tools
	for _, tf := range toolFiles {
		if t, err := scanner.LoadToolFromFile(tf); err == nil { svc.Tools = append(svc.Tools, t) }
	}

	if listObjects {
		pats, _ := scanner.GetPatterns()
		fmt.Printf("%s Patterns: %s\n", au.Bold("Available"), strings.Join(pats, ", "))
		fmt.Printf("%s Tools:\n", au.Bold("Available"))
		for _, t := range svc.Tools { fmt.Printf("  - %s: %s\n", au.Cyan(t.ID), t.Description) }
		return
	}

	if webMode != "" {
		srv := api.NewServer(webMode, svc)
		srv.Start()
		return
	}

	if diagnose != "" {
		pats := flag.Args()
		if allMode { pats, _ = scanner.GetPatterns() }
		for _, l := range svc.DiagnoseLine(diagnose, pats) { fmt.Println(l) }
		return
	}

	// 3. Resolve Targets
	var targets []string
	if stdin {
		s := bufio.NewScanner(os.Stdin)
		for s.Scan() { targets = append(targets, s.Text()) }
	} else if listFile != "" {
		f, _ := os.Open(listFile)
		s := bufio.NewScanner(f)
		for s.Scan() { targets = append(targets, s.Text()) }
		f.Close()
	} else {
		targets = flag.Args()
		if len(targets) > 1 && !allMode { targets = flag.Args()[1:] }
	}
	if len(targets) == 0 { targets = []string{"."} }

	// 4. Execution
	var runPats []string
	if allMode {
		runPats, _ = scanner.GetPatterns()
	} else if flag.NArg() > 0 {
		runPats = []string{flag.Arg(0)}
	}

	var activeToolIDs []string
	if toolIDs != "" { activeToolIDs = strings.Split(toolIDs, ",") }

	ctx := context.Background()
	var resultChan <-chan *models.Result
	if processFile != "" {
		resultChan, _ = svc.ProcessResults(ctx, processFile, activeToolIDs)
	} else {
		resultChan, _ = svc.RunScan(ctx, scanner.ScannerOptions{
			TargetPaths: targets,
			Patterns:    runPats,
			ToolIDs:     activeToolIDs,
			SmartMode:   smartMode,
			EntropyMode: entropyMode,
		})
	}

	// 5. Output Management
	if jsonMode { fmt.Print("[") }
	first := true
	var reportFile *os.File
	if reportMode {
		name := fmt.Sprintf("zetgrep_report_%d.md", time.Now().Unix())
		reportFile, _ = os.Create(name)
		fmt.Fprintln(reportFile, "# ZetGrep Intelligence Report\n")
		fmt.Printf("%s Generating report: %s\n", au.Cyan("[*]"), name)
	}

	hitCount := 0
	for res := range resultChan {
		hitCount++
		if reportFile != nil {
			fmt.Fprintf(reportFile, "### [%s] %s\n- Content: `%s`\n", res.Pattern, res.File, res.Content)
		}
		if jsonMode {
			if !first { fmt.Print(",") }; b, _ := json.Marshal(res); fmt.Print(string(b)); first = false
		} else if outputTemplate != "" {
			fmt.Println(formatResult(outputTemplate, *res))
		} else if !silent {
			fmt.Printf("[%s] %s:%d: %s\n", au.Yellow(res.Pattern), au.Cyan(res.File), res.Line, res.Content)
			for _, td := range res.ToolData { fmt.Printf("   %s %s\n", au.Bold(au.Magenta("↳ "+td.Label+":")), td.Value) }
		} else {
			fmt.Println(res.Content)
		}
		scanner.PutResult(res)
	}
	if jsonMode { fmt.Println("]") }
	if reportFile != nil { reportFile.Close() }
	if !silent { fmt.Printf("\n%s Finished. Total Hits: %d\n", au.Green("[+]"), hitCount) }
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
	for i, m := range res.Matches { out = strings.ReplaceAll(out, fmt.Sprintf("{{match[%d]}}", i), m) }
	for _, td := range res.ToolData {
		out = strings.ReplaceAll(out, fmt.Sprintf("{{tool:%s}}", td.ToolID), td.Value)
		out = strings.ReplaceAll(out, fmt.Sprintf("{{tool:%s}}", td.Label), td.Value)
	}
	return out
}
