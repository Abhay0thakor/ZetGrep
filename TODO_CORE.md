# TODO: Core Engine Optimization (Algorithmic)

## Phase 1: Literals & Fast-Path
- [ ] Implement **Aho-Corasick** multi-pattern matcher for literal-only patterns.
- [ ] Implement **Bloom Filter** for rapid "negative" checks (skip content that definitely doesn't match).
- [ ] Integrate a PCRE2 or RE2 based engine for faster regex execution than Go's standard `regexp`.

## Phase 2: Vectorization & Parallelism
- [ ] Use **SIMD** instructions (where available) for byte searching.
- [ ] Optimize the concurrent worker pool to reduce channel contention.
- [ ] Implement **Work Stealing** algorithm for better load balancing across cores.

## Phase 3: Memory Efficiency
- [ ] Switch to `[]byte` based processing throughout the pipeline to avoid `string` allocations.
- [ ] Implement a **Zero-Copy** record passing mechanism.
- [ ] Audit and reduce pointers in critical paths to help GC.
