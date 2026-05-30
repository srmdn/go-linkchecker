package main

import (
	"strings"
	"testing"
	"time"
)

func TestFormatVersion(t *testing.T) {
	originalVersion := buildVersion
	originalCommit := buildCommit
	t.Cleanup(func() {
		buildVersion = originalVersion
		buildCommit = originalCommit
	})

	t.Run("version only", func(t *testing.T) {
		buildVersion = "v0.5.3"
		buildCommit = ""

		if got := formatVersion(); got != "go-linkchecker v0.5.3" {
			t.Fatalf("unexpected version output: %q", got)
		}
	})

	t.Run("version with commit", func(t *testing.T) {
		buildVersion = "v0.5.3"
		buildCommit = "abc1234"

		if got := formatVersion(); got != "go-linkchecker v0.5.3 (abc1234)" {
			t.Fatalf("unexpected version output: %q", got)
		}
	})
}

func TestValidateCLIConfig(t *testing.T) {
	t.Run("valid config", func(t *testing.T) {
		if err := validateCLIConfig(10*time.Second, 5); err != nil {
			t.Fatalf("validateCLIConfig returned error for valid config: %v", err)
		}
	})

	t.Run("rejects zero timeout", func(t *testing.T) {
		err := validateCLIConfig(0, 5)
		if err == nil {
			t.Fatal("expected error for zero timeout")
		}
		if !strings.Contains(err.Error(), "--timeout") {
			t.Fatalf("expected timeout error, got %v", err)
		}
	})

	t.Run("rejects zero concurrency", func(t *testing.T) {
		err := validateCLIConfig(10*time.Second, 0)
		if err == nil {
			t.Fatal("expected error for zero concurrency")
		}
		if !strings.Contains(err.Error(), "--concurrency") {
			t.Fatalf("expected concurrency error, got %v", err)
		}
	})
}

func TestBuildSMTPConfig(t *testing.T) {
	t.Run("no smtp configured", func(t *testing.T) {
		cfg, hasSMTP, err := buildSMTPConfig("", "465", "", "", "", "")
		if err != nil {
			t.Fatalf("buildSMTPConfig returned error: %v", err)
		}
		if hasSMTP {
			t.Fatal("expected SMTP to be disabled when no config is provided")
		}
		if cfg != (SMTPConfig{}) {
			t.Fatalf("expected empty config, got %#v", cfg)
		}
	})

	t.Run("defaults from to smtp user", func(t *testing.T) {
		cfg, hasSMTP, err := buildSMTPConfig("smtp.example.com", "465", "bot@example.com", "secret", "", "you@example.com")
		if err != nil {
			t.Fatalf("buildSMTPConfig returned error: %v", err)
		}
		if !hasSMTP {
			t.Fatal("expected SMTP to be enabled")
		}
		if cfg.From != "bot@example.com" {
			t.Fatalf("expected From to default to smtp user, got %q", cfg.From)
		}
	})

	t.Run("rejects partial smtp config", func(t *testing.T) {
		_, hasSMTP, err := buildSMTPConfig("", "465", "bot@example.com", "secret", "", "you@example.com")
		if err == nil {
			t.Fatal("expected error for incomplete SMTP config")
		}
		if hasSMTP {
			t.Fatal("expected SMTP to remain disabled on invalid config")
		}
		if !strings.Contains(err.Error(), "--smtp-host") {
			t.Fatalf("expected missing host error, got %v", err)
		}
	})
}
