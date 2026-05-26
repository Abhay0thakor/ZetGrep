package cmd

import (
	"bytes"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/spf13/pflag"
)

func getProjectRoot() string {
	_, filename, _, _ := runtime.Caller(0)
	// cmd/zetgrep/cmd/cmd_test.go -> cmd/zetgrep/cmd -> cmd/zetgrep -> cmd -> .
	return filepath.Join(filepath.Dir(filename), "..", "..", "..")
}

func executeCommand(args ...string) (output string, err error) {
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs(args)
	
	// Explicitly reset global variables to avoid state leakage
	verbose = false
	silent = false
	noColor = false
	configFiles = nil
	patternsDir = ""
	toolsDir = ""
	inputConfigs = nil
	listFile = ""
	stdin = false
	inputMode = ""
	toolFiles = nil
	allMode = false
	uniqueMode = false
	smartMode = false
	entropyMode = false
	tags = nil
	jsonMode = false
	reportMode = false
	outputFile = ""
	outputTemplate = ""
	toolIDs = ""
	resumeFile = ""
	processFile = ""
	concurrency = 0
	dryRun = false
	format = ""
	targetField = ""
	targetFields = nil
	preProcess = ""
	csvSeparator = ""
	csvNoHeader = false
	csvIDIndex = 0
	csvTargetIndex = nil

	// Reset flags
	rootCmd.Flags().VisitAll(func(f *pflag.Flag) {
		f.Value.Set(f.DefValue)
	})
	rootCmd.PersistentFlags().VisitAll(func(f *pflag.Flag) {
		f.Value.Set(f.DefValue)
	})

	err = rootCmd.Execute()
	return buf.String(), err
}

func TestRootCommand(t *testing.T) {
	root := getProjectRoot()
	samplePath := filepath.Join(root, "test_env/data/sample.txt")

	tests := []struct {
		name    string
		args    []string
		wantErr bool
	}{
		{"version", []string{"version"}, false},
		{"no-args", []string{}, false}, 
		{"invalid-flag", []string{"--invalid"}, true},
		{"non-existent-pd", []string{"ip", samplePath, "--pd", "/non/existent"}, false}, 
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := executeCommand(tt.args...)
			if (err != nil) != tt.wantErr {
				t.Errorf("executeCommand() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestScanEdgeCases(t *testing.T) {
	root := getProjectRoot()
	emptyListPath := filepath.Join(root, "test_env/data/empty.txt")

	tests := []struct {
		name    string
		args    []string
		wantErr bool
	}{
		{"missing-targets", []string{"ip"}, false}, 
		{"non-existent-file", []string{"ip", "non_existent_file.txt"}, false},
		{"empty-list-file", []string{"ip", "-l", emptyListPath}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := executeCommand(tt.args...)
			if (err != nil) != tt.wantErr {
				t.Errorf("executeCommand() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestScanFlags(t *testing.T) {
	root := getProjectRoot()
	samplePath := filepath.Join(root, "test_env/data/sample.txt")

	tests := []struct {
		name    string
		args    []string
		wantErr bool
	}{
		{"dry-run-all", []string{"ip", samplePath, "--dry-run"}, false},
		{"invalid-concurrency", []string{"ip", samplePath, "-c", "-1"}, false}, 
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := executeCommand(tt.args...)
			if (err != nil) != tt.wantErr {
				t.Errorf("executeCommand() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
