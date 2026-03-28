package scanner

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"

	"github.com/Abhay0thakor/ZetGrep/pkg/models"
	"github.com/Abhay0thakor/ZetGrep/pkg/utils"
	fileutil "github.com/projectdiscovery/utils/file"
)

var resultPool = sync.Pool{
	New: func() interface{} {
		return &models.Result{}
	},
}

func GetResult() *models.Result {
	r := resultPool.Get().(*models.Result)
	r.Reset()
	return r
}

func PutResult(r *models.Result) {
	if r == nil {
		return
	}
	resultPool.Put(r)
}

// Engine defines the interface for a pattern matching engine.
type Engine interface {
	Execute(ctx context.Context, p models.Pattern, targets ...string) (<-chan *models.Result, error)
}

// BaseEngine provides common functionality for engines.
type BaseEngine struct {
	BinaryPath string
}

func (e *BaseEngine) executeCommand(ctx context.Context, p models.Pattern, args []string) (<-chan *models.Result, error) {
	cmd := exec.CommandContext(ctx, e.BinaryPath, args...)
	
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("error creating stdout pipe: %w", err)
	}
	
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("error starting command: %w", err)
	}
	
	resChan := make(chan *models.Result)
	go func() {
		defer close(resChan)
		defer cmd.Wait()
		
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			select {
			case <-ctx.Done():
				return
			default:
				line := scanner.Text()
				if line == "" {
					continue
				}
				res := GetResult()
				if err := parseLineInto(p.Name, line, res); err != nil {
					PutResult(res)
					continue
				}
				resChan <- res
			}
		}
	}()
	
	return resChan, nil
}

// RipgrepEngine uses 'rg' for pattern matching.
type RipgrepEngine struct {
	BaseEngine
}

func NewRipgrepEngine() (*RipgrepEngine, error) {
	path, err := exec.LookPath("rg")
	if err != nil {
		return nil, err
	}
	return &RipgrepEngine{BaseEngine{BinaryPath: path}}, nil
}

func (e *RipgrepEngine) Execute(ctx context.Context, p models.Pattern, targets ...string) (<-chan *models.Result, error) {
	flags := []string{"--with-filename", "--line-number", "--no-heading"}
	if strings.Contains(p.Flags, "i") {
		flags = append(flags, "--ignore-case")
	}
	
	args := append(flags, p.Pattern)
	args = append(args, targets...)
	return e.executeCommand(ctx, p, args)
}

// GrepEngine uses 'grep' for pattern matching.
type GrepEngine struct {
	BaseEngine
}

func NewGrepEngine() (*GrepEngine, error) {
	path, err := exec.LookPath("grep")
	if err != nil {
		return nil, err
	}
	return &GrepEngine{BaseEngine{BinaryPath: path}}, nil
}

func (e *GrepEngine) Execute(ctx context.Context, p models.Pattern, targets ...string) (<-chan *models.Result, error) {
	flags := []string{"-Hn"}
	if strings.Contains(p.Flags, "i") {
		flags = append(flags, "-i")
	}
	
	args := append(flags, p.Pattern)
	args = append(args, targets...)
	return e.executeCommand(ctx, p, args)
}

func parseLineInto(patternName, line string, res *models.Result) error {
	res.Pattern = patternName
	res.Content = line
	
	// Better parsing that handles colons in filenames (at least for the first two colons)
	// Typical output: filename:line:content
	firstColon := strings.Index(line, ":")
	if firstColon == -1 {
		return nil
	}
	res.File = line[:firstColon]
	res.Ext = filepath.Ext(res.File)
	
	remaining := line[firstColon+1:]
	secondColon := strings.Index(remaining, ":")
	if secondColon == -1 {
		res.Content = remaining
		return nil
	}
	
	lineNumStr := remaining[:secondColon]
	fmt.Sscanf(lineNumStr, "%d", &res.Line)
	res.Content = remaining[secondColon+1:]
	
	return nil
}

// Helper functions for loading patterns and tools

func GetPatternDir() (string, error) {
	if d := os.Getenv("ZetGrep_PATTERNS_DIR"); d != "" {
		return d, nil
	}
	for _, d := range []string{"patterns", "examples"} {
		if fileutil.FolderExists(d) {
			return d, nil
		}
	}
	exe, _ := os.Executable()
	binDir := filepath.Dir(exe)
	for _, d := range []string{"patterns", "examples"} {
		path := filepath.Join(binDir, d)
		if fileutil.FolderExists(path) {
			return path, nil
		}
	}
	home, _ := os.UserHomeDir()
	for _, d := range []string{".zetgrep", ".config/gf/patterns"} {
		path := filepath.Join(home, d)
		if fileutil.FolderExists(path) {
			return path, nil
		}
	}
	return "examples", fmt.Errorf("no patterns directory found")
}

func GetPatterns(dir string) ([]string, error) {
	if dir == "" {
		var err error
		dir, err = GetPatternDir()
		if err != nil {
			return nil, err
		}
	}
	
	dir = utils.ExpandPath(dir)
	fsList, _ := filepath.Glob(filepath.Join(dir, "*.json"))
	var res []string
	for _, f := range fsList {
		res = append(res, strings.TrimSuffix(filepath.Base(f), ".json"))
	}
	return res, nil
}

func LoadPattern(f string) (models.Pattern, error) {
	b, err := os.ReadFile(f)
	if err != nil {
		return models.Pattern{}, err
	}
	var p models.Pattern
	if err := json.Unmarshal(b, &p); err != nil {
		return models.Pattern{}, err
	}
	if p.Pattern == "" && len(p.Patterns) > 0 {
		p.Pattern = "(" + strings.Join(p.Patterns, "|") + ")"
	}
	return p, nil
}
