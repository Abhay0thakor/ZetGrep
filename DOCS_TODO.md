# ZetGrep Documentation Roadmap

## 1. Getting Started
- [x] **Installation Guide** - Go, Binary, and Makefile methods.
- [x] **Basic Usage** - Scanning files and folders.
- [x] **Smart Swapping** - How ZetGrep handles argument order automatically.

## 2. Advanced Engines
- [x] **Bloom Filter Fast-Path** - Accelerating scans by skipping non-matching data.
- [x] **Aho-Corasick Matcher** - Multi-pattern literal optimization.
- [x] **PCRE2 Integration** - Using advanced regex patterns.

## 3. System Orchestration
- [x] **Thermal Watchdog** - Protecting hardware from overheating.
- [x] **RAM Pressure Management** - Automatic pausing during high memory usage.
- [x] **Hardware Auto-Scaling** - Dynamic concurrency based on CPU load.

## 4. Stateful Operations
- [x] **Incremental Scanning** - Skipping unchanged files.
- [x] **Global Deduplication** - Cross-scan result uniqueness via bbolt.
- [x] **Stateful Resume** - Picking up where you left off.

## 5. Output & Integration
- [x] **Intelligent Routing** - Decoupling Terminal UI from Data Streams.
- [x] **Multi-Stream JSON/Text** - Saving multiple formats at once.
- [x] **Zstd Compression** - On-the-fly output compression.
- [x] **Interactive Webhooks** - Slack and Discord rich notifications.

## 6. Visualization
- [x] **Live Web Dashboard** - Real-time match streaming via SSE.
- [x] **Professional HTML Reports** - Analytics and Charts with Chart.js.

---
**ZetGrep v0.8.0 is officially feature-complete.**
Project sponsored by **[Toolsura](https://www.toolsura.com/)**.
Verified and Hardened on 2026-05-30.
