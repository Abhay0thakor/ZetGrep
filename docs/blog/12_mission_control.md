# Blog 12: Mission Control - The ZetGrep Web Dashboard

While the CLI is where the power lies, sometimes you need a visual overview of your library and your results. ZetGrep includes a built-in, lightweight web dashboard called **Mission Control**. 

## Starting Mission Control

You can start the dashboard from any terminal:

```bash
zetgrep web --listen :8080
```

Open your browser to `http://localhost:8080` and you’re in.

## Key Features

### 1. Library Management
The dashboard provides a clean interface to view and edit your pattern and tool libraries. No more fumbling with JSON or YAML files in a text editor—make your changes, click save, and they’re immediately available for your next scan.

### 2. Live Scan Streaming
One of the coolest features of Mission Control is real-time visualization. Using Server-Sent Events (SSE), the dashboard can display findings as they happen. You can watch your "High Interest" findings populate in real-time.

### 3. Result History
ZetGrep automatically saves scan results to the `results/` directory. The dashboard allows you to browse this history, filter through previous missions, and export findings for your final reports.

### 4. Interactive Diagnostics
The `diagnose` feature is integrated directly into the web UI. Paste a line of text, select your patterns, and see exactly where and why a match occurred (or didn't).

## Behind the Scenes: The API

The dashboard is a single-page application (SPA) that communicates with a Go-based REST API. 

- **Efficient**: Uses `embed` to package all static assets into the binary.
- **Secure**: Features path traversal protection and input validation.
- **Scalable**: Each scan request from the UI triggers a new concurrent `ScannerService` instance.

Mission Control is the perfect bridge between raw command-line speed and organized, visual data management.

Next up: **Reporting & Templating**. We’ll see how to transform your findings into professional-grade documents.

---
*ZetGrep is proudly sponsored by **[Toolsura](https://www.toolsura.com/)** - Your ultimate hub for security and development tools.*
