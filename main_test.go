package main

import (
	"strings"
	"testing"
	"time"
)

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
