package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/exploded/riseset"
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

	var result riseset.RiseSet
	err = json.Unmarshal(rr.Body.Bytes(), &result)
	if err != nil {
		t.Errorf("handler returned invalid JSON: %v", err)
	}

	if result.Rise == "" || result.Set == "" {
		t.Errorf("handler returned empty rise or set times")
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

	var result riseset.RiseSet
	err = json.Unmarshal(rr.Body.Bytes(), &result)
	if err != nil {
		t.Errorf("handler returned invalid JSON: %v", err)
	}

	if result.Rise != "error" || result.Set != "error" {
		t.Errorf("handler should return error for missing parameters")
	}
}

// Test the maps key handler
func TestMapsKey(t *testing.T) {
	// Set a test API key for this test
	originalKey := os.Getenv("GOOGLE_MAPS_API_KEY")
	testKey := "test_api_key_for_testing"
	os.Setenv("GOOGLE_MAPS_API_KEY", testKey)
	defer func() {
		if originalKey != "" {
			os.Setenv("GOOGLE_MAPS_API_KEY", originalKey)
		} else {
			os.Unsetenv("GOOGLE_MAPS_API_KEY")
		}
	}()

	req, err := http.NewRequest("GET", "/api/maps-key", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(mapsKey)
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	var result map[string]string
	err = json.Unmarshal(rr.Body.Bytes(), &result)
	if err != nil {
		t.Errorf("handler returned invalid JSON: %v", err)
	}

	if key, exists := result["key"]; !exists || key != testKey {
		t.Errorf("handler should return the test API key, got %v", key)
	}
}

// Test the maps key handler with environment variable
func TestMapsKeyWithEnv(t *testing.T) {
	originalKey := os.Getenv("GOOGLE_MAPS_API_KEY")
	testKey := "test_api_key_12345"
	os.Setenv("GOOGLE_MAPS_API_KEY", testKey)
	defer os.Setenv("GOOGLE_MAPS_API_KEY", originalKey)

	req, err := http.NewRequest("GET", "/api/maps-key", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(mapsKey)
	handler.ServeHTTP(rr, req)

	var result map[string]string
	json.Unmarshal(rr.Body.Bytes(), &result)

	if result["key"] != testKey {
		t.Errorf("handler should use environment variable: got %v want %v", result["key"], testKey)
	}
}

// Test security headers middleware
func TestSecurityHeaders(t *testing.T) {
	handler := securityHeaders(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
		{"X-XSS-Protection", "1; mode=block"},
		{"Referrer-Policy", "strict-origin-when-cross-origin"},
	}

	for _, tt := range tests {
		if got := rr.Header().Get(tt.header); got != tt.want {
			t.Errorf("security header %s = %v, want %v", tt.header, got, tt.want)
		}
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

	// Check that response contains expected HTML elements
	body := rr.Body.String()
	expectedStrings := []string{"Moon Rise and Set", "mapholder", "Latitude", "Longitude"}
	for _, expected := range expectedStrings {
		if !contains(body, expected) {
			t.Errorf("response body should contain %q", expected)
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

// Test content-type headers for JSON endpoints
func TestContentTypeHeaders(t *testing.T) {
	tests := []struct {
		endpoint string
		handler  http.HandlerFunc
	}{
		{"/gettimes?lon=144&lat=-37&zon=10", gettimes},
		{"/api/maps-key", mapsKey},
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

// Helper function to check if string contains substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 || (len(s) > 0 && (s[0:len(substr)] == substr || contains(s[1:], substr))))
}
