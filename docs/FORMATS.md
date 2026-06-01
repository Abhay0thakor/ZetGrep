# Supported Data Formats (v0.8.7)

ZetGrep is more than a line-matcher; it understands the structure of your data.

## 1. JSONL (JSON Lines)
Ideal for outputs from `httpx`, `nuclei`, or `katana`.

```bash
# Scan a specific field
zetgrep scan ... --im jsonl --target response.body
```
*   **Parallel Parsing**: Multiple threads parse different chunks of the JSONL file simultaneously.
*   **SIMD Acceleration**: Uses `buger/jsonparser` for zero-allocation field extraction.
*   **Nested Support**: Access fields like `metadata.user.id` using dot notation.

## 2. CSV / TSV
Optimized for database dumps and inventory lists.

```bash
# Semicolon separated, no header, scan column 2
zetgrep scan ... --im csv --csv-sep ";" --csv-no-header --csv-targets 2
```
*   **Header Awareness**: Automatically skips the first row if header is present.
*   **Identifier Mapping**: Use `--csv-id 0` to use the first column as the "file" name in results.

## 3. Raw Text / Logs
Standard behavior for `.txt`, `.log`, or piped input.

```bash
cat data.txt | zetgrep scan pattern
```
*   **Line Alignment**: Ensures findings are reported with correct line numbers.
*   **Mmap Fast-Path**: Even raw text uses the memory-mapped engine for speed.

## 4. Compressed Streams (`.zst`, `.gz`)
Scan compressed files without extracting them to disk.

```bash
# Real-time decompression and scanning
zetgrep scan archive.json.zst --pre-process "zstd -dc"
```

---

## Output Routing
ZetGrep separates Human UI from Machine Data.

| Channel | Format | Destination |
| :--- | :--- | :--- |
| **UI** | Professional (Colors/Symbols) | `stderr` |
| **Data** | Content Only (Raw/JSON) | `stdout` |

---
ZetGrep is proudly sponsored by **[Toolsura](https://www.toolsura.com/)**.
