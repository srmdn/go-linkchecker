package main

import (
	"strings"
	"testing"
)

func TestFormatEmailText_NoBrokenUsesFriendlySummary(t *testing.T) {
	results := []Result{
		{
			URL:        "https://example.com",
			StatusCode: 200,
			Files:      []string{"/tmp/site/content/blog/post/index.md"},
		},
		{
			URL:        "https://blocked.example.com",
			StatusCode: 403,
			Skipped:    true,
			Files:      []string{"/tmp/site/content/blog/post/index.md"},
		},
	}

	body := FormatEmailText(results, "/tmp/site/content/blog")

	if !strings.Contains(body, "Weekly Link Check Report") {
		t.Fatal("expected friendly email heading")
	}
	if !strings.Contains(body, "No broken links found in checked sources.") {
		t.Fatal("expected all-clear summary")
	}
	if !strings.Contains(body, "- 1 link checked") {
		t.Fatal("expected checked count summary")
	}
	if !strings.Contains(body, "Found in: post/index.md") {
		t.Fatal("expected relative file path in email body")
	}
}

func TestFormatEmailHTML_BrokenSectionAndRelativePaths(t *testing.T) {
	results := []Result{
		{
			URL:        "https://example.com/missing",
			StatusCode: 404,
			Files:      []string{"/tmp/site/content/blog/post/index.md"},
		},
	}

	body, err := FormatEmailHTML(results, "/tmp/site/content/blog")
	if err != nil {
		t.Fatalf("FormatEmailHTML returned error: %v", err)
	}

	if !strings.Contains(body, "Broken Links Requiring Attention") {
		t.Fatal("expected broken-links section heading")
	}
	if !strings.Contains(body, "HTTP 404") {
		t.Fatal("expected HTTP status label")
	}
	if !strings.Contains(body, "post/index.md") {
		t.Fatal("expected relative file path in HTML body")
	}
}

func TestBuildEmailMessage_UsesMultipartAlternative(t *testing.T) {
	cfg := SMTPConfig{
		From: "Link Checker <bot@example.com>",
		To:   "you@example.com",
	}

	msg, err := buildEmailMessage(cfg, "subject", "plain body", "<strong>html body</strong>")
	if err != nil {
		t.Fatalf("buildEmailMessage returned error: %v", err)
	}

	if !strings.Contains(msg, "Content-Type: multipart/alternative;") {
		t.Fatal("expected multipart content type")
	}
	if !strings.Contains(msg, "Content-Type: text/plain; charset=utf-8") {
		t.Fatal("expected plain-text part")
	}
	if !strings.Contains(msg, "Content-Type: text/html; charset=utf-8") {
		t.Fatal("expected html part")
	}
	if !strings.Contains(msg, "plain body") || !strings.Contains(msg, "<strong>html body</strong>") {
		t.Fatal("expected both plain and html bodies in message")
	}
}
