# Creating Custom Plugins (Tools)

`zetgrep` is more than a search tool; it's an orchestration engine. You can define YAML files that take every match found by `zetgrep` and pipe it into any Linux command.

## 🛠 Tool Anatomy
Tool files are YAML files stored in your `tools_dir`.

```yaml
id: my_lookup           # Unique ID for the -tools flag
name: Custom Lookup     # Friendly name
description: Runs a custom command on matches
extract: "[a-z]+"       # (Optional) Sub-regex to extract data from the match
command: "echo {{match}} | my-tool" # The command to execute
field: custom_output    # The label used in output
```

## 🔄 Template Variables in Tools
You can use these variables inside the `command` string:

| Variable | Description |
| :--- | :--- |
| `{{match}}` | The full string matched by the pattern regex. |
| `{{extracted}}` | The part of the match caught by the tool's `extract` regex. |
| `{{file}}` | The name of the source file. |
| `{{pattern}}` | The name of the active pattern. |
| `{{match[1]}}` | Specific capture group from the original pattern regex. |

## 💡 Real-world Plugin Example: Base64 to JSON Formatter
This tool extracts a base64 string, decodes it, and then uses `jq` to pretty-print the resulting JSON.

```yaml
id: b64_json
name: Base64 JSON Decoder
description: Decodes base64 and formats as JSON
command: "echo {{match}} | base64 -d | jq '.'"
field: json_payload
```

**Usage**:
```bash
./zetgrep -tools b64_json auth-tokens .
```
