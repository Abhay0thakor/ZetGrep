# Production Examples (v0.8.7)

ZetGrep is optimized for high-throughput reconnaissance. Here are verified production workflows.

## 1. 100GB+ Data Lake Secret Hunting
Scan massive datasets using **Parallel Mmap** and **Bloom Filters**.

```bash
# Scan a huge compressed dump for all patterns
zetgrep scan lake.json.zst --all --bloom --mmap --oJ hits.json.zst --web
```
*   `--bloom`: Skips non-matching lines instantly.
*   `--mmap`: Maps the file into virtual memory for extreme speed.
*   `--web`: Stream hits to your browser in real-time.

## 2. CI/CD Incremental Scan
Integrate into your pipeline to only scan changed files.

```bash
# Only scan new/modified files in the repo
zetgrep scan . --all --incremental --global-dedupe --webhook "https://..."
```
*   `--incremental`: Skips files already scanned.
*   `--global-dedupe`: Ensures you don't get the same finding twice.
*   `--webhook`: Alerts the team instantly.

## 3. High-Signal Intelligence Reporting
Extract and enrich data with visualization.

```bash
# Scan for IPs and enrich with GeoIP
zetgrep scan ip logs/ -w ip_info --oH report.html --webhook-level high-interest
```
*   `-w ip_info`: Runs the WHOIS/GeoIP tool on every hit.
*   `--oH`: Generates a clean HTML dashboard with analytics.
*   `--webhook-level`: Only alerts if the interest classifier is triggered.

## 4. Historical Delta Analysis
Find what changed between two reconnaissance runs.

```bash
# Compare last week's scan with today's
zetgrep delta results_may_24.json results_june_01.json
```
*   Outputs only the findings that are **new** in the second file.

## 5. Precise CSV Parsing
Extracting credentials from specialized CSV outputs.

```bash
# Scan column 3 of a semicolon-separated file
zetgrep scan secrets users.csv --csv-sep ";" --csv-targets 3 --csv-no-header
```

---
ZetGrep is proudly sponsored by **[Toolsura](https://www.toolsura.com/)**.
