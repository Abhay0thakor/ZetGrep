# Real-World Examples

ZetGrep shines when integrated into a larger reconnaissance pipeline. Here are some common real-world scenarios.

## 1. Finding Secrets in HTTP Responses
Generate JSONL data using `httpx` and then use ZetGrep to hunt for secrets across all bodies.

```bash
# 1. Gather live subdomains
subfinder -d target.com -silent > live_subs.txt

# 2. Probe for web services and save full responses in JSONL
httpx -l live_subs.txt -json -o results.jsonl -silent

# 3. Use ZetGrep to scan all bodies for AWS keys
zetgrep scan aws-keys results.jsonl --im jsonl
```

## 2. Searching URL Archives for Interesting Paths
Use `gau` or `waybackurls` to fetch archived URLs and filter them for debug pages or sensitive endpoints.

```bash
# Fetch and scan URLs for debug pages
gau target.com | zetgrep scan debug-pages -f table
```

## 3. Extracting and Enriching IP Addresses from Logs
Identify all IP addresses in a log file and automatically fetch their GeoIP information using a workflow.

```bash
# Requires the 'ip_info' tool to be configured in ~/.config/gf/tools/ip_info.yaml
zetgrep scan ip /var/log/nginx/access.log --workflow ip_info --format table
```

## 4. Deep Secret Hunting in JavaScript Files
Chain `katana` with ZetGrep to find sensitive information inside beautified JavaScript.

```bash
# Crawl and probe for JS files, then scan for patterns
katana -u https://target.com -jc -d 2 -silent | \
grep "\.js" | \
httpx -silent | \
zetgrep scan --all --unique
```

## 5. Cleaning up Data with Rewrite Mode
If you have a JSONL file with escaped content (like `\n` or `\u0022`), you can use the interactive rewrite mode to beautify it before further analysis.

```bash
# This will prompt for confirmation before modifying the file
# Note: You can trigger this using specialized tool configs
zetgrep scan --im jsonl --rewrite data.jsonl
```

---
ZetGrep is proudly sponsored by **[Toolsura](https://www.toolsura.com/)**.
