# <img src="logo.svg" width="40" height="40" align="center"> ZetGrep (v0.1.8)

A professional-grade pattern matching and orchestration engine designed for security auditors, bug hunters, and data engineers. 

`ZetGrep` simplifies complex regex analysis and allows you to pipe matches into custom tool workflows. It is optimized for both standard file systems and massive (100GB+) JSONL/CSV datasets with stateful resume capabilities.

---

## 🚀 Key Features
- **ProjectDiscovery-Style CLI**: Categorized, intuitive, and highly configurable.
- **Stateful Resume**: Pause and resume long-running scans without losing progress.
- **Multi-Format Streaming**: Native high-speed support for JSONL, JSON, and CSV.
- **Advanced Orchestration**: Chain multiple tools together (e.g., `base64_decode` -> `whois`).
- **Tag-Based Filtering**: Run groups of patterns using tags (e.g., `-tags secrets,cloud`).
- **Modern Web Dashboard**: Real-time analytics, metrics, and an integrated code editor.

---

## 🛠️ Installation

```bash
go install github.com/Abhay0thakor/ZetGrep/cmd/zetgrep@latest
```

---

## 📖 Usage

### Quick Start
```bash
# Scan a directory for all patterns
zetgrep -all ./recon_data

# Scan a JSONL file using a specific input config
zetgrep -input-config inputs/httpx.yaml -all httpx_output.jsonl

# Resume a paused scan
zetgrep -resume resume.cfg -all ./massive_dump
```

### Advanced Examples
```bash
# Filter patterns by tags
zetgrep -tags secrets,cve -l targets.txt

# Chain tools and output using a template
zetgrep -all -w ip_info,whois -o "[{{file}}] {{tool:ip_info}} -> {{tool:whois}}" data.jsonl
```

---

## 🏳️ Flags

### INPUT
- `-l, -list string`: File containing a list of targets to scan.
- `-stdin`: Read targets from standard input.
- `-input-config string`: Path to input configuration file (YAML).

### CONFIG
- `-config-file string`: Path to global configuration file.
- `-pd string`: Directory containing pattern definitions.
- `-td string`: Directory containing tool definitions.

### FILTER
- `-all`: Run all available patterns in the library.
- `-tags string`: Filter patterns by tag (comma-separated).
- `-smart`: Use AI-based interest filtering.
- `-entropy`: Filter by high-entropy content.
- `-diagnose string`: Step-by-step diagnostic for a single line.

### OUTPUT
- `-json`: Output results in JSON format.
- `-report`: Generate a professional Markdown intelligence report.
- `-o string`: Define a custom output template.
- `-silent`: Display only the results (no banner/info).
- `-nc, -no-color`: Disable colorized output.

### LOGIC
- `-web string`: Start the web-based Mission Control dashboard.
- `-resume string`: Path to the scan state file for pausing/resuming.
- `-w, -workflow string`: Comma-separated tool IDs to execute.
- `-update`: Self-update ZetGrep to the latest version.

---

## 🛡️ License
Distributed under the MIT License. See `LICENSE` for more information.
