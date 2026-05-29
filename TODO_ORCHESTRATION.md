# TODO: Advanced Orchestration & Intelligence

## Phase 1: Hardware-Aware Pausing
- [ ] Implement **CPU Temperature Monitoring** to automatically pause when thermal throttling is detected.
- [ ] Dynamic **Concurrency Scaling** (auto-adjust workers based on disk I/O wait times).
- [ ] Memory-pressure aware scan (pause if RAM usage exceeds 90%).

## Phase 2: Smart Resume & Deltas
- [x] Implement **Incremental Scanning** (only scan files that changed since last run).
- [ ] Add **Delta Reporting** (show only NEW findings compared to previous scan).
- [x] Cross-scan **Global Deduplication** using a persistent key-value store (bbolt).


## Phase 3: Reporting & UI
- [x] Add **Live Web Dashboard** SSE streaming for multiple concurrent scans.
- [ ] Implement **Templated PDF Reports** with visualization charts.
- [ ] Add **Slack/Discord/Webhook** interactive integrations.
