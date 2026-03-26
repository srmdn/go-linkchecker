package main

import (
	"fmt"
	"net/http"
	"regexp"
	"sync"
	"time"
)

// Result holds the outcome of checking a single URL.
type Result struct {
	File       string
	URL        string
	StatusCode int
	Err        error
}

// IsBroken returns true if the link is broken (error or non-2xx/3xx status).
func (r Result) IsBroken() bool {
	return r.Err != nil || (r.StatusCode != 0 && r.StatusCode >= 400)
}

// CheckConfig holds configuration for the link checker.
type CheckConfig struct {
	Timeout     time.Duration
	Concurrency int
	SkipPattern *regexp.Regexp
}

// CheckLinks checks all URLs extracted from the given files concurrently.
func CheckLinks(files []string, cfg CheckConfig) []Result {
	type job struct {
		file string
		url  string
	}

	client := &http.Client{
		Timeout: cfg.Timeout,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 10 {
				return fmt.Errorf("too many redirects")
			}
			return nil
		},
	}

	jobs := make(chan job)
	results := make(chan Result)

	var wg sync.WaitGroup
	for i := 0; i < cfg.Concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := range jobs {
				results <- checkURL(client, j.file, j.url)
			}
		}()
	}

	go func() {
		wg.Wait()
		close(results)
	}()

	go func() {
		seen := make(map[string]bool)
		for _, file := range files {
			urls, err := ExtractURLs(file)
			if err != nil {
				continue
			}
			for _, u := range urls {
				if cfg.SkipPattern != nil && cfg.SkipPattern.MatchString(u) {
					continue
				}
				key := file + "|" + u
				if !seen[key] {
					seen[key] = true
					jobs <- job{file: file, url: u}
				}
			}
		}
		close(jobs)
	}()

	var all []Result
	for r := range results {
		all = append(all, r)
	}
	return all
}

func checkURL(client *http.Client, file, url string) Result {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return Result{File: file, URL: url, Err: err}
	}
	req.Header.Set("User-Agent", "go-linkchecker/1.0 (+https://github.com/srmdn/go-linkchecker)")

	resp, err := client.Do(req)
	if err != nil {
		return Result{File: file, URL: url, Err: err}
	}
	defer resp.Body.Close()

	return Result{File: file, URL: url, StatusCode: resp.StatusCode}
}
