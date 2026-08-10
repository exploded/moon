package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// proxiedRequest builds a request as moon actually receives it in production:
// arriving over loopback from Caddy, with whatever proxy headers are supplied.
func proxiedRequest(headers map[string]string) *http.Request {
	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "127.0.0.1:54321"
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	return req
}

// Caddy fills X-Real-IP from {client_ip}, which it resolves using its
// trusted_proxies list. That is the value the limiter should key on.
func TestClientKeyPrefersCaddyRealIP(t *testing.T) {
	req := proxiedRequest(map[string]string{
		"X-Real-IP":       "203.0.113.7",
		"X-Forwarded-For": "203.0.113.7, 162.158.1.1",
	})
	if got, want := clientKey(req), "203.0.113.7"; got != want {
		t.Errorf("clientKey = %q, want %q", got, want)
	}
}

// This is the bug that caused the 429s: without a per-visitor key every request
// in the world lands in the same bucket.
func TestClientKeyDistinguishesVisitors(t *testing.T) {
	a := clientKey(proxiedRequest(map[string]string{"X-Real-IP": "203.0.113.7"}))
	b := clientKey(proxiedRequest(map[string]string{"X-Real-IP": "198.51.100.9"}))
	if a == b {
		t.Fatalf("two different visitors share a bucket: both keyed as %q", a)
	}
}

// A client can put anything in CF-Connecting-IP or in the leading entries of
// X-Forwarded-For -- the origin answers on :443 directly, so those values are
// not vouched for by Cloudflare. None of them may influence the key.
func TestClientKeyIgnoresForgedHeaders(t *testing.T) {
	base := clientKey(proxiedRequest(map[string]string{"X-Real-IP": "203.0.113.7"}))

	forgeries := []map[string]string{
		{"X-Real-IP": "203.0.113.7", "CF-Connecting-IP": "1.2.3.4"},
		{"X-Real-IP": "203.0.113.7", "CF-Connecting-IP": "5.6.7.8"},
		{"X-Real-IP": "203.0.113.7", "X-Forwarded-For": "1.2.3.4, 203.0.113.7"},
		{"X-Real-IP": "203.0.113.7", "X-Forwarded-For": "9.9.9.9, 8.8.8.8, 203.0.113.7"},
		{"X-Real-IP": "203.0.113.7", "True-Client-IP": "1.2.3.4"},
		{"X-Real-IP": "203.0.113.7", "X-Client-IP": "1.2.3.4"},
		{"X-Real-IP": "203.0.113.7", "Forwarded": "for=1.2.3.4"},
	}
	for _, h := range forgeries {
		if got := clientKey(proxiedRequest(h)); got != base {
			t.Errorf("forged headers %v changed the key: got %q, want %q", h, got, base)
		}
	}
}

// Without X-Real-IP the last X-Forwarded-For hop is used, because that entry is
// the one Caddy appended itself. Leading entries remain untrusted.
func TestClientKeyFallsBackToLastForwardedHop(t *testing.T) {
	req := proxiedRequest(map[string]string{
		"X-Forwarded-For": "1.2.3.4, 198.51.100.9, 162.158.1.1",
	})
	if got, want := clientKey(req), "162.158.1.1"; got != want {
		t.Errorf("clientKey = %q, want the last (Caddy-appended) hop %q", got, want)
	}
}

// If moon is ever reachable without the proxy in front, no header is
// trustworthy and only the peer address counts.
func TestClientKeyIgnoresHeadersFromNonLoopbackPeer(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "203.0.113.50:443"
	req.Header.Set("X-Real-IP", "1.2.3.4")
	req.Header.Set("X-Forwarded-For", "1.2.3.4")

	if got, want := clientKey(req), "203.0.113.50"; got != want {
		t.Errorf("clientKey = %q, want the real peer %q", got, want)
	}
}

// One IPv6 subscriber owns a whole prefix, so addresses within a /64 must share
// a bucket or the limiter is trivially sidestepped.
func TestClientKeyGroupsIPv6By64(t *testing.T) {
	a := clientKey(proxiedRequest(map[string]string{"X-Real-IP": "2001:db8:abcd:1234::1"}))
	b := clientKey(proxiedRequest(map[string]string{"X-Real-IP": "2001:db8:abcd:1234:ffff:ffff:ffff:ffff"}))
	if a != b {
		t.Errorf("addresses in one /64 got different buckets: %q vs %q", a, b)
	}

	c := clientKey(proxiedRequest(map[string]string{"X-Real-IP": "2001:db8:abcd:9999::1"}))
	if a == c {
		t.Errorf("addresses in different /64s share a bucket: %q", a)
	}
}

// Two visitors must not consume each other's allowance.
func TestRateLimitIsPerVisitor(t *testing.T) {
	rl := newRateLimiter(3, time.Minute, time.Minute)

	for i := 0; i < 3; i++ {
		if allowed, _ := rl.allow("203.0.113.7"); !allowed {
			t.Fatalf("request %d from first visitor rejected inside the limit", i+1)
		}
	}
	if allowed, _ := rl.allow("203.0.113.7"); allowed {
		t.Error("first visitor allowed past the limit")
	}
	if allowed, _ := rl.allow("198.51.100.9"); !allowed {
		t.Error("second visitor rejected because of the first visitor's traffic")
	}
}

// The security property, end to end through the middleware: rotating a forged
// header must not hand an attacker a fresh bucket each request.
func TestRateLimitForgedHeaderCannotMintBuckets(t *testing.T) {
	rl := newRateLimiter(10, time.Minute, time.Minute)
	handler := rateLimit(rl, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	var served, throttled int
	for i := 0; i < 50; i++ {
		req := proxiedRequest(map[string]string{
			"X-Real-IP": "203.0.113.7",
			// Every request claims to be someone new.
			"CF-Connecting-IP": fmt.Sprintf("10.0.0.%d", i),
			"X-Forwarded-For":  fmt.Sprintf("10.0.0.%d, 203.0.113.7", i),
		})
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)
		if rr.Code == http.StatusTooManyRequests {
			throttled++
		} else {
			served++
		}
	}

	if served != 10 {
		t.Errorf("attacker got %d requests through a limit of 10 by rotating headers", served)
	}
	if throttled != 40 {
		t.Errorf("throttled %d requests, want 40", throttled)
	}
}

// A genuine visitor is limited but never silently locked out beyond the window.
func TestRateLimitWindowResets(t *testing.T) {
	rl := newRateLimiter(2, 50*time.Millisecond, time.Minute)

	rl.allow("203.0.113.7")
	rl.allow("203.0.113.7")
	if allowed, _ := rl.allow("203.0.113.7"); allowed {
		t.Fatal("visitor allowed past the limit within the window")
	}

	time.Sleep(80 * time.Millisecond)
	if allowed, _ := rl.allow("203.0.113.7"); !allowed {
		t.Error("visitor still blocked after the window elapsed")
	}
}

// Warning volume must be bounded by time, not by traffic. This is the property
// that keeps a flood off monitor, whose logship handler is pinned at LevelWarn.
func TestRateLimitWarningsAreThrottled(t *testing.T) {
	rl := newRateLimiter(1, time.Minute, time.Hour)

	var reports []*blockReport
	for i := 0; i < 500; i++ {
		// A different visitor each time, so this also covers a distributed
		// flood rather than a single noisy client.
		key := fmt.Sprintf("203.0.113.%d", i%250)
		for j := 0; j < 3; j++ {
			if _, rep := rl.allow(key); rep != nil {
				reports = append(reports, rep)
			}
		}
	}

	if len(reports) != 1 {
		t.Fatalf("emitted %d warnings for 1500 requests, want exactly 1 within the log interval", len(reports))
	}
	if reports[0].Requests < 1 {
		t.Errorf("report carried no request count: %+v", reports[0])
	}
	if reports[0].Sample == "" {
		t.Error("report carried no sample key to investigate")
	}
}

// After the interval passes, the next rejection reports again and carries the
// aggregate of everything suppressed in between.
func TestRateLimitWarningResumesWithAggregate(t *testing.T) {
	rl := newRateLimiter(1, time.Minute, 50*time.Millisecond)

	rl.allow("203.0.113.7") // consumes the single allowance
	_, first := rl.allow("203.0.113.7")
	if first == nil {
		t.Fatal("first rejection did not report immediately")
	}

	rl.allow("198.51.100.9")
	for i := 0; i < 20; i++ {
		if _, rep := rl.allow("198.51.100.9"); rep != nil {
			t.Fatal("reported again inside the log interval")
		}
	}

	time.Sleep(80 * time.Millisecond)
	rl.allow("192.0.2.5")
	_, next := rl.allow("192.0.2.5")
	if next == nil {
		t.Fatal("no report after the log interval elapsed")
	}
	if next.Requests != 21 {
		t.Errorf("aggregate covered %d requests, want the 21 suppressed since the last report", next.Requests)
	}
	if next.Visitors != 2 {
		t.Errorf("aggregate covered %d visitors, want 2", next.Visitors)
	}
}

// Test the gettimes handler with valid parameters
func TestGettimesValid(t *testing.T) {
	req, err := http.NewRequest("GET", "/gettimes?lon=144&lat=-37&zon=10", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(gettimes)
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	var result timesResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &result); err != nil {
		t.Fatalf("handler returned invalid JSON: %v", err)
	}

	if result.Error != "" {
		t.Errorf("unexpected error field: %q", result.Error)
	}
	// Melbourne-ish coords at a reasonable zone: the moon is usually either
	// rising/setting or always-above/below. Either way, not both blank.
	hasTimes := result.Rise != "" || result.Set != ""
	if !hasTimes && !result.AlwaysAbove && !result.AlwaysBelow {
		t.Errorf("handler returned empty rise/set with no always-above/below flag")
	}
}

// Test the gettimes handler with missing parameters
func TestGettimesMissingParams(t *testing.T) {
	req, err := http.NewRequest("GET", "/gettimes?lon=144&lat=-37", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(gettimes)
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	var result timesResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &result); err != nil {
		t.Fatalf("handler returned invalid JSON: %v", err)
	}

	if result.Error == "" {
		t.Errorf("handler should return an Error field for missing parameters, got %+v", result)
	}
}

// Test the gettimes handler with invalid parameters
func TestGettimesInvalidParams(t *testing.T) {
	cases := []string{
		"/gettimes?lon=999&lat=-37&zon=10",
		"/gettimes?lon=144&lat=999&zon=10",
		"/gettimes?lon=144&lat=-37&zon=99",
		"/gettimes?lon=abc&lat=-37&zon=10",
	}
	for _, url := range cases {
		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			t.Fatal(err)
		}
		rr := httptest.NewRecorder()
		http.HandlerFunc(gettimes).ServeHTTP(rr, req)

		var result timesResponse
		if err := json.Unmarshal(rr.Body.Bytes(), &result); err != nil {
			t.Fatalf("%s: invalid JSON: %v", url, err)
		}
		if result.Error == "" {
			t.Errorf("%s: expected Error field, got %+v", url, result)
		}
	}
}

// Test security headers middleware
func TestSecurityHeaders(t *testing.T) {
	handler := securityHeaders(false, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req, err := http.NewRequest("GET", "/", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	tests := []struct {
		header string
		want   string
	}{
		{"X-Content-Type-Options", "nosniff"},
		{"X-Frame-Options", "DENY"},
		{"Referrer-Policy", "strict-origin-when-cross-origin"},
	}

	for _, tt := range tests {
		if got := rr.Header().Get(tt.header); got != tt.want {
			t.Errorf("security header %s = %v, want %v", tt.header, got, tt.want)
		}
	}

	// X-XSS-Protection should NOT be set (deprecated)
	if got := rr.Header().Get("X-XSS-Protection"); got != "" {
		t.Errorf("X-XSS-Protection should not be set, got %v", got)
	}

	// HSTS should NOT be set in non-prod mode
	if got := rr.Header().Get("Strict-Transport-Security"); got != "" {
		t.Errorf("HSTS should not be set in non-prod mode, got %v", got)
	}
}

// Test HSTS header is set in prod mode
func TestSecurityHeadersProd(t *testing.T) {
	handler := securityHeaders(true, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req, err := http.NewRequest("GET", "/", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	want := "max-age=63072000; includeSubDomains"
	if got := rr.Header().Get("Strict-Transport-Security"); got != want {
		t.Errorf("HSTS header = %v, want %v", got, want)
	}
}

// Test cache headers middleware
func TestCacheStaticAssets(t *testing.T) {
	handler := cacheStaticAssets(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req, err := http.NewRequest("GET", "/static/script.js", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	cacheControl := rr.Header().Get("Cache-Control")
	if cacheControl != "public, max-age=604800, immutable" {
		t.Errorf("Cache-Control = %v, want 'public, max-age=604800, immutable'", cacheControl)
	}
}

// Test the handleIndex function
func TestHandleIndex(t *testing.T) {
	req, err := http.NewRequest("GET", "/", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(handleIndex)
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	body := rr.Body.String()
	expectedStrings := []string{"Moon Rise and Set", "mapholder", "Latitude", "Longitude"}
	for _, expected := range expectedStrings {
		if !strings.Contains(body, expected) {
			t.Errorf("response body should contain %q", expected)
		}
	}
}

// Test that invalid URLs return 404
func TestHandleIndexNotFound(t *testing.T) {
	paths := []string{"/nonexistent", "/foo/bar", "/random-page"}
	for _, p := range paths {
		req, err := http.NewRequest("GET", p, nil)
		if err != nil {
			t.Fatal(err)
		}

		rr := httptest.NewRecorder()
		handler := http.HandlerFunc(handleIndex)
		handler.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusNotFound {
			t.Errorf("path %s: got status %v, want %v", p, status, http.StatusNotFound)
		}
		if !strings.Contains(rr.Body.String(), "404 - Page Not Found") {
			t.Errorf("path %s: response should contain 404 message", p)
		}
	}
}

// Test the about handler
func TestAbout(t *testing.T) {
	req, err := http.NewRequest("GET", "/about", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(about)
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}
}

// Test the calendar handler renders a valid month page
func TestCalendar(t *testing.T) {
	req, err := http.NewRequest("GET", "/calendar?lat=-37&lon=144&zon=10&year=2026&month=3", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	http.HandlerFunc(calendar).ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("calendar returned status %v, want 200", rr.Code)
	}

	body := rr.Body.String()
	// March 2026 has 31 rows — check the header, month name, and a few dates.
	for _, want := range []string{"March 2026", "01-03-2026", "31-03-2026", "Moon Rise", "Sun Set"} {
		if !strings.Contains(body, want) {
			t.Errorf("calendar body missing %q", want)
		}
	}
}

// Test the archive page embeds the QBASIC source and wires the copy button
func TestArchive(t *testing.T) {
	req, err := http.NewRequest("GET", "/archive", nil)
	if err != nil {
		t.Fatal(err)
	}
	rr := httptest.NewRecorder()
	http.HandlerFunc(handleArchive).ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("archive returned status %v, want 200", rr.Code)
	}
	body := rr.Body.String()
	for _, want := range []string{
		"Keith Burnett",
		"DECLARE FUNCTION hm",
		`id="copy-bas"`,
		`id="bas-code"`,
		`src="/static/archive.js"`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("archive body missing %q", want)
		}
	}
}

// Test that calendar handles invalid params by falling back to defaults
func TestCalendarInvalidParams(t *testing.T) {
	req, err := http.NewRequest("GET", "/calendar?lat=abc&lon=xyz&zon=99&year=0&month=13", nil)
	if err != nil {
		t.Fatal(err)
	}
	rr := httptest.NewRecorder()
	http.HandlerFunc(calendar).ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("calendar returned status %v, want 200 (should fall back to defaults)", rr.Code)
	}
}

// Test content-type headers for JSON endpoints
func TestContentTypeHeaders(t *testing.T) {
	tests := []struct {
		endpoint string
		handler  http.HandlerFunc
	}{
		{"/gettimes?lon=144&lat=-37&zon=10", gettimes},
	}

	for _, tt := range tests {
		req, err := http.NewRequest("GET", tt.endpoint, nil)
		if err != nil {
			t.Fatal(err)
		}

		rr := httptest.NewRecorder()
		tt.handler.ServeHTTP(rr, req)

		contentType := rr.Header().Get("Content-Type")
		if contentType != "application/json" {
			t.Errorf("endpoint %s Content-Type = %v, want 'application/json'", tt.endpoint, contentType)
		}
	}
}
