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
	"sync/atomic"
	"time"

	"github.com/Abhay0thakor/ZetGrep/pkg/classifier"
	"github.com/Abhay0thakor/ZetGrep/pkg/models"
	"github.com/Abhay0thakor/ZetGrep/pkg/utils"
	"github.com/logrusorgru/aurora"
)

type ScannerOptions struct {
	TargetPaths      []string
	Patterns         []string
	Tags             []string
	ToolIDs          []string
	SmartMode        bool
	EntropyMode      bool
	Unique           bool
	ResumeFile       string
	Silent           bool
	Concurrency      int
	Notify           bool
	NotifyInterval   int
	CooldownEvery    int
	CooldownTime     string
	ThermalThreshold float64
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
	isPaused     int32
}

var au = aurora.NewAurora(true)

type progressReader struct {
	r      io.Reader
	onRead func(int)
}

func (pr *progressReader) Read(p []byte) (n int, err error) {
	n, err = pr.r.Read(p)
	if n > 0 && pr.onRead != nil {
		pr.onRead(n)
	}
	return
}

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
		p = &FastJSONParser{Config: cfg.Input}
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
	s = strings.ReplaceAll(s, "\\n", "\n")
	s = strings.ReplaceAll(s, "\\r", "\r")
	s = strings.ReplaceAll(s, "\\t", "\t")
	s = strings.ReplaceAll(s, "\\\"", "\"")
	if strings.Contains(s, "\\u") {
		var decoded string
		if err := json.Unmarshal([]byte("\""+s+"\""), &decoded); err == nil {
			s = decoded
		}
	}
	return s
}

func (s *ScannerService) resolveTargets(paths []string) []string {
	var resolved []string
	for _, path := range paths {
		if path == "stdin" || path == "-" {
			resolved = append(resolved, path)
			continue
		}
		info, err := os.Stat(path)
		if err != nil {
			slog.Debug("Error stating path", "path", path, "error", err)
			continue
		}
		if info.IsDir() {
			filepath.Walk(path, func(p string, i os.FileInfo, e error) error {
				if e != nil {
					return nil
				}
				if !i.IsDir() {
					resolved = append(resolved, p)
				}
				return nil
			})
		} else {
			resolved = append(resolved, path)
		}
	}
	return resolved
}

func (s *ScannerService) sendNotification(msg string) {
	cmd := exec.Command("notify")
	cmd.Stdin = strings.NewReader(msg)
	if err := cmd.Run(); err != nil {
		slog.Debug("Failed to send notification", "error", err)
	}
}

func (s *ScannerService) handleMatch(res *models.Result, opts ScannerOptions, activeTools []models.Tool, hitCounter interface{}, resultChan chan<- *models.Result, ctx context.Context) {
	if opts.Unique {
		key := res.Pattern + ":" + res.Content
		if _, seen := s.seenMatches.LoadOrStore(key, true); seen {
			PutResult(res)
			return
		}
	}

	if opts.SmartMode && s.Classifier.Classify(res.Content) != "high-interest" {
		PutResult(res)
		return
	}
	if opts.EntropyMode && res.Entropy < 3.5 {
		PutResult(res)
		return
	}

	for _, t := range activeTools {
		if val, _ := s.executeToolWithLimit(t, *res); val != "" {
			res.ToolData = append(res.ToolData, models.ToolOutput{ToolID: t.ID, Label: t.Field, Value: val})
		}
	}

	if hc, ok := hitCounter.(*struct {
		sync.Mutex
		count int
	}); ok {
		hc.Lock()
		hc.count++
		hc.Unlock()
	}

	select {
	case <-ctx.Done():
		return
	case resultChan <- res:
	}
}

func (s *ScannerService) RunScan(ctx context.Context, opts ScannerOptions) (<-chan *models.Result, error) {
	slog.Debug("Scan started", "format", s.Config.Input.Format)
	resultChan := make(chan *models.Result, 1000)

	if opts.ThermalThreshold > 0 {
		go utils.MonitorThermal(opts.ThermalThreshold, opts.ThermalThreshold-15.0, func() {
			atomic.StoreInt32(&s.isPaused, 1)
			msg := fmt.Sprintf("🌡️ Thermal Protection: CPU temperature reached %.1f°C. Pausing scan...", opts.ThermalThreshold)
			slog.Warn(msg)
			if opts.Notify {
				s.sendNotification(msg)
			}
		}, func() {
			atomic.StoreInt32(&s.isPaused, 0)
			msg := "🟢 CPU Cooled down. Resuming scan."
			slog.Info(msg)
			if opts.Notify {
				s.sendNotification(msg)
			}
		})
	}

	var compiledPatterns []struct {
		p    models.Pattern
		comp *regexp.Regexp
	}
	literals := make(map[string]string)

	for _, pName := range opts.Patterns {
		p, err := s.getPattern(pName)
		if err != nil {
			slog.Debug("Failed to get pattern", "name", pName, "error", err)
			continue
		}

		finalPattern := p.Pattern
		isCaseInsensitive := strings.Contains(p.Flags, "i")

		if !isCaseInsensitive && IsLiteral(finalPattern) {
			literals[finalPattern] = pName
			continue
		}

		if isCaseInsensitive && !strings.HasPrefix(finalPattern, "(?i)") {
			finalPattern = "(?i)" + finalPattern
		}

		if comp, err := regexp.Compile(finalPattern); err == nil {
			compiledPatterns = append(compiledPatterns, struct {
				p    models.Pattern
				comp *regexp.Regexp
			}{p, comp})
		}
	}

	litMatcher := NewLiteralMatcher(literals)
	activeTools := s.getActiveTools(opts.ToolIDs)
	recordChan := make(chan ScanRecord, 1000)
	var wg sync.WaitGroup
	numWorkers := opts.Concurrency
	if numWorkers <= 0 {
		numWorkers = runtime.NumCPU() * 2
	}

	hitCounter := struct {
		sync.Mutex
		count int
	}{}

	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for rec := range recordChan {
				for atomic.LoadInt32(&s.isPaused) == 1 {
					time.Sleep(5 * time.Second)
				}

				content := rec.Content
				if s.Config.Input.Decode {
					content = []byte(unescapeContent(string(content)))
				}

				postCmd := ""
				if cmd, ok := s.Config.Input.PostProcess[rec.ID]; ok {
					postCmd = cmd
				} else if cmd, ok := s.Config.Input.PostProcess["$"]; ok {
					postCmd = cmd
				}

				if postCmd != "" {
					var cmd *exec.Cmd
					if runtime.GOOS == "windows" {
						cmd = exec.CommandContext(ctx, "cmd", "/c", "echo "+string(content)+" | "+postCmd)
					} else {
						cmd = exec.CommandContext(ctx, "sh", "-c", "echo '"+strings.ReplaceAll(string(content), "'", "'\\''")+"' | "+postCmd)
					}
					if out, err := cmd.Output(); err == nil {
						content = []byte(strings.TrimSpace(string(out)))
					}
				}

				if litMatcher != nil {
					matches := litMatcher.Match(string(content))
					for _, m := range matches {
						res := GetResult()
						res.Pattern = m.Name
						res.Content = m.Literal
						res.Entropy = utils.ShannonEntropy(string(content))
						res.Line = rec.Line
						res.File = rec.File
						s.handleMatch(res, opts, activeTools, &hitCounter, resultChan, ctx)
					}
				}

				for _, cp := range compiledPatterns {
					matches := cp.comp.FindAllSubmatch(content, -1)
					for _, matchGroup := range matches {
						if len(matchGroup) == 0 || len(matchGroup[0]) == 0 {
							continue
						}
						res := GetResult()
						res.Pattern = cp.p.Name
						res.Content = string(matchGroup[0])
						res.Matches = make([]string, len(matchGroup))
						for idx, mg := range matchGroup {
							res.Matches[idx] = string(mg)
						}
						res.Entropy = utils.ShannonEntropy(res.Content)
						res.Line = rec.Line
						res.File = rec.File
						s.handleMatch(res, opts, activeTools, &hitCounter, resultChan, ctx)
					}
				}
			}
		}()
	}

	go func() {
		defer close(resultChan)
		startTime := time.Now()
		var innerWg sync.WaitGroup
		innerWg.Add(1)
		go func() {
			defer innerWg.Done()
			wg.Wait()
		}()

		targets := s.resolveTargets(opts.TargetPaths)
		var globalTotalSize int64
		for _, path := range targets {
			if info, err := os.Stat(path); err == nil {
				globalTotalSize += info.Size()
			}
		}

		var globalBytesRead int64
		var lastNotifiedPct int = -1
		var lastCooldownPct int = 0
		var progressMu sync.Mutex

		for i, path := range targets {
			if opts.ResumeFile != "" && i < s.Resume.FileIndex {
				if info, err := os.Stat(path); err == nil {
					progressMu.Lock()
					globalBytesRead += info.Size()
					progressMu.Unlock()
				}
				continue
			}

			var sourceReader io.ReadCloser
			if path == "-" || path == "stdin" {
				sourceReader = io.NopCloser(os.Stdin)
			} else {
				f, err := os.Open(path)
				if err != nil {
					slog.Error("Error opening file", "path", path, "error", err)
					continue
				}
				sourceReader = f
			}

			trackedReader := &progressReader{
				r: sourceReader,
				onRead: func(n int) {
					progressMu.Lock()
					defer progressMu.Unlock()
					globalBytesRead += int64(n)
					
					if globalTotalSize > 0 {
						pct := int((float64(globalBytesRead) / float64(globalTotalSize)) * 100)
						if pct > 100 { pct = 100 }

						if opts.Notify && pct > lastNotifiedPct {
							shouldNotify := false
							if pct < 95 {
								if pct%opts.NotifyInterval == 0 {
									shouldNotify = true
								}
							} else {
								shouldNotify = true
							}

							if shouldNotify && pct != lastNotifiedPct {
								s.sendNotification(fmt.Sprintf("ZetGrep Progress: %d%% | Target: %s", pct, filepath.Base(path)))
								lastNotifiedPct = pct
							}
						}

						if opts.CooldownEvery > 0 && pct > 0 && pct < 100 && pct % opts.CooldownEvery == 0 && pct > lastCooldownPct {
							lastCooldownPct = pct
							duration, _ := time.ParseDuration(opts.CooldownTime)
							msg := fmt.Sprintf("❄️ Cooldown Paused (%d%%) for %s", pct, opts.CooldownTime)
							slog.Info(msg)
							if opts.Notify { s.sendNotification(msg) }
							time.Sleep(duration)
						}
					}
				},
			}

			var finalReader io.ReadCloser = io.NopCloser(trackedReader)
			if s.Config.Input.PreProcess != "" && path != "stdin" && path != "-" {
				cmd := exec.CommandContext(ctx, "sh", "-c", s.Config.Input.PreProcess)
				cmd.Stdin = trackedReader
				if stdout, err := cmd.StdoutPipe(); err == nil {
					if err := cmd.Start(); err == nil {
						finalReader = stdout
						go cmd.Wait()
					}
				}
			}

			recs, err := s.Parser.GetRecords(ctx, finalReader, path)
			if err == nil {
				for rec := range recs {
					select {
					case <-ctx.Done():
						sourceReader.Close()
						finalReader.Close()
						return
					case recordChan <- rec:
					}
				}
			}
			sourceReader.Close()
			finalReader.Close()
			
			if !opts.Silent && globalTotalSize > 0 {
				fmt.Fprintf(os.Stderr, "\r%s Scanned %s: 100%%          \n", au.Green("[+]"), filepath.Base(path))
			}
		}
		close(recordChan)
		innerWg.Wait()
		
		if opts.Notify {
			duration := time.Since(startTime).Round(time.Second)
			hitCounter.Lock()
			finalHits := hitCounter.count
			hitCounter.Unlock()
			s.sendNotification(fmt.Sprintf("ZetGrep Completed! ✅\nDuration: %s\nHits: %d\nTargets: %d", duration, finalHits, len(targets)))
		}
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
