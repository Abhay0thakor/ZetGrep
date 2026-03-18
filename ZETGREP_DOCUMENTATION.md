# ZetGrep Intelligence Tool - Documentation

`zetgrep` is a high-performance pattern matching wrapper designed for security auditing, large-scale data analysis, and automated post-processing. It supports standard file scanning and massive (40GB+) JSONL streaming.

---

## 🚀 Core Commands

### Basic Scanning
Scan a file or directory for a specific pattern:
```bash
./zetgrep php-sources ./my-project
```

### Run All Patterns
Scan for every pattern in your patterns directory:
```bash
./zetgrep -all ./my-project
```

### List Available Patterns & Tools
Show what patterns and plugins are currently loaded:
```bash
./zetgrep -list
```

---

## 🛠 Command Line Flags

| Flag | Description |
| :--- | :--- |
| `-all` | Run all available patterns against the target. |
| `-config-file` | Path to a global config (JSON or YAML). |
| `-input-config` | Path to a YAML file specifically for JSONL processing. |
| `-tools` | Comma-separated list of Tool IDs to run on matches (e.g., `-tools b64,dase`). |
| `-o` | **Output Template.** Define exactly how results look (see Templating). |
| `-json` | Output results in raw JSON format. |
| `-smart` | Enable AI-based "high-interest" filtering. |
| `-entropy` | Filter matches by Shannon Entropy (defaults to > 3.5). |
| `-list` | List all patterns and tools and exit. |
| `-config` | Show the current paths for patterns and tools directories. |
| `-web` | Start the Web UI on a specific port (e.g., `:8080`). |

---

## 💎 Max Power: Output Templating (`-o`)

The `-o` flag allows you to customize the output string. You can use the following placeholders:

| Placeholder | Description |
| :--- | :--- |
| `{{pattern}}` | The name of the pattern that matched. |
| `{{file}}` | The filename (or ID for JSONL). |
| `{{line}}` | The line number of the match. |
| `{{match}}` | The full string that matched the regex. |
| `{{match[1]}}` | The first capture group from your regex. |
| `{{content}}` | The full line or content block containing the match. |
| `{{tool:ID}}` | The output from a specific post-processing tool. |

### Example Template:
```bash
./zetgrep -o "[{{pattern}}] Found {{match}} in {{file}} (Tool Result: {{tool:b64_decode}})" aws-keys .
```

---

## 📦 Configuration

### 1. Global Config (`config.yaml`)
Control where your patterns live and what files to ignore globally.

```yaml
patterns_dir: "/root/my-patterns"
tools_dir: "/root/my-tools"
globals:
  ignore_extensions: [".jpg", ".png", ".css", ".exe"]
  ignore_files: ["jquery.js", "bootstrap.min.js"]
```

### 2. JSONL Input Config (`input.yaml`)
Use this for scanning massive data dumps (30GB - 50GB files).

```yaml
format: jsonl
target: body    # The field inside the JSON line to scan
id: url         # The field to use as the identifier (Source)
decode: true    # Auto-decode unicode/escaped characters
```

**Usage:**
```bash
./zetgrep -input-config input.yaml -all /path/to/massive_dump.jsonl
```

---

## 🔧 Creating Custom Tools

Tools are defined in YAML files inside your `tools_dir`. They allow you to pipe matches into other Linux commands.

**Example Tool (`tools/whois.yaml`):**
```yaml
id: domain_whois
name: Whois Lookup
description: Performs a whois lookup on the match
command: "whois {{match}} | grep 'Registrar:'"
field: registrar
```

---

## 💡 Practical Examples

### Example 1: Extract and Decode Base64 from JSONL
If you are scanning a 40GB file where the `body` field might contain base64 strings:
1. Define a pattern for base64.
2. Use the `b64_decode` tool.
3. Use a custom output template.
```bash
./zetgrep -input-config input.yaml -tools b64_decode -o "{{file}} -> {{tool:b64_decode}}" b64_pattern dump.jsonl
```

### Example 2: Filtering Noise
Run a scan but ignore all `.js` and `.html` files:
```bash
# In your config.yaml
globals:
  ignore_extensions: [".js", ".html"]

# Run command
./zetgrep -config-file config.yaml -all .
```

### Example 3: Extracting Specific Regex Groups
If your pattern is `API_KEY=([a-z0-9]+)`, you can extract just the key:
```bash
./zetgrep -o "Key Found: {{match[1]}}" api-pattern .
```

---

## ⚡ Performance Tips
- **Worker Pool:** By default, `zetgrep` uses 4 workers for file scanning and 8 workers for JSONL regex matching.
- **Ripgrep:** For standard files, `zetgrep` will automatically use `rg` if installed for maximum speed.
- **Streaming:** JSONL scanning is streamed line-by-line, meaning it can scan a 100GB file using only a few megabytes of RAM.
