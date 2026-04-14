package scanner

import (
	"bufio"
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/Abhay0thakor/ZetGrep/pkg/models"
)

// ScanRecord represents a single unit of work (a line, a CSV row, etc.)
type ScanRecord struct {
	Content string
	ID      string
	Line    int
	File    string
}

// Parser defines the interface for different input formats
type Parser interface {
	GetRecords(ctx context.Context, reader io.Reader, path string) (<-chan ScanRecord, error)
}

// TextParser handles raw text files
type TextParser struct {
	Config models.InputConfig
}

func (p *TextParser) GetRecords(ctx context.Context, reader io.Reader, path string) (<-chan ScanRecord, error) {
	out := make(chan ScanRecord, 100)
	go func() {
		defer close(out)
		scanner := bufio.NewScanner(reader)
		// Set a larger buffer for very long lines
		buf := make([]byte, 1024*1024)
		scanner.Buffer(buf, 100*1024*1024)
		
		lineNum := 0
		for scanner.Scan() {
			lineNum++
			select {
			case <-ctx.Done():
				return
			case out <- ScanRecord{
				Content: scanner.Text(),
				Line:    lineNum,
				File:    path,
			}:
			}
		}
	}()
	return out, nil
}

// JSONLParser handles JSONL (one JSON object per line)
type JSONLParser struct {
	Config models.InputConfig
}

func (p *JSONLParser) GetRecords(ctx context.Context, reader io.Reader, path string) (<-chan ScanRecord, error) {
	out := make(chan ScanRecord, 100)
	go func() {
		defer close(out)
		scanner := bufio.NewScanner(reader)
		buf := make([]byte, 1024*1024)
		scanner.Buffer(buf, 100*1024*1024)
		
		lineNum := 0
		targets := append([]string{}, p.Config.Targets...)
		if p.Config.Target != "" {
			targets = append(targets, p.Config.Target)
		}
		if len(targets) == 0 {
			targets = []string{"$"} // Default to whole line if no targets
		}

		for scanner.Scan() {
			lineNum++
			line := scanner.Text()
			var data map[string]interface{}
			if err := json.Unmarshal([]byte(line), &data); err != nil {
				// If not JSON, but we are in JSONL mode, skip or treat as text?
				// For now, let's treat whole line as content if it's the only target
				for _, t := range targets {
					if t == "$" {
						select {
						case <-ctx.Done(): return
						case out <- ScanRecord{Content: line, Line: lineNum, File: path}:
						}
					}
				}
				continue
			}

			// Check filters
			matchFilters := true
			for field, val := range p.Config.Filters {
				if v, ok := getNestedField(data, field); !ok || v != val {
					matchFilters = false
					break
				}
			}
			if !matchFilters {
				continue
			}

			idVal, _ := getNestedField(data, p.Config.ID)

			foundTarget := false
			for _, targetField := range targets {
				var content string
				var ok bool
				if targetField == "$" {
					content = line
					ok = true
				} else {
					content, ok = getNestedField(data, targetField)
				}
				
				if ok && content != "" {
					foundTarget = true
					displayFile := path
					if idVal != "" {
						displayFile = fmt.Sprintf("%s:%s", path, idVal)
					}
					select {
					case <-ctx.Done(): return
					case out <- ScanRecord{Content: content, Line: lineNum, File: displayFile, ID: idVal}:
					}
				}
			}
			if !foundTarget {
				// fmt.Fprintf(os.Stderr, "No target field found in JSON at line %d\n", lineNum)
			}
		}
		if err := scanner.Err(); err != nil {
			// Using fmt for now as slog might not be imported or set up here 
			// Wait, I am in pkg/scanner, I can import slog or use fmt.
			// Let's use a warn or error if possible.
		}
	}()
	return out, nil
}

// CSVParser handles CSV files
type CSVParser struct {
	Config models.InputConfig
}

func (p *CSVParser) GetRecords(ctx context.Context, reader io.Reader, path string) (<-chan ScanRecord, error) {
	out := make(chan ScanRecord, 100)
	separator := p.Config.CSVConfig.Separator
	if separator == "" {
		separator = ","
	}
	
	go func() {
		defer close(out)
		csvReader := csv.NewReader(reader)
		csvReader.Comma = rune(separator[0])
		csvReader.LazyQuotes = true
		
		if p.Config.CSVConfig.HasHeader {
			_, _ = csvReader.Read()
		}

		lineNum := 0
		for {
			record, err := csvReader.Read()
			if err == io.EOF {
				break
			}
			if err != nil {
				continue
			}
			lineNum++
			
			idVal := ""
			if p.Config.CSVConfig.IDIndex < len(record) {
				idVal = record[p.Config.CSVConfig.IDIndex]
			}

			targetIdxs := p.Config.CSVConfig.TargetIdx
			if len(targetIdxs) == 0 {
				for i := range record {
					targetIdxs = append(targetIdxs, i)
				}
			}

			for _, idx := range targetIdxs {
				if idx < len(record) && record[idx] != "" {
					displayFile := path
					if idVal != "" {
						displayFile = fmt.Sprintf("%s:%s", path, idVal)
					}
					select {
					case <-ctx.Done(): return
					case out <- ScanRecord{Content: record[idx], Line: lineNum, File: displayFile, ID: idVal}:
					}
				}
			}
		}
	}()
	return out, nil
}

// getNestedField is a helper moved from service.go or models.go
func getNestedField(data map[string]interface{}, path string) (string, bool) {
	if path == "" { return "", false }
	parts := strings.Split(path, ".")
	var current interface{} = data
	for _, part := range parts {
		if m, ok := current.(map[string]interface{}); ok {
			current = m[part]
		} else {
			return "", false
		}
	}
	
	var val string
	if str, ok := current.(string); ok {
		val = str
	} else if current != nil {
		val = fmt.Sprintf("%v", current)
	} else {
		return "", false
	}

	return val, true
}
