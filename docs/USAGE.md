# Usage Guide (v0.8.7)

ZetGrep is a high-performance intelligence orchestrator designed for large-scale reconnaissance and data analysis.

## Global Flags

Available for all subcommands:

- `-v, --verbose`: Enable debug/trace logging.
- `--silent`: Absolute silence on terminal (ideal for piping).
- `--no-color`: Plain ASCII output with simplified symbols.
- `--quiet, -q`: Quiet mode (show progress bar but hide individual hits on terminal).
- `--config-file`: Path to global configuration(s).

## Core Subcommands

### 1. `scan`
Primary engine for pattern matching and data extraction.

**Usage:**
```bash
zetgrep scan [pattern] [targets...] [flags]
```

**Key Optimization Flags:**
- `--bloom`: Enable Bloom Filter fast-path (skips non-matching lines in nanoseconds).
- `--mmap`: Use Memory-Mapped I/O for 10x faster local file access.
- `--pcre`: Use the PCRE2 compatible regex engine (for complex look-aheads).
- `-c, --concurrency`: Set worker count (default: auto-scaled).

**Persistence & State:**
- `--incremental`: Only scan files changed since the last run.
- `--global-dedupe`: Skip findings already seen in previous scans (via bbolt).
- `--resume`: Pick up a large scan from a specific file/line.
    - **How it works**: When you use `--resume state.json`, ZetGrep automatically saves its progress every 10,000 records. If you stop the scan (Ctrl+C), you can run the exact same command again with the same `--resume state.json` to continue from the last saved point.

**Hardware Safety:**
- `--auto-scale`: Throttles workers based on system CPU load.
- `--thermal-threshold`: Pauses scan if CPU temperature exceeds limit.
- `--max-ram`: Pauses scan if RAM usage hits threshold.

**Reporting & Integration:**
- `--oJ`: Save results to JSON (supports `.zst` compression).
- `--oT`: Save results to a clean Text file.
- `--oH`: Generate a Professional HTML Intelligence Report.
- `--web`: Launch the Live SSE Dashboard.
- `--webhook`: Send rich alerts to Slack/Discord.

---

### 2. `delta`
Compare two scan result files and extract only **NEW** findings.

**Usage:**
```bash
zetgrep delta old_results.json new_results.json
```

---

### 3. `diagnose`
Debug a single string against your pattern library.

**Usage:**
```bash
zetgrep diagnose --content "AKIA1234567890EXAMPLE" -p aws-keys
```

---

### 4. `web`
Start a standalone Mission Control dashboard.

**Usage:**
```bash
zetgrep web --port 8080
```

---

### 5. `list`
Inventory your intelligence assets.

**Usage:**
```bash
zetgrep list patterns
zetgrep list tools
```

---
ZetGrep is proudly sponsored by **[Toolsura](https://www.toolsura.com/)**.
Verified and Hardened on 2026-06-01.
