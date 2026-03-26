# Changelog

## v0.3.0 — 2026-03-26

- Three-section report: Broken, OK, Skipped — skipped URLs are visible, not hidden inside OK count
- `--skip-pattern` docs improved: explains when and why to use it (bot-hostile sites, affiliate links, local URLs)

## v0.2.0 — 2026-03-26

- HEAD → GET fallback: tries HEAD first, retries with GET on 403/405
- Global URL deduplication: same URL across multiple files checked once
- Retry once on 5xx or timeout before marking as broken
- Report shows all files containing a broken URL (not just the first)

## v0.1.1 — 2026-03-26

- Fix: skip URLs inside fenced and inline code blocks
- Fix: skip URLs containing shell variables (`$`, `{}`) or backticks

## v0.1.0 — 2026-03-26

Initial release.

- Recursive `.md` file scanning
- Concurrent HTTP/HTTPS link checking
- Plain-text report with broken link details
- SMTPS email delivery (port 465)
- Skip pattern via regex
- `--only-broken` flag
- `--output` flag for file reports
- CI-friendly exit codes (0 = healthy, 1 = broken links)
- Zero external dependencies
