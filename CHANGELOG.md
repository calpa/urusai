# Changelog

All notable changes to this project will be documented in this file.

## [1.1.0] - 2026-06-09

### Features

- **Concurrent crawling** — Configurable concurrent workers (default: 3) with mutex-protected shared state ([#10], PR #21)
- **Graceful shutdown** — SIGINT/SIGTERM cancellation propagates through all goroutines via `context.Context` ([#8], PR #17)
- **SSRF protection** — Blocks loopback, RFC1918, link-local, and IPv6 ULA addresses; limits redirects to 5 ([#13], PR #20)

### Bug Fixes

- **Recursive crawl → iterative loop** — Eliminates stack overflow risk from unbounded recursion in `browseFromLinks` ([#7], PR #15)
- **Config validation** — Rejects invalid config (empty root URLs, invalid sleep range) with descriptive errors ([#9], PR #16)
- **Timeout flag override** — CLI `--timeout` default sentinel changed from `0` to `-1` to preserve config value ([#9], PR #16)
- **Response body truncation** — `Read(buf)` replaced with `io.ReadAll(io.LimitReader)` to capture full response up to 1 MB ([#12], PR #14)
- **Regex recompilation** — `regexp.MustCompile` moved to package-level vars, compiled once instead of every call ([#12], PR #14)

### Miscellaneous

- **Code quality sweep** — `math/rand/v2`, blacklist map for O(1) lookup, updated user agents (2025-era), `--version` flag with ldflags injection ([#11], PR #19)
- **Module import paths** — Fixed to use `github.com/calpa/urusai` consistently

[v1.1.0]: https://github.com/calpa/urusai/compare/v1.0.3...v1.1.0
