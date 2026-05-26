package scanner

import (
	"context"
	"strings"
	"testing"
	"github.com/Abhay0thakor/ZetGrep/pkg/models"
)

func BenchmarkTextParser(b *testing.B) {
	data := strings.Repeat("This is a sample line with an IP 1.1.1.1\n", 1000)
	parser := &TextParser{Config: models.InputConfig{}}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r := strings.NewReader(data)
		ch, _ := parser.GetRecords(context.Background(), r, "test")
		for range ch {}
	}
}

func BenchmarkJSONLParser(b *testing.B) {
	line := `{"id": "test", "content": "Sample 1.1.1.1"}` + "\n"
	data := strings.Repeat(line, 1000)
	parser := &JSONLParser{Config: models.InputConfig{Target: "content"}}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r := strings.NewReader(data)
		ch, _ := parser.GetRecords(context.Background(), r, "test")
		for range ch {}
	}
}

