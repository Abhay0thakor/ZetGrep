# Blog 4: Mastering JSONL - Hunting Secrets in Structured Data

In modern reconnaissance, tools like `httpx` and `subfinder` have standardized on JSONL (JSON Lines) for output. Why? Because it's machine-readable and supports streaming. In this post, we'll see how ZetGrep makes you a JSONL power user.

## The Problem with Grep and JSON

If you `grep` a JSONL file, you often get the entire object as one line. If that object contains a full HTML response body, your terminal becomes a mess. You lose context, and finding the exact field that matched is nearly impossible.

## The ZetGrep Solution: Dot Notation

ZetGrep allows you to target specific fields within a JSON object using simple dot notation.

### Scenario: Scanning HTTPX results
Imagine an `httpx` output:
```json
{"url":"https://kiwi.com","host_ip":"151.101.3.42","response":{"body":"...lots of html...","headers":{"server":"Varnish"}}}
```

### 1. Target a Single Field
To scan only the IP address field:
```bash
zetgrep scan ip results.jsonl --im jsonl --target host_ip
```

### 2. Deep Nesting
To scan the HTML body inside the `response` object:
```bash
zetgrep scan aws-keys results.jsonl --im jsonl --target response.body
```

### 3. Multiple Targets
Want to scan both the URL and the Body simultaneously?
```bash
zetgrep scan secrets results.jsonl --im jsonl --targets url,response.body
```

## Advanced Feature: Automatic Unescaping

JSON often escapes characters (e.g., `\n`, `\u0022`). If your regex depends on cleartext, standard grep might miss it. 

ZetGrep's `JSONLParser` automatically unescapes the content of your target fields before running the regex, ensuring that your patterns match the *actual* data, not the JSON-encoded version.

## Pro Tip: Pipelining from httpx
You don't even need to save a file. Pipe directly for real-time intelligence:

```bash
subfinder -d target.com -silent | httpx -json -silent | zetgrep scan secrets --stdin --im jsonl --target response.body
```

By targeting only the relevant fields, you significantly reduce the amount of data the regex engine has to process, leading to **massive performance gains** on large datasets.

Next up: **CSV Excellence**. We’ll see how to handle spreadsheets and column-based data with the same level of precision.

---
*ZetGrep is proudly sponsored by **[Toolsura](https://www.toolsura.com/)** - Your ultimate hub for security and development tools.*
