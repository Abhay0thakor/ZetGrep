# Blog 11: Stateful Operations - Resuming Missions and Post-Processing

In the world of professional reconnaissance, "one-and-done" scans are a myth. Real missions take time, sometimes days. If your terminal crashes or your server restarts, you shouldn't have to start from scratch. In this post, we’ll look at ZetGrep’s stateful features: `--resume` and `--process`.

## 1. The Resume System (`--resume`)

ZetGrep features a robust, stateful resume system. As it scans, it periodically saves its progress (current file, current line) to a JSON state file.

### How to use it:
```bash
# Start a scan and save state
zetgrep scan --all massive_data/ --resume state.json
```

If the scan is interrupted, simply run the **exact same command** again. ZetGrep will detect `state.json`, read the last processed line, and jump straight to where it left off.

## 2. The Post-Processing Workflow (`--process`)

Sometimes you want to perform a "Fast Scan" first to find all possible matches, and then "Enrich" them later. This is where the `--process` flag shines.

### Step 1: Fast Scan to JSON
```bash
# Scan 100GB of logs for IPs, save only the raw matches
zetgrep scan ip massive_logs/ --format json --silent > initial_hits.json
```

### Step 2: Delayed Enrichment
Later, you can take that `initial_hits.json` and run your heavy workflows (like Shodan lookups or WHOIS) without ever touching the original 100GB of logs again.

```bash
# Enrich the previously found hits
zetgrep scan --process initial_hits.json --workflow shodan_lookup,whois
```

## Why State Matters

1. **Reliability**: Protect your time investment on massive datasets.
2. **Efficiency**: Only perform heavy API lookups on verified matches.
3. **Auditability**: Keep a record of your scan state for reporting and reproducibility.

Stateful operations turn ZetGrep from a simple tool into a reliable **Recon Database**.

Next up: **Mission Control**. We’ll leave the terminal for a moment and explore the **Web Dashboard**.

---
*ZetGrep is proudly sponsored by **[Toolsura](https://www.toolsura.com/)** - Your partner in professional security engineering.*
