# go-linkchecker

A fast, zero-dependency link checker for markdown files. Scans `.md` files recursively, checks every HTTP/HTTPS URL, and reports broken links. Built for self-hosted blogs and static sites.

## Features

- Scans all `.md` files in a directory recursively
- Concurrent HTTP checks (configurable)
- HEAD → GET fallback — handles sites that block HEAD requests
- Global URL deduplication — same URL across multiple files checked once
- Three-section report: **Broken**, **OK**, and **Skipped**
- Optional email delivery via SMTPS (port 465)
- Skip URLs by regex pattern (useful for bot-hostile or trusted domains)
- CI-friendly: exits with code `1` if broken links found
- Zero external dependencies — standard library only

## Install

```sh
go install github.com/srmdn/go-linkchecker@latest
```

Or build from source:

```sh
git clone https://github.com/srmdn/go-linkchecker.git
cd go-linkchecker
go build -o go-linkchecker .
```

## Usage

```sh
go-linkchecker [flags] <directory>
```

Scan current directory:

```sh
go-linkchecker .
```

Scan a specific blog content directory:

```sh
go-linkchecker ./content/blog
```

Only show broken links:

```sh
go-linkchecker --only-broken ./content/blog
```

Save report to file:

```sh
go-linkchecker --only-broken --output report.txt ./content/blog
```

## Skipping URLs

Use `--skip-pattern` to skip URLs you don't want checked. Skipped URLs still appear in the report under a **SKIPPED** section so you always have full visibility — they are not silently hidden.

Common reasons to skip a URL:

- **Bot-hostile sites** — some sites return HTTP 403 to all automated requests even though the page is live. They aren't broken, just blocking crawlers. Common examples: Wikipedia, OpenAI, Cloudflare community forum (`community.cloudflare.com`).
- **Affiliate or redirect links** — short links that redirect to third-party destinations you don't control. See also `--no-follow-redirects`.
- **Local/dev URLs** — `localhost`, `127.0.0.1`, staging domains.

```sh
# Skip local URLs
go-linkchecker --skip-pattern "localhost|127\.0\.0\.1" ./content/blog

# Skip known bot-hostile domains
go-linkchecker --skip-pattern "wikipedia\.org|openai\.com|community\.cloudflare\.com" ./content/blog

# Combine multiple patterns
go-linkchecker --skip-pattern "localhost|wikipedia\.org|openai\.com|yourshortlinks\.com" ./content/blog
```

If you use a URL shortener or affiliate links that redirect to bot-hostile destinations, use `--no-follow-redirects` instead. This treats any HTTP 3xx response as OK without following the chain:

```sh
go-linkchecker --no-follow-redirects ./content/blog
```

The pattern is a regular expression matched against the full URL. Dots in domain names should be escaped (`\.`).

## Report Format

The report has three sections:

```
Checked: 24 | Broken: 1 | OK: 23 | Skipped: 3
------------------------------------------------------------

BROKEN LINKS (1)

  [HTTP 404]
  https://example.com/old-page
  File: ./content/blog/my-post/index.md

------------------------------------------------------------

OK LINKS (23)

  [200] https://github.com/...
      File: ./content/blog/my-post/index.md
  ...

------------------------------------------------------------

SKIPPED LINKS (3)
(matched --skip-pattern, not checked)

  https://wikipedia.org/...
      File: ./content/blog/my-post/index.md
  ...
```

- **Broken** — checked and returned 4xx/5xx or a network error
- **OK** — checked and returned 2xx/3xx
- **Skipped** — matched `--skip-pattern`, not checked

Use `--only-broken` to hide the OK and Skipped sections (useful for email reports or CI).

## Email Reports

Pass SMTP credentials via flags or environment variables:

```sh
export LINKCHECKER_SMTP_HOST=smtp.example.com
export LINKCHECKER_SMTP_PORT=465
export LINKCHECKER_SMTP_USER=user@example.com
export LINKCHECKER_SMTP_PASS=yourpassword
export LINKCHECKER_SMTP_FROM="Link Checker <user@example.com>"
export LINKCHECKER_SMTP_TO=you@example.com

go-linkchecker --only-broken ./content/blog
```

By default, email is only sent if broken links are found (`--email-only-broken=true`). Set `--email-only-broken=false` to always send.

## All Flags

| Flag | Default | Description |
|---|---|---|
| `--timeout` | `10s` | HTTP request timeout per link |
| `--concurrency` | `5` | Concurrent link checks |
| `--only-broken` | `false` | Only show broken links in report |
| `--skip-pattern` | `` | Regex — skip matching URLs (shown as Skipped in report) |
| `--no-follow-redirects` | `false` | Treat HTTP 3xx as OK — do not follow redirects |
| `--output` | `` | Write report to file |
| `--smtp-host` | `` | SMTP host |
| `--smtp-port` | `465` | SMTP port (TLS) |
| `--smtp-user` | `` | SMTP username |
| `--smtp-pass` | `` | SMTP password |
| `--smtp-from` | `` | From address |
| `--smtp-to` | `` | Recipient address |
| `--email-only-broken` | `true` | Only email if broken links exist |

## Automating with systemd

Example weekly timer on a Linux server:

**`/etc/systemd/system/linkchecker.service`**
```ini
[Unit]
Description=Weekly link checker

[Service]
Type=oneshot
User=youruser
WorkingDirectory=/your/site/dir
EnvironmentFile=/etc/linkchecker.env
ExecStart=/usr/local/bin/go-linkchecker --only-broken --skip-pattern "localhost|wikipedia\.org" ./content/blog
StandardOutput=journal
StandardError=journal
```

**`/etc/systemd/system/linkchecker.timer`**
```ini
[Unit]
Description=Weekly link checker timer
Requires=linkchecker.service

[Timer]
OnCalendar=weekly
Persistent=true

[Install]
WantedBy=timers.target
```

```sh
systemctl enable --now linkchecker.timer
```

## Exit Codes

| Code | Meaning |
|---|---|
| `0` | All checked links healthy (skipped links do not affect exit code) |
| `1` | One or more broken links found |

## License

MIT
