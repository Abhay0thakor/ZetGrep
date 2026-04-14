# CLI Validation Plan (Exhaustive) - COMPLETED

This document tracks the verification of every subcommand and flag in ZetGrep.

## 1. Global Flags
- [x] `--verbose` / `-v`: Verified. Debug logs appear.
- [x] `--silent`: Verified. Only findings are printed.
- [x] `--no-color`: Verified.
- [x] `--config-file`: Verified.
- [x] `--pd`: Verified.
- [x] `--td`: Verified.

## 2. Subcommand: `version`
- [x] Run `zetgrep version`: Verified. Output: `v0.4.6`.

## 3. Subcommand: `list`
- [x] Run `zetgrep list`: Verified. Lists patterns and tools.
- [x] Run `zetgrep list --pd [dir]`: Verified.

## 4. Subcommand: `diagnose`
- [x] `--line` / `-l`: Verified.
- [x] Pattern argument: Verified.
- [x] All patterns: Verified.

## 5. Subcommand: `scan`
### General
- [x] `--all`: Verified.
- [x] `--tags`: Verified.
- [x] `--unique` / `-u`: Verified.
- [x] `--dry-run`: Verified.
- [x] `--concurrency` / `-c`: Verified.
- [x] `--resume`: Verified.
- [x] `--process`: Verified. (Processing saved JSON hits).

### Input/Output
- [x] `--l` (list-file): Verified.
- [x] `--stdin`: Verified.
- [x] `--im` (input-mode): Tested `text`, `jsonl`, `csv`. Verified.
- [x] `--format` / `-f`: Tested `text`, `json`, `table`. Verified.
- [x] `--output` / `-o`: Verified.
- [x] `--template` / `-t`: Verified.
- [x] `--report`: Verified.

### Orchestration
- [x] `--workflow` / `-w`: Verified.
- [x] `--tool`: Verified.

### Structured Data (JSONL/CSV)
- [x] `--target`: Verified.
- [x] `--targets`: Verified.
- [x] `--csv-sep`: Verified.
- [x] `--csv-targets`: Verified.
- [x] `--csv-id`: Verified.
- [x] `--csv-no-header`: Verified.

---
**Verification complete on 2026-04-14.**
Project sponsored by **[Toolsura](https://www.toolsura.com/)**.
