# Blog 14: The Ultimate Recon Pipeline - Integrating ZetGrep with the Best

In this final installment of our series, we're bringing everything together. ZetGrep was never meant to live in isolation—it was designed to be the "Intelligence Hub" of a modern reconnaissance pipeline. 

Here is how you can integrate ZetGrep with industry-standard tools like `subfinder`, `httpx`, and `nuclei`.

## The "Subdomain Secret Hunter" Pipeline

This pipeline gathers subdomains, probes them for web services, extracts full response bodies, and then uses ZetGrep to hunt for secrets with high-entropy filtering.

```bash
# 1. Enumerate subdomains
subfinder -d target.com -silent > subs.txt

# 2. Probe for alive web services and save full JSONL output
httpx -l subs.txt -json -o results.jsonl -silent

# 3. Use ZetGrep to hunt for secrets across all response bodies
# We use:
# - im jsonl: To handle httpx format
# - target response.body: To look inside the HTML/JS
# - tags secrets: To run only high-value patterns
# - entropy: To find random keys, not English words
# - workflow ip_info: To enrich any IPs found in the code
# - format table: For clean terminal review
zetgrep scan --tags secrets results.jsonl \
  --im jsonl --target response.body \
  --entropy --unique \
  --workflow ip_info \
  --format table
```

## The "Archive Exploiter" Pipeline

Looking for sensitive endpoints or debug pages in historical URL data.

```bash
# 1. Fetch historical URLs from Wayback Machine, AlienVault, etc.
gau target.com --subs | grep -vE "\.(jpg|jpeg|gif|png|css|woff)" > urls.txt

# 2. Filter for debug pages, PHP errors, or sensitive files using ZetGrep
zetgrep scan debug-pages,php-errors,sec urls.txt --unique -f table
```

## The "Nuclei Enrichment" Pipeline

Enriching vulnerabilities found by `nuclei` with additional environmental context.

```bash
# 1. Run nuclei and save as JSONL
nuclei -l subs.txt -jsonl -o nuclei_findings.jsonl -silent

# 2. Extract and enrich IP addresses from nuclei findings
zetgrep scan ip nuclei_findings.jsonl \
  --im jsonl --target ip \
  --workflow ip_info,whois \
  --template "VULN IP: {{match}} ({{tool:ip_info}})"
```

## Final Thoughts

By integrating ZetGrep into your existing toolkit, you move from "Data Gathering" to "Intelligence Generation." You no longer just have a list of subdomains; you have an enriched, pruned, and prioritized database of actionable findings.

Thank you for following the ZetGrep Deep-Dive Series. Now go forth and hunt with superpowers!

---
**ZetGrep is proudly sponsored by [Toolsura](https://www.toolsura.com/)** - Powering the next generation of security researchers.
