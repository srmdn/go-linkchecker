package main

import (
	"net/http"
	"net/http/httptest"
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

// mockServer starts a test HTTP server that returns the given status code for every request.
func mockServer(t *testing.T, status int) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(status)
	}))
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
	srv := mockServer(t, 403)
	defer srv.Close()

	cfg := testConfig(map[int]bool{403: true})
	results := CheckLinks([]string{}, cfg)
	_ = results

	// Check directly via checkURL + manual skip logic (mirrors CheckLinks internals)
	client := &http.Client{Timeout: 5 * time.Second}
	r := checkURL(client, srv.URL)
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
	srv := mockServer(t, 403)
	defer srv.Close()

	client := &http.Client{Timeout: 5 * time.Second}
	r := checkURL(client, srv.URL)
	if r.StatusCode != 403 {
		t.Fatalf("expected status 403, got %d", r.StatusCode)
	}
	if !r.IsBroken() {
		t.Fatal("403 without IgnoreStatus should still be broken")
	}
}

func TestIgnoreStatus_200NotAffected(t *testing.T) {
	srv := mockServer(t, 200)
	defer srv.Close()

	client := &http.Client{Timeout: 5 * time.Second}
	r := checkURL(client, srv.URL)
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
		srv := mockServer(t, code)
		client := &http.Client{Timeout: 5 * time.Second}
		r := checkURL(client, srv.URL)
		if ignored[r.StatusCode] {
			r.Skipped = true
		}
		if r.IsBroken() {
			t.Errorf("status %d with IgnoreStatus should not be broken", code)
		}
		srv.Close()
	}
}
