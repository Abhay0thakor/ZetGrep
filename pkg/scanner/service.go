package scanner

import (
	"bufio"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"regexp"
	"runtime"
	"fmt"
	"encoding/csv"
	"io"
	"os/exec"

	"github.com/logrusorgru/aurora"
	"github.com/Abhay0thakor/ZetGrep/pkg/classifier"
	"github.com/Abhay0thakor/ZetGrep/pkg/models"
	"github.com/Abhay0thakor/ZetGrep/pkg/utils"
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
}

type ScannerService struct {
	Engine       Engine
	Fallback     Engine
	Config       models.Config
	Classifier   *classifier.Classifier
	Tools        []models.Tool
	patternCache sync.Map
	seenMatches  sync.Map
	processSem chan struct{}
	Resume     models.ResumeConfig
}

var au = aurora.NewAurora(true)

func NewScannerService(cfg models.Config) (*ScannerService, error) {
	engine, err := NewRipgrepEngine()
	var fallback Engine
	if err != nil {
		fallback, _ = NewGrepEngine()
	}

	if cfg.PatternsDir == "" {
		cfg.PatternsDir, _ = GetPatternDir()
	}
	if cfg.ToolsDir == "" {
		cfg.ToolsDir, _ = GetToolDir()
	}

	maxProc := runtime.NumCPU() * 2
	if maxProc > 50 { maxProc = 50 }

	return &ScannerService{
		Engine:       engine,
		Fallback:     fallback,
		Config:       cfg,
		Classifier:   classifier.DefaultClassifier(),
		Tools:        LoadToolsFrom(cfg.ToolsDir),
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

// getNestedField retrieves a value from a nested map using dot notation
func getNestedField(data map[string]interface{}, path string) (string, bool) {
	parts := strings.Split(path, ".")
	var current interface{} = data
	for _, part := range parts {
		if m, ok := current.(map[string]interface{}); ok {
			current = m[part]
		} else {
			return "", false
		}
	}
	
	var val string
	if str, ok := current.(string); ok {
		val = str
	} else if current != nil {
		val = fmt.Sprintf("%v", current)
	} else {
		return "", false
	}

	return val, true
}

func unescapeContent(s string) string {
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
	if !opts.Silent { fmt.Fprintf(os.Stderr, "%s Detected Format: %s\n", au.Cyan("[DEBUG]"), s.Config.Input.Format) }
	if s.Config.Input.Format == "jsonl" || s.Config.Input.Format == "json" {
		return s.RunJSONLScan(ctx, opts)
	}
	if s.Config.Input.Format == "csv" {
		return s.RunCSVScan(ctx, opts)
	}

	// Use RunTextScan for everything else (text, html, js) to support pipelines
	return s.RunTextScan(ctx, opts)
}

func (s *ScannerService) RunTextScan(ctx context.Context, opts ScannerOptions) (<-chan *models.Result, error) {
	if !opts.Silent { fmt.Fprintf(os.Stderr, "%s Using Text Scanning Mode\n", au.Cyan("[*]")) }
	resultChan := make(chan *models.Result, 1000)

	var compiledPatterns []struct {
		p    models.Pattern
		comp *regexp.Regexp
	}

	for _, pName := range opts.Patterns {
		p, err := s.getPattern(pName); if err != nil { continue }
		
		finalPattern := p.Pattern
		if strings.Contains(p.Flags, "i") && !strings.HasPrefix(finalPattern, "(?i)") {
			finalPattern = "(?i)" + finalPattern
		}

		if comp, err := regexp.Compile(finalPattern); err == nil {
			compiledPatterns = append(compiledPatterns, struct { p models.Pattern; comp *regexp.Regexp }{p, comp})
		}
	}

	activeTools := s.getActiveTools(opts.ToolIDs)
	lineChan := make(chan string, 1000)
	var wg sync.WaitGroup
	numWorkers := runtime.NumCPU() * 2

	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for line := range lineChan {
				if line == "" { continue }
				
				content := line
				if s.Config.Input.Decode { content = unescapeContent(content) }
				
				// Post-process for text usually applies to the whole line
				if cmdStr, exists := s.Config.Input.PostProcess["$"]; exists {
					cmd := exec.CommandContext(ctx, "bash", "-c", "echo '"+strings.ReplaceAll(content, "'", "'\\''")+"' | "+cmdStr)
					if out, err := cmd.Output(); err == nil { content = string(out) }
				}

				for _, cp := range compiledPatterns {
					matches := cp.comp.FindAllStringSubmatch(content, -1)
					for _, matchGroup := range matches {
						if len(matchGroup) == 0 || matchGroup[0] == "" { continue }
						res := GetResult()
						res.Pattern = cp.p.Name; res.Content = matchGroup[0]; res.Matches = matchGroup; res.Entropy = utils.ShannonEntropy(res.Content)
						
						if opts.Unique {
							key := res.Pattern + ":" + res.Content
							if _, seen := s.seenMatches.LoadOrStore(key, true); seen {
								PutResult(res); continue
							}
						}

						// ONLY filter if explicitly requested
						if opts.SmartMode && s.Classifier.Classify(res.Content) != "high-interest" {
							PutResult(res); continue
						}
						if opts.EntropyMode && res.Entropy < 3.5 {
							PutResult(res); continue
						}
						for _, t := range activeTools {
							if val, _ := s.executeToolWithLimit(t, *res); val != "" {
								res.ToolData = append(res.ToolData, models.ToolOutput{ToolID: t.ID, Label: t.Field, Value: val})
							}
						}
						select { case <-ctx.Done(): return; case resultChan <- res: }
					}
				}
			}
		}()
	}

	go func() {
		var lineCount int
		for _, path := range opts.TargetPaths {
			var fileReader io.ReadCloser; var totalSize int64
			if path != "stdin" && path != "-" { if info, err := os.Stat(path); err == nil { totalSize = info.Size() } }
			
			if s.Config.Input.PreProcess != "" && path != "stdin" && path != "-" {
				cmd := exec.CommandContext(ctx, "bash", "-c", s.Config.Input.PreProcess+" "+path)
				if stdout, err := cmd.StdoutPipe(); err == nil && cmd.Start() == nil {
					fileReader = stdout; defer cmd.Wait()
				}
			}
			if fileReader == nil {
				if path == "-" || path == "stdin" { fileReader = io.NopCloser(os.Stdin)
				} else { f, err := os.Open(path); if err != nil { continue }; fileReader = f }
			}
			scanner := bufio.NewScanner(fileReader); buf := make([]byte, 1024*1024); scanner.Buffer(buf, 100*1024*1024)
			var bytesRead int64
			for scanner.Scan() {
				text := scanner.Text(); lineCount++
				bytesRead += int64(len(text)) + 1
				select { case <-ctx.Done(): fileReader.Close(); goto done; case lineChan <- text: }
				if lineCount%500 == 0 && totalSize > 0 && !opts.Silent {
					pct := (float64(bytesRead) / float64(totalSize)) * 100
					fmt.Fprintf(os.Stderr, "\r%s Scanning %s: %.1f%% (%d lines)", au.Cyan("[*]"), filepath.Base(path), pct, lineCount)
				}
			}
			fileReader.Close()
			if !opts.Silent && totalSize > 0 { fmt.Fprintf(os.Stderr, "\r%s Scanned %s: 100%%          \n", au.Green("[+]"), filepath.Base(path)) }
		}
	done:
		close(lineChan); wg.Wait(); close(resultChan)
	}()
	return resultChan, nil
}

func (s *ScannerService) RunJSONLScan(ctx context.Context, opts ScannerOptions) (<-chan *models.Result, error) {
	resultChan := make(chan *models.Result, 1000)

	var compiledPatterns []struct {
		p    models.Pattern
		comp *regexp.Regexp
	}

	for _, pName := range opts.Patterns {
		p, err := s.getPattern(pName)
		if err != nil { continue }
		comp, err := regexp.Compile(p.Pattern)
		if err == nil {
			compiledPatterns = append(compiledPatterns, struct {
				p    models.Pattern
				comp *regexp.Regexp
			}{p, comp})
		}
	}

	activeTools := s.getActiveTools(opts.ToolIDs)
	
	var targets []string
	if s.Config.Input.Target != "" { targets = append(targets, s.Config.Input.Target) }
	targets = append(targets, s.Config.Input.Targets...)
	
	idField := s.Config.Input.ID
	filters := s.Config.Input.Filters

	lineChan := make(chan string, 1000)
	var wg sync.WaitGroup
	numWorkers := runtime.NumCPU() * 2

	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for line := range lineChan {
				if line == "" { continue }
				var data map[string]interface{}
				err := json.Unmarshal([]byte(line), &data)
				if err == nil {
					matchFilters := true
					for field, val := range filters {
						if v, ok := getNestedField(data, field); !ok || v != val {
							matchFilters = false
							break
						}
					}
					if !matchFilters { continue }
				}

				idVal := "unknown"
				if err == nil {
					idVal, _ = getNestedField(data, idField)
					if idVal == "" { idVal = "unknown" }
				}

				for _, targetField := range targets {
					var content string
					var ok bool
					if targetField == "$" {
						content = line
						ok = true
					} else if err == nil {
						content, ok = getNestedField(data, targetField)
					}
					if !ok || content == "" { continue }

					if s.Config.Input.Decode {
						content = unescapeContent(content)
					}

					if cmdStr, exists := s.Config.Input.PostProcess[targetField]; exists {
						cmd := exec.CommandContext(ctx, "bash", "-c", "echo '"+strings.ReplaceAll(content, "'", "'\\''")+"' | "+cmdStr)
						if out, err := cmd.Output(); err == nil { content = string(out) }
					}

					for _, cp := range compiledPatterns {
						matches := cp.comp.FindAllStringSubmatch(content, -1)
						for _, matchGroup := range matches {
							if len(matchGroup) == 0 || matchGroup[0] == "" { continue }
							match := matchGroup[0]
							res := GetResult()
							res.Pattern = cp.p.Name
							res.File = idVal
							res.Content = match
							res.Matches = matchGroup
							res.Entropy = utils.ShannonEntropy(match)

							if opts.Unique {
								key := res.Pattern + ":" + res.Content
								if _, seen := s.seenMatches.LoadOrStore(key, true); seen {
									PutResult(res); continue
								}
							}

							if opts.SmartMode && s.Classifier.Classify(res.Content) != "high-interest" {
								PutResult(res); continue
							}
							if opts.EntropyMode && res.Entropy < 3.5 {
								PutResult(res); continue
							}
							for _, t := range activeTools {
								if val, _ := s.executeToolWithLimit(t, *res); val != "" {
									res.ToolData = append(res.ToolData, models.ToolOutput{ToolID: t.ID, Label: t.Field, Value: val})
								}
							}
							select {
							case <-ctx.Done(): return
							case resultChan <- res:
							}
						}
					}
				}
			}
		}()
	}

	go func() {
		var lineCount int
		for i, path := range opts.TargetPaths {
			if opts.ResumeFile != "" && i < s.Resume.FileIndex { continue }
			var fileReader io.ReadCloser
			var totalSize int64
			if path != "stdin" && path != "-" {
				if info, err := os.Stat(path); err == nil { totalSize = info.Size() }
			}
			if s.Config.Input.PreProcess != "" && path != "stdin" && path != "-" {
				cmd := exec.CommandContext(ctx, "bash", "-c", s.Config.Input.PreProcess+" "+path)
				if stdout, err := cmd.StdoutPipe(); err == nil && cmd.Start() == nil {
					fileReader = stdout
					defer cmd.Wait()
				}
			}
			if fileReader == nil {
				if path == "-" || path == "stdin" { fileReader = io.NopCloser(os.Stdin)
				} else {
					f, err := os.Open(path); if err != nil { continue }; fileReader = f
				}
			}
			scanner := bufio.NewScanner(fileReader)
			buf := make([]byte, 64*1024); scanner.Buffer(buf, 20*1024*1024)
			var bytesRead int64
			for scanner.Scan() {
				text := scanner.Text(); lineCount++
				bytesRead += int64(len(text)) + 1
				if opts.ResumeFile != "" && i == s.Resume.FileIndex && lineCount <= s.Resume.LineIndex { continue }
				select {
				case <-ctx.Done(): fileReader.Close(); goto done
				case lineChan <- text:
				}
				if lineCount%100 == 0 {
					if totalSize > 0 && !opts.Silent {
						pct := (float64(bytesRead) / float64(totalSize)) * 100
						if pct > 100 { pct = 100 }
						fmt.Fprintf(os.Stderr, "\r%s Scanning %s: %.1f%% (%d lines)", au.Cyan("[*]"), filepath.Base(path), pct, lineCount)
					}
					if opts.ResumeFile != "" {
						s.Resume.FileIndex = i; s.Resume.LineIndex = lineCount; s.Resume.Target = path
						s.SaveResumeState(opts.ResumeFile)
					}
				}
			}
			fileReader.Close()
			if !opts.Silent && totalSize > 0 { fmt.Fprintf(os.Stderr, "\r%s Scanned %s: 100%%          \n", au.Green("[+]"), filepath.Base(path)) }
			if opts.ResumeFile != "" {
				s.Resume.FileIndex = i + 1; s.Resume.LineIndex = 0; s.Resume.Target = ""; s.SaveResumeState(opts.ResumeFile)
			}
			lineCount = 0
		}
	done:
		close(lineChan); wg.Wait(); close(resultChan)
	}()
	return resultChan, nil
}

func (s *ScannerService) RunCSVScan(ctx context.Context, opts ScannerOptions) (<-chan *models.Result, error) {
	resultChan := make(chan *models.Result, 1000)
	var compiledPatterns []struct {
		p    models.Pattern
		comp *regexp.Regexp
	}
	for _, pName := range opts.Patterns {
		p, err := s.getPattern(pName); if err != nil { continue }
		
		finalPattern := p.Pattern
		if strings.Contains(p.Flags, "i") && !strings.HasPrefix(finalPattern, "(?i)") {
			finalPattern = "(?i)" + finalPattern
		}

		if comp, err := regexp.Compile(finalPattern); err == nil {
			compiledPatterns = append(compiledPatterns, struct { p models.Pattern; comp *regexp.Regexp }{p, comp})
		}
	}
	activeTools := s.getActiveTools(opts.ToolIDs)
	separator := s.Config.Input.CSVConfig.Separator; if separator == "" { separator = "," }
	idIdx := s.Config.Input.CSVConfig.IDIndex; targetIdxs := s.Config.Input.CSVConfig.TargetIdx
	recordChan := make(chan []string, 1000)
	var wg sync.WaitGroup
	numWorkers := runtime.NumCPU() * 2
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for record := range recordChan {
				currentTargets := targetIdxs
				if len(currentTargets) == 0 { for idx := range record { currentTargets = append(currentTargets, idx) } }
				idVal := "unknown"; if idIdx < len(record) { idVal = record[idIdx] }
				for _, tIdx := range currentTargets {
					if tIdx >= len(record) { continue }
					content := record[tIdx]
					for _, cp := range compiledPatterns {
						matches := cp.comp.FindAllStringSubmatch(content, -1)
						for _, matchGroup := range matches {
							if len(matchGroup) == 0 || matchGroup[0] == "" { continue }
							match := matchGroup[0]; res := GetResult()
							res.Pattern = cp.p.Name; res.File = idVal; res.Content = match; res.Matches = matchGroup; res.Entropy = utils.ShannonEntropy(match)
							if (opts.SmartMode && s.Classifier.Classify(res.Content) != "high-interest") || (opts.EntropyMode && res.Entropy < 3.5) {
								PutResult(res); continue
							}
							for _, t := range activeTools {
								if val, _ := s.executeToolWithLimit(t, *res); val != "" {
									res.ToolData = append(res.ToolData, models.ToolOutput{ToolID: t.ID, Label: t.Field, Value: val})
								}
							}
							select { case <-ctx.Done(): return; case resultChan <- res: }
						}
					}
				}
			}
		}()
	}
	go func() {
		for _, path := range opts.TargetPaths {
			var fileReader io.ReadCloser; var totalSize int64
			if path != "-" && path != "stdin" { if info, err := os.Stat(path); err == nil { totalSize = info.Size() } }
			if path == "-" || path == "stdin" { fileReader = io.NopCloser(os.Stdin)
			} else { f, err := os.Open(path); if err != nil { continue }; fileReader = f }
			reader := csv.NewReader(fileReader); reader.Comma = rune(separator[0]); reader.LazyQuotes = true
			if s.Config.Input.CSVConfig.HasHeader { reader.Read() }
			var bytesRead int64; var rowCount int
			for {
				record, err := reader.Read(); if err == io.EOF { break }; if err != nil { continue }
				rowCount++
				bytesRead += int64(len(strings.Join(record, separator))) + 1
				if rowCount%500 == 0 && totalSize > 0 && !opts.Silent {
					pct := (float64(bytesRead) / float64(totalSize)) * 100; if pct > 100 { pct = 100 }
					fmt.Fprintf(os.Stderr, "\r%s Scanning CSV %s: %.1f%%", au.Cyan("[*]"), filepath.Base(path), pct)
				}
				select { case <-ctx.Done(): fileReader.Close(); goto done; case recordChan <- record: }
			}
			fileReader.Close()
			if !opts.Silent && totalSize > 0 { fmt.Fprintf(os.Stderr, "\r%s Scanned CSV %s: 100%%          \n", au.Green("[+]"), filepath.Base(path)) }
		}
	done:
		close(recordChan); wg.Wait(); close(resultChan)
	}()
	return resultChan, nil
}

func (s *ScannerService) getActiveTools(toolIDs []string) []models.Tool {
	if len(toolIDs) == 0 { return nil }
	var active []models.Tool
	for _, id := range toolIDs {
		id = strings.TrimSpace(id)
		for _, t := range s.Tools { if t.ID == id { active = append(active, t) } }
	}
	return active
}

func (s *ScannerService) ProcessResults(ctx context.Context, resultsFile string, toolIDs []string) (<-chan *models.Result, error) {
	resultsFile = utils.ExpandPath(resultsFile)
	resultChan := make(chan *models.Result, 2000)
	activeTools := s.getActiveTools(toolIDs)
	b, err := os.ReadFile(resultsFile); if err != nil { return nil, err }
	var results []*models.Result; if err := json.Unmarshal(b, &results); err != nil { return nil, err }
	go func() {
		defer close(resultChan)
		for _, res := range results {
			for _, t := range activeTools {
				if val, _ := s.executeToolWithLimit(t, *res); val != "" {
					res.ToolData = append(res.ToolData, models.ToolOutput{ToolID: t.ID, Label: t.Field, Value: val})
				}
			}
			select { case <-ctx.Done(): return; case resultChan <- res: }
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
		if sep == "" { sep = "," }
		reader := csv.NewReader(strings.NewReader(line))
		reader.Comma = rune(sep[0])
		record, err := reader.Read()
		if err != nil {
			logs = append(logs, fmt.Sprintf("%s CSV Parse failed: %v", au.Red("[ERROR]"), err))
			return logs
		}
		logs = append(logs, fmt.Sprintf("%s CSV parsed successfully (%d columns)", au.Green("[SUCCESS]"), len(record)))
		
		idIdx := s.Config.Input.CSVConfig.IDIndex
		if idIdx < len(record) { idVal = record[idIdx] }
		
		targetIdxs := s.Config.Input.CSVConfig.TargetIdx
		if len(targetIdxs) == 0 {
			for i := range record { targetIdxs = append(targetIdxs, i) }
		}
		
		for _, idx := range targetIdxs {
			if idx < len(record) { contents = append(contents, record[idx]) }
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
			if idVal == "" { idVal = "unknown" }
		}

		var targets []string
		if s.Config.Input.Target != "" { targets = append(targets, s.Config.Input.Target) }
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
		if content == "" { continue }
		for _, pName := range patterns {
			if pName == "" { continue }
			p, perr := s.getPattern(pName)
			if perr != nil { continue }
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
	file = utils.ExpandPath(file); b, err := os.ReadFile(file); if err != nil { return err }
	return json.Unmarshal(b, &s.Resume)
}

func (s *ScannerService) SaveResumeState(file string) error {
	file = utils.ExpandPath(file); b, _ := json.MarshalIndent(s.Resume, "", "  ")
	return os.WriteFile(file, b, 0644)
}

func (s *ScannerService) FilterPatternsByTag(tags []string) []string {
	if len(tags) == 0 { return nil }
	var matched []string; pats, _ := GetPatterns(s.Config.PatternsDir)
	for _, pName := range pats {
		p, _ := s.getPattern(pName)
		for _, t := range tags { for _, pt := range p.Tags { if t == pt { matched = append(matched, pName); break } } }
	}
	return matched
}

func (s *ScannerService) RewriteFile(ctx context.Context, path string) error {
	path = utils.ExpandPath(path)
	fmt.Printf("%s %s: This will modify the target file in-place! Continue? [y/N]: ", au.Bold(au.Red("[WARNING]")), path)
	var confirm string; fmt.Scanln(&confirm)
	if strings.ToLower(confirm) != "y" { return fmt.Errorf("operation cancelled by user") }
	tempPath := path + ".tmp"; f, err := os.Open(path); if err != nil { return err }; defer f.Close()
	out, err := os.Create(tempPath); if err != nil { return err }; defer out.Close()
	scanner := bufio.NewScanner(f); buf := make([]byte, 64*1024); scanner.Buffer(buf, 20*1024*1024)
	lineCount := 0
	for scanner.Scan() {
		line := scanner.Text(); lineCount++
		var data map[string]interface{}
		if err := json.Unmarshal([]byte(line), &data); err == nil {
			for field, cmdStr := range s.Config.Input.PostProcess {
				if val, ok := getNestedField(data, field); ok {
					cmd := exec.CommandContext(ctx, "bash", "-c", "echo '"+strings.ReplaceAll(val, "'", "'\\''")+"' | "+cmdStr)
					if processed, err := cmd.Output(); err == nil { data[field] = strings.TrimSpace(string(processed)) }
				}
			}
			newData, _ := json.Marshal(data); out.Write(newData); out.WriteString("\n")
		} else { out.WriteString(line + "\n") }
		if lineCount%100 == 0 { fmt.Fprintf(os.Stderr, "\r%s Rewriting %s: %d lines processed", au.Yellow("[*]"), filepath.Base(path), lineCount) }
	}
	os.Rename(tempPath, path); fmt.Printf("\n%s Successfully beautified %s\n", au.Green("[+]"), path); return nil
}
