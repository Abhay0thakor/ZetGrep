# Blog 8: Workflow Orchestration - From Regex to Intelligence

In most tools, finding a match is the end of the journey. In ZetGrep, it's just the beginning. Today, we’re diving into **Workflow Orchestration**, the feature that allows you to transform raw data into actionable intelligence in real-time.

## What is a Workflow?

A workflow in ZetGrep is a sequence of external tools that are executed for every pattern match. This allows you to "enrich" your findings.

- Found an IP? **Run a GeoIP lookup.**
- Found a Base64 string? **Decode it.**
- Found a subdomain? **Check if it's alive.**

## Step 1: Defining a Tool

Tools are defined in YAML files (default: `~/.config/gf/tools`). They use a powerful placeholder system to interact with the scan results.

### Example: `ip_info.yaml`
```yaml
id: ip_info
name: IP Enrichment
description: Fetches ASN and Organization for an IP
command: "curl -s https://ipapi.co/{{match}}/json | jq -r '.org'"
field: organization
```

### Placeholders:
- `{{match}}`: The string that triggered the pattern.
- `{{file}}`: The source file.
- `{{line}}`: The line number.
- `{{pattern}}`: The name of the matching pattern.

## Step 2: Executing the Workflow

To trigger your tools, use the `--workflow` (or `-w`) flag followed by the tool ID.

```bash
# Scan for IPs and immediately fetch their organization info
zetgrep scan ip logs/ --workflow ip_info
```

## The Power of Fields

Note the `field: organization` in the YAML. ZetGrep uses this label when printing results:

`[ip] logs/access.log:10: 1.1.1.1`
`   ↳ organization: Cloudflare, Inc.`

## Why this is a Game Changer

Orchestration allows you to build **self-healing and self-enriching pipelines**. Instead of manually reviewing 1,000 matches, you can use workflows to automatically filter out known organizations, decode encoded data, or even send critical hits to a Slack webhook.

In the next post, we’ll take this to the next level with **Dynamic Tool Chaining**, where we’ll see how one tool can pass its output to another.

---
*ZetGrep is proudly sponsored by **[Toolsura](https://www.toolsura.com/)** - Elevate your security workflow with elite tools.*
