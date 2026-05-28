# TODO: High-Performance Parsing (I/O)

## Phase 1: JSONL Mastery
- [ ] Integrate **simdjson-go** for high-speed JSON parsing.
- [ ] Implement **Parallel JSON Decoders** (split file chunks and parse in parallel).
- [ ] Add support for **mmap** (Memory Mapped Files) for faster random-access reading of huge files.

## Phase 2: CSV Excellence
- [ ] Rewrite CSV parser to use **Vectorized Column Extraction**.
- [ ] Support for **custom row-delimiters** and binary data.
- [ ] Optimize separator detection using byte-frequency analysis.

## Phase 3: Streaming & Compression
- [ ] Optimize **zstd** stream management (buffer pooling for decompressors).
- [ ] Implement **Look-ahead buffering** to minimize I/O wait times.
- [ ] Support for **Parquet/Avro** formats for even better data density in big recon dumps.
