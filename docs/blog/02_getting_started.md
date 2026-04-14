# Blog 2: Getting Started like a Pro - Your First 10 Seconds with ZetGrep

Welcome to the second installment of the ZetGrep Deep-Dive Series. Today, we're moving from vision to action. We'll set up ZetGrep and run a scan that would make standard `grep` cry.

## Installation

ZetGrep is a single binary written in Go. You can install it in seconds:

```bash
go install github.com/Abhay0thakor/ZetGrep/cmd/zetgrep@latest
```

*Pro Tip: Ensure your `GOPATH/bin` is in your system `PATH`.*

## Setting Up Your Intelligence Library

ZetGrep relies on **Patterns** (Regex) and **Tools** (Scripts). To get the most out of it, you should initialize your library:

```bash
mkdir -p ~/.config/gf
git clone https://github.com/Abhay0thakor/ZetGrep.git /tmp/zetgrep
cp -r /tmp/zetgrep/library/patterns ~/.config/gf/
cp -r /tmp/zetgrep/library/tools ~/.config/gf/
```

Now, ZetGrep has the "brains" to find secrets, IPs, and more.

## Your First Power Scan

Let's say you have a list of subdomains and you want to find every IP address mentioned.

### The Basic Way:
```bash
zetgrep scan ip subdomains.txt
```

### The Professional Way (Piped & Formatted):
```bash
subfinder -d target.com -silent | zetgrep scan ip --stdin --format table
```

### Why this is better:
1. **Context**: ZetGrep doesn't just show the line; it identifies it as an `ip` match.
2. **Readability**: The `--format table` flag gives you a beautiful, structured view of your findings.
3. **No Mess**: No temp files needed—everything flows through standard pipes.

## Verification

To ensure your environment is healthy, run the built-in health check:

```bash
zetgrep list
```

If you see a list of patterns (like `aws-keys`, `firebase`, `ip`), you are ready for mission-critical recon.

In the next post, we’ll dive deep into the **Unified Parser Architecture** and see how ZetGrep handles structured data formats like JSONL and CSV.

---
*ZetGrep is proudly sponsored by **[Toolsura](https://www.toolsura.com/)** - Elevate your security workflow.*
