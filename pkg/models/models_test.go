package models

import (
	"testing"
)

func TestToolExecutePlaceholders(t *testing.T) {
	tool := Tool{
		ID:      "test-tool",
		Command: "echo '{{match}} {{extracted}} {{file}}'",
		Extract: `(\d+\.\d+\.\d+\.\d+)`,
	}

	res := Result{
		Content: "IP is 1.2.3.4 here",
		Matches: []string{"1.2.3.4"},
		File:    "test.txt",
	}

	got, err := tool.Execute(res)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := "1.2.3.4 1.2.3.4 test.txt"
	if got != expected {
		t.Errorf("expected %q, got %q", expected, got)
	}
}

func TestToolExecuteChaining(t *testing.T) {
	tool := Tool{
		ID:      "chain-tool",
		Command: "echo '{{tool:prev}}'",
	}

	res := Result{
		Content: "dummy",
		ToolData: []ToolOutput{
			{ToolID: "prev", Value: "chained-value"},
		},
	}

	got, err := tool.Execute(res)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := "chained-value"
	if got != expected {
		t.Errorf("expected %q, got %q", expected, got)
	}
}

func TestToolExecuteShell(t *testing.T) {
	tool := Tool{
		ID:      "shell-tool",
		Command: "echo 'hello' | tr 'a-z' 'A-Z'",
	}

	res := Result{Content: "dummy"}

	got, err := tool.Execute(res)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := "HELLO"
	if got != expected {
		t.Errorf("expected %q, got %q", expected, got)
	}
}
