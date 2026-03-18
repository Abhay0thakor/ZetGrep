# ZetGrep (v0.0.1)

A high-performance pattern matching and orchestration wrapper designed for security auditors, bug hunters, and data engineers. 

`ZetGrep` simplifies the use of complex regex patterns and allows you to pipe matches into custom tools for automated analysis. It is optimized for both standard file systems and massive (100GB+) JSONL datasets.

## ✨ Key Features
- **🚀 Turbo JSONL Engine**: Stream massive datasets with multi-core concurrency and low memory overhead.
- **🛠️ Plugin Orchestration**: Pipe regex matches directly into external commands (whois, nmap, decoders).
- **💎 Max Power Templating**: Full control over output format with custom variables.
- **📦 Multi-Config Support**: Merge multiple YAML/JSON configurations on the fly.
- **⚡ Engine Auto-Detection**: Uses `ripgrep` (rg) if available, falling back to optimized Go regex or `grep`.
- **🖥️ Intelligence Dashboard**: Built-in Web UI for real-time visualization of findings.
- **🔄 Self-Update**: Keep your tool current with a single command.

## 📥 Installation

```bash
go install github.com/Abhay0thakor/ZetGrep/cmd/zetgrep@latest
```

### 🔄 Updating
To update `ZetGrep` to the latest version:
```bash
zetgrep -update
```

## 📖 Documentation
Detailed guides for every feature:

1.  [**Basic Usage**](docs/USAGE.md) - Flags, filtering, and patterns.
2.  [**Advanced Configuration**](docs/ADVANCED.md) - JSONL streaming, templating, and "Max Power" features.
3.  [**Custom Plugins**](docs/PLUGINS.md) - How to build your own tools.
4.  [**AI Prompt Templates**](docs/PROMPTS.md) - Use AI to generate complex configs.
5.  [**Real-World Examples**](docs/EXAMPLES.md) - Scenarios and safe sample data.

## 🏁 Quick Start
Scan for IP addresses in a JSONL file and format the output:
```bash
./zetgrep -input-config httpx.yaml -o "IP FOUND: {{match}} [{{file}}]" ip targets.jsonl
```

## 📜 Acknowledgements
Original concept by [tomnomnom](https://github.com/tomnomnom/gf). This fork (**ZetGrep**) adds the high-performance streaming engine, plugin orchestration, self-update mechanism, and advanced templating system.
