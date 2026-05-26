# Blog 15: Handling Compressed Data - Zstd, Gzip, and Beyond

In high-volume reconnaissance, storage is expensive. Savvy engineers compress their logs using high-performance algorithms like `zstd`. But how do you scan a 100GB compressed file without decompressing it to disk first? In this post, we’ll explore ZetGrep’s `--pre-process` flag.

## The Problem: Storage vs. Speed

Decompressing a massive file just to `grep` it wastes time and disk space. You want to stream the decompressed data directly into your regex engine.

## The ZetGrep Solution: `--pre-process`

ZetGrep allows you to specify a command that will be executed for every input file before scanning. The output of this command is then streamed directly into the ZetGrep parser.

### Scenario: Scanning Zstd-compressed logs
Imagine you have `access.log.zst`.

```bash
# Stream decompression directly into ZetGrep
zetgrep scan ip access.log.zst --pre-process "zstd -dc"
```

### How it works:
1. ZetGrep sees the file `access.log.zst`.
2. It executes `zstd -dc access.log.zst`.
3. It captures the `stdout` of that command and pipes it into the unified parser (Text, JSONL, or CSV).
4. Matches are found and reported in real-time.

## Versatility for any format

The `--pre-process` flag isn't limited to `zstd`. You can use it for any decompression tool or even custom decryption scripts:

- **Gzip**: `--pre-process "gzip -dc"`
- **Bzip2**: `--pre-process "bzip2 -dc"`
- **Custom Decryption**: `--pre-process "./my_decrypt_tool --key XXX"`

## Performance Tip

Since the decompression happens in a separate process, it doesn't block ZetGrep's worker pool. This effectively parallelizes the decompression and the pattern matching, making it incredibly fast.

By using `--pre-process`, ZetGrep allows you to maintain a compressed "Cold Storage" of your recon data while still being able to perform "Hot Scans" in seconds.

---
*ZetGrep is proudly sponsored by **[Toolsura](https://www.toolsura.com/)** - Powering your high-speed intelligence pipelines.*
