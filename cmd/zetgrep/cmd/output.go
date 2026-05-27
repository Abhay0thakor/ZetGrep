package cmd

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/Abhay0thakor/ZetGrep/pkg/models"
	"github.com/Abhay0thakor/ZetGrep/pkg/scanner"
	"github.com/olekukonko/tablewriter"
)

func outputResults(resultChan <-chan *models.Result, start time.Time) {
	// 1. Setup Intelligent Routing
	// uiOut: Where the Pro UI goes (Screen/Human) -> Always Stderr
	// dataOut: Where the structured data goes (Machine) -> Stdout or File
	var uiOut io.Writer = os.Stderr
	var dataOut io.Writer = os.Stdout

	var saveFile *os.File
	if outputFile != "" && !reportMode {
		var err error
		saveFile, err = os.Create(outputFile)
		if err != nil {
			slog.Error("Error creating output file", "path", outputFile, "error", err)
		} else {
			dataOut = saveFile
		}
	}

	var csvWriter *csv.Writer
	if format == "csv" {
		csvWriter = csv.NewWriter(dataOut)
	}

	var table *tablewriter.Table
	if format == "table" {
		table = tablewriter.NewWriter(dataOut)
		table.Header("Pattern", "File", "Line", "Content")
	}

	var reportFile *os.File
	if reportMode {
		name := outputFile
		if name == "" {
			name = fmt.Sprintf("zetgrep_report_%d.md", time.Now().Unix())
		}
		var err error
		reportFile, err = os.Create(name)
		if err == nil {
			fmt.Fprintln(reportFile, "# ZetGrep Intelligence Report")
			fmt.Fprintf(reportFile, "- **Generated at**: %s\n\n---\n\n", time.Now().Format(time.RFC1123))
		}
	}

	// Start JSON array if needed
	if (jsonMode || format == "json") && dataOut != nil {
		fmt.Fprintf(dataOut, "[\n")
	}

	// 2. Processing Loop
	hitCount := 0
	first := true
	for res := range resultChan {
		hitCount++

		// Terminal UI (Professional Layout)
		if !silent {
			entropyStr := ""
			if res.Entropy > 4.0 {
				entropyStr = au.Bold(au.Red(fmt.Sprintf(" (H:%.1f)", res.Entropy))).String()
			}
			matchPrefix := fmt.Sprintf("[%s] %s:%d%s", au.Bold(au.Yellow(res.Pattern)), au.Cyan(res.File), res.Line, entropyStr)
			fmt.Fprintf(uiOut, "%s\n  %s %s\n", matchPrefix, au.Gray(15, "➜"), au.White(res.Content))
			for _, td := range res.ToolData {
				fmt.Fprintf(uiOut, "    %s %s: %s\n", au.Gray(15, "└"), au.Magenta(td.Label), au.White(td.Value))
			}
		}

		// Persistent Reporting
		if reportFile != nil {
			fmt.Fprintf(reportFile, "### [%s] %s\n- Line: %d\n- Content: `%s`\n", res.Pattern, res.File, res.Line, res.Content)
			for _, td := range res.ToolData {
				fmt.Fprintf(reportFile, "  - **%s**: %s\n", td.Label, td.Value)
			}
			fmt.Fprintln(reportFile, "")
		}

		// Structured Data Stream
		if jsonMode || format == "json" {
			b, _ := json.Marshal(res)
			if !first {
				fmt.Fprintf(dataOut, ",\n")
			}
			fmt.Fprint(dataOut, string(b))
		} else if format == "csv" {
			csvWriter.Write([]string{res.Pattern, res.File, fmt.Sprintf("%d", res.Line), res.Content})
		} else if format == "table" {
			table.Append(res.Pattern, res.File, fmt.Sprintf("%d", res.Line), res.Content)
		} else if outputTemplate != "" {
			fmt.Fprintln(dataOut, formatResult(outputTemplate, res))
		} else {
			if format == "text" || format == "" {
				fmt.Fprintln(dataOut, res.Content)
			}
		}
		first = false
		scanner.PutResult(res)
	}

	// 3. Finalize Streams
	if (jsonMode || format == "json") && dataOut != nil {
		fmt.Fprintf(dataOut, "\n]\n")
	}
	if format == "csv" {
		csvWriter.Flush()
	}
	if format == "table" {
		table.Render()
	}
	if saveFile != nil {
		saveFile.Close()
	}
	if reportFile != nil {
		reportFile.Close()
	}

	// Summary
	duration := time.Since(start).Round(time.Millisecond)
	fmt.Fprintf(os.Stderr, "\n%s\n", au.Gray(15, strings.Repeat("─", 80)))
	summary := fmt.Sprintf("Summary: %s hits | %s", au.Bold(fmt.Sprintf("%d", hitCount)), au.Bold(duration))
	if outputFile != "" {
		summary += fmt.Sprintf(" | Saved to: %s", au.Underline(outputFile))
	}
	fmt.Fprintf(os.Stderr, "%s %s\n\n", au.Green("✔"), summary)
}

func formatResult(tmpl string, res *models.Result) string {
	out := tmpl
	out = strings.ReplaceAll(out, "{{pattern}}", res.Pattern)
	out = strings.ReplaceAll(out, "{{file}}", res.File)
	out = strings.ReplaceAll(out, "{{line}}", fmt.Sprintf("%d", res.Line))
	out = strings.ReplaceAll(out, "{{content}}", res.Content)
	out = strings.ReplaceAll(out, "{{entropy}}", fmt.Sprintf("%.2f", res.Entropy))
	for _, td := range res.ToolData {
		out = strings.ReplaceAll(out, fmt.Sprintf("{{tool:%s}}", td.ToolID), td.Value)
		out = strings.ReplaceAll(out, fmt.Sprintf("{{tool:%s}}", td.Label), td.Value)
	}
	return out
}
