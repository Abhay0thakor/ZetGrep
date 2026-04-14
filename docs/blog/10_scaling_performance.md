# Blog 10: Scaling to 100GB+ - Performance Tuning for the Elite

When you’re dealing with reconnaissance data from a Fortune 500 company, you aren't looking at kilobytes—you're looking at hundreds of gigabytes of logs and responses. If your tool isn't optimized, it will either take days to finish or crash your server. In this post, we’ll see how ZetGrep handles the heavy lifting.

## The Worker Pool Model

ZetGrep uses a unified concurrent engine built on Go's goroutines. When you start a scan, ZetGrep spins up a pool of workers. 

1. **The Reader**: A single high-speed goroutine reads the input (file or stdin) and pushes records into a queue.
2. **The Workers**: Multiple worker goroutines pull records from the queue, perform regex matching, execute workflows, and push findings to the output.

## Tuning Concurrency (`--concurrency`)

By default, ZetGrep uses `CPU_CORES * 2` workers. However, every system is different. 

- **CPU-Bound Scans**: If you're running complex regex patterns against raw text, you are CPU-bound. Increasing workers beyond your core count won't help much.
- **I/O-Bound Scans**: If your workflow involves `curl` commands or API calls, you are I/O-bound. You can significantly increase the worker count to handle the network wait time.

```bash
# High concurrency for network-heavy workflows
zetgrep scan ip targets.txt --workflow ip_info --concurrency 50
```

## Buffer Optimization

ZetGrep uses a 100MB internal buffer for reading files. This allows it to handle extremely long lines (common in minified JavaScript or large JSON bodies) that would cause other tools to fail.

## Memory Management

Unlike `jq` or some Python scripts that load the entire file into RAM, ZetGrep is a **Streaming Engine**. It only keeps a few hundred records in memory at any given time. This means you can scan a 500GB file on a laptop with 8GB of RAM without any issues.

## Pro Tips for Maximum Speed

1. **Use `ripgrep`**: ZetGrep will automatically use `rg` if it's installed. It is significantly faster than standard `grep`.
2. **Filter at the Source**: Use the `filters` block in your input configs to discard useless data before it hits the regex engine.
3. **Use `--unique`**: Deduplication happens in memory. If you have millions of duplicate hits, `--unique` will save you massive amounts of disk I/O on the output side.

Scaling isn't just about going fast—it's about going fast *safely*. In the next post, we’ll look at **Stateful Operations** and how to manage these long-running missions without losing progress.

---
*ZetGrep is proudly sponsored by **[Toolsura](https://www.toolsura.com/)** - High-performance tools for high-performance engineers.*
