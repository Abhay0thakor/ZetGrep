package scanner

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Abhay0thakor/ZetGrep/pkg/models"
)

func TestScannerService_RunJSONLScan(t *testing.T) {
	// Setup patterns dir
	tmpDir, err := os.MkdirTemp("", "zetgrep-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	pattern := models.Pattern{
		Name:    "ip",
		Pattern: `\d+\.\d+\.\d+\.\d+`,
	}
	patternData, _ := json.Marshal(pattern)
	os.WriteFile(filepath.Join(tmpDir, "ip.json"), patternData, 0644)

	cfg := models.Config{
		PatternsDir: tmpDir,
		Input: models.InputConfig{
			Format:  "jsonl",
			Targets: []string{"data"},
			ID:      "id",
		},
	}

	svc, err := NewScannerService(cfg)
	if err != nil {
		t.Fatal(err)
	}

	// Create dummy JSONL file
	jsonlFile := filepath.Join(tmpDir, "test.jsonl")
	os.WriteFile(jsonlFile, []byte(`{"id": "test1", "data": "ip is 1.1.1.1"}
{"id": "test2", "data": "no ip here"}
{"id": "test3", "data": "another 2.2.2.2 ip"}`), 0644)

	opts := ScannerOptions{
		TargetPaths: []string{jsonlFile},
		Patterns:    []string{"ip"},
	}

	resChan, err := svc.RunScan(context.Background(), opts)
	if err != nil {
		t.Fatal(err)
	}

	var results []*models.Result
	for res := range resChan {
		results = append(results, res)
	}

	if len(results) != 2 {
		t.Errorf("expected 2 results, got %d", len(results))
	}

	found := make(map[string]string)
	for _, r := range results {
		parts := strings.Split(r.File, ":")
		id := parts[len(parts)-1]
		found[id] = r.Content
	}

	if found["test1"] != "1.1.1.1" {
		t.Errorf("expected test1 to have 1.1.1.1, got %s", found["test1"])
	}
	if found["test3"] != "2.2.2.2" {
		t.Errorf("expected test3 to have 2.2.2.2, got %s", found["test3"])
	}
}

func TestScannerService_RunCSVScan(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "zetgrep-csv-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	pattern := models.Pattern{Name: "ip", Pattern: `\d+\.\d+\.\d+\.\d+`}
	patternData, _ := json.Marshal(pattern)
	os.WriteFile(filepath.Join(tmpDir, "ip.json"), patternData, 0644)

	cfg := models.Config{
		PatternsDir: tmpDir,
		Input: models.InputConfig{
			Format:    "csv",
			CSVConfig: models.CSVConfig{Separator: ",", IDIndex: 0, TargetIdx: []int{1}},
		},
	}
	svc, _ := NewScannerService(cfg)

	csvFile := filepath.Join(tmpDir, "test.csv")
	os.WriteFile(csvFile, []byte("id1,1.2.3.4\nid2,no-ip\nid3,5.6.7.8"), 0644)

	opts := ScannerOptions{TargetPaths: []string{csvFile}, Patterns: []string{"ip"}}
	resChan, err := svc.RunScan(context.Background(), opts)
	if err != nil {
		t.Fatal(err)
	}

	count := 0
	for range resChan {
		count++
	}
	if count != 2 {
		t.Errorf("expected 2 results, got %d", count)
	}
}

func TestScannerService_RunTextScan(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "zetgrep-text-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	pattern := models.Pattern{Name: "ip", Pattern: `\d+\.\d+\.\d+\.\d+`}
	patternData, _ := json.Marshal(pattern)
	os.WriteFile(filepath.Join(tmpDir, "ip.json"), patternData, 0644)

	cfg := models.Config{PatternsDir: tmpDir, Input: models.InputConfig{Format: "text"}}
	svc, _ := NewScannerService(cfg)

	textFile := filepath.Join(tmpDir, "test.txt")
	os.WriteFile(textFile, []byte("some text with 1.2.3.4\nand another 5.6.7.8"), 0644)

	opts := ScannerOptions{TargetPaths: []string{textFile}, Patterns: []string{"ip"}}
	resChan, err := svc.RunScan(context.Background(), opts)
	if err != nil {
		t.Fatal(err)
	}

	count := 0
	for range resChan {
		count++
	}
	if count != 2 {
		t.Errorf("expected 2 results, got %d", count)
	}
}

func TestScannerService_DiagnoseLine(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "zetgrep-diag-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	pattern := models.Pattern{Name: "ip", Pattern: `\d+\.\d+\.\d+\.\d+`}
	patternData, _ := json.Marshal(pattern)
	os.WriteFile(filepath.Join(tmpDir, "ip.json"), patternData, 0644)

	cfg := models.Config{
		PatternsDir: tmpDir,
		Input:       models.InputConfig{Format: "jsonl", Targets: []string{"data"}},
	}
	svc, _ := NewScannerService(cfg)

	line := `{"data": "found 1.1.1.1 here"}`
	logs := svc.DiagnoseLine(line, []string{"ip"})

	hitFound := false
	for _, l := range logs {
		if strings.Contains(l, "[MATCH]") && strings.Contains(l, "hit") {
			hitFound = true
			break
		}
	}

	if !hitFound {
		t.Errorf("DiagnoseLine failed to find match in logs: %v", logs)
	}
}
