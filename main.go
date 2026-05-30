package main

import (
	"flag"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var (
	buildVersion = "dev"
	buildCommit  = ""
)

func main() {
	var (
		timeout           = flag.Duration("timeout", 10*time.Second, "HTTP request timeout per link")
		concurrency       = flag.Int("concurrency", 5, "Number of concurrent link checks")
		onlyBroken        = flag.Bool("only-broken", false, "Only show broken links in report")
		skipPattern       = flag.String("skip-pattern", "", "Regex pattern -- skip matching URLs")
		ignoreStatus      = flag.String("ignore-status", "", "Comma-separated HTTP status codes to treat as OK (e.g. 403,429)")
		noFollowRedirects = flag.Bool("no-follow-redirects", false, "Treat HTTP 3xx as OK -- do not follow redirects")
		output            = flag.String("output", "", "Write report to file (default: stdout only)")

		smtpHost        = flag.String("smtp-host", envOr("LINKCHECKER_SMTP_HOST", ""), "SMTP host")
		smtpPort        = flag.String("smtp-port", envOr("LINKCHECKER_SMTP_PORT", "465"), "SMTP port (TLS)")
		smtpUser        = flag.String("smtp-user", envOr("LINKCHECKER_SMTP_USER", ""), "SMTP username")
		smtpPass        = flag.String("smtp-pass", envOr("LINKCHECKER_SMTP_PASS", ""), "SMTP password")
		smtpFrom        = flag.String("smtp-from", envOr("LINKCHECKER_SMTP_FROM", ""), "Email from address")
		smtpTo          = flag.String("smtp-to", envOr("LINKCHECKER_SMTP_TO", ""), "Email recipient address")
		emailOnlyBroken = flag.Bool("email-only-broken", true, "Only send email if broken links found")
		showVersion     = flag.Bool("version", false, "Print version and exit")
	)

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: go-linkchecker [flags] <directory>\n\n")
		fmt.Fprintf(os.Stderr, "Scans all .md files in <directory> for broken HTTP/HTTPS links.\n\n")
		fmt.Fprintf(os.Stderr, "Flags:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nSMTP flags can also be set via environment variables:\n")
		fmt.Fprintf(os.Stderr, "  LINKCHECKER_SMTP_HOST, LINKCHECKER_SMTP_PORT\n")
		fmt.Fprintf(os.Stderr, "  LINKCHECKER_SMTP_USER, LINKCHECKER_SMTP_PASS\n")
		fmt.Fprintf(os.Stderr, "  LINKCHECKER_SMTP_FROM, LINKCHECKER_SMTP_TO\n")
	}

	flag.Parse()

	if *showVersion {
		fmt.Println(formatVersion())
		return
	}

	if err := validateCLIConfig(*timeout, *concurrency); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	smtpCfg, hasSMTP, err := buildSMTPConfig(*smtpHost, *smtpPort, *smtpUser, *smtpPass, *smtpFrom, *smtpTo)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	dir := flag.Arg(0)
	if dir == "" {
		dir = "."
	}

	// Validate directory
	if _, err := os.Stat(dir); err != nil {
		fmt.Fprintf(os.Stderr, "error: directory %q not found\n", dir)
		os.Exit(1)
	}

	// Compile skip pattern if provided
	var skip *regexp.Regexp
	if *skipPattern != "" {
		var err error
		skip, err = regexp.Compile(*skipPattern)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: invalid skip-pattern: %v\n", err)
			os.Exit(1)
		}
	}

	// Find markdown files
	files, err := FindMarkdownFiles(dir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	if len(files) == 0 {
		fmt.Fprintf(os.Stderr, "no .md files found in %q\n", dir)
		os.Exit(0)
	}

	fmt.Fprintf(os.Stderr, "scanning %d markdown file(s) in %s ...\n", len(files), dir)

	// Parse ignored status codes
	ignoredStatus := map[int]bool{}
	if *ignoreStatus != "" {
		for _, s := range strings.Split(*ignoreStatus, ",") {
			code, err := strconv.Atoi(strings.TrimSpace(s))
			if err != nil || code < 100 || code > 599 {
				fmt.Fprintf(os.Stderr, "error: invalid status code in --ignore-status: %q\n", s)
				os.Exit(1)
			}
			ignoredStatus[code] = true
		}
	}

	// Run checks
	cfg := CheckConfig{
		Timeout:           *timeout,
		Concurrency:       *concurrency,
		SkipPattern:       skip,
		NoFollowRedirects: *noFollowRedirects,
		IgnoreStatus:      ignoredStatus,
	}
	results := CheckLinks(files, cfg)

	// Build report
	report := FormatReport(results, *onlyBroken, dir)

	// Print to stdout
	fmt.Print(report)

	// Write to file if requested
	if *output != "" {
		if err := WriteReport(*output, report); err != nil {
			fmt.Fprintf(os.Stderr, "error writing report: %v\n", err)
		} else {
			fmt.Fprintf(os.Stderr, "report written to %s\n", *output)
		}
	}

	// Send email if SMTP is configured
	if hasSMTP {
		hasBroken := false
		for _, r := range results {
			if r.IsBroken() {
				hasBroken = true
				break
			}
		}

		if !*emailOnlyBroken || hasBroken {
			subject := "[go-linkchecker] Weekly report: all checked links healthy"
			if hasBroken {
				brokenCount := 0
				for _, r := range results {
					if r.IsBroken() {
						brokenCount++
					}
				}
				subject = fmt.Sprintf("[go-linkchecker] Weekly report: %d broken link(s)", brokenCount)
			}

			emailText := FormatEmailText(results, dir)
			emailHTML, err := FormatEmailHTML(results, dir)
			if err != nil {
				fmt.Fprintf(os.Stderr, "error rendering email: %v\n", err)
				os.Exit(1)
			}

			if err := SendEmail(smtpCfg, subject, emailText, emailHTML); err != nil {
				fmt.Fprintf(os.Stderr, "error sending email: %v\n", err)
				os.Exit(1)
			}
			fmt.Fprintf(os.Stderr, "report emailed to %s\n", *smtpTo)
		}
	}

	// Exit 1 if broken links found (useful for CI)
	for _, r := range results {
		if r.IsBroken() {
			os.Exit(1)
		}
	}
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func formatVersion() string {
	if buildCommit != "" {
		return fmt.Sprintf("go-linkchecker %s (%s)", buildVersion, buildCommit)
	}
	return fmt.Sprintf("go-linkchecker %s", buildVersion)
}

func validateCLIConfig(timeout time.Duration, concurrency int) error {
	if timeout <= 0 {
		return fmt.Errorf("invalid --timeout %q: must be greater than 0", timeout)
	}
	if concurrency <= 0 {
		return fmt.Errorf("invalid --concurrency %d: must be greater than 0", concurrency)
	}
	return nil
}

func buildSMTPConfig(host, port, user, pass, from, to string) (SMTPConfig, bool, error) {
	hasSMTP := host != "" || user != "" || pass != "" || from != "" || to != ""
	if !hasSMTP {
		return SMTPConfig{}, false, nil
	}

	if from == "" {
		from = user
	}

	var missing []string
	if host == "" {
		missing = append(missing, "--smtp-host")
	}
	if user == "" {
		missing = append(missing, "--smtp-user")
	}
	if pass == "" {
		missing = append(missing, "--smtp-pass")
	}
	if to == "" {
		missing = append(missing, "--smtp-to")
	}
	if from == "" {
		missing = append(missing, "--smtp-from")
	}
	if len(missing) > 0 {
		return SMTPConfig{}, false, fmt.Errorf("incomplete SMTP configuration: missing %s", strings.Join(missing, ", "))
	}

	if port == "" {
		port = "465"
	}

	return SMTPConfig{
		Host: host,
		Port: port,
		User: user,
		Pass: pass,
		From: from,
		To:   to,
	}, true, nil
}
