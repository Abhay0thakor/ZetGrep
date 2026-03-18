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
	Format string `json:"format" yaml:"format"` // "jsonl"
	Target string `json:"target" yaml:"target"` // e.g. "body"
	ID     string `json:"id" yaml:"id"`         // e.g. "url"
	Decode bool   `json:"decode" yaml:"decode"` // Whether to decode the target field (e.g. if it's a JSON string inside JSON)
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
