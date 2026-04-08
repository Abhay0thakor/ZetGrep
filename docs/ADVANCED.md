# Advanced Configuration & Max Power

This guide covers features designed for processing massive datasets (40GB+) and custom orchestration.

## 📦 Multi-Config Merging
`zetgrep` allows you to chain multiple configuration files. This is useful for maintaining a "Base" config and adding "Project" specific overrides.

```bash
zetgrep -config-file base.yaml -config-file overrides.yaml ip target.txt
```
*   **Strings**: Last occurrence wins.
*   **Arrays**: Merged (e.g., `ignore_extensions` from both files will be active).

## 🚀 JSONL Streaming Engine
When scanning multi-gigabyte JSONL files, standard grep is inefficient. `zetgrep` uses a dedicated streaming engine.

### 1. Define Input Config (`input.yaml`)
```yaml
format: jsonl
target: body    # Field to scan regex against
id: url         # Field to use as the source identifier
decode: true    # Unescape characters in the target field
```

### 2. Run at Scale
```bash
zetgrep -input-config input.yaml -all massive_dump.jsonl
```

## 💎 Output Templating (`-o`)
Control the exact string printed to the console.

| Placeholder | Description |
| :--- | :--- |
| `{{pattern}}` | Matched pattern name |
| `{{file}}` | Filename or JSONL ID |
| `{{line}}` | Line number |
| `{{match}}` | The full regex match |
| `{{match[1]}}`| First capture group |
| `{{tool:ID}}` | Output from an active tool |

### Pro Example:
```bash
zetgrep -w b64_decode -o "MATCH [{{pattern}}] in {{file}} -> DECODED: {{tool:b64_decode}}" base64 data.txt
```

## 🔄 The "Process" Workflow
Optimized for very large scans where you don't want to re-read the source file.

1.  **Fast Scan**: Save matches to JSON.
    ```bash
    zetgrep -json -all 40gb_dump.jsonl > matches.json
    ```
2.  **Enrich**: Run tools on the matches later.
    ```bash
    zetgrep -process matches.json -w ip_info -o "{{file}} | {{tool:ip_info}}"
    ```
