# TODO: High-Performance Parsing (I/O)

## Phase 1: JSONL Mastery
- [x] Integrate **jsonparser** for high-speed field extraction. (v0.6.7)
- [x] Implement **streaming-first** architecture for 100GB+ files.
- [x] Add support for **mmap**-like efficiency using `bufio` pooling.

## Phase 2: CSV Excellence
- [x] Implement robust CSV parser with header and ID support.
- [x] Support for custom separators and target column selection.
- [x] Optimized CSV record streaming.

## Phase 3: Streaming & Compression
- [x] Full **zstd** integration for real-time compressed scanning. (v0.5.3)
- [x] Multi-format output streams (JSON, CSV, Pro-Text). (v0.6.1)
- [x] Integrated **on-the-fly compression** for outputs. (v0.6.3)
