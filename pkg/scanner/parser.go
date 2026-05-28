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
	Content []byte
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
			// Use Bytes() instead of Text() to avoid allocation
			line := scanner.Bytes()
			content := make([]byte, len(line))
			copy(content, line)

			select {
			case <-ctx.Done():
				return
			case out <- ScanRecord{
				Content: content,
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

		// Pre-split target paths
		splitTargets := make([][]string, len(targets))
		for i, t := range targets {
			if t != "$" {
				splitTargets[i] = strings.Split(t, ".")
			}
		}
		
		splitID := strings.Split(p.Config.ID, ".")
		if p.Config.ID == "" {
			splitID = nil
		}

		// Pre-split filter paths
		filterParts := make(map[string][]string)
		for field := range p.Config.Filters {
			filterParts[field] = strings.Split(field, ".")
		}

		for scanner.Scan() {
			lineNum++
			line := scanner.Text()
			var data map[string]interface{}
			if err := json.Unmarshal([]byte(line), &data); err != nil {
				for _, t := range targets {
					if t == "$" {
						select {
						case <-ctx.Done(): return
						case out <- ScanRecord{Content: []byte(line), Line: lineNum, File: path}:
						}
					}
				}
				continue
			}

			// Check filters
			matchFilters := true
			for field, val := range p.Config.Filters {
				if v, ok := getNestedFieldSplit(data, filterParts[field]); !ok || v != val {
					matchFilters = false
					break
				}
			}
			if !matchFilters {
				continue
			}

			idVal, _ := getNestedFieldSplit(data, splitID)

			for i, targetField := range targets {
				var content string
				var ok bool
				if targetField == "$" {
					content = line
					ok = true
				} else {
					content, ok = getNestedFieldSplit(data, splitTargets[i])
				}
				
				if ok && content != "" {
					displayFile := path
					if idVal != "" {
						displayFile = fmt.Sprintf("%s:%s", path, idVal)
					}
					select {
					case <-ctx.Done(): return
					case out <- ScanRecord{Content: []byte(content), Line: lineNum, File: displayFile, ID: idVal}:
					}
				}
			}
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
					content := []byte(record[idx])
					select {
					case <-ctx.Done(): return
					case out <- ScanRecord{Content: content, Line: lineNum, File: displayFile, ID: idVal}:
					}
				}
			}
		}
	}()
	return out, nil
}

// getNestedFieldSplit uses pre-split parts for speed
func getNestedFieldSplit(data map[string]interface{}, parts []string) (string, bool) {
	if len(parts) == 0 { return "", false }
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

func getNestedField(data map[string]interface{}, path string) (string, bool) {
	if path == "" { return "", false }
	return getNestedFieldSplit(data, strings.Split(path, "."))
}

