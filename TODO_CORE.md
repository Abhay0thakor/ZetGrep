# TODO: Core Engine Optimization (Algorithmic)

## Phase 1: Literals & Fast-Path
- [x] Implement **Aho-Corasick** multi-pattern matcher for literal-only patterns. (v0.6.7)
- [x] Implement **Bloom Filter** for rapid "negative" checks. (v0.7.5)
- [x] Integrate a PCRE2 based engine (`dlclark/regexp2`). (v0.7.0)

## Phase 2: Vectorization & Parallelism
- [x] Use **SIMD**-accelerated parsing via `buger/jsonparser`. (v0.6.7)
- [x] Optimize concurrent worker pool with channel-based distribution.
- [x] Implement system-aware auto-scaling (CPU load throttling). (v0.6.9)

## Phase 3: Memory Efficiency
- [x] Switch to `[]byte` based processing throughout. (v0.6.7)
- [x] Implement a **Zero-Copy** record passing mechanism. (v0.6.7)
- [x] Implement global **Buffer Pooling** via `sync.Pool`. (v0.7.6)
