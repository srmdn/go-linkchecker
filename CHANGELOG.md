# Changelog

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
