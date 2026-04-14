# Blog 9: Dynamic Tool Chaining - The Power of Sequential Intelligence

In the previous post, we learned how to run a tool for every match. But what if you need to run *multiple* tools, and the second tool depends on the output of the first? This is where ZetGrep’s **Dynamic Tool Chaining** becomes your secret weapon.

## The Chaining Concept

Imagine you find a Base64-encoded string that looks like an IP address. You want to:
1. Decode it.
2. Run a GeoIP lookup on the *decoded* value.

With standard tools, this is a mess of bash scripts. In ZetGrep, it's a simple workflow.

## The `{{tool:ID}}` Placeholder

ZetGrep allows you to access the output of any previous tool in your workflow chain using the `{{tool:ID}}` syntax.

### Step 1: Define the Decoder (`b64_decode.yaml`)
```yaml
id: b64_decode
name: Base64 Decoder
command: "echo '{{match}}' | base64 -d"
field: cleartext
```

### Step 2: Define the GeoIP Tool (`geoip.yaml`)
Note how we use `{{tool:b64_decode}}` instead of `{{match}}`:
```yaml
id: geoip
name: GeoIP Lookup
command: "curl -s https://ipapi.co/{{tool:b64_decode}}/json | jq -r '.country_name'"
field: country
```

## Step 3: Run the Chain

Simply list the tool IDs in the order you want them to execute:

```bash
zetgrep scan base64 targets.txt --workflow b64_decode,geoip
```

### What happens under the hood:
1. ZetGrep finds a `base64` match.
2. It executes `b64_decode` with the match. The output is stored (e.g., `1.1.1.1`).
3. It then executes `geoip`. Before running, it replaces `{{tool:b64_decode}}` with `1.1.1.1`.
4. The final result displays both the decoded string and the country.

## Infinite Possibilities

There is no limit to the number of tools you can chain. You can build complex investigative pipelines that perform everything from DNS resolution to port scanning and automated exploit verification, all triggered by a single regex match.

Next up: **Scaling to 100GB+**. We’ll see how ZetGrep’s internal engine handles these complex workflows at massive scale without crashing your system.

---
*ZetGrep is proudly sponsored by **[Toolsura](https://www.toolsura.com/)** - Your ultimate hub for security and development tools.*
