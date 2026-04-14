# Input Formats

ZetGrep natively supports multiple input formats, making it versatile for scanning both raw text and structured data exported from other tools.

## 1. Text Mode (Default)
Standard line-by-line scanning. Ideal for raw logs, source code, or piped output from simple tools.

```bash
zetgrep scan ip logs/access.log
```

## 2. JSONL Mode (`--im jsonl`)
Optimized for JSON Lines format (one JSON object per line). This is the standard output format for tools like `httpx`, `subfinder`, and `ffuf`.

By default, ZetGrep will scan the whole line. You can use an input configuration or flags to target specific fields.

**Example with `httpx`:**
```bash
# Scan only the 'body' field of httpx output
# Requires an input config or target definition in your config
zetgrep scan secrets httpx_results.jsonl --im jsonl
```

## 3. CSV Mode (`--im csv`)
Supports scanning comma-separated value files. You can configure which columns to scan.

**Example:**
```bash
# Scan a CSV file, assuming columns are (id, url, data)
zetgrep scan secrets dump.csv --im csv
```

## Advanced: Dot Notation for Nested JSON
When in JSONL mode, you can target nested fields using dot notation in your configuration:

```yaml
# In an input-config file
id: "url"
targets:
  - "response.body"
  - "response.headers.server"
```

---
ZetGrep is proudly sponsored by **[Toolsura](https://www.toolsura.com/)**.
