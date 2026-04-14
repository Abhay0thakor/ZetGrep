# Usage Guide

ZetGrep provides a powerful CLI interface with several subcommands to manage and execute your reconnaissance workflows.

## Global Flags

These flags are available for all subcommands:

- `-v, --verbose`: Enable debug logging.
- `--silent`: Disable all output except for findings.
- `--no-color`: Disable ANSI colors in output.
- `--config-file`: Path to a global configuration file (Viper compatible).

## Subcommands

### 1. `scan`
The primary command used to perform pattern matching on targets.

**Usage:**
```bash
zetgrep scan [pattern] [targets...] [flags]
```

**Key Flags:**
- `--all`: Run all available patterns in the library.
- `-u, --unique`: Deduplicate results across all patterns and targets.
- `--tags`: Filter patterns by specific tags (e.g., `--tags secrets,aws`).
- `-f, --format`: Output format (`text`, `json`, `table`).
- `-w, --workflow`: Tool IDs to chain for each match (e.g., `--workflow ip_info,whois`).
- `-c, --concurrency`: Number of concurrent workers (default: CPU * 2).
- `--dry-run`: Show what patterns and targets would be processed without executing.
- `--resume`: Path to a state file to resume a previous scan.

**Structured Data Flags (JSONL/CSV):**
- `--im`: Input mode (`jsonl`, `csv`, `text`).
- `--target`: (JSONL) Single field to scan (e.g., `--target msg`).
- `--targets`: (JSONL) Multiple fields to scan (e.g., `--targets msg,response.body`).
- `--csv-sep`: (CSV) Column separator (default: `,`).
- `--csv-targets`: (CSV) Column indices to scan (e.g., `--csv-targets 1,3`).
- `--csv-id`: (CSV) Column index to use as a source identifier.
- `--csv-no-header`: (CSV) Set if the file does not have a header row.

**Examples:**
```bash
# Scan a file for IP addresses and output a table
zetgrep scan ip data.txt -f table

# Scan specific JSONL fields from HTTPX output
zetgrep scan aws-keys results.jsonl --im jsonl --targets msg,response.body

# Scan a specific column in a CSV file
zetgrep scan ip data.csv --im csv --csv-targets 1
```

### 2. `web`
Starts the interactive Mission Control dashboard.

**Usage:**
```bash
zetgrep web --listen :8080
```

### 3. `list`
Lists all available patterns and tools in your library.

```bash
zetgrep list
```

### 4. `diagnose`
Debug a single line of input against your patterns. Useful for testing new regex patterns.

**Usage:**
```bash
zetgrep diagnose --line "some sample data with an API_KEY=12345" [pattern]
```

---
ZetGrep is proudly sponsored by **[Toolsura](https://www.toolsura.com/)**.
