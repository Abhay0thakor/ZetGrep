package scanner

import (
	"bufio"
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"runtime"
	"strings"
	"sync"

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
	GetRecordsParallel(ctx context.Context, data []byte, path string, concurrency int) (<-chan ScanRecord, error)
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
		buf := make([]byte, 1024*1024)
		scanner.Buffer(buf, 100*1024*1024)
		
		lineNum := 0
		for scanner.Scan() {
			lineNum++
			line := scanner.Bytes()
			content := GetBuffer()
			if cap(content) < len(line) {
				content = make([]byte, len(line))
			}
			content = content[:len(line)]
			copy(content, line)

			select {
			case <-ctx.Done():
				PutBuffer(content)
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

func (p *TextParser) GetRecordsParallel(ctx context.Context, data []byte, path string, concurrency int) (<-chan ScanRecord, error) {
	out := make(chan ScanRecord, 500)
	if concurrency <= 0 {
		concurrency = runtime.NumCPU()
	}

	var wg sync.WaitGroup
	chunkSize := int64(len(data)) / int64(concurrency)

	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			start := int64(id) * chunkSize
			end := int64(id+1) * chunkSize
			if id == concurrency-1 {
				end = int64(len(data))
			}

			if start > 0 {
				for start < int64(len(data)) && data[start-1] != '\n' {
					start++
				}
			}

			if end < int64(len(data)) {
				for end < int64(len(data)) && data[end-1] != '\n' {
					end++
				}
			}

			if start >= end {
				return
			}

			curr := start
			for curr < end {
				lineEnd := curr
				for lineEnd < end && data[lineEnd] != '\n' {
					lineEnd++
				}
				
				line := data[curr:lineEnd]
				content := GetBuffer()
				if cap(content) < len(line) {
					content = make([]byte, len(line))
				}
				content = content[:len(line)]
				copy(content, line)

				select {
				case <-ctx.Done():
					PutBuffer(content)
					return
				case out <- ScanRecord{
					Content: content,
					Line:    int(curr),
					File:    path,
				}:
				}
				curr = lineEnd + 1
			}
		}(i)
	}

	go func() {
		wg.Wait()
		close(out)
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
		csvReader.ReuseRecord = true
		
		idIdx := p.Config.CSVConfig.IDIndex
		targetIdxs := p.Config.CSVConfig.TargetIdx

		lineNum := 0
		for {
			record, err := csvReader.Read()
			if err != nil {
				break
			}
			lineNum++

			if lineNum == 1 && p.Config.CSVConfig.HasHeader {
				continue
			}

			idVal := ""
			if idIdx < len(record) {
				idVal = record[idIdx]
			}

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

func (p *CSVParser) GetRecordsParallel(ctx context.Context, data []byte, path string, concurrency int) (<-chan ScanRecord, error) {
	return p.GetRecords(ctx, strings.NewReader(string(data)), path)
}

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
