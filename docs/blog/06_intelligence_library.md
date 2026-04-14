# Blog 6: Building Your Intelligence Library - Patterns that Matter

So far, we've discussed how ZetGrep reads data. Now, let's talk about the "brains" of the operation: **Patterns**. In ZetGrep, a pattern is more than just a regex; it's a modular piece of intelligence.

## The Pattern Schema

ZetGrep patterns are stored as simple JSON files in your patterns directory (default: `~/.config/gf/patterns`). 

### Anatomy of a Pattern (`aws-keys.json`)
```json
{
  "name": "aws-keys",
  "pattern": "(AKIA|ASIA)[A-Z0-9]{16}",
  "flags": "-i",
  "tags": ["cloud", "secrets"]
}
```

### Breakdown:
- **`name`**: The unique identifier you use in the CLI.
- **`pattern`**: The Go-compatible regular expression.
- **`flags`**: (Optional) Standard regex flags like `-i` for case-insensitive.
- **`tags`**: (Optional) Metadata used for grouping and filtering.

## Organizing with Tags

As your library grows to 100+ patterns, running all of them against every file becomes inefficient. Tags allow you to run targeted scans.

```bash
# Run only patterns related to AWS
zetgrep scan --tags aws target_data.jsonl

# Run patterns related to both cloud and secrets
zetgrep scan --tags cloud,secrets target_data.jsonl
```

## Advanced: The `patterns` Array

Sometimes, a single regex isn't enough to capture a concept. ZetGrep allows you to define multiple regexes in a single pattern file.

```json
{
  "name": "php-errors",
  "patterns": [
    "Fatal error:",
    "Parse error:",
    "Warning: .* in .*"
  ]
}
```
ZetGrep will automatically combine these into a single optimized search pass.

## Best Practices for Pattern Design

1. **Be Specific**: Avoid overly broad patterns like `.*password.*`. They generate massive amounts of noise.
2. **Use Capture Groups**: If you want to extract specific parts of a match for use in tools (more on this in Blog 8), ensure your regex uses parentheses `()`.
3. **Test with `diagnose`**: Before adding a pattern to your library, use the `diagnose` command to verify it matches your sample data correctly.

```bash
zetgrep diagnose --line "My key is AKIA1234567890ABCDEF" aws-keys
```

By building a high-quality library of patterns, you transform ZetGrep from a simple scanner into a sophisticated automated auditor.

Next up: **Smart Pruning**. We’ll see how to use AI and Entropy to filter out the noise and find the needles in the haystack.

---
*ZetGrep is proudly sponsored by **[Toolsura](https://www.toolsura.com/)** - Powering the next generation of security tools.*
