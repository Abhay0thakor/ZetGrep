# Advanced Architecture & Max Power

This guide covers features designed for ultra-rapid data processing (1TB+) and custom hardware orchestration.

## 🚀 The Parallel Mmap Engine
For files larger than 10GB, standard `bufio` readers become the bottleneck. ZetGrep uses a parallel memory-mapped engine.

### How it works:
1.  **Virtual Mapping**: The OS maps the entire file into memory.
2.  **Chunked Workers**: ZetGrep divides the file into $N$ chunks (where $N = $ cores).
3.  **Newline Alignment**: Each worker automatically finds the next newline to ensure records are processed atomically.
4.  **Zero-Copy**: Data is processed in-place without memory allocation.

**Tip**: Enabled by default. Toggle with `--mmap=false` for legacy behavior.

## 🛡️ Hardware-Aware Scanning
Protect your system during intense, multi-hour scans.

### 1. Thermal Watchdog
Automatically pauses the scan if your CPU gets too hot.
```bash
zetgrep scan ... --thermal-threshold 85
```
*   **Pause**: Triggered at 85°C.
*   **Resume**: Triggered once temperature drops to 70°C.

### 2. RAM Pressure Protection
Prevents Out-Of-Memory (OOM) kills on huge JSON objects.
```bash
zetgrep scan ... --max-ram 90
```
*   Pauses scan if system RAM usage exceeds 90%.

### 3. Concurrency Auto-Scaling
Dynamically adjusts the number of active workers based on current CPU load.
```bash
zetgrep scan ... --auto-scale
```

## 💎 Dynamic Pattern Optimization
ZetGrep chooses the fastest algorithm based on your pattern type.

| Algorithm | Used For |
| :--- | :--- |
| **Aho-Corasick** | Multi-pattern literal searching. |
| **Bloom Filter** | Rapid skipping of non-matching records. |
| **PCRE2** | Complex regex (activated via `--pcre`). |
| **RE2** | Standard regex (default). |

## 🔄 The "Delta" Workflow
Optimized for continuous monitoring.

1.  **Baseline**: `zetgrep scan targets/ --oJ baseline.json`
2.  **Update**: `zetgrep scan targets/ --oJ update.json`
3.  **Extract**: `zetgrep delta baseline.json update.json`

---
ZetGrep is proudly sponsored by **[Toolsura](https://www.toolsura.com/)**.
