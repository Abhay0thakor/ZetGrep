package scanner

import (
	"context"
	"strings"
	"testing"
	"github.com/Abhay0thakor/ZetGrep/pkg/models"
)

func TestLargeLine(t *testing.T) {
	// Create a line larger than the default 64KB buffer of bufio.Scanner
	largeLine := strings.Repeat("A", 1024*1024) + " 1.1.1.1"
	r := strings.NewReader(largeLine)
	
	parser := &TextParser{Config: models.InputConfig{}}
	ctx := context.Background()
	ch, _ := parser.GetRecords(ctx, r, "large.txt")
	
	count := 0
	for rec := range ch {
		count++
		if !strings.Contains(rec.Content, "1.1.1.1") {
			t.Errorf("Expected line to contain 1.1.1.1")
		}
	}
	
	if count != 1 {
		t.Errorf("Expected 1 record, got %d", count)
	}
}

func TestUnicodeInput(t *testing.T) {
	unicodeData := "こんにちは 1.1.1.1\n🚀 Rocket 2.2.2.2"
	r := strings.NewReader(unicodeData)
	
	parser := &TextParser{Config: models.InputConfig{}}
	ctx := context.Background()
	ch, _ := parser.GetRecords(ctx, r, "unicode.txt")
	
	var records []ScanRecord
	for rec := range ch {
		records = append(records, rec)
	}
	
	if len(records) != 2 {
		t.Fatalf("Expected 2 records, got %d", len(records))
	}
	
	if !strings.Contains(records[0].Content, "こんにちは") {
		t.Errorf("Unicode content mismatch in line 1")
	}
}
