package scanner

import (
	"bufio"
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"sync"

	"github.com/Abhay0thakor/ZetGrep/pkg/classifier"
	"github.com/Abhay0thakor/ZetGrep/pkg/models"
	"github.com/Abhay0thakor/ZetGrep/pkg/utils"
	"github.com/logrusorgru/aurora"
)

type ScannerOptions struct {
	TargetPaths []string
	Patterns    []string
	Tags        []string
	ToolIDs     []string
	SmartMode   bool
	EntropyMode bool
	Unique      bool
	ResumeFile  string
	Silent      bool
	Concurrency int
}

type ScannerService struct {
	Engine       Engine
	Fallback     Engine
	Config       models.Config
	Classifier   *classifier.Classifier
	Tools        []models.Tool
	Parser       Parser
	patternCache sync.Map
	seenMatches  sync.Map
	processSem   chan struct{}
	Resume       models.ResumeConfig
}

var au = aurora.NewAurora(true)

func NewScannerService(cfg models.Config) (*ScannerService, error) {
	engine, err := NewRipgrepEngine()
	var fallback Engine
	if err != nil {
		fallback, err = NewGrepEngine()
		if err != nil {
			return nil, fmt.Errorf("no suitable scanning engine found (ripgrep or grep): %w", err)
		}
	}

	if cfg.PatternsDir == "" {
		cfg.PatternsDir, err = GetPatternDir()
		if err != nil {
			// fallback to current directory patterns if any
			if _, err := os.Stat("patterns"); err == nil {
				cfg.PatternsDir = "patterns"
			}
		}
	}
	if cfg.ToolsDir == "" {
		cfg.ToolsDir, _ = GetToolDir()
	}

	maxProc := runtime.NumCPU() * 2
	if maxProc > 50 {
		maxProc = 50
	}

	tools, err := LoadToolsFrom(cfg.ToolsDir)
	if err != nil {
		slog.Warn("Error loading tools", "error", err)
	}

	var p Parser
	switch cfg.Input.Format {
	case "jsonl", "json":
		p = &JSONLParser{Config: cfg.Input}
	case "csv":
		p = &CSVParser{Config: cfg.Input}
	default:
		p = &TextParser{Config: cfg.Input}
	}

	return &ScannerService{
		Engine:       engine,
		Fallback:     fallback,
		Config:       cfg,
		Classifier:   classifier.DefaultClassifier(),
		Tools:        tools,
		Parser:       p,
		patternCache: sync.Map{},
		seenMatches:  sync.Map{},
		processSem:   make(chan struct{}, maxProc),
	}, nil
}

func (s *ScannerService) getPattern(name string) (models.Pattern, error) {
	if val, ok := s.patternCache.Load(name); ok {
		return val.(models.Pattern), nil
	}
	p, err := LoadPattern(filepath.Join(s.Config.PatternsDir, name+".json"))
	if err != nil {
		return models.Pattern{}, err
	}
	p.Name = name
	s.patternCache.Store(name, p)
	return p, nil
}

func (s *ScannerService) executeToolWithLimit(t models.Tool, res models.Result) (string, error) {
	s.processSem <- struct{}{}
	defer func() { <-s.processSem }()
	return t.Execute(res)
}

func unescapeContent(s string) string {
	if !strings.Contains(s, "\\") {
		return s
	}
	// 1. Handle JSON Escaped Newlines
	s = strings.ReplaceAll(s, "\\n", "\n")
	s = strings.ReplaceAll(s, "\\r", "\r")
	s = strings.ReplaceAll(s, "\\t", "\t")
	s = strings.ReplaceAll(s, "\\\"", "\"")

	// 2. Handle Unicode escapes if any
	if strings.Contains(s, "\\u") {
		var decoded string
		if err := json.Unmarshal([]byte("\""+s+"\""), &decoded); err == nil {
			s = decoded
		}
	}
	return s
}

func (s *ScannerService) RunScan(ctx context.Context, opts ScannerOptions) (<-chan *models.Result, error) {
	slog.Debug("Scan started", "format", s.Config.Input.Format)
	resultChan := make(chan *models.Result, 1000)

	var compiledPatterns []struct {
		p    models.Pattern
		comp *regexp.Regexp
	}

	for _, pName := range opts.Patterns {
		p, err := s.getPattern(pName)
		if err != nil {
			slog.Debug("Failed to get pattern", "name", pName, "error", err)
			continue
		}

		finalPattern := p.Pattern
		if strings.Contains(p.Flags, "i") && !strings.HasPrefix(finalPattern, "(?i)") {
			finalPattern = "(?i)" + finalPattern
		}

		if comp, err := regexp.Compile(finalPattern); err == nil {
			slog.Debug("Compiled pattern", "name", pName, "regex", finalPattern)
			compiledPatterns = append(compiledPatterns, struct {
				p    models.Pattern
				comp *regexp.Regexp
			}{p, comp})
		} else {
			slog.Debug("Failed to compile pattern", "name", pName, "error", err)
		}
	}

	activeTools := s.getActiveTools(opts.ToolIDs)
	recordChan := make(chan ScanRecord, 1000)
	var wg sync.WaitGroup
	numWorkers := opts.Concurrency
	if numWorkers <= 0 {
		numWorkers = runtime.NumCPU() * 2
	}

	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for rec := range recordChan {
				content := rec.Content
				if s.Config.Input.Decode {
					content = unescapeContent(content)
				}

				// Post-process usually applies to the whole content if not specified
				// For unified, we check if there's a post-process for the specific field or "$"
				// This part might need more tuning to match exact old behavior
				postCmd := ""
				if cmd, ok := s.Config.Input.PostProcess[rec.ID]; ok {
					postCmd = cmd
				} else if cmd, ok := s.Config.Input.PostProcess["$"]; ok {
					postCmd = cmd
				}

				if postCmd != "" {
					cmd := exec.CommandContext(ctx, "bash", "-c", "echo '"+strings.ReplaceAll(content, "'", "'\\''")+"' | "+postCmd)
					if out, err := cmd.Output(); err == nil {
						content = string(out)
					}
				}

				for _, cp := range compiledPatterns {
					matches := cp.comp.FindAllStringSubmatch(content, -1)
					for _, matchGroup := range matches {
						if len(matchGroup) == 0 || matchGroup[0] == "" {
							continue
						}
						res := GetResult()
						res.Pattern = cp.p.Name
						res.Content = matchGroup[0]
						res.Matches = matchGroup
						res.Entropy = utils.ShannonEntropy(res.Content)
						res.Line = rec.Line
						res.File = rec.File

						if opts.Unique {
							key := res.Pattern + ":" + res.Content
							if _, seen := s.seenMatches.LoadOrStore(key, true); seen {
								PutResult(res)
								continue
							}
						}

						// ONLY filter if explicitly requested
						if opts.SmartMode && s.Classifier.Classify(res.Content) != "high-interest" {
							PutResult(res)
							continue
						}
						if opts.EntropyMode && res.Entropy < 3.5 {
							PutResult(res)
							continue
						}
						for _, t := range activeTools {
							if val, _ := s.executeToolWithLimit(t, *res); val != "" {
								res.ToolData = append(res.ToolData, models.ToolOutput{ToolID: t.ID, Label: t.Field, Value: val})
							}
						}
						select {
						case <-ctx.Done():
							return
						case resultChan <- res:
						}
					}
				}
			}
		}()
	}

	go func() {
		defer close(resultChan)
		var innerWg sync.WaitGroup
		innerWg.Add(1)
		go func() {
			defer innerWg.Done()
			wg.Wait()
		}()

		for i, path := range opts.TargetPaths {
			if opts.ResumeFile != "" && i < s.Resume.FileIndex {
				continue
			}

			var fileReader io.ReadCloser
			var totalSize int64
			if path != "stdin" && path != "-" {
				if info, err := os.Stat(path); err == nil {
					totalSize = info.Size()
				}
			}

			if s.Config.Input.PreProcess != "" && path != "stdin" && path != "-" {
				cmd := exec.CommandContext(ctx, "bash", "-c", s.Config.Input.PreProcess+" "+path)
				if stdout, err := cmd.StdoutPipe(); err == nil {
					if err := cmd.Start(); err == nil {
						fileReader = stdout
						go cmd.Wait()
					}
				}
			}

			if fileReader == nil {
				if path == "-" || path == "stdin" {
					fileReader = io.NopCloser(os.Stdin)
				} else {
					f, err := os.Open(path)
					if err != nil {
						slog.Error("Error opening file", "path", path, "error", err)
						continue
					}
					fileReader = f
				}
			}

			recs, err := s.Parser.GetRecords(ctx, fileReader, path)
			if err != nil {
				slog.Error("Error starting parser", "path", path, "error", err)
				fileReader.Close()
				continue
			}

			var lineCount int
			var bytesRead int64
			for rec := range recs {
				lineCount = rec.Line
				select {
				case <-ctx.Done():
					fileReader.Close()
					close(recordChan) // Close recordChan to unblock workers
					return
				case recordChan <- rec:
				}

				if lineCount%100 == 0 {
					if totalSize > 0 && !opts.Silent {
						bytesRead += int64(len(rec.Content))
						pct := (float64(bytesRead) / float64(totalSize)) * 100
						if pct > 100 {
							pct = 100
						}
						fmt.Fprintf(os.Stderr, "\r%s Scanning %s: %.1f%% (%d records)", au.Cyan("[*]"), filepath.Base(path), pct, lineCount)
					}
					if opts.ResumeFile != "" {
						s.Resume.FileIndex = i
						s.Resume.LineIndex = lineCount
						s.Resume.Target = path
						s.SaveResumeState(opts.ResumeFile)
					}
				}
			}
			fileReader.Close()
			if !opts.Silent && totalSize > 0 {
				fmt.Fprintf(os.Stderr, "\r%s Scanned %s: 100%%          \n", au.Green("[+]"), filepath.Base(path))
			}
			if opts.ResumeFile != "" {
				s.Resume.FileIndex = i + 1
				s.Resume.LineIndex = 0
				s.Resume.Target = ""
				s.SaveResumeState(opts.ResumeFile)
			}
		}
		close(recordChan) // All records sent, close channel to unblock workers
		innerWg.Wait()    // Wait for all workers to finish
	}()

	return resultChan, nil
}

func (s *ScannerService) getActiveTools(toolIDs []string) []models.Tool {
	if len(toolIDs) == 0 {
		return nil
	}
	var active []models.Tool
	for _, id := range toolIDs {
		id = strings.TrimSpace(id)
		for _, t := range s.Tools {
			if t.ID == id {
				active = append(active, t)
			}
		}
	}
	return active
}

func (s *ScannerService) ProcessResults(ctx context.Context, resultsFile string, toolIDs []string) (<-chan *models.Result, error) {
	resultsFile = utils.ExpandPath(resultsFile)
	resultChan := make(chan *models.Result, 2000)
	activeTools := s.getActiveTools(toolIDs)
	b, err := os.ReadFile(resultsFile)
	if err != nil {
		return nil, err
	}
	var results []*models.Result
	if err := json.Unmarshal(b, &results); err != nil {
		return nil, err
	}
	go func() {
		defer close(resultChan)
		for _, res := range results {
			for _, t := range activeTools {
				if val, _ := s.executeToolWithLimit(t, *res); val != "" {
					res.ToolData = append(res.ToolData, models.ToolOutput{ToolID: t.ID, Label: t.Field, Value: val})
				}
			}
			select {
			case <-ctx.Done():
				return
			case resultChan <- res:
			}
		}
	}()
	return resultChan, nil
}

func (s *ScannerService) DiagnoseLine(line string, patterns []string) []string {
	var logs []string
	logs = append(logs, fmt.Sprintf("%s Testing input line (Format: %s): %s", au.Bold(au.Cyan("[DEBUG]")), s.Config.Input.Format, line))

	if line == "" {
		logs = append(logs, fmt.Sprintf("%s Line is empty", au.Red("[ERROR]")))
		return logs
	}

	var contents []string
	var idVal string = "unknown"

	if s.Config.Input.Format == "csv" {
		sep := s.Config.Input.CSVConfig.Separator
		if sep == "" {
			sep = ","
		}
		reader := csv.NewReader(strings.NewReader(line))
		reader.Comma = rune(sep[0])
		record, err := reader.Read()
		if err != nil {
			logs = append(logs, fmt.Sprintf("%s CSV Parse failed: %v", au.Red("[ERROR]"), err))
			return logs
		}
		logs = append(logs, fmt.Sprintf("%s CSV parsed successfully (%d columns)", au.Green("[SUCCESS]"), len(record)))

		idIdx := s.Config.Input.CSVConfig.IDIndex
		if idIdx < len(record) {
			idVal = record[idIdx]
		}

		targetIdxs := s.Config.Input.CSVConfig.TargetIdx
		if len(targetIdxs) == 0 {
			for i := range record {
				targetIdxs = append(targetIdxs, i)
			}
		}

		for _, idx := range targetIdxs {
			if idx < len(record) {
				contents = append(contents, record[idx])
			}
		}
	} else {
		var data map[string]interface{}
		err := json.Unmarshal([]byte(line), &data)
		if err != nil {
			if s.Config.Input.Format != "text" {
				logs = append(logs, fmt.Sprintf("%s JSON Unmarshal failed: %v. Only '$' target will work.", au.Yellow("[WARN]"), err))
			}
		} else {
			logs = append(logs, fmt.Sprintf("%s JSON parsed successfully", au.Green("[SUCCESS]")))
		}

		if err == nil {
			for field, val := range s.Config.Input.Filters {
				v, ok := getNestedField(data, field)
				if !ok {
					logs = append(logs, fmt.Sprintf("%s Field '%s' missing. %s", au.Yellow("[FILTER]"), field, au.Red("SKIP.")))
					return logs
				}
				if v != val {
					logs = append(logs, fmt.Sprintf("%s Field '%s' value '%s' != '%s'. %s", au.Yellow("[FILTER]"), field, v, val, au.Red("SKIP.")))
					return logs
				}
				logs = append(logs, fmt.Sprintf("%s Field '%s' matches '%s'. %s", au.Yellow("[FILTER]"), field, val, au.Green("PASS.")))
			}
		}

		idField := s.Config.Input.ID
		if err == nil {
			idVal, _ = getNestedField(data, idField)
			if idVal == "" {
				idVal = "unknown"
			}
		}

		var targets []string
		if s.Config.Input.Target != "" {
			targets = append(targets, s.Config.Input.Target)
		}
		targets = append(targets, s.Config.Input.Targets...)

		// If no targets defined or format is text, default to raw line ($)
		if len(targets) == 0 || s.Config.Input.Format == "text" {
			targets = append(targets, "$")
		}

		for _, targetField := range targets {
			var content string
			var ok bool
			if targetField == "$" {
				content = line
				ok = true
				logs = append(logs, fmt.Sprintf("%s Added target '$' (Raw Line)", au.Blue("[TARGET]")))
			} else if err == nil {
				content, ok = getNestedField(data, targetField)
				if ok {
					logs = append(logs, fmt.Sprintf("%s Found field '%s'.", au.Blue("[TARGET]"), targetField))
				}
			}

			if ok {
				if s.Config.Input.Decode {
					oldLen := len(content)
					content = unescapeContent(content)
					logs = append(logs, fmt.Sprintf("%s Unescaped content (Length: %d -> %d)", au.Magenta("[DECODE]"), oldLen, len(content)))
				}

				if cmdStr, exists := s.Config.Input.PostProcess[targetField]; exists {
					logs = append(logs, fmt.Sprintf("%s Running PostProcess: %s", au.Yellow("[PRE]"), cmdStr))
					cmd := exec.CommandContext(context.Background(), "bash", "-c", "echo '"+strings.ReplaceAll(content, "'", "'\\''")+"' | "+cmdStr)
					if out, err := cmd.Output(); err == nil {
						content = string(out)
						logs = append(logs, fmt.Sprintf("%s Transformation complete.", au.Green("[POST]")))
					}
				}
				contents = append(contents, content)
			}
		}
	}

	if len(contents) == 0 && s.Config.Input.Format != "csv" {
		logs = append(logs, fmt.Sprintf("%s No targets matched!", au.Red("[ERROR]")))
		return logs
	}

	for _, content := range contents {
		if content == "" {
			continue
		}
		for _, pName := range patterns {
			if pName == "" {
				continue
			}
			p, perr := s.getPattern(pName)
			if perr != nil {
				continue
			}
			re, rerr := regexp.Compile(p.Pattern)
			if rerr != nil {
				logs = append(logs, fmt.Sprintf("%s Pattern '%s' invalid regex: %v", au.Red("[PATTERN]"), pName, rerr))
				continue
			}
			if matches := re.FindAllStringSubmatch(content, -1); len(matches) > 0 {
				logs = append(logs, fmt.Sprintf("%s Pattern '%s' %s %d times in content: %s", au.Green("[MATCH]"), pName, au.Bold("hit"), len(matches), content))
			}
		}
	}

	return logs
}

func (s *ScannerService) LoadResumeState(file string) error {
	file = utils.ExpandPath(file)
	b, err := os.ReadFile(file)
	if err != nil {
		return err
	}
	return json.Unmarshal(b, &s.Resume)
}

func (s *ScannerService) SaveResumeState(file string) error {
	file = utils.ExpandPath(file)
	b, err := json.MarshalIndent(s.Resume, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(file, b, 0644)
}

func (s *ScannerService) FilterPatternsByTag(tags []string) []string {
	if len(tags) == 0 {
		return nil
	}
	var matched []string
	pats, _ := GetPatterns(s.Config.PatternsDir)
	for _, pName := range pats {
		p, _ := s.getPattern(pName)
		for _, t := range tags {
			for _, pt := range p.Tags {
				if t == pt {
					matched = append(matched, pName)
					break
				}
			}
		}
	}
	return matched
}

func (s *ScannerService) RewriteFile(ctx context.Context, path string) error {
	path = utils.ExpandPath(path)
	fmt.Printf("%s %s: This will modify the target file in-place! Continue? [y/N]: ", au.Bold(au.Red("[WARNING]")), path)
	var confirm string
	fmt.Scanln(&confirm)
	if strings.ToLower(confirm) != "y" {
		return fmt.Errorf("operation cancelled by user")
	}
	tempPath := path + ".tmp"
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	out, err := os.Create(tempPath)
	if err != nil {
		return err
	}
	defer out.Close()
	scanner := bufio.NewScanner(f)
	buf := make([]byte, 64*1024)
	scanner.Buffer(buf, 20*1024*1024)
	lineCount := 0
	for scanner.Scan() {
		line := scanner.Text()
		lineCount++
		var data map[string]interface{}
		if err := json.Unmarshal([]byte(line), &data); err == nil {
			for field, cmdStr := range s.Config.Input.PostProcess {
				if val, ok := getNestedField(data, field); ok {
					cmd := exec.CommandContext(ctx, "bash", "-c", "echo '"+strings.ReplaceAll(val, "'", "'\\''")+"' | "+cmdStr)
					if processed, err := cmd.Output(); err == nil {
						data[field] = strings.TrimSpace(string(processed))
					}
				}
			}
			newData, _ := json.Marshal(data)
			out.Write(newData)
			out.WriteString("\n")
		} else {
			out.WriteString(line + "\n")
		}
		if lineCount%100 == 0 {
			fmt.Fprintf(os.Stderr, "\r%s Rewriting %s: %d lines processed", au.Yellow("[*]"), filepath.Base(path), lineCount)
		}
	}
	os.Rename(tempPath, path)
	fmt.Printf("\n%s Successfully beautified %s\n", au.Green("[+]"), path)
	return nil
}
