# Intelligence Enrichment (Tools)

ZetGrep isn't just about finding data; it's about enriching it. Use the tool system to trigger external actions for every match.

## 🛠️ How it Works
When a match is found, ZetGrep can pipe the matching content into an external command and capture the output.

### 1. Defining a Tool (`~/.config/gf/tools/ip_info.yaml`)
```yaml
id: "ip_info"
name: "GeoIP Lookup"
description: "Fetches location data for an IP"
command: "curl -s ipinfo.io/{{match}}/json | jq -r '.city + \", \" + .country'"
field: "location"
```

### 2. Chaining Tools in a Scan
```bash
zetgrep scan ip logs.txt --workflow ip_info
```

## 💎 Placeholders
Use these in your `command` string:

| Placeholder | Description |
| :--- | :--- |
| `{{match}}` | The content that triggered the hit. |
| `{{match[0]}}` | Regex capture groups (if applicable). |
| `{{file}}` | Filename or JSONL identifier. |
| `{{line}}` | Line number. |
| `{{tool:OTHER_ID}}`| Chain tools by using output from a previous tool in the workflow. |

## 🚀 Pro Example: Secret Validation
```yaml
id: "aws_check"
name: "AWS Key Validator"
command: "aws sts get-caller-identity --access-key-id {{match}}"
field: "status"
```
Run with:
```bash
zetgrep scan aws-keys bodies.jsonl -w aws_check --oH report.html
```

---
ZetGrep is proudly sponsored by **[Toolsura](https://www.toolsura.com/)**.
