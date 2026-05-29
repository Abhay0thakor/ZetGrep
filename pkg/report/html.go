package report

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/Abhay0thakor/ZetGrep/pkg/models"
)

type ReportData struct {
	Title      string           `json:"title"`
	StartTime  time.Time        `json:"start_time"`
	EndTime    time.Time        `json:"end_time"`
	Duration   string           `json:"duration"`
	TotalHits  int              `json:"total_hits"`
	Targets    []string         `json:"targets"`
	Patterns   []string         `json:"patterns"`
	Results    []*models.Result `json:"results"`
	Stats      map[string]int   `json:"stats"`
}

func GenerateHTMLReport(data ReportData, outputPath string) error {
	f, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	defer f.Close()

	// Calculate Pattern Stats
	data.Stats = make(map[string]int)
	for _, res := range data.Results {
		data.Stats[res.Pattern]++
	}

	jsonData, _ := json.Marshal(data)

	html := fmt.Sprintf(`
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>ZetGrep Intelligence Report</title>
    <script src="https://cdn.jsdelivr.net/npm/chart.js"></script>
    <style>
        body { font-family: 'Segoe UI', Tahoma, Geneva, Verdana, sans-serif; background: #0f172a; color: #e2e8f0; margin: 0; padding: 20px; }
        .container { max-width: 1200px; margin: 0 auto; }
        header { border-bottom: 2px solid #1e293b; padding-bottom: 20px; margin-bottom: 30px; }
        h1 { color: #38bdf8; margin: 0; }
        .summary-grid { display: grid; grid-template-columns: repeat(auto-fit, minmax(200px, 1fr)); gap: 20px; margin-bottom: 40px; }
        .card { background: #1e293b; padding: 20px; border-radius: 12px; border: 1px solid #334155; }
        .card-label { color: #94a3b8; font-size: 0.875rem; margin-bottom: 5px; }
        .card-value { font-size: 1.5rem; font-weight: bold; color: #f8fafc; }
        .chart-container { background: #1e293b; padding: 20px; border-radius: 12px; margin-bottom: 40px; border: 1px solid #334155; }
        table { width: 100%%; border-collapse: collapse; background: #1e293b; border-radius: 12px; overflow: hidden; }
        th { background: #334155; color: #38bdf8; text-align: left; padding: 12px 15px; }
        td { padding: 12px 15px; border-bottom: 1px solid #334155; font-family: monospace; }
        tr:hover { background: #334155; }
        .badge { padding: 4px 8px; border-radius: 6px; font-size: 0.75rem; font-weight: bold; background: #38bdf8; color: #0f172a; }
    </style>
</head>
<body>
    <div class="container">
        <header>
            <h1>ZetGrep : Intelligence Report</h1>
            <p style="color: #94a3b8">Scan Duration: %s</p>
        </header>

        <div class="summary-grid">
            <div class="card">
                <div class="card-label">Total Hits</div>
                <div class="card-value">%d</div>
            </div>
            <div class="card">
                <div class="card-label">Files Scanned</div>
                <div class="card-value">%d</div>
            </div>
            <div class="card">
                <div class="card-label">Patterns Used</div>
                <div class="card-value">%d</div>
            </div>
        </div>

        <div class="chart-container">
            <canvas id="patternChart" height="100"></canvas>
        </div>

        <div class="card" style="padding: 0">
            <table>
                <thead>
                    <tr>
                        <th>Pattern</th>
                        <th>File:Line</th>
                        <th>Match Content</th>
                    </tr>
                </thead>
                <tbody>
                    %s
                </tbody>
            </table>
        </div>
    </div>

    <script>
        const data = %s;
        const ctx = document.getElementById('patternChart').getContext('2d');
        new Chart(ctx, {
            type: 'bar',
            data: {
                labels: Object.keys(data.stats),
                datasets: [{
                    label: 'Hits by Pattern',
                    data: Object.values(data.stats),
                    backgroundColor: '#38bdf8',
                    borderRadius: 6
                }]
            },
            options: {
                responsive: true,
                plugins: { legend: { display: false } },
                scales: {
                    y: { beginAtZero: true, grid: { color: '#334155' } },
                    x: { grid: { display: false } }
                }
            }
        });
    </script>
</body>
</html>
`, data.Duration, data.TotalHits, len(data.Targets), len(data.Patterns), generateRows(data.Results), string(jsonData))

	_, err = f.Write([]byte(html))
	return err
}

func generateRows(results []*models.Result) string {
	var rows string
	// Show only top 100 results in HTML for performance
	limit := 100
	if len(results) < limit {
		limit = len(results)
	}
	for i := 0; i < limit; i++ {
		res := results[i]
		rows += fmt.Sprintf(`
            <tr>
                <td><span class="badge">%s</span></td>
                <td style="color: #3498db">%s:%d</td>
                <td>%s</td>
            </tr>
        `, res.Pattern, res.File, res.Line, res.Content)
	}
	if len(results) > limit {
		rows += fmt.Sprintf(`<tr><td colspan="3" style="text-align: center; color: #94a3b8">... and %d more results saved to data streams ...</td></tr>`, len(results)-limit)
	}
	return rows
}
