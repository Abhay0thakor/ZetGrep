# Blog 5: CSV Excellence - Surgical Precision in Spreadsheets

While JSONL is the king of APIs, CSV (Comma Separated Values) remains the king of data exports and business intelligence. From bug bounty payouts to inventory dumps, CSVs are everywhere. In this post, we’ll explore how ZetGrep handles them with surgical precision.

## The CSV Dilemma

A large CSV file might have 50 columns. If you `grep` for an IP address, you might get a match in the `client_ip` column (good), but you might also get one in a `description` column where someone just happened to write an IP (noise). 

## The ZetGrep Solution: Column Mapping

ZetGrep’s `CSVParser` allows you to target specific columns by their index (starting at 0).

### Scenario: Processing a user dump
Imagine a CSV named `users.csv`:
`id,username,email,last_login_ip,bio`

### 1. Target a Specific Column
To scan only the `last_login_ip` (Column 3):
```bash
zetgrep scan ip users.csv --im csv --csv-targets 3
```

### 2. Multiple Target Columns
To scan both the `email` (Column 2) and `bio` (Column 4) columns:
```bash
zetgrep scan secrets users.csv --im csv --csv-targets 2,4
```

### 3. Handle Custom Separators
Not all CSVs use commas. If you’re dealing with a semicolon-separated file (common in some regions):
```bash
zetgrep scan ip data.csv --im csv --csv-sep ";" --csv-targets 1
```

## Advanced Logic: Source Identification

When processing thousands of rows, you need to know *who* or *what* triggered a match. ZetGrep can use a specific column as a stable identifier for each finding.

```bash
# Use column 0 (id) as the identifier
zetgrep scan ip users.csv --im csv --csv-targets 3 --csv-id 0
```

The output will now look like this:
`[ip] users.csv:user_123: 1.1.1.1`

Instead of just a line number, you get the actual ID from the CSV, making it trivial to cross-reference the finding back to your database.

## Header Management

By default, ZetGrep assumes the first row is a header and skips it. If you’re scanning a raw data dump without a header row, simply add the `--csv-no-header` flag.

---

With JSONL and CSV under our belt, we’ve mastered **how** data is read. In the next post, we’ll look at **what** is being matched: **Building Your Intelligence Library**.

---
*ZetGrep is proudly sponsored by **[Toolsura](https://www.toolsura.com/)** - Your ultimate hub for security and development tools.*
