# Real-World Scenarios & Examples

Learn how to use `zetgrep` for common security and data analysis tasks with real output examples.

## 1. 🔑 Find Leaked Secrets in a Repository
Run a fast, comprehensive secrets scan while ignoring noisy binary and vendor files.

**Config (`config.yaml`)**:
```yaml
globals:
  ignore_extensions: [".png", ".jpg", ".exe", ".bin"]
  ignore_files: ["node_modules/", "vendor/"]
```

**Command**:
```bash
./zetgrep -config-file config.yaml -all ./target-repo
```

**Real Output**:
`CRITICAL: aws-keys found in https://aws-console.amazon.com/s3 (AKIAIOSFODNN7EXAMPLE)`

## 2. 🛡️ Decoding Hidden Base64 in Source Code
Search for base64-like strings and automatically decode them to see if they contain sensitive data.

**Command**:
```bash
./zetgrep -tools b64_decode -o "FILE: {{file}} | DECODED: {{tool:b64_decode}}" base64 ./src
```

**Real Output**:
`FILE: https://api.internal.corp/v1/auth | DECODED: {"alg":"HS256","typ":"JWT"}`

## 3. 🌐 Scanning Massive HTTP Response Bodies (40GB)
If you have a massive JSONL dump from `httpx`, use this method to find specific IPs or patterns.

**Input Config (`httpx.yaml`)**:
```yaml
format: jsonl
target: body
id: url
decode: true
```

**Command**:
```bash
./zetgrep -input-config httpx.yaml -json ip huge_dump.jsonl > matches.json
```

**Real Output (formatted)**:
```json
{
  "id": 1,
  "pattern": "ip",
  "file": "https://example.com/debug",
  "content": "172.16.254.1",
  "entropy": 3.12
}
```

## 4. 🗄️ Database String Identification
Extract hashes from a file and try to identify their type using a custom tool.

**Tool (`tools/hash_id.yaml`)**:
```yaml
id: hash_id
command: "python3 -c \"import sys; h=sys.stdin.read().strip(); print('MD5' if len(h)==32 else 'SHA256' if len(h)==64 else 'Unknown')\""
field: hash_type
```

**Command**:
```bash
./zetgrep -tool tools/hash_id.yaml -tools hash_id sec dump.sql
```
