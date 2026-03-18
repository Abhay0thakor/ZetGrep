# ZetGrep: Maximum Potential Test Prompts

Use these prompts to challenge an AI (like Gemini or GPT) to create the most complex configurations possible for `ZetGrep`. These are designed to showcase the orchestration power of the tool.

---

## 📦 Part 1: Input Configuration Prompts (JSONL)
*Give these to an AI along with the **AI Master Context** to generate complex parsing rules.*

1.  **Nested CloudWatch Logs**: "Create an input config for AWS CloudWatch JSONL logs where the actual message is inside `logStreams[0].events[].message` and the ID is `owner`. Handle the case where the message might be a escaped JSON string."
2.  **Kubernetes Pod Logs**: "I have K8s logs in JSONL. I need to scan the `content` field but use a concatenated string of `pod_name` + `namespace` + `container_id` as the source ID. How do I define this in ZetGrep YAML?"
3.  **Burp Suite Advanced**: "Create a config for Burp Suite logs where the `response_body` is the target, but only if the `status_code` is 200. Use the `url` field as the ID and set `decode: true` to handle HTML entities."
4.  **Elasticsearch Export**: "Generate a config for an Elasticsearch `_search` export. The target field is `_source.data.raw_html` and the identifier is `_id`. Ensure it handles the deep nesting correctly."
5.  **Multi-Identifier Logs**: "I need an input config where the ID field is `metadata.source.ip` and the target is `payload.hex_dump`. Ensure `decode` is true."
6.  **Crawl Data (GAU/Wayback)**: "Create a config for `gau` JSON output where the target is the `url` itself (to find patterns in parameters) and the ID is the `timestamp`."
7.  **Authentication Audit**: "I have JSONL auth logs. Scan the `user_agent` field for suspicious patterns, and use the `user_email` field as the source ID."
8.  **API Response Collector**: "Create a config for a file containing raw API responses. The target is `json_response.results[0].body` and the ID is `endpoint_path`."
9.  **Security Hub Events**: "Parse AWS Security Hub JSONL. The target is `Findings[0].Description` and the ID is `Findings[0].Resources[0].Id`."
10. **Database SQL-in-JSON**: "I have a JSONL dump of a database. Scan the `query_text` field for SQL injection patterns, using `session_id` as the ID."

---

## 🛠️ Part 2: Tool Configuration Prompts (Plugins)
*Use these to create complex post-processing logic.*

1.  **JWT Deep Inspector**: "Create a tool that takes a JWT match, extracts the payload, decodes it, and then uses `date` to convert the `exp` timestamp into a human-readable format."
2.  **DNS Security Checker**: "Create a tool that extracts a domain from a URL, runs `dig +short`, and then greps the output to see if it points to a known Cloudflare IP range."
3.  **Automatic Nmap**: "Create a tool called `port_scan` that takes an IP match and runs a fast `nmap -sV -T4` on the top 20 ports, returning only the service versions."
4.  **Base64-to-Jq**: "Create a tool that extracts a Base64 string, decodes it, and pipes it into `jq` to extract a specific field called `'access_token'`."
5.  **Sensitive File Checker**: "Create a tool that takes a path match (e.g., `/etc/passwd`) and runs a `ls -l` to check permissions, returning the owner and group."
6.  **Whois Privacy Audit**: "Create a tool that runs `whois` on a domain match and extracts the 'Registrar' and 'Expiration Date' into a single line."
7.  **API Validator**: "Create a tool that takes an API key match and runs a `curl` request to an 'identity' endpoint to see if the key is still active (HTTP 200)."
8.  **Entropy Threshold Alert**: "Create a python-based tool for ZetGrep that calculates the exact Shannon Entropy of a match and returns 'CRITICAL' if it is above 5.0."
9.  **Binary Header Analysis**: "Create a tool that takes a hex-encoded match, decodes it to binary, and uses `file -b` to identify the actual file type."
10. **Github Repo Hunter**: "Create a tool that takes a Github URL match and uses the `gh` CLI to fetch the number of stars and the last commit date."
11. **Subdomain Takeover Check**: "Create a tool that runs `host -t CNAME` on a domain match and checks if the output contains common 'dead' pointers like 'herokudns.com'."
12. **HTML Meta Extractor**: "Create a tool that takes a URL, curls it, and uses `grep` to extract the `<title>` and `<meta description>` tags."
13. **Python String Unescaper**: "Create a complex tool that uses a Python one-liner to unescape double-encoded Unicode strings from a match."
14. **S3 Bucket ACL Check**: "Create a tool that takes an S3 bucket name match and uses `aws s3api get-bucket-acl` to check if 'AllUsers' have access."
15. **Hash Type Auto-ID**: "Create a tool that takes a 32, 40, or 64 character hash and identifies if it is MD5, SHA1, or SHA256 based on length."
16. **Directory Listing Auditor**: "Create a tool that curls a URL match and returns 'VULNERABLE' if the body contains the string 'Index of /'."
17. **Email Breach Check**: "Create a tool that takes an email match and uses `curl` to query a breach database API (like HaveIBeenPwned)."
18. **SQLMap Auto-Trigger**: "Create a tool that takes a URL match with parameters and spawns a `sqlmap` instance in `--batch --banner` mode."
19. **IP Geolocation Pipe**: "Create a tool that takes an IP match and uses `curl -s ipapi.co/{{match}}/city/` to return the city name."
20. **Slack Webhook Alerter**: "Create a tool that takes a high-priority match and sends a POST request to a Slack Webhook with the match details and filename."
