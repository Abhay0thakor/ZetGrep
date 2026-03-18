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
	// Global semaphore to limit concurrent external processes
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

	// Limit to number of CPUs or 20, whichever is smaller, to prevent OS thrashing
	maxProc := runtime.NumCPU() * 2
	if maxProc > 50 {
		maxProc = 50
	}

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

// executeToolWithLimit ensures we don't spawn too many processes at once
func (s *ScannerService) executeToolWithLimit(t models.Tool, res models.Result) (string, error) {
	s.processSem <- struct{}{}
	defer func() { <-s.processSem }()
	return t.Execute(res)
}

func (s *ScannerService) RunScan(ctx context.Context, opts ScannerOptions) (<-chan *models.Result, error) {
	if s.Config.Input.Format == "jsonl" {
		return s.RunJSONLScan(ctx, opts)
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
					
					// Tool execution moved to separate pool if many matches, but here we keep it simple with the semaphore
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
	targetField := s.Config.Input.Target
	idField := s.Config.Input.ID

	lineChan := make(chan string, 5000)
	var wg sync.WaitGroup
	numWorkers := runtime.NumCPU() * 2

	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for line := range lineChan {
				var data map[string]interface{}
				// Use a more efficient decoder for large objects if needed, but json.Unmarshal is standard
				if err := json.Unmarshal([]byte(line), &data); err != nil { continue }

				content, ok := data[targetField].(string)
				if !ok { continue }
				
				idVal, _ := data[idField].(string)
				if idVal == "" { idVal = "unknown" }

				for _, cp := range compiledPatterns {
					matches := cp.comp.FindAllStringSubmatch(content, -1)
					for _, matchGroup := range matches {
						if len(matchGroup) == 0 { continue }
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
		}()
	}

	go func() {
		for _, path := range opts.TargetPaths {
			file, err := os.Open(path)
			if err != nil { continue }
			scanner := bufio.NewScanner(file)
			// Efficient buffer management
			buf := make([]byte, 64*1024)
			scanner.Buffer(buf, 20*1024*1024) // 20MB max line

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
