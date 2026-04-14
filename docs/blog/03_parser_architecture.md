# Blog 3: The Unified Parser Architecture - Efficiency by Design

Welcome back. In this post, we're lifting the hood of ZetGrep to understand the engine that powers it: the **Unified Parser Architecture**.

## The Challenge: Heterogeneous Data

In a typical recon session, your data looks like a messy jigsaw puzzle:
- `subdomains.txt`: Raw text lines.
- `httpx_results.jsonl`: Complex, nested JSON objects.
- `inventory.csv`: Comma-separated spreadsheet data.

Most tools force you to use different scripts for each format. ZetGrep uses a unified interface to treat them all identically once they enter the scanning pipeline.

## The `Parser` Interface

At the core of ZetGrep's Go source code is the `Parser` interface:

```go
type Parser interface {
    GetRecords(ctx context.Context, reader io.Reader, path string) (<-chan ScanRecord, error)
}
```

This interface is the "Translator." It takes a raw data stream and converts it into a stream of **ScanRecords**. 

### How it works:
1. **Source Agnostic**: Whether it's a file on disk or data piped through `stdin`, the parser doesn't care.
2. **Concurrent Streaming**: Records are sent through a Go channel. This allows the scanner to start matching patterns as soon as the first record is parsed, without waiting for the whole file to load.
3. **Format Detection**: ZetGrep automatically detects the format based on the file extension (`.jsonl`, `.csv`, `.txt`) or you can override it using the `--im` (Input Mode) flag.

## The Three Horsemen

ZetGrep currently implements three specialized parsers:

1. **TextParser**: The classic. One line = one record.
2. **JSONLParser**: The power-lifter. It unmarshals JSON on the fly and can target specific fields (or the whole object).
3. **CSVParser**: The surgeon. It respects separators and can target specific columns by index.

## Why This Matters for You

Because of this architecture, **pattern matching logic is decoupled from data format**. You can write a single regex for an AWS key and run it against a text file, a JSON dump, and a CSV export with zero changes to the pattern itself.

```bash
# Same pattern, three different worlds:
zetgrep scan aws-keys raw_text.txt
zetgrep scan aws-keys structured.jsonl --im jsonl --target response.body
zetgrep scan aws-keys dump.csv --im csv --csv-targets 2
```

In the next installment, we’ll take a deep dive into **Mastering JSONL**, exploring how to use dot notation to hunt for secrets in the most complex nested structures.

---
*ZetGrep is proudly sponsored by **[Toolsura](https://www.toolsura.com/)** - Your ultimate hub for security and development tools.*
