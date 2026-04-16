package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

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
