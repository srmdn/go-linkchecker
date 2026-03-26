package main

import (
	"fmt"
	"net/http"
	"regexp"
	"sort"
	"sync"
	"time"
)

// Result holds the outcome of checking a single URL.
// Files lists every markdown file that contains this URL.
type Result struct {
	URL        string
	Files      []string
	StatusCode int
	Err        error
	Skipped    bool // true if matched --skip-pattern
}

// IsBroken returns true if the link is broken (error or non-2xx/3xx status).
func (r Result) IsBroken() bool {
	return !r.Skipped && (r.Err != nil || (r.StatusCode != 0 && r.StatusCode >= 400))
}

// CheckConfig holds configuration for the link checker.
type CheckConfig struct {
	Timeout     time.Duration
	Concurrency int
	SkipPattern *regexp.Regexp
}

// CheckLinks checks all unique URLs extracted from the given files concurrently.
// Each URL is checked once regardless of how many files contain it.
// Skipped URLs (matched by SkipPattern) are included in results with Skipped=true.
func CheckLinks(files []string, cfg CheckConfig) []Result {
	// Build global URL → files map (deduplication across all files)
	urlFiles := make(map[string][]string)
	var skippedURLs []string

	for _, file := range files {
		urls, err := ExtractURLs(file)
		if err != nil {
			continue
		}
		for _, u := range urls {
			if cfg.SkipPattern != nil && cfg.SkipPattern.MatchString(u) {
				// Track skipped URLs with their files too
				urlFiles[u] = appendUnique(urlFiles[u], file)
				if !contains(skippedURLs, u) {
					skippedURLs = append(skippedURLs, u)
				}
				continue
			}
			urlFiles[u] = appendUnique(urlFiles[u], file)
		}
	}

	// Separate URLs to check vs skip
	var toCheck []string
	skippedSet := make(map[string]bool)
	for _, u := range skippedURLs {
		skippedSet[u] = true
	}
	for u := range urlFiles {
		if !skippedSet[u] {
			toCheck = append(toCheck, u)
		}
	}
	sort.Strings(toCheck)
	sort.Strings(skippedURLs)

	client := &http.Client{
		Timeout: cfg.Timeout,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 10 {
				return fmt.Errorf("too many redirects")
			}
			return nil
		},
	}

	jobs := make(chan string, len(toCheck))
	results := make(chan Result, len(toCheck))

	var wg sync.WaitGroup
	for i := 0; i < cfg.Concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for u := range jobs {
				r := checkURL(client, u)
				r.Files = urlFiles[u]
				results <- r
			}
		}()
	}

	for _, u := range toCheck {
		jobs <- u
	}
	close(jobs)

	wg.Wait()
	close(results)

	var all []Result
	for r := range results {
		all = append(all, r)
	}

	// Append skipped results
	for _, u := range skippedURLs {
		all = append(all, Result{
			URL:     u,
			Files:   urlFiles[u],
			Skipped: true,
		})
	}

	return all
}

// checkURL tries HEAD first; falls back to GET on 403/405 or request error.
// Retries once on 5xx or timeout before marking as broken.
func checkURL(client *http.Client, url string) Result {
	r := doRequest(client, http.MethodHead, url)

	// Fall back to GET if HEAD is blocked or not allowed
	if r.Err != nil || r.StatusCode == 403 || r.StatusCode == 405 {
		r = doRequest(client, http.MethodGet, url)
	}

	// Retry once on transient server errors or timeouts
	if r.Err != nil || r.StatusCode >= 500 {
		time.Sleep(2 * time.Second)
		retry := doRequest(client, http.MethodGet, url)
		if !retry.IsBroken() || (retry.StatusCode > 0 && retry.StatusCode < r.StatusCode) {
			r = retry
		}
	}

	return r
}

func doRequest(client *http.Client, method, url string) Result {
	req, err := http.NewRequest(method, url, nil)
	if err != nil {
		return Result{URL: url, Err: err}
	}
	req.Header.Set("User-Agent", "go-linkchecker/1.0 (+https://github.com/srmdn/go-linkchecker)")

	resp, err := client.Do(req)
	if err != nil {
		return Result{URL: url, Err: err}
	}
	defer resp.Body.Close()

	return Result{URL: url, StatusCode: resp.StatusCode}
}

func appendUnique(slice []string, s string) []string {
	for _, v := range slice {
		if v == s {
			return slice
		}
	}
	return append(slice, s)
}

func contains(slice []string, s string) bool {
	for _, v := range slice {
		if v == s {
			return true
		}
	}
	return false
}
