# go-linkchecker

A fast, zero-dependency link checker for markdown files. Scans `.md` files recursively, checks every HTTP/HTTPS URL, and reports broken links. Built for self-hosted blogs and static sites.

## Features

- Scans all `.md` files in a directory recursively
- Concurrent HTTP checks (configurable)
- Plain-text report: broken links with file location and error reason
- Optional email delivery via SMTPS (port 465)
- Skip URLs by regex pattern
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

Skip URLs matching a pattern:

```sh
go-linkchecker --skip-pattern "localhost|127\.0\.0\.1" ./content/blog
```

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
| `--skip-pattern` | `` | Regex — skip matching URLs |
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
ExecStart=/usr/local/bin/go-linkchecker --only-broken ./content/blog
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
| `0` | All links healthy (or no links found) |
| `1` | One or more broken links found |

## License

MIT
