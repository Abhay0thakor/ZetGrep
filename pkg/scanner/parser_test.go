package scanner

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"github.com/Abhay0thakor/ZetGrep/pkg/models"
)

func TestTextParser(t *testing.T) {
	data := "line1\nline2\nline3"
	r := strings.NewReader(data)
	p := &TextParser{Config: models.InputConfig{}}
	
	ch, err := p.GetRecords(context.Background(), r, "test")
	if err != nil {
		t.Fatalf("GetRecords failed: %v", err)
	}

	count := 0
	for rec := range ch {
		count++
		expected := fmt.Sprintf("line%d", count)
		if string(rec.Content) != expected {
			t.Errorf("Expected %q, got %q", expected, string(rec.Content))
		}
	}

	if count != 3 {
		t.Errorf("Expected 3 lines, got %d", count)
	}
}
