# Installation Guide

ZetGrep is written in Go and designed to run on Linux, Windows, and macOS.

## 1. Quick Install (Go)
Requires Go 1.25 or higher.

```bash
go install github.com/Abhay0thakor/ZetGrep/cmd/zetgrep@latest
```

## 2. Build from Source
```bash
# Clone the repository
git clone https://github.com/Abhay0thakor/ZetGrep.git
cd ZetGrep

# Build the binary
make build

# Move to path (optional)
mv zetgrep /usr/local/bin/
```

## 3. Post-Installation (Setup Library)
ZetGrep is most powerful when used with the pattern library.

```bash
# Create config directory
mkdir -p ~/.config/gf

# Clone/Copy patterns (e.g. from the library folder)
cp -r library/patterns ~/.config/gf/
cp -r library/tools ~/.config/gf/
```

---

## 🏗️ Docker (Coming Soon)
A high-performance containerized version of ZetGrep is under development.

---
ZetGrep is proudly sponsored by **[Toolsura](https://www.toolsura.com/)**.
