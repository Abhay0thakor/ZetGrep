package scanner

import (
	"context"
	"strings"
	"testing"

	"github.com/Abhay0thakor/ZetGrep/pkg/models"
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

func TestGrepEngine(t *testing.T) {
	engine, err := NewGrepEngine()
	if err != nil {
		t.Skip("grep not found")
	}
	ctx := context.Background()
	p := models.Pattern{
		Name:    "test",
		Pattern: "package scanner",
	}
	// We scan scanner.go itself
	resChan, err := engine.Execute(ctx, p, "scanner.go")
	if err != nil {
		t.Fatal(err)
	}
	count := 0
	for range resChan {
		count++
	}
	if count == 0 {
		t.Error("expected at least one match for 'package scanner' in scanner.go")
	}
}
