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

	pdregexp "github.com/projectdiscovery/utils/regexp"
	"github.com/Abhay0thakor/ZetGrep/pkg/classifier"
	"github.com/Abhay0thakor/ZetGrep/pkg/models"
	"github.com/Abhay0thakor/ZetGrep/pkg/utils"
)

type ScannerOptions struct {
	TargetPaths []string
	Patterns    []string
	ToolIDs     []string
	SmartMode   bool
	EntropyMode bool
}

type ScannerService struct {
	Engine       Engine
	Fallback     Engine
	Config       models.Config
	Classifier   *classifier.Classifier
	Tools        []models.Tool
	patternCache sync.Map
	processSem chan struct{}
}

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
	if str, ok := current.(string); ok {
		return str, true
	}
	if current != nil {
		return fmt.Sprintf("%v", current), true
	}
	return "", false
}

func (s *ScannerService) RunScan(ctx context.Context, opts ScannerOptions) (<-chan *models.Result, error) {
	if s.Config.Input.Format == "jsonl" || s.Config.Input.Format == "json" {
		return s.RunJSONLScan(ctx, opts)
	}
	if s.Config.Input.Format == "csv" {
		return s.RunCSVScan(ctx, opts)
	}

	resultChan := make(chan *models.Result, 2000)
	activeTools := s.getActiveTools(opts.ToolIDs)
	patternQueue := make(chan string)
	var wg sync.WaitGroup

	ignoreFiles := make(map[string]bool)
	for _, f := range s.Config.Globals.IgnoreFiles { ignoreFiles[f] = true }
	ignoreExts := make(map[string]bool)
	for _, e := range s.Config.Globals.IgnoreExtensions {
		if !strings.HasPrefix(e, ".") { e = "." + e }; ignoreExts[e] = true
	}

	numWorkers := runtime.NumCPU()
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for name := range patternQueue {
				p, err := s.getPattern(name)
				if err != nil { continue }
				comp, _ := pdregexp.Compile(p.Pattern)

				var resChan <-chan *models.Result
				if s.Engine != nil {
					resChan, _ = s.Engine.Execute(ctx, p, opts.TargetPaths...)
				} else if s.Fallback != nil {
					resChan, _ = s.Fallback.Execute(ctx, p, opts.TargetPaths...)
				}

				if resChan == nil { continue }

				for res := range resChan {
					if ignoreFiles[filepath.Base(res.File)] || ignoreExts[res.Ext] {
						PutResult(res); continue
					}
					if comp != nil {
						if m := comp.FindStringSubmatch(res.Content); len(m) > 0 {
							res.Matches = m
							res.Entropy = utils.ShannonEntropy(m[0])
						}
					}
					if (opts.SmartMode && s.Classifier.Classify(res.Content) != "high-interest") || (opts.EntropyMode && res.Entropy < 3.5) {
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
		}()
	}

	go func() {
		for _, pName := range opts.Patterns {
			if pName == "" { continue }
			select {
			case <-ctx.Done(): break
			case patternQueue <- pName:
			}
		}
		close(patternQueue)
		wg.Wait()
		close(resultChan)
	}()

	return resultChan, nil
}

func (s *ScannerService) RunJSONLScan(ctx context.Context, opts ScannerOptions) (<-chan *models.Result, error) {
	resultChan := make(chan *models.Result, 5000)

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
	
	// Collect all targets (support both legacy and new array)
	var targets []string
	if s.Config.Input.Target != "" { targets = append(targets, s.Config.Input.Target) }
	targets = append(targets, s.Config.Input.Targets...)
	
	idField := s.Config.Input.ID
	filters := s.Config.Input.Filters

	lineChan := make(chan string, 5000)
	var wg sync.WaitGroup
	numWorkers := runtime.NumCPU() * 2

	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for line := range lineChan {
				var data map[string]interface{}
				if err := json.Unmarshal([]byte(line), &data); err != nil { continue }

				// Apply Filters (e.g. status: "200")
				matchFilters := true
				for field, val := range filters {
					if v, ok := getNestedField(data, field); !ok || v != val {
						matchFilters = false
						break
					}
				}
				if !matchFilters { continue }

				idVal, _ := getNestedField(data, idField)
				if idVal == "" { idVal = "unknown" }

				// Scan multiple targets
				for _, targetField := range targets {
					var content string
					var ok bool
					
					if targetField == "$" {
						content = line
						ok = true
					} else {
						content, ok = getNestedField(data, targetField)
					}
					
					if !ok { continue }

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

							if (opts.SmartMode && s.Classifier.Classify(res.Content) != "high-interest") || (opts.EntropyMode && res.Entropy < 3.5) {
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
		for _, path := range opts.TargetPaths {
			file, err := os.Open(path)
			if err != nil { continue }
			scanner := bufio.NewScanner(file)
			buf := make([]byte, 64*1024)
			scanner.Buffer(buf, 20*1024*1024)

			for scanner.Scan() {
				select {
				case <-ctx.Done(): file.Close(); goto done
				case lineChan <- scanner.Text():
				}
			}
			file.Close()
		}
	done:
		close(lineChan)
		wg.Wait()
		close(resultChan)
	}()

	return resultChan, nil
}

func (s *ScannerService) RunCSVScan(ctx context.Context, opts ScannerOptions) (<-chan *models.Result, error) {
	resultChan := make(chan *models.Result, 5000)

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
	separator := s.Config.Input.CSVConfig.Separator
	if separator == "" { separator = "," }
	
	idIdx := s.Config.Input.CSVConfig.IDIndex
	targetIdxs := s.Config.Input.CSVConfig.TargetIdx
	if len(targetIdxs) == 0 { targetIdxs = []int{0} }

	recordChan := make(chan []string, 5000)
	var wg sync.WaitGroup
	numWorkers := runtime.NumCPU() * 2

	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for record := range recordChan {
				idVal := "unknown"
				if idIdx < len(record) { idVal = record[idIdx] }

				for _, tIdx := range targetIdxs {
					if tIdx >= len(record) { continue }
					content := record[tIdx]

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

							if (opts.SmartMode && s.Classifier.Classify(res.Content) != "high-interest") || (opts.EntropyMode && res.Entropy < 3.5) {
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
		for _, path := range opts.TargetPaths {
			file, err := os.Open(path)
			if err != nil { continue }
			
			reader := csv.NewReader(file)
			reader.Comma = rune(separator[0])
			reader.LazyQuotes = true
			
			if s.Config.Input.CSVConfig.HasHeader {
				reader.Read() // Skip header
			}

			for {
				record, err := reader.Read()
				if err == io.EOF { break }
				if err != nil { continue }
				
				select {
				case <-ctx.Done():
					file.Close()
					goto done
				case recordChan <- record:
				}
			}
			file.Close()
		}
	done:
		close(recordChan)
		wg.Wait()
		close(resultChan)
	}()

	return resultChan, nil
}

func (s *ScannerService) getActiveTools(toolIDs []string) []models.Tool {
	if len(toolIDs) == 0 { return nil }
	var active []models.Tool
	for _, id := range toolIDs {
		id = strings.TrimSpace(id)
		for _, t := range s.Tools {
			if t.ID == id { active = append(active, t) }
		}
	}
	return active
}

func (s *ScannerService) ProcessResults(ctx context.Context, resultsFile string, toolIDs []string) (<-chan *models.Result, error) {
	resultChan := make(chan *models.Result, 2000)
	activeTools := s.getActiveTools(toolIDs)

	b, err := os.ReadFile(resultsFile)
	if err != nil { return nil, err }

	var results []*models.Result
	if err := json.Unmarshal(b, &results); err != nil { return nil, err }

	go func() {
		defer close(resultChan)
		for _, res := range results {
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
	}()

	return resultChan, nil
}
