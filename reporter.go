package main

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"net/smtp"
	"os"
	"strings"
	"time"
)

// SMTPConfig holds email delivery settings.
type SMTPConfig struct {
	Host string
	Port string
	User string
	Pass string
	From string
	To   string
}

// FormatReport builds a plain-text report from results.
func FormatReport(results []Result, onlyBroken bool, dir string) string {
	var buf bytes.Buffer

	var broken, ok, skipped []Result
	for _, r := range results {
		switch {
		case r.Skipped:
			skipped = append(skipped, r)
		case r.IsBroken():
			broken = append(broken, r)
		default:
			ok = append(ok, r)
		}
	}

	fmt.Fprintf(&buf, "go-linkchecker report\n")
	fmt.Fprintf(&buf, "Generated: %s\n", time.Now().Format(time.RFC1123))
	fmt.Fprintf(&buf, "Directory: %s\n", dir)
	fmt.Fprintf(&buf, "Checked: %d | Broken: %d | OK: %d | Skipped: %d\n",
		len(broken)+len(ok), len(broken), len(ok), len(skipped))
	fmt.Fprintf(&buf, "%s\n\n", strings.Repeat("-", 60))

	if len(broken) == 0 {
		fmt.Fprintln(&buf, "All checked links are healthy.")
	} else {
		fmt.Fprintf(&buf, "BROKEN LINKS (%d)\n\n", len(broken))
		for _, r := range broken {
			reason := ""
			if r.Err != nil {
				reason = r.Err.Error()
			} else {
				reason = fmt.Sprintf("HTTP %d", r.StatusCode)
			}
			fmt.Fprintf(&buf, "  [%s]\n  %s\n", reason, r.URL)
			for _, f := range r.Files {
				fmt.Fprintf(&buf, "  File: %s\n", f)
			}
			fmt.Fprintln(&buf)
		}
	}

	if !onlyBroken {
		if len(ok) > 0 {
			fmt.Fprintf(&buf, "%s\n\nOK LINKS (%d)\n\n", strings.Repeat("-", 60), len(ok))
			for _, r := range ok {
				fmt.Fprintf(&buf, "  [%d] %s\n", r.StatusCode, r.URL)
				for _, f := range r.Files {
					fmt.Fprintf(&buf, "      File: %s\n", f)
				}
			}
		}

		if len(skipped) > 0 {
			fmt.Fprintf(&buf, "%s\n\nSKIPPED LINKS (%d)\n", strings.Repeat("-", 60), len(skipped))
			fmt.Fprintf(&buf, "(matched --skip-pattern, not checked)\n\n")
			for _, r := range skipped {
				fmt.Fprintf(&buf, "  %s\n", r.URL)
				for _, f := range r.Files {
					fmt.Fprintf(&buf, "      File: %s\n", f)
				}
			}
		}
	}

	return buf.String()
}

// WriteReport writes the report to a file.
func WriteReport(path, content string) error {
	return os.WriteFile(path, []byte(content), 0644)
}

// SendEmail sends the report via SMTPS (TLS on connect, port 465).
func SendEmail(cfg SMTPConfig, subject, body string) error {
	tlsCfg := &tls.Config{ServerName: cfg.Host}
	conn, err := tls.Dial("tcp", cfg.Host+":"+cfg.Port, tlsCfg)
	if err != nil {
		return fmt.Errorf("dial: %w", err)
	}

	client, err := smtp.NewClient(conn, cfg.Host)
	if err != nil {
		return fmt.Errorf("smtp client: %w", err)
	}
	defer client.Close()

	auth := smtp.PlainAuth("", cfg.User, cfg.Pass, cfg.Host)
	if err := client.Auth(auth); err != nil {
		return fmt.Errorf("auth: %w", err)
	}

	if err := client.Mail(cfg.From); err != nil {
		return fmt.Errorf("MAIL FROM: %w", err)
	}
	if err := client.Rcpt(cfg.To); err != nil {
		return fmt.Errorf("RCPT TO: %w", err)
	}

	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("DATA: %w", err)
	}

	msg := fmt.Sprintf(
		"From: %s\r\nTo: %s\r\nSubject: %s\r\nMIME-Version: 1.0\r\nContent-Type: text/plain; charset=utf-8\r\n\r\n%s",
		cfg.From, cfg.To, subject, body,
	)
	if _, err := w.Write([]byte(msg)); err != nil {
		return fmt.Errorf("write: %w", err)
	}
	return w.Close()
}
