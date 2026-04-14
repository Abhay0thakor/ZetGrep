# Blog 7: Smart Pruning - Quality Over Quantity

In large-scale reconnaissance, the biggest enemy is **Noise**. A single broad regex can generate 10,000 false positives, burying the one critical finding you actually need. In this post, we’ll explore ZetGrep’s sophisticated pruning features that help you find the needle without the haystack.

## 1. Global Deduplication (`--unique`)

When scanning multiple files or using multiple patterns, you often get redundant hits. The `--unique` (or `-u`) flag ensures that you only see each unique combination of pattern and content once per scan.

```bash
# Don't show the same IP 100 times
zetgrep scan ip access.log --unique
```

## 2. High-Entropy Filtering (`--entropy`)

Secrets like API keys, SSH private keys, and session tokens share a common characteristic: **High Randomness (Entropy)**. 

Standard text has low entropy. Randomly generated strings have high entropy. By using the `--entropy` flag, ZetGrep calculates the Shannon Entropy of every match and only reports those that cross a threshold (usually > 3.5).

```bash
# Find strings that look like random keys, not English words
zetgrep scan base64 data.txt --entropy
```

## 3. The "Smart" AI Classifier (`--smart`)

ZetGrep includes a built-in interest classifier. This engine analyzes the context and structure of a match to determine if it "looks" like something a security researcher would care about. 

When you enable `--smart`, ZetGrep runs every match through this heuristic engine and discards low-interest items.

```bash
# Let the machine decide what's interesting
zetgrep scan --all massive_dump.jsonl --smart
```

## 4. Input Filtering (Pruning at the Source)

Sometimes the best way to reduce noise is to not scan it in the first place. ZetGrep allows you to define filters in your input configuration files to skip entire records based on their metadata.

### Example: Scanning only successful requests
```yaml
format: jsonl
targets: ["response.body"]
filters:
  status_code: "200"
  content_type: "application/json"
```
By using `filters`, the regex engine never even sees 404 pages or image binary data, leading to **massive performance savings**.

---

By combining these four strategies, you can reduce your finding volume by up to 90% while actually increasing the signal-to-noise ratio of your final report.

In the next post, we’re moving into the **Orchestration Masterclass**, where we’ll see how to transform these pruned findings into actionable intelligence using **Workflow Orchestration**.

---
*ZetGrep is proudly sponsored by **[Toolsura](https://www.toolsura.com/)** - Your ultimate hub for security and development tools.*
