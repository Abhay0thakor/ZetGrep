package cmd

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/Abhay0thakor/ZetGrep/pkg/models"
	"github.com/Abhay0thakor/ZetGrep/pkg/report"
	"github.com/Abhay0thakor/ZetGrep/pkg/scanner"
	"github.com/klauspost/compress/zstd"
	"github.com/olekukonko/tablewriter"
)

// SmartWriter handles optional zstd compression
type SmartWriter struct {
	file   *os.File
	zstd   *zstd.Encoder
	writer io.Writer
}

func newSmartWriter(path string) (*SmartWriter, error) {
	if path == "" {
		return nil, nil
	}
	f, err := os.Create(path)
	if err != nil {
		return nil, err
	}

	sw := &SmartWriter{file: f, writer: f}
	if strings.HasSuffix(strings.ToLower(path), ".zst") {
		enc, err := zstd.NewWriter(f)
		if err != nil {
			f.Close()
			return nil, err
		}
		sw.zstd = enc
		sw.writer = enc
	}
	return sw, nil
}

func (sw *SmartWriter) Write(p []byte) (n int, err error) {
	return sw.writer.Write(p)
}

func (sw *SmartWriter) Close() error {
	if sw.zstd != nil {
		sw.zstd.Close()
	}
	return sw.file.Close()
}

func outputResults(resultChan <-chan *models.Result, start time.Time) {
	// 1. Setup Stream Routing
	var uiOut io.Writer = os.Stderr
	var dataOut io.Writer = os.Stdout

	if silent {
		uiOut = io.Discard
	}

	symbolArrow := "➜"
	symbolBranch := "└"
	symbolCheck := "✔"
	lineChar := "─"

	if noColor {
		symbolArrow = ">"
		symbolBranch = "|"
		symbolCheck = "[+]"
		lineChar = "-"
	}

	var htmlResults []*models.Result

	// New Dedicated Multi-Output Streams
	jsonSW, _ := newSmartWriter(outputJSON)
	textSW, _ := newSmartWriter(outputText)

	// Legacy -o support
	legacySW, _ := newSmartWriter(outputFile)
	var legacyDataOut io.Writer = dataOut
	if legacySW != nil && !reportMode {
		legacyDataOut = legacySW
	}

	var csvWriter *csv.Writer
	if format == "csv" {
		csvWriter = csv.NewWriter(legacyDataOut)
	}

	var table *tablewriter.Table
	if format == "table" {
		table = tablewriter.NewWriter(legacyDataOut)
		table.Header("Pattern", "File", "Line", "Content")
	}

	var reportFile *os.File
	if reportMode && outputFile != "" {
		reportFile, _ = os.Create(outputFile)
		if reportFile != nil {
			fmt.Fprintln(reportFile, "# ZetGrep Intelligence Report")
			fmt.Fprintf(reportFile, "- **Generated at**: %s\n\n---\n\n", time.Now().Format(time.RFC1123))
		}
	}

	if (jsonMode || format == "json") && legacyDataOut != nil {
		fmt.Fprintf(legacyDataOut, "[\n")
	}
	if jsonSW != nil {
		fmt.Fprintf(jsonSW, "[\n")
	}

	// 2. Processing Loop
	hitCount := 0
	first := true
	for res := range resultChan {
		hitCount++

		if outputHTML != "" {
			htmlResults = append(htmlResults, res)
		}

		// Generate the Professional UI string
		entropyStr := ""
		if res.Entropy > 4.0 {
			entropyStr = au.Bold(au.Red(fmt.Sprintf(" (H:%.1f)", res.Entropy))).String()
		}
		matchPrefix := fmt.Sprintf("[%s] %s:%d%s", au.Bold(au.Yellow(res.Pattern)), au.Cyan(res.File), res.Line, entropyStr)
		proUI := fmt.Sprintf("%s\n  %s %s\n", matchPrefix, au.Gray(15, symbolArrow), au.White(res.Content))
		for _, td := range res.ToolData {
			proUI += fmt.Sprintf("    %s %s: %s\n", au.Gray(15, symbolBranch), au.Magenta(td.Label), au.White(td.Value))
		}

		// A. Show Pro UI
		fmt.Fprint(uiOut, proUI)

		// B. Save Pro UI to dedicated text file (oT)
		if textSW != nil {
			fmt.Fprint(textSW, stripANSI(proUI))
		}

		// C. Save JSON to dedicated file (oJ)
		if jsonSW != nil {
			b, _ := json.Marshal(res)
			if !first {
				fmt.Fprint(jsonSW, ",\n")
			}
			jsonSW.Write(b)
		}

		// D. Persistent Reporting (Legacy)
		if reportFile != nil {
			fmt.Fprintf(reportFile, "### [%s] %s\n- Line: %d\n- Content: `%s`\n", res.Pattern, res.File, res.Line, res.Content)
			for _, td := range res.ToolData {
				fmt.Fprintf(reportFile, "  - **%s**: %s\n", td.Label, td.Value)
			}
			fmt.Fprintln(reportFile, "")
		}

		// E. Legacy Data Stream (Stdout or -o)
		if jsonMode || format == "json" {
			b, _ := json.Marshal(res)
			if !first {
				fmt.Fprintf(legacyDataOut, ",\n")
			}
			fmt.Fprint(legacyDataOut, string(b))
		} else if format == "csv" {
			csvWriter.Write([]string{res.Pattern, res.File, fmt.Sprintf("%d", res.Line), res.Content})
		} else if format == "table" {
			table.Append(res.Pattern, res.File, fmt.Sprintf("%d", res.Line), res.Content)
		} else if outputTemplate != "" {
			fmt.Fprintln(legacyDataOut, formatResult(outputTemplate, res))
		} else {
			if format == "text" || format == "" {
				fmt.Fprintln(legacyDataOut, res.Content)
			}
		}

		first = false
		scanner.PutResult(res)
	}

	// 3. Finalize Streams
	if (jsonMode || format == "json") && legacyDataOut != nil {
		fmt.Fprintf(legacyDataOut, "\n]\n")
	}
	if jsonSW != nil {
		fmt.Fprintf(jsonSW, "\n]\n")
		jsonSW.Close()
	}
	if textSW != nil {
		textSW.Close()
	}
	if format == "csv" {
		csvWriter.Flush()
	}
	if format == "table" {
		table.Render()
	}
	if legacySW != nil {
		legacySW.Close()
	}
	if reportFile != nil {
		reportFile.Close()
	}

	// Generate HTML Report
	if outputHTML != "" {
		reportData := report.ReportData{
			Title:     "ZetGrep Scan Results",
			StartTime: start,
			EndTime:   time.Now(),
			Duration:  time.Since(start).Round(time.Millisecond).String(),
			TotalHits: hitCount,
			Results:   htmlResults,
		}
		_ = report.GenerateHTMLReport(reportData, outputHTML)
	}

	// Summary
	if !silent {
		duration := time.Since(start).Round(time.Millisecond)
		fmt.Fprintf(os.Stderr, "\n%s\n", au.Gray(15, strings.Repeat(lineChar, 80)))
		summary := fmt.Sprintf("Summary: %s hits | %s", au.Bold(fmt.Sprintf("%d", hitCount)), au.Bold(duration))
		
		var saved []string
		if outputJSON != "" { saved = append(saved, outputJSON) }
		if outputText != "" { saved = append(saved, outputText) }
		if outputHTML != "" { saved = append(saved, outputHTML) }
		if outputFile != "" { saved = append(saved, outputFile) }
		
		if len(saved) > 0 {
			summary += fmt.Sprintf(" | Saved to: %s", au.Underline(strings.Join(saved, ", ")))
		}
		fmt.Fprintf(os.Stderr, "%s %s\n\n", au.Green(symbolCheck), summary)
	}
}

func formatResult(tmpl string, res *models.Result) string {
	out := tmpl
	out = strings.ReplaceAll(out, "{{pattern}}", res.Pattern)
	out = strings.ReplaceAll(out, "{{file}}", res.File)
	out = strings.ReplaceAll(out, "{{line}}", fmt.Sprintf("%d", res.Line))
	out = strings.ReplaceAll(out, "{{content}}", res.Content)
	out = strings.ReplaceAll(out, "{{entropy}}", fmt.Sprintf("%.2f", res.Entropy))
	
	mainMatch := res.Content
	if len(res.Matches) > 0 {
		mainMatch = res.Matches[0]
	}
	out = strings.ReplaceAll(out, "{{match}}", mainMatch)

	for i, m := range res.Matches {
		out = strings.ReplaceAll(out, fmt.Sprintf("{{match[%d]}}", i), m)
	}

	for _, td := range res.ToolData {
		out = strings.ReplaceAll(out, fmt.Sprintf("{{tool:%s}}", td.ToolID), td.Value)
		out = strings.ReplaceAll(out, fmt.Sprintf("{{tool:%s}}", td.Label), td.Value)
	}
	return out
}

func stripANSI(str string) string {
	const ansi = "[\u001B\u009B][[()#;?]*(?:[0-9]{1,4}(?:;[0-9]{0,4})*)?[0-9A-ORZcf-nqry=><]"
	var re = regexp.MustCompile(ansi)
	return re.ReplaceAllString(str, "")
}
