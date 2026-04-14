# Blog 13: Reporting & Templating - Professional Output for Stakeholders

Reconnaissance is only valuable if you can communicate the results. Whether you are reporting to a CISO or documenting findings for your own database, the format matters. In this post, we’ll explore ZetGrep’s reporting and templating engine.

## 1. Custom CLI Templates (`--template`)

If you are piping ZetGrep into another tool or just want a specific terminal view, use the `--template` flag. It supports all standard placeholders plus tool outputs.

```bash
# Output format: [PATTERN] FILE -> MATCH
zetgrep scan ip data/ --template "[{{pattern}}] {{file}} -> {{match}}"
```

### Advanced: Workflow Templates
You can include the output of any chained tool in your template:
```bash
zetgrep scan ip data/ -w ip_info --template "{{match}} belongs to {{tool:ip_info}}"
```

## 2. Professional Markdown Reports (`--report`)

For formal engagements, ZetGrep can generate a structured Markdown report. This report includes pattern names, source files, and the matching content formatted for readability.

```bash
# Generate a report file
zetgrep scan --all massive_dump.jsonl --report --output engagement_report.md
```

### What's in the report:
- **High-level summary**: Pattern identifiers and file sources.
- **Contextual snippets**: The exact matching content formatted as code blocks.
- **Metadata**: Timestamps and hit counts (when combined with other flags).

## 3. Machine-Readable Formats (`--format json/table`)

- **JSON**: Perfect for integration with other automation scripts or databases. Each result is a JSON object with full metadata.
- **Table**: Best for human review in the terminal. Uses an optimized grid layout that auto-adjusts to your terminal width.

```bash
# JSON for the machines
zetgrep scan ip logs/ -f json

# Table for the humans
zetgrep scan ip logs/ -f table
```

## Why This Matters

Standard `grep` output is hard to parse and even harder to present. ZetGrep’s templating engine gives you **full control over the data presentation layer**, saving you hours of post-processing and formatting work.

In our final post, we’ll see how all of these features come together in a **Real-World Recon Pipeline**.

---
*ZetGrep is proudly sponsored by **[Toolsura](https://www.toolsura.com/)** - Elevate your reporting with professional tools.*
