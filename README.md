# <img src="logo.svg" width="40" height="40" align="center"> ZetGrep (v0.4.4)

A professional-grade regex orchestration framework designed for massive reconnaissance data.

`ZetGrep` transforms standard patterns and custom tools into a high-speed intelligence pipeline. Optimized for **JSONL**, **CSV**, and **JSON** streaming.

---

## 🚀 Key Features
- **Industry Standard CLI**: Categorized, intuitive, and clean flags.
- **Stateful Resume**: Never lose progress on massive scans (`-resume`).
- **Deep Orchestration**: Chain tools dynamically using `{{tool:ID}}` placeholders.
- **High-Performance Streaming**: Process 100GB+ files with minimal RAM.
- **Library of 50+ Patterns**: Built-in intelligence for secrets, cloud keys, and more.
- **Interactive Diagnostics**: Step-by-step breakdown of your configs (`-diagnose`).

---

## 📖 Usage

### Standard Scan
```bash
# Scan a directory using all available patterns
zetgrep -all ./data

# Pipe results from other tools
subfinder -d target.com | zetgrep -stdin -all
```

### Structured Data
```bash
# Scan specific fields in HTTPX JSONL
zetgrep -im jsonl -input-config inputs/httpx.yaml -all data.jsonl

# Scan spreadsheet data (CSV)
zetgrep -im csv -all dump.csv
```

### Orchestration & Reporting
```bash
# Chain tools: Extract IP -> Fetch Info -> Perform WHOIS
zetgrep -tags recon -w ip_info,whois -report -o "[{{file}}] {{tool:ip_info}} | {{tool:whois}}" data.jsonl
```

---

## 🏳️ CLI Flags

### INPUT
- `-im, -input-mode string`: Input format (`jsonl`, `csv`, `text`).
- `-input-config string`: Path to specialized input YAML.
- `-l, -list string`: File containing list of targets.
- `-stdin`: Read targets from standard input.

### CONFIG
- `-pd string`: Pattern directory (default: `~/.config/gf/patterns`).
- `-td string`: Tool directory (default: `~/.config/gf/tools`).
- `-config-file string`: Global config path.

### FILTER
- `-all`: Run all patterns in the library.
- `-tags string`: Filter patterns by tag (e.g. `secrets,cve`).
- `-smart`: AI-based high-interest filtering.
- `-entropy`: Filter by Shannon Entropy (> 3.5).
- `-diagnose string`: Debug a single line of input.

### OUTPUT
- `-json`: Output results in raw JSON.
- `-report`: Generate a professional Markdown report.
- `-o string`: Custom output template (e.g. `{{match}}`).
- `-silent`: Display only findings.
- `-nc, -no-color`: Disable colors.

### LOGIC
- `-resume string`: Resume scan from state file.
- `-w, -workflow string`: Chain of tool IDs to execute.
- `-update`: Self-update to latest version.
- `-health-check`: Verify environment paths.

---

## 🛠️ Installation
```bash
go install github.com/Abhay0thakor/ZetGrep/cmd/zetgrep@latest
```

---

## 🛡️ License
MIT License. Created for the security community.
