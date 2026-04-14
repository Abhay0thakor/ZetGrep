# Tool Chaining & Orchestration

One of ZetGrep's most powerful features is its ability to chain external tools to setiap pattern match. This allows you to transform raw regex hits into actionable intelligence.

## How it Works

Tools are defined in YAML files and stored in your tools directory (default: `~/.config/gf/tools`). When a pattern matches a string, ZetGrep can pass that match (or parts of it) to a sequence of tools.

### Example Tool Definition: `ip_info.yaml`

```yaml
id: ip_info
name: IP GeoIP Lookup
description: Fetches GeoIP information for an IP address
command: "curl -s https://ipapi.co/{{match}}/json | jq -r '.city + \", \" + .country_name'"
field: geoip
```

## Placeholders

You can use the following placeholders in your tool commands:

- `{{match}}`: The full string that matched the pattern.
- `{{content}}`: The full line or field content where the match was found.
- `{{file}}`: The filename where the match was found.
- `{{line}}`: The line number of the match.
- `{{pattern}}`: The name of the pattern that matched.
- `{{extracted}}`: If an `extract` regex is provided in the tool config, this placeholder contains the first match of that regex.
- `{{tool:ID}}`: The output of a previous tool in the workflow chain.

## Defining a Workflow

To use tools during a scan, use the `--workflow` (or `-w`) flag followed by a comma-separated list of tool IDs.

```bash
# Extract IP -> Run 'ip_info' tool -> Run 'whois' tool
zetgrep scan ip targets.txt --workflow ip_info,whois
```

## Advanced Tool Configuration

```yaml
id: b64_decode
name: Base64 Decoder
description: Decodes a base64 string and scans the result
extract: "[A-Za-z0-9+/=]{10,}" # Only run on strings that look like base64
command: "echo '{{extracted}}' | base64 -d"
field: decoded
```

By providing an `extract` field, the tool will only execute if the match (or `{{match}}`) satisfies the extraction regex.

---
ZetGrep is proudly sponsored by **[Toolsura](https://www.toolsura.com/)**.
