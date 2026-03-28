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
	version = "v0.2.2"
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
	// Flag Groups
	var (
		// Input
		inputConfigs multiFlag
		listFile     string
		stdin        bool
		inputMode    string

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
		tags        multiFlag

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
		resumeFile  string
		updateMode  bool
		versionMode bool
		listObjects bool
		healthCheck bool
	)

	// Define Flags
	flag.Var(&inputConfigs, "input-config", "path to input config file (YAML, multiple allowed)")
	flag.StringVar(&listFile, "l", "", "file containing list of targets to scan")
	flag.BoolVar(&stdin, "stdin", false, "read targets from stdin")
	flag.StringVar(&inputMode, "im", "", "input mode (jsonl, csv, text)")

	flag.Var(&configFiles, "config-file", "path to global config file (YAML/JSON, multiple allowed)")
	flag.Var(&toolFiles, "tool", "path to individual tool YAML file (multiple allowed)")
	flag.StringVar(&patternsDir, "pd", "", "directory containing patterns")
	flag.StringVar(&toolsDir, "td", "", "directory containing tool definitions")

	flag.BoolVar(&allMode, "all", false, "run all available patterns")
	flag.BoolVar(&smartMode, "smart", false, "use AI-based interest filtering")
	flag.BoolVar(&entropyMode, "entropy", false, "filter by high-entropy content")
	flag.StringVar(&diagnose, "diagnose", "", "diagnose a single line against current config")
	flag.Var(&tags, "tags", "filter patterns by tag (comma-separated)")

	flag.BoolVar(&jsonMode, "json", false, "output in JSON format")
	flag.BoolVar(&reportMode, "report", false, "generate markdown intelligence report")
	flag.StringVar(&outputTemplate, "o", "", "custom output template")
	flag.BoolVar(&silent, "silent", false, "display only results")
	flag.BoolVar(&verbose, "verbose", false, "display verbose information")
	flag.BoolVar(&noColor, "no-color", false, "disable colorized output")

	flag.StringVar(&webMode, "web", "", "start web dashboard (e.g. :8080)")
	flag.StringVar(&toolIDs, "w", "", "comma-separated tool IDs (workflow) to execute")
	flag.StringVar(&toolIDs, "workflow", "", "comma-separated tool IDs (workflow) to execute") // Alias
	flag.StringVar(&processFile, "process", "", "path to previous results.json to re-process")
	flag.StringVar(&resumeFile, "resume", "", "resume scan from state file (e.g. resume.cfg)")
	flag.BoolVar(&updateMode, "update", false, "update zetgrep to the latest version")
	flag.BoolVar(&versionMode, "version", false, "show version")
	flag.BoolVar(&listObjects, "list", false, "list all available patterns and tools")
	flag.BoolVar(&healthCheck, "health-check", false, "run diagnostic self-check")

	flag.Usage = func() {
		showBanner()
		fmt.Fprintf(os.Stderr, "Usage: zetgrep [flags] [pattern] [targets...]\n\n")
		
		groups := map[string][]string{
			"INPUT": {"input-config", "im", "l", "stdin"},
			"CONFIG": {"config-file", "tool", "pd", "td"},
			"FILTER": {"all", "smart", "entropy", "diagnose", "tags"},
			"OUTPUT": {"json", "report", "o", "silent", "verbose", "no-color"},
			"LOGIC": {"web", "w", "workflow", "process", "resume", "update", "version", "list", "health-check"},
		}

		// Helper to print flags
		printGroup := func(name string) {
			fmt.Fprintf(os.Stderr, "%s:\n", name)
			for _, fname := range groups[name] {
				f := flag.Lookup(fname)
				if f == nil { continue }
				s := fmt.Sprintf("   -%s", f.Name)
				if len(s) <= 4 { s += "\t" } else { s += " " }
				s += fmt.Sprintf(" %s", f.Usage)
				if f.DefValue != "false" && f.DefValue != "" { s += fmt.Sprintf(" (default %s)", f.DefValue) }
				fmt.Fprintf(os.Stderr, "%-40s\n", s)
			}
			fmt.Fprintln(os.Stderr)
		}

		printGroup("INPUT")
		printGroup("CONFIG")
		printGroup("FILTER")
		printGroup("OUTPUT")
		printGroup("LOGIC")
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
		cmd.Env = append(os.Environ(), "GOPROXY=direct")
		cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
		if err := cmd.Run(); err == nil {
			fmt.Println(au.Green("[+] ZetGrep updated successfully!"))
		}
		return
	}

	// 1. Resolve Global Configuration
	var finalCfg models.Config
	if len(configFiles) == 0 {
		def := utils.ExpandPath("~/.config/gf/config.yaml")
		if _, err := os.Stat(def); err == nil { configFiles = append(configFiles, def) }
	}
	for _, cf := range configFiles {
		var cfg models.Config
		b, _ := os.ReadFile(cf)
		if strings.HasSuffix(cf, ".json") { json.Unmarshal(b, &cfg) } else { yaml.Unmarshal(b, &cfg) }
		mergeConfigs(&finalCfg, cfg, cf)
	}
	if patternsDir != "" { finalCfg.PatternsDir = utils.ExpandPath(patternsDir) }
	if toolsDir != "" { finalCfg.ToolsDir = utils.ExpandPath(toolsDir) }
// 2. Resolve Input Configuration (Manual overrides via -input-config)
for _, ic := range inputConfigs {
	ic = utils.ExpandPath(ic)
	var inc models.InputConfig
	b, _ := os.ReadFile(ic)
	yaml.Unmarshal(b, &inc)
	// Map inc into finalCfg.Input (Explicit flags override global config)
	if inc.Format != "" { finalCfg.Input.Format = inc.Format }
	if inc.Target != "" { finalCfg.Input.Target = inc.Target }
	if len(inc.Targets) > 0 { finalCfg.Input.Targets = inc.Targets }
	if inc.ID != "" { finalCfg.Input.ID = inc.ID }
	if len(inc.Filters) > 0 {
		if finalCfg.Input.Filters == nil { finalCfg.Input.Filters = make(map[string]string) }
		for k, v := range inc.Filters { finalCfg.Input.Filters[k] = v }
	}
	if inc.CSVConfig.Separator != "" { finalCfg.Input.CSVConfig = inc.CSVConfig }
	finalCfg.Input.Decode = finalCfg.Input.Decode || inc.Decode
}

if inputMode != "" { finalCfg.Input.Format = inputMode }

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
		pats, _ := scanner.GetPatterns(svc.Config.PatternsDir)
		fmt.Printf("%s Patterns: %s\n", au.Bold("Available"), strings.Join(pats, ", "))
		fmt.Printf("%s Tools:\n", au.Bold("Available"))
		for _, t := range svc.Tools { fmt.Printf("  - %s: %s\n", au.Cyan(t.ID), t.Description) }
		return
	}

	if healthCheck {
		fmt.Printf("%s Running health check...\n", au.Bold(au.Cyan("[*]")))
		fmt.Printf("Patterns Directory: %s\n", svc.Config.PatternsDir)
		if _, err := os.Stat(svc.Config.PatternsDir); err == nil { fmt.Printf("%s Directory exists\n", au.Green("[+]")) } else { fmt.Printf("%s Directory NOT FOUND\n", au.Red("[-]")) }
		fmt.Printf("Tools Directory: %s\n", svc.Config.ToolsDir)
		if _, err := os.Stat(svc.Config.ToolsDir); err == nil { fmt.Printf("%s Directory exists\n", au.Green("[+]")) } else { fmt.Printf("%s Directory NOT FOUND\n", au.Red("[-]")) }
		return
	}

	if webMode != "" {
		srv := api.NewServer(webMode, svc)
		srv.Start()
		return
	}

	// Resume Logic
	if resumeFile != "" {
		if err := svc.LoadResumeState(resumeFile); err == nil {
			if !silent { fmt.Printf("%s Resuming scan from file %d, line %d\n", au.Cyan("[*]"), svc.Resume.FileIndex, svc.Resume.LineIndex) }
		} else {
			if !silent { fmt.Printf("%s Starting new scan session: %s\n", au.Cyan("[*]"), resumeFile) }
		}
	}

	if diagnose != "" {
		pats := flag.Args()
		if allMode { pats, _ = scanner.GetPatterns(svc.Config.PatternsDir) }
		for _, l := range svc.DiagnoseLine(diagnose, pats) { fmt.Println(l) }
		return
	}

	// 3. Resolve Targets
	var targets []string
	if stdin {
		targets = []string{"stdin"}
	} else if listFile != "" {
		f, _ := os.Open(utils.ExpandPath(listFile))
		s := bufio.NewScanner(f)
		for s.Scan() { targets = append(targets, utils.ExpandPath(s.Text())) }
		f.Close()
	} else {
		for _, arg := range flag.Args() {
			targets = append(targets, utils.ExpandPath(arg))
		}
		if len(targets) > 1 && !allMode { targets = targets[1:] }
	}
	if len(targets) == 0 { targets = []string{"."} }

	if inputMode != "" { finalCfg.Input.Format = inputMode }

	// 4. Execution
	var runPats []string
	if allMode {
		runPats, _ = scanner.GetPatterns(svc.Config.PatternsDir)
	} else if len(tags) > 0 {
		runPats = svc.FilterPatternsByTag(tags)
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
			Tags:        tags,
			ToolIDs:     activeToolIDs,
			SmartMode:   smartMode,
			EntropyMode: entropyMode,
			ResumeFile:  resumeFile,
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

func mergeConfigs(dest *models.Config, src models.Config, configPath string) {
	configDir := filepath.Dir(configPath)

	resolve := func(path string) string {
		if path == "" || filepath.IsAbs(path) {
			return path
		}
		// Expand tilde if present in the config value
		if strings.HasPrefix(path, "~") {
			return utils.ExpandPath(path)
		}
		return filepath.Join(configDir, path)
	}

	if src.PatternsDir != "" {
		dest.PatternsDir = resolve(src.PatternsDir)
	}
	if src.ToolsDir != "" {
		dest.ToolsDir = resolve(src.ToolsDir)
	}
	dest.Globals.IgnoreExtensions = append(dest.Globals.IgnoreExtensions, src.Globals.IgnoreExtensions...)
	dest.Globals.IgnoreFiles = append(dest.Globals.IgnoreFiles, src.Globals.IgnoreFiles...)

	// Merge Input Config
	if src.Input.Format != "" { dest.Input.Format = src.Input.Format }
	if src.Input.Target != "" { dest.Input.Target = src.Input.Target }
	if len(src.Input.Targets) > 0 { dest.Input.Targets = src.Input.Targets }
	if src.Input.ID != "" { dest.Input.ID = src.Input.ID }
	if src.Input.Decode { dest.Input.Decode = true }
	if len(src.Input.Filters) > 0 {
		if dest.Input.Filters == nil { dest.Input.Filters = make(map[string]string) }
		for k, v := range src.Input.Filters { dest.Input.Filters[k] = v }
	}
	if src.Input.CSVConfig.Separator != "" { dest.Input.CSVConfig = src.Input.CSVConfig }
}

func formatResult(tmpl string, res models.Result) string {
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
	
	// Replace both {{tool:ID}} and {{tool:Label}}
	for _, td := range res.ToolData {
		out = strings.ReplaceAll(out, fmt.Sprintf("{{tool:%s}}", td.ToolID), td.Value)
		out = strings.ReplaceAll(out, fmt.Sprintf("{{tool:%s}}", td.Label), td.Value)
	}
	return out
}
