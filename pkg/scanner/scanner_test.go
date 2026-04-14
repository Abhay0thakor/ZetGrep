package scanner

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Abhay0thakor/ZetGrep/pkg/models"
	"github.com/Abhay0thakor/ZetGrep/pkg/utils"
)

func TestShannonEntropy(t *testing.T) {
	// I'll test it via the utils package if I want to be thorough,
	// but let's test the scanner logic here.
}

func TestParseLine(t *testing.T) {
	line := "main.go:10:func main() {"
	res := GetResult()
	err := parseLineInto("test", line, res)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if res.File != "main.go" {
		t.Errorf("expected main.go, got %s", res.File)
	}
	if res.Line != 10 {
		t.Errorf("expected 10, got %d", res.Line)
	}
	if !strings.Contains(res.Content, "func main()") {
		t.Errorf("expected content to contain func main(), got %s", res.Content)
	}
}

type MockEngine struct {
	Results []*models.Result
}

func (e *MockEngine) Execute(ctx context.Context, p models.Pattern, targets ...string) (<-chan *models.Result, error) {
	resChan := make(chan *models.Result)
	go func() {
		defer close(resChan)
		for _, res := range e.Results {
			select {
			case <-ctx.Done():
				return
			case resChan <- res:
			}
		}
	}()
	return resChan, nil
}

func TestScannerService_MalformedJSONL(t *testing.T) {
	// Enable debug logs for testing
	utils.InitLogger(true, false)
	
	tmpDir, err := os.MkdirTemp("", "zetgrep-malformed-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	pattern := models.Pattern{Name: "test", Pattern: "good"}
	patternData, _ := json.Marshal(pattern)
	os.WriteFile(filepath.Join(tmpDir, "test.json"), patternData, 0644)

	cfg := models.Config{
		PatternsDir: tmpDir,
		Input:       models.InputConfig{Format: "jsonl", Targets: []string{"data"}},
	}
	svc, _ := NewScannerService(cfg)

	jsonlFile := filepath.Join(tmpDir, "malformed.jsonl")
	os.WriteFile(jsonlFile, []byte(`{"data": "good"}
not json at all
{"data": "still good"}`), 0644)

	opts := ScannerOptions{TargetPaths: []string{jsonlFile}, Patterns: []string{"test"}}
	resChan, err := svc.RunScan(context.Background(), opts)
	if err != nil {
		t.Fatal(err)
	}

	count := 0
	for range resChan {
		count++
	}
	// It should skip "not json at all" but process the other two.
	// Both "good" and "still good" contain "good".
	if count != 2 {
		t.Errorf("expected 2 results, got %d", count)
	}
}
