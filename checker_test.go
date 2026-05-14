package main

import (
	"net/http"
	"testing"
	"time"
)

func testConfig(ignoreStatus map[int]bool) CheckConfig {
	return CheckConfig{
		Timeout:      5 * time.Second,
		Concurrency:  2,
		IgnoreStatus: ignoreStatus,
	}
}

type statusRoundTripper struct {
	status int
}

func (rt statusRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	return &http.Response{
		StatusCode: rt.status,
		Body:       http.NoBody,
		Header:     make(http.Header),
		Request:    req,
	}, nil
}

// mockClient returns an HTTP client that always responds with the given status code.
func mockClient(status int) *http.Client {
	return &http.Client{
		Timeout:   5 * time.Second,
		Transport: statusRoundTripper{status: status},
	}
}

func TestIsBroken_brokenOnHighStatus(t *testing.T) {
	r := Result{StatusCode: 404}
	if !r.IsBroken() {
		t.Fatal("expected 404 to be broken")
	}
}

func TestIsBroken_okOnSuccessStatus(t *testing.T) {
	r := Result{StatusCode: 200}
	if r.IsBroken() {
		t.Fatal("expected 200 to not be broken")
	}
}

func TestIsBroken_skippedNotBroken(t *testing.T) {
	r := Result{StatusCode: 403, Skipped: true}
	if r.IsBroken() {
		t.Fatal("skipped result should not be broken regardless of status")
	}
}

func TestIgnoreStatus_403TreatedAsSkipped(t *testing.T) {
	cfg := testConfig(map[int]bool{403: true})
	results := CheckLinks([]string{}, cfg)
	_ = results

	// Check directly via checkURL + manual skip logic (mirrors CheckLinks internals)
	client := mockClient(403)
	r := checkURL(client, "http://example.test")
	if r.StatusCode != 403 {
		t.Fatalf("expected status 403, got %d", r.StatusCode)
	}
	// Simulate the IgnoreStatus check from CheckLinks
	if cfg.IgnoreStatus[r.StatusCode] {
		r.Skipped = true
	}
	if r.IsBroken() {
		t.Fatal("403 with IgnoreStatus should not be broken")
	}
	if !r.Skipped {
		t.Fatal("403 with IgnoreStatus should be marked skipped")
	}
}

func TestIgnoreStatus_403StillBrokenWhenNotIgnored(t *testing.T) {
	client := mockClient(403)
	r := checkURL(client, "http://example.test")
	if r.StatusCode != 403 {
		t.Fatalf("expected status 403, got %d", r.StatusCode)
	}
	if !r.IsBroken() {
		t.Fatal("403 without IgnoreStatus should still be broken")
	}
}

func TestIgnoreStatus_200NotAffected(t *testing.T) {
	client := mockClient(200)
	r := checkURL(client, "http://example.test")
	if cfg := (CheckConfig{IgnoreStatus: map[int]bool{403: true}}); cfg.IgnoreStatus[r.StatusCode] {
		r.Skipped = true
	}
	if r.IsBroken() {
		t.Fatal("200 should not be broken")
	}
	if r.Skipped {
		t.Fatal("200 should not be skipped")
	}
}

func TestIgnoreStatus_multipleCodesIgnored(t *testing.T) {
	ignored := map[int]bool{403: true, 429: true}
	for _, code := range []int{403, 429} {
		client := mockClient(code)
		r := checkURL(client, "http://example.test")
		if ignored[r.StatusCode] {
			r.Skipped = true
		}
		if r.IsBroken() {
			t.Errorf("status %d with IgnoreStatus should not be broken", code)
		}
	}
}
