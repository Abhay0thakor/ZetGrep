# Installation Guide

ZetGrep is written in Go and can be installed on any platform that supports the Go runtime.

## Prerequisites

- [Go](https://golang.org/doc/install) (v1.21 or later recommended)
- `ripgrep` (Optional, but highly recommended for performance)
- `grep` (Fallback engine)

## Installation Methods

### 1. Using `go install` (Recommended)
The easiest way to install ZetGrep is using the `go install` command:

```bash
go install github.com/Abhay0thakor/ZetGrep/cmd/zetgrep@latest
```

Ensure your `GOPATH/bin` directory is in your system's `PATH`.

### 2. Building from Source
If you want to build the binary manually:

```bash
# Clone the repository
git clone https://github.com/Abhay0thakor/ZetGrep.git
cd ZetGrep

# Build using Makefile
make build

# Move the binary to your path
sudo mv zetgrep /usr/local/bin/
```

### 3. Docker (Coming Soon)
A Dockerfile will be provided in future releases for containerized environments.

## Post-Installation

### Verify Installation
Check if ZetGrep is installed correctly by running:

```bash
zetgrep version
```

### Setup Patterns and Tools
ZetGrep looks for patterns and tools in the following locations by default:
1. `~/.config/gf/patterns` and `~/.config/gf/tools`
2. `./patterns` and `./tools` (relative to current directory)

You can copy the provided library to your config directory:
```bash
mkdir -p ~/.config/gf
cp -r library/patterns ~/.config/gf/
cp -r library/tools ~/.config/gf/
```

---
ZetGrep is proudly sponsored by **[Toolsura](https://www.toolsura.com/)**.
