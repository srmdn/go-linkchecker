# Changelog

## v0.5.2 - 2026-05-30

- Validate `--timeout` and `--concurrency` up front so invalid values fail fast instead of producing false healthy reports
- Harden SMTP setup validation and default `--smtp-from` to the SMTP user when omitted
- Replace the misnamed `pre-commit` hook with a proper `commit-msg` hook and add AI commit cleanup documentation

## v0.3.1 - 2026-03-26

- `--only-broken` now hides only the OK section; skipped links remain visible in reports
- Fix: always show the Skipped section so intentional skips stay visible

## v0.5.1 - 2026-05-14

- Email summary cards now stack on mobile so stats stay visible in narrow inboxes
- Added a responsive HTML email layout update for smaller screens

## v0.5.0 - 2026-05-14

- Email reports now send as `multipart/alternative` with both plain-text and styled HTML bodies
- Email wording reworked to read like a weekly report instead of raw terminal output
- Email bodies now show relative file paths when links are found inside the scanned directory
- GitHub Pages landing page added with install, usage, and project links

## v0.4.0 - 2026-04-27

- `--no-follow-redirects` flag: treat HTTP 3xx as OK without following the redirect chain; fixes false positives from short links or affiliate links whose final destination blocks bots
- README: document `community.cloudflare.com` as a known bot-hostile domain alongside Wikipedia and OpenAI

## v0.3.0 - 2026-03-26

- Three-section report: Broken, OK, Skipped; skipped URLs are visible, not hidden inside OK count
- `--skip-pattern` docs improved: explains when and why to use it (bot-hostile sites, affiliate links, local URLs)

## v0.2.0 - 2026-03-26

- HEAD → GET fallback: tries HEAD first, retries with GET on 403/405
- Global URL deduplication: same URL across multiple files checked once
- Retry once on 5xx or timeout before marking as broken
- Report shows all files containing a broken URL (not just the first)

## v0.1.1 - 2026-03-26

- Fix: skip URLs inside fenced and inline code blocks
- Fix: skip URLs containing shell variables (`$`, `{}`) or backticks

## v0.1.0 - 2026-03-26

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
