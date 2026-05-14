package main

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"html/template"
	"mime/multipart"
	"net/smtp"
	"net/textproto"
	"os"
	"path/filepath"
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

type reportSections struct {
	broken  []Result
	ok      []Result
	skipped []Result
}

type emailSummary struct {
	Generated string
	Checked   int
	Broken    int
	OK        int
	Skipped   int
}

type emailItem struct {
	URL    string
	Reason string
	Files  []string
}

type emailViewData struct {
	Summary          emailSummary
	HasBroken        bool
	BrokenIntro      string
	BrokenItems      []emailItem
	SkippedItems     []emailItem
	SkippedExplainer string
}

func splitResults(results []Result) reportSections {
	var sections reportSections
	for _, r := range results {
		switch {
		case r.Skipped:
			sections.skipped = append(sections.skipped, r)
		case r.IsBroken():
			sections.broken = append(sections.broken, r)
		default:
			sections.ok = append(sections.ok, r)
		}
	}
	return sections
}

func formatReason(r Result) string {
	if r.Err != nil {
		return r.Err.Error()
	}
	if r.StatusCode != 0 {
		return fmt.Sprintf("HTTP %d", r.StatusCode)
	}
	return "Unreachable"
}

func pluralize(n int, singular, plural string) string {
	if n == 1 {
		return singular
	}
	return plural
}

func displayFile(file, dir string) string {
	if dir != "" {
		if rel, err := filepath.Rel(dir, file); err == nil && rel != "." && !strings.HasPrefix(rel, "..") {
			return filepath.ToSlash(rel)
		}
	}
	return filepath.ToSlash(file)
}

func buildEmailViewData(results []Result, dir string) emailViewData {
	sections := splitResults(results)
	summary := emailSummary{
		Generated: time.Now().Format(time.RFC1123),
		Checked:   len(sections.broken) + len(sections.ok),
		Broken:    len(sections.broken),
		OK:        len(sections.ok),
		Skipped:   len(sections.skipped),
	}

	data := emailViewData{
		Summary:          summary,
		HasBroken:        len(sections.broken) > 0,
		SkippedExplainer: "These links were intentionally skipped because they matched configured skip rules or returned an ignored status such as bot protection or authentication.",
	}

	if data.HasBroken {
		data.BrokenIntro = fmt.Sprintf("go-linkchecker found %d broken link(s) that need attention.", summary.Broken)
	} else {
		data.BrokenIntro = "No broken links found in checked sources."
	}

	for _, r := range sections.broken {
		item := emailItem{
			URL:    r.URL,
			Reason: formatReason(r),
		}
		for _, file := range r.Files {
			item.Files = append(item.Files, displayFile(file, dir))
		}
		data.BrokenItems = append(data.BrokenItems, item)
	}

	for _, r := range sections.skipped {
		item := emailItem{
			URL: r.URL,
		}
		if r.StatusCode != 0 {
			item.Reason = fmt.Sprintf("HTTP %d ignored", r.StatusCode)
		} else {
			item.Reason = "Matched skip rule"
		}
		for _, file := range r.Files {
			item.Files = append(item.Files, displayFile(file, dir))
		}
		data.SkippedItems = append(data.SkippedItems, item)
	}

	return data
}

// FormatReport builds a plain-text report from results.
func FormatReport(results []Result, onlyBroken bool, dir string) string {
	var buf bytes.Buffer

	sections := splitResults(results)
	broken := sections.broken
	ok := sections.ok
	skipped := sections.skipped

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
			reason := formatReason(r)
			fmt.Fprintf(&buf, "  [%s]\n  %s\n", reason, r.URL)
			for _, f := range r.Files {
				fmt.Fprintf(&buf, "  File: %s\n", f)
			}
			fmt.Fprintln(&buf)
		}
	}

	if !onlyBroken && len(ok) > 0 {
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
		fmt.Fprintf(&buf, "(matched --skip-pattern or --ignore-status)\n\n")
		for _, r := range skipped {
			if r.StatusCode != 0 {
				fmt.Fprintf(&buf, "  [HTTP %d] %s\n", r.StatusCode, r.URL)
			} else {
				fmt.Fprintf(&buf, "  %s\n", r.URL)
			}
			for _, f := range r.Files {
				fmt.Fprintf(&buf, "      File: %s\n", f)
			}
		}
	}

	return buf.String()
}

// WriteReport writes the report to a file.
func WriteReport(path, content string) error {
	return os.WriteFile(path, []byte(content), 0644)
}

// FormatEmailText builds a human-friendly plain-text email report.
func FormatEmailText(results []Result, dir string) string {
	var buf bytes.Buffer
	data := buildEmailViewData(results, dir)

	fmt.Fprintln(&buf, "Weekly Link Check Report")
	fmt.Fprintf(&buf, "Generated: %s\n\n", data.Summary.Generated)
	fmt.Fprintf(&buf, "%s\n\n", data.BrokenIntro)

	fmt.Fprintln(&buf, "Summary")
	fmt.Fprintf(&buf, "- %d %s checked\n", data.Summary.Checked, pluralize(data.Summary.Checked, "link", "links"))
	fmt.Fprintf(&buf, "- %d %s\n", data.Summary.Broken, pluralize(data.Summary.Broken, "broken link", "broken links"))
	fmt.Fprintf(&buf, "- %d %s\n", data.Summary.OK, pluralize(data.Summary.OK, "healthy link", "healthy links"))
	fmt.Fprintf(&buf, "- %d %s\n", data.Summary.Skipped, pluralize(data.Summary.Skipped, "skipped link", "skipped links"))

	if len(data.BrokenItems) > 0 {
		fmt.Fprintf(&buf, "\nBroken Links Requiring Attention (%d)\n\n", len(data.BrokenItems))
		for i, item := range data.BrokenItems {
			fmt.Fprintf(&buf, "%d. %s\n", i+1, item.URL)
			fmt.Fprintf(&buf, "   Status: %s\n", item.Reason)
			for _, file := range item.Files {
				fmt.Fprintf(&buf, "   Found in: %s\n", file)
			}
			fmt.Fprintln(&buf)
		}
	}

	if len(data.SkippedItems) > 0 {
		fmt.Fprintf(&buf, "\nSkipped Links (%d)\n", len(data.SkippedItems))
		fmt.Fprintf(&buf, "%s\n\n", data.SkippedExplainer)
		for _, item := range data.SkippedItems {
			fmt.Fprintf(&buf, "- %s\n", item.URL)
			fmt.Fprintf(&buf, "  Reason: %s\n", item.Reason)
			for _, file := range item.Files {
				fmt.Fprintf(&buf, "  Found in: %s\n", file)
			}
		}
	}

	fmt.Fprintln(&buf, "\nGenerated by go-linkchecker.")
	return buf.String()
}

const emailHTMLTemplate = `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>Weekly Link Check Report</title>
</head>
<body style="margin:0;padding:24px;background:#f3f6fb;font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',sans-serif;color:#10233a;">
  <div style="max-width:760px;margin:0 auto;background:#ffffff;border:1px solid #dbe5f0;border-radius:18px;overflow:hidden;">
    <div style="padding:28px 32px;background:linear-gradient(135deg,#0f2744,#174a7a);color:#ffffff;">
      <div style="font-size:12px;letter-spacing:0.12em;text-transform:uppercase;opacity:0.78;">go-linkchecker</div>
      <h1 style="margin:10px 0 8px;font-size:28px;line-height:1.2;">Weekly Link Check Report</h1>
      <p style="margin:0;font-size:15px;line-height:1.6;opacity:0.92;">{{.BrokenIntro}}</p>
      <p style="margin:14px 0 0;font-size:12px;opacity:0.72;">Generated {{.Summary.Generated}}</p>
    </div>

    <div style="padding:24px 32px 8px;">
      <table role="presentation" width="100%" cellpadding="0" cellspacing="0" style="border-collapse:separate;border-spacing:12px 12px;">
        <tr>
          <td style="width:25%;background:#f6f9fc;border:1px solid #dbe5f0;border-radius:14px;padding:16px;">
            <div style="font-size:12px;text-transform:uppercase;letter-spacing:0.08em;color:#5b7289;">Checked</div>
            <div style="margin-top:8px;font-size:28px;font-weight:700;color:#10233a;">{{.Summary.Checked}}</div>
          </td>
          <td style="width:25%;background:#fff4f2;border:1px solid #f2c1b8;border-radius:14px;padding:16px;">
            <div style="font-size:12px;text-transform:uppercase;letter-spacing:0.08em;color:#9a4738;">Broken</div>
            <div style="margin-top:8px;font-size:28px;font-weight:700;color:#bf3b23;">{{.Summary.Broken}}</div>
          </td>
          <td style="width:25%;background:#f3faf5;border:1px solid #c7e6cf;border-radius:14px;padding:16px;">
            <div style="font-size:12px;text-transform:uppercase;letter-spacing:0.08em;color:#3b6b48;">Healthy</div>
            <div style="margin-top:8px;font-size:28px;font-weight:700;color:#1f7a3d;">{{.Summary.OK}}</div>
          </td>
          <td style="width:25%;background:#f8f7fc;border:1px solid #ddd8ee;border-radius:14px;padding:16px;">
            <div style="font-size:12px;text-transform:uppercase;letter-spacing:0.08em;color:#695b8c;">Skipped</div>
            <div style="margin-top:8px;font-size:28px;font-weight:700;color:#594888;">{{.Summary.Skipped}}</div>
          </td>
        </tr>
      </table>
    </div>

    {{if .HasBroken}}
    <div style="padding:8px 32px 0;">
      <h2 style="margin:0 0 14px;font-size:18px;color:#10233a;">Broken Links Requiring Attention</h2>
      {{range .BrokenItems}}
      <div style="margin-bottom:14px;padding:18px;border:1px solid #f0c0b7;border-radius:14px;background:#fff8f7;">
        <div style="margin-bottom:8px;font-size:13px;font-weight:700;color:#bf3b23;">{{.Reason}}</div>
        <div style="margin-bottom:10px;word-break:break-word;"><a href="{{.URL}}" style="color:#174a7a;text-decoration:none;">{{.URL}}</a></div>
        {{range .Files}}
        <div style="margin-top:6px;font-size:13px;color:#4f6277;">Found in: <span style="font-family:ui-monospace,SFMono-Regular,Menlo,monospace;color:#10233a;">{{.}}</span></div>
        {{end}}
      </div>
      {{end}}
    </div>
    {{end}}

    {{if .SkippedItems}}
    <div style="padding:18px 32px 0;">
      <h2 style="margin:0 0 10px;font-size:18px;color:#10233a;">Skipped Links</h2>
      <p style="margin:0 0 14px;font-size:14px;line-height:1.6;color:#4f6277;">{{.SkippedExplainer}}</p>
      {{range .SkippedItems}}
      <div style="margin-bottom:12px;padding:16px;border:1px solid #ddd8ee;border-radius:14px;background:#faf9fd;">
        <div style="margin-bottom:7px;font-size:13px;font-weight:700;color:#594888;">{{.Reason}}</div>
        <div style="margin-bottom:8px;word-break:break-word;"><a href="{{.URL}}" style="color:#174a7a;text-decoration:none;">{{.URL}}</a></div>
        {{range .Files}}
        <div style="margin-top:6px;font-size:13px;color:#4f6277;">Found in: <span style="font-family:ui-monospace,SFMono-Regular,Menlo,monospace;color:#10233a;">{{.}}</span></div>
        {{end}}
      </div>
      {{end}}
    </div>
    {{end}}

    <div style="padding:24px 32px 30px;font-size:12px;line-height:1.6;color:#6a7f93;">
      Generated by go-linkchecker.
    </div>
  </div>
</body>
</html>`

// FormatEmailHTML builds a styled HTML email report.
func FormatEmailHTML(results []Result, dir string) (string, error) {
	data := buildEmailViewData(results, dir)
	tmpl, err := template.New("email").Parse(emailHTMLTemplate)
	if err != nil {
		return "", fmt.Errorf("parse email template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("render email template: %w", err)
	}
	return buf.String(), nil
}

func buildEmailMessage(cfg SMTPConfig, subject, plainBody, htmlBody string) (string, error) {
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)

	plainHeader := textproto.MIMEHeader{}
	plainHeader.Set("Content-Type", "text/plain; charset=utf-8")
	plainHeader.Set("Content-Transfer-Encoding", "8bit")
	plainPart, err := writer.CreatePart(plainHeader)
	if err != nil {
		return "", fmt.Errorf("create plain part: %w", err)
	}
	if _, err := plainPart.Write([]byte(plainBody)); err != nil {
		return "", fmt.Errorf("write plain part: %w", err)
	}

	htmlHeader := textproto.MIMEHeader{}
	htmlHeader.Set("Content-Type", "text/html; charset=utf-8")
	htmlHeader.Set("Content-Transfer-Encoding", "8bit")
	htmlPart, err := writer.CreatePart(htmlHeader)
	if err != nil {
		return "", fmt.Errorf("create html part: %w", err)
	}
	if _, err := htmlPart.Write([]byte(htmlBody)); err != nil {
		return "", fmt.Errorf("write html part: %w", err)
	}

	if err := writer.Close(); err != nil {
		return "", fmt.Errorf("close multipart writer: %w", err)
	}

	msg := fmt.Sprintf(
		"From: %s\r\nTo: %s\r\nSubject: %s\r\nMIME-Version: 1.0\r\nContent-Type: multipart/alternative; boundary=%q\r\n\r\n%s",
		cfg.From, cfg.To, subject, writer.Boundary(), body.String(),
	)
	return msg, nil
}

// SendEmail sends the report via SMTPS (TLS on connect, port 465).
func SendEmail(cfg SMTPConfig, subject, plainBody, htmlBody string) error {
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

	msg, err := buildEmailMessage(cfg, subject, plainBody, htmlBody)
	if err != nil {
		return err
	}
	if _, err := w.Write([]byte(msg)); err != nil {
		return fmt.Errorf("write: %w", err)
	}
	return w.Close()
}
