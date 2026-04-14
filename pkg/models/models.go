package models

import (
	"context"
	"fmt"
	"os/exec"
	"regexp"
	"strings"
	"time"

	"github.com/google/shlex"
)

type ToolOutput struct {
	ToolID string `json:"tool_id"`
	Label  string `json:"label"`
	Value  string `json:"value"`
}

type Result struct {
	ID       int          `json:"id"`
	Pattern  string       `json:"pattern"`
	File     string       `json:"file,omitempty"`
	Ext      string       `json:"ext,omitempty"`
	Line     int          `json:"line,omitempty"`
	Content  string       `json:"content"`
	Matches  []string     `json:"matches,omitempty"`
	Entropy  float64      `json:"entropy,omitempty"`
	ToolData []ToolOutput `json:"tool_data,omitempty"`
}

func (r *Result) Reset() {
	r.ID = 0
	r.Pattern = ""
	r.File = ""
	r.Ext = ""
	r.Line = 0
	r.Content = ""
	r.Matches = r.Matches[:0]
	r.Entropy = 0
	r.ToolData = r.ToolData[:0]
}

type Tool struct {
	ID          string   `yaml:"id" json:"id"`
	Name        string   `yaml:"name" json:"name"`
	Description string   `yaml:"description" json:"description"`
	Extract     string   `yaml:"extract" json:"extract"`
	Command     string   `yaml:"command" json:"command"`
	Field       string   `yaml:"field" json:"field"`
	ApplyTo     []string `yaml:"apply_to" json:"apply_to"`
}

type Pattern struct {
	Name     string   `json:"-"`
	Flags    string   `json:"flags,omitempty"`
	Pattern  string   `json:"pattern,omitempty"`
	Patterns []string `json:"patterns,omitempty"`
	Engine   string   `json:"engine,omitempty"`
	Tags     []string `json:"tags,omitempty" yaml:"tags,omitempty"`
}

type ResumeConfig struct {
	FileIndex int    `json:"file_index"`
	LineIndex int    `json:"line_index"`
	Target    string `json:"target"`
}

type Config struct {
	PatternsDir string      `json:"patterns_dir" yaml:"patterns_dir"`
	ToolsDir    string      `json:"tools_dir" yaml:"tools_dir"`
	Input       InputConfig `json:"input" yaml:"input"`
	Globals     Globals     `json:"globals" yaml:"globals"`
}

type Globals struct {
	IgnoreExtensions []string `json:"ignore_extensions" yaml:"ignore_extensions"`
	IgnoreFiles      []string `json:"ignore_files" yaml:"ignore_files"`
}

type InputConfig struct {
	Format      string            `json:"format" yaml:"format"`             // "jsonl", "json", "csv"
	PreProcess  string            `json:"pre_process" yaml:"pre_process"`   // Command to run on input (e.g. js-beautify)
	Target      string            `json:"target" yaml:"target"`             // Legacy single target
	Targets     []string          `json:"targets" yaml:"targets"`           // Multiple fields to scan
	ID          string            `json:"id" yaml:"id"`                     // Source identifier field
	Decode      bool              `json:"decode" yaml:"decode"`             // Unescape target content
	Filters     map[string]string `json:"filters" yaml:"filters"`           // Conditional matching (e.g. status: "200")
	PostProcess map[string]string `json:"post_process" yaml:"post_process"` // Field-specific commands
	CSVConfig   CSVConfig         `json:"csv_config" yaml:"csv_config"`
}

type CSVConfig struct {
	Separator string `json:"separator" yaml:"separator"` // default ","
	HasHeader bool   `json:"has_header" yaml:"has_header"`
	IDIndex   int    `json:"id_index" yaml:"id_index"`
	TargetIdx []int  `json:"target_indices" yaml:"target_indices"`
}

func (t *Tool) Execute(res Result) (string, error) {
	input := res.Content
	if len(res.Matches) > 0 {
		input = res.Matches[0]
	}
	extracted := input
	if t.Extract != "" {
		re, err := regexp.Compile(t.Extract)
		if err != nil {
			return "", fmt.Errorf("invalid extract regex: %w", err)
		}
		if m := re.FindString(input); m != "" {
			extracted = m
		} else {
			return "", nil
		}
	}

	cmdStr := t.Command
	// Core Variables
	cmdStr = strings.ReplaceAll(cmdStr, "{{extracted}}", extracted)
	cmdStr = strings.ReplaceAll(cmdStr, "{{match}}", input)
	cmdStr = strings.ReplaceAll(cmdStr, "{{content}}", res.Content)
	cmdStr = strings.ReplaceAll(cmdStr, "{{file}}", res.File)
	cmdStr = strings.ReplaceAll(cmdStr, "{{line}}", fmt.Sprintf("%d", res.Line))
	cmdStr = strings.ReplaceAll(cmdStr, "{{pattern}}", res.Pattern)
	cmdStr = strings.ReplaceAll(cmdStr, "{{ext}}", res.Ext)

	// Indexed Matches (e.g., {{match[0]}}, {{match[1]}})
	for i, m := range res.Matches {
		placeholder := fmt.Sprintf("{{match[%d]}}", i)
		cmdStr = strings.ReplaceAll(cmdStr, placeholder, m)
	}

	// Tool Data Chaining (e.g., {{tool:b64_decode}})
	for _, td := range res.ToolData {
		placeholder := fmt.Sprintf("{{tool:%s}}", td.ToolID)
		cmdStr = strings.ReplaceAll(cmdStr, placeholder, td.Value)
		placeholderLabel := fmt.Sprintf("{{tool:%s}}", td.Label)
		cmdStr = strings.ReplaceAll(cmdStr, placeholderLabel, td.Value)
	}

	// Create context with timeout for tool execution
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	var cmd *exec.Cmd
	// If the command contains shell-specific characters, use bash -c as fallback
	if strings.ContainsAny(cmdStr, "|&><$();*") {
		cmd = exec.CommandContext(ctx, "bash", "-c", cmdStr)
	} else {
		parts, err := shlex.Split(cmdStr)
		if err != nil || len(parts) == 0 {
			// Fallback to bash -c if splitting fails
			cmd = exec.CommandContext(ctx, "bash", "-c", cmdStr)
		} else {
			cmd = exec.CommandContext(ctx, parts[0], parts[1:]...)
		}
	}

	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("tool execution error: %w (output: %s)", err, string(out))
	}
	return strings.TrimSpace(string(out)), nil
}
