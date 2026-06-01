# Intelligence Library (Patterns)

ZetGrep's power comes from its library of highly-curated patterns.

## 📁 Structure
Patterns are stored in `~/.config/gf/patterns/` as JSON files.

```json
{
  "name": "aws-keys",
  "pattern": "AKIA[A-Z0-9]{16}",
  "flags": "i",
  "tags": ["cloud", "secrets", "aws"]
}
```

## 🚀 Engine Optimization
ZetGrep automatically optimizes patterns:
*   **Simple Strings**: Handled by **Aho-Corasick** (10x faster).
*   **Advanced Patterns**: Can use **PCRE2** via `--pcre`.
*   **Rapid Negative-Check**: Automatic keyword extraction for **Bloom Filters**.

## 🏷️ Tagging
Use tags to organize your library and run specialized scans.

```bash
# Only run patterns tagged with 'secrets' or 'cloud'
zetgrep scan ... --tags secrets,cloud
```

## 🤝 Contribution
The intelligence library is a community effort. Please submit PRs with new patterns to expand the default reconnaissance capabilities.

---
ZetGrep is proudly sponsored by **[Toolsura](https://www.toolsura.com/)**.
