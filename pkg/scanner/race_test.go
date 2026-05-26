package scanner

import (
	"context"
	"os"
	"sync"
	"testing"
	"github.com/Abhay0thakor/ZetGrep/pkg/models"
)

func TestScanRaceAndDuplicates(t *testing.T) {
	// Setup a temporary file with duplicate content
	content := "1.1.1.1\n1.1.1.1\n1.1.1.1\n"
	tmpFile := "test_race.txt"
	os.WriteFile(tmpFile, []byte(content), 0644)
	defer os.Remove(tmpFile)

	cfg := models.Config{
		PatternsDir: "../../library/patterns",
	}
	svc, _ := NewScannerService(cfg)

	// Run multiple scans concurrently to check for races
	var wg sync.WaitGroup
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			ctx := context.Background()
			resChan, err := svc.RunScan(ctx, ScannerOptions{
				TargetPaths: []string{tmpFile},
				Patterns:    []string{"ip"},
				Unique:      true,
				Concurrency: 2,
			})
			if err != nil {
				return
			}
			for range resChan {}
		}(i)
	}
	wg.Wait()
}
