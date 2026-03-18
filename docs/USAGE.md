# Usage Guide

`zetgrep` is designed to be both simple for quick lookups and powerful for complex data processing.

## 🏁 Basic Commands

### Search for a specific pattern
```bash
./zetgrep ip access.log
```

### Search multiple targets
```bash
./zetgrep aws-keys ./src ./backups logs/
```

### Run all registered patterns
```bash
./zetgrep -all .
```

## 🛠 Command Line Flags

| Flag | Type | Description |
| :--- | :--- | :--- |
| `-all` | bool | Runs all patterns in your library against the target. |
| `-config-file` | string | Path to a global config (YAML/JSON). Supports multiple uses. |
| `-input-config` | string | Path to a JSONL input definition (YAML). Supports multiple uses. |
| `-tool` | string | Path to an individual tool YAML file. Supports multiple uses. |
| `-tools` | string | Comma-separated list of Tool IDs to execute on matches. |
| `-o` | string | **Output Template.** Customize the visual format of results. |
| `-json` | bool | Output everything in structured JSON format. |
| `-process` | string | Re-process a previously saved `results.json` with new tools. |
| `-smart` | bool | AI-assisted filtering for high-interest findings. |
| `-entropy` | bool | Filter results by Shannon Entropy (default > 3.5). |
| `-list` | bool | Lists all patterns and available plugins. |
| `-config` | bool | Prints currently active configuration paths. |
| `-web` | string | Starts the Intelligence Dashboard (e.g., `-web :8080`). |

## 🔍 Filtering Results

### Entropy Filtering
Useful for finding random-looking strings like API keys or secrets while ignoring standard English text.
```bash
./zetgrep -entropy aws-keys secrets.txt
```

### Smart AI Filtering
Uses internal heuristics to separate "boring" matches from "high-interest" security findings.
```bash
./zetgrep -smart php-sources ./html
```
