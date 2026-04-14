# <img src="logo.svg" width="40" height="40" align="center"> ZetGrep (v0.4.6)

A professional-grade regex orchestration framework designed for massive reconnaissance data.

[![Sponsor](https://img.shields.io/badge/Sponsor-Toolsura-blue?style=for-the-badge)](https://www.toolsura.com/)

`ZetGrep` transforms standard patterns and custom tools into a high-speed intelligence pipeline. Optimized for **JSONL**, **CSV**, and **JSON** streaming, it is built for the modern security engineer.

---

## 🚀 Key Features

- **Modern CLI Architecture**: Powered by Cobra with dedicated subcommands (`scan`, `web`, `diagnose`, `list`).
- **Unified Configuration**: Robust management with Viper (Config files, Env Vars, Defaults).
- **Deep Orchestration**: Chain external tools dynamically using `{{tool:ID}}` placeholders.
- **High-Performance Streaming**: Process 100GB+ files with minimal RAM using a unified concurrent engine.
- **Multiple Formats**: Native support for JSONL, CSV, and raw Text data.
- **Stateful Resume**: Never lose progress on massive scans (`--resume`).
- **Library of 50+ Patterns**: Built-in intelligence for secrets, cloud keys, and more.

---

## 🛠️ Installation

### From Source
```bash
go install github.com/Abhay0thakor/ZetGrep/cmd/zetgrep@latest
```

### Using Makefile
```bash
git clone https://github.com/Abhay0thakor/ZetGrep.git
cd ZetGrep
make build
sudo mv zetgrep /usr/local/bin/
```

---

## 📖 Quick Start

### 1. List Available Intelligence
```bash
zetgrep list
```

### 2. Standard Pattern Scan
```bash
# Scan a directory using a specific pattern
zetgrep scan aws-keys ./data

# Scan everything using all patterns
zetgrep scan --all ./data
```

### 3. Pipeline Integration (The Power User Way)
```bash
# Combine with subfinder and httpx for deep secret hunting
subfinder -d target.com -silent | httpx -json -silent | zetgrep scan --all -f jsonl
```

### 4. Tool Chaining & Workflow
```bash
# Extract IP -> Run custom tool -> Output Table
zetgrep scan ip --workflow ip_info --format table data.txt
```

---

## 🌐 Mission Control (Web Dashboard)
Start the interactive dashboard to manage patterns, tools, and view live scan results:
```bash
zetgrep web --listen :8080
```

---

## 📚 Documentation

Detailed documentation is available in the `docs/` directory:

- [Installation Guide](docs/INSTALLATION.md)
- [Comprehensive Usage](docs/USAGE.md)
- [Input Formats (JSONL, CSV, Text)](docs/FORMATS.md)
- [Tool Chaining & Orchestration](docs/TOOLS.md)
- [Real-World Examples](docs/EXAMPLES.md)
- [Web Dashboard Guide](docs/DASHBOARD.md)

---

## 💖 Sponsored By
ZetGrep is proudly sponsored by **[Toolsura](https://www.toolsura.com/)** - Your ultimate hub for security and development tools.

---

## 🛡️ License
MIT License. Created for the security community.
