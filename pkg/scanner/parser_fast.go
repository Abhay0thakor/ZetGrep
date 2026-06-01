package scanner

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"runtime"
	"strings"
	"sync"

	"github.com/Abhay0thakor/ZetGrep/pkg/models"
	"github.com/buger/jsonparser"
)

// FastJSONParser uses buger/jsonparser for lightning-fast field extraction without full unmarshal
type FastJSONParser struct {
	Config models.InputConfig
}

func (p *FastJSONParser) GetRecords(ctx context.Context, reader io.Reader, path string) (<-chan ScanRecord, error) {
	out := make(chan ScanRecord, 500)
	
	targets := append([]string{}, p.Config.Targets...)
	if p.Config.Target != "" {
		targets = append(targets, p.Config.Target)
	}
	if len(targets) == 0 {
		targets = []string{"$"}
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

	go func() {
		defer close(out)
		scanner := bufio.NewScanner(reader)
		buf := make([]byte, 1024*1024)
		scanner.Buffer(buf, 100*1024*1024)
		
		lineNum := 0
		for scanner.Scan() {
			lineNum++
			line := scanner.Bytes()
			
			// Extract ID
			idVal := ""
			if splitID != nil {
				if v, _, _, err := jsonparser.Get(line, splitID...); err == nil {
					idVal = string(v)
				}
			}

			// Check Filters
			matchFilters := true
			for field, val := range p.Config.Filters {
				parts := strings.Split(field, ".")
				if v, _, _, err := jsonparser.Get(line, parts...); err != nil || string(v) != val {
					matchFilters = false
					break
				}
			}
			if !matchFilters {
				continue
			}

			for i, targetField := range targets {
				var content []byte
				if targetField == "$" {
					content = make([]byte, len(line))
					copy(content, line)
				} else {
					if v, _, _, err := jsonparser.Get(line, splitTargets[i]...); err == nil {
						content = make([]byte, len(v))
						copy(content, v)
					}
				}
				
				if len(content) > 0 {
					displayFile := path
					if idVal != "" {
						displayFile = fmt.Sprintf("%s:%s", path, idVal)
					}
					select {
					case <-ctx.Done(): return
					case out <- ScanRecord{Content: content, Line: lineNum, File: displayFile, ID: idVal, RawLength: len(line) + 1}:
					}
				}
			}
		}
	}()
	return out, nil
}

func (p *FastJSONParser) GetRecordsParallel(ctx context.Context, data []byte, path string, concurrency int) (<-chan ScanRecord, error) {
	out := make(chan ScanRecord, 500)
	if concurrency <= 0 {
		concurrency = runtime.NumCPU()
	}

	targets := append([]string{}, p.Config.Targets...)
	if p.Config.Target != "" {
		targets = append(targets, p.Config.Target)
	}
	if len(targets) == 0 {
		targets = []string{"$"}
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
				
				// Extract ID
				idVal := ""
				if splitID != nil {
					if v, _, _, err := jsonparser.Get(line, splitID...); err == nil {
						idVal = string(v)
					}
				}

				// Check Filters
				matchFilters := true
				for field, val := range p.Config.Filters {
					parts := strings.Split(field, ".")
					if v, _, _, err := jsonparser.Get(line, parts...); err != nil || string(v) != val {
						matchFilters = false
						break
					}
				}
				if !matchFilters {
					curr = lineEnd + 1
					continue
				}

				for i, targetField := range targets {
					var content []byte
					if targetField == "$" {
						content = make([]byte, len(line))
						copy(content, line)
					} else {
						if v, _, _, err := jsonparser.Get(line, splitTargets[i]...); err == nil {
							content = make([]byte, len(v))
							copy(content, v)
						}
					}
					
					if len(content) > 0 {
						displayFile := path
						if idVal != "" {
							displayFile = fmt.Sprintf("%s:%s", path, idVal)
						}
						select {
						case <-ctx.Done(): return
						case out <- ScanRecord{Content: content, Line: int(curr), File: displayFile, ID: idVal, RawLength: len(line) + 1}:
						}
					}
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
