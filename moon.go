/*
Rise and set times for Sun and Moon.

# Copyright 2020,2023 James McHugh

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package main

import (
	"context"
	"encoding/json"
	"html/template"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"syscall"
	"time"

	"github.com/exploded/monitor/pkg/logship"
	"github.com/exploded/riseset"
)

// Template cache
var templates *template.Template

// Keith Burnett's QBASIC source, mirrored alongside the archive page and
// injected into templates/archive.html for display + copy-to-clipboard.
var risetBasSource string

// Initialize templates at startup. Panic on failure: the handlers depend
// on every template being present, and a silent fallback would mask
// configuration errors in prod.
func init() {
	var err error
	templates, err = template.ParseGlob("templates/*.html")
	if err != nil {
		panic("failed to parse templates: " + err.Error())
	}

	bas, err := os.ReadFile("templates/riset.bas")
	if err != nil {
		panic("failed to read templates/riset.bas: " + err.Error())
	}
	risetBasSource = string(bas)
}

// rateLimiter tracks request counts per IP using a sliding window.
type rateLimiter struct {
	mu       sync.Mutex
	visitors map[string]*visitor
	rate     int           // max requests per window
	window   time.Duration // window duration
}

type visitor struct {
	count    int
	windowStart time.Time
}

func newRateLimiter(rate int, window time.Duration) *rateLimiter {
	rl := &rateLimiter{
		visitors: make(map[string]*visitor),
		rate:     rate,
		window:   window,
	}
	go rl.cleanup()
	return rl
}

// cleanup removes stale entries every minute.
func (rl *rateLimiter) cleanup() {
	for {
		time.Sleep(time.Minute)
		rl.mu.Lock()
		for ip, v := range rl.visitors {
			if time.Since(v.windowStart) > rl.window {
				delete(rl.visitors, ip)
			}
		}
		rl.mu.Unlock()
	}
}

// allow returns true if the request from ip should be allowed.
func (rl *rateLimiter) allow(ip string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	v, exists := rl.visitors[ip]
	now := time.Now()

	if !exists || now.Sub(v.windowStart) > rl.window {
		rl.visitors[ip] = &visitor{count: 1, windowStart: now}
		return true
	}

	v.count++
	return v.count <= rl.rate
}

// rateLimit is HTTP middleware that returns 429 when an IP exceeds the limit.
func rateLimit(limiter *rateLimiter, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip, _, err := net.SplitHostPort(r.RemoteAddr)
		if err != nil {
			ip = r.RemoteAddr
		}
		if !limiter.allow(ip) {
			slog.Warn("rate limit exceeded", "ip", ip)
			http.Error(w, "Too Many Requests", http.StatusTooManyRequests)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// Get Google Maps API key from environment variable
func getGoogleMapsKey() string {
	key := os.Getenv("GOOGLE_MAPS_API_KEY")
	if key == "" {
		slog.Warn("GOOGLE_MAPS_API_KEY not set in environment")
	}
	return key
}

// Add request logging
func requestLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		slog.Info("request", "method", r.Method, "uri", r.RequestURI, "duration", time.Since(start))
	})
}

// Add security headers to all responses
func securityHeaders(isProd bool, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		if isProd {
			w.Header().Set("Strict-Transport-Security", "max-age=63072000; includeSubDomains")
		}
		// CSP: all allowances beyond 'self' are driven by Google Maps
		// (the only remaining external origin).
		//   script-src: 'unsafe-eval' + blob: + maps.googleapis.com — required by Maps.
		//   style-src:  fonts.googleapis.com — Maps injects a Roboto/Google Sans
		//              stylesheet at runtime. 'unsafe-inline' — Maps sets inline
		//              styles on markers/controls.
		//   font-src:   fonts.gstatic.com — font files for the above stylesheet.
		//   img-src:    *.googleapis.com / *.gstatic.com — Maps tiles and icons.
		w.Header().Set("Content-Security-Policy", "default-src 'self'; script-src 'self' 'unsafe-eval' blob: https://maps.googleapis.com; style-src 'self' 'unsafe-inline' https://fonts.googleapis.com; img-src 'self' data: blob: https://*.googleapis.com https://*.gstatic.com; font-src 'self' https://fonts.gstatic.com; connect-src 'self' data: https://maps.googleapis.com https://*.gstatic.com; worker-src blob:")
		next.ServeHTTP(w, r)
	})
}

// Add caching headers for static assets
func cacheStaticAssets(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Cache static assets for 7 days
		w.Header().Set("Cache-Control", "public, max-age=604800, immutable")
		next.ServeHTTP(w, r)
	})
}

// limiter allows 60 requests per minute per IP.
var limiter = newRateLimiter(60, time.Minute)

func makeServerFromMux(mux *http.ServeMux, isProd bool) *http.Server {
	// set timeouts so that a slow or malicious client doesn't
	// hold resources forever
	return &http.Server{
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
		Handler:      requestLogger(rateLimit(limiter, securityHeaders(isProd, mux))),
	}
}

func makeHTTPServer(isProd bool) *http.Server {
	mux := &http.ServeMux{}
	mux.HandleFunc("/", handleIndex)
	mux.HandleFunc("/about", about)
	mux.HandleFunc("/gettimes", gettimes)
	mux.HandleFunc("/calendar", calendar)
	mux.HandleFunc("/archive", handleArchive)
	mux.HandleFunc("/favicon.ico", handleFavicon)
	path, _ := os.Getwd()
	slog.Info("Working directory", "path", path)
	fileServer := http.FileServer(http.Dir(path + "/static"))
	mux.Handle("/static/", http.StripPrefix("/static/", cacheStaticAssets(fileServer)))
	return makeServerFromMux(mux, isProd)
}

func main() {
	flgProduction := os.Getenv("PROD") == "True"

	httpPort := os.Getenv("PORT")
	if httpPort == "" {
		httpPort = "8484"
	}
	httpPort = ":" + httpPort

	// Set up log shipping to monitor portal
	monitorURL := os.Getenv("MONITOR_URL")
	monitorKey := os.Getenv("MONITOR_API_KEY")

	if monitorURL != "" && monitorKey != "" {
		ship := logship.New(logship.Options{
			Endpoint: monitorURL + "/api/logs",
			APIKey:   monitorKey,
			App:      "moon",
			Level:    slog.LevelWarn,
		})
		defer ship.Shutdown()

		logger := slog.New(logship.Multi(
			slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo}),
			ship,
		))
		slog.SetDefault(logger)
		slog.Warn("moon app started, log shipping active", "endpoint", monitorURL+"/api/logs")
	} else {
		slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo})))
	}

	slog.Info("Production", "enabled", flgProduction)
	slog.Info("HTTP Port", "port", httpPort)

	httpSrv := makeHTTPServer(flgProduction)
	httpSrv.Addr = httpPort

	// Start server in goroutine
	go func() {
		slog.Info("Starting HTTP server", "addr", httpSrv.Addr)
		if err := httpSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("httpSrv.ListenAndServe() failed", "error", err)
			os.Exit(1)
		}
	}()

	// Wait for interrupt signal for graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	slog.Info("Shutting down server...")

	// Give outstanding requests 5 seconds to complete
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := httpSrv.Shutdown(ctx); err != nil {
		slog.Error("Server forced to shutdown", "error", err)
	}

	slog.Info("Server exited")
}

func about(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := templates.ExecuteTemplate(w, "about.html", nil); err != nil {
		slog.Error("Error executing about template", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

func calendar(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	Lon, err := strconv.ParseFloat(r.URL.Query().Get("lon"), 64)
	if err != nil || Lon < -180 || Lon > 180 {
		Lon = 144 // Default
	}
	Lat, err := strconv.ParseFloat(r.URL.Query().Get("lat"), 64)
	if err != nil || Lat < -90 || Lat > 90 {
		Lat = -37 // Default
	}
	Zon, err := strconv.ParseFloat(r.URL.Query().Get("zon"), 64)
	if err != nil || Zon < -12 || Zon > 14 {
		Zon = 10 // Default
	}

	// Determine which month to show, defaulting to the current month in the
	// user's timezone. We shift from UTC explicitly so the result doesn't
	// depend on the server's local time zone.
	zondur := time.Hour * time.Duration(Zon)
	now := time.Now().UTC().Add(zondur)

	year, err := strconv.Atoi(r.URL.Query().Get("year"))
	if err != nil || year < 1 || year > 9999 {
		year = now.Year()
	}
	month, err := strconv.Atoi(r.URL.Query().Get("month"))
	if err != nil || month < 1 || month > 12 {
		month = int(now.Month())
	}

	// Previous / next month navigation (handles year rollovers).
	prevMonth, prevYear := month-1, year
	if prevMonth < 1 {
		prevMonth = 12
		prevYear--
	}
	nextMonth, nextYear := month+1, year
	if nextMonth > 12 {
		nextMonth = 1
		nextYear++
	}

	// Today's date string in local time, used to highlight the current row.
	today := now.Format("02-01-2006")

	// Last day of the requested month: day 0 of the following month.
	// time.Date normalizes month=13 to January of year+1, so December works
	// without a special case.
	lastDay := time.Date(year, time.Month(month+1), 0, 0, 0, 0, 0, time.UTC).Day()

	type gridrow struct {
		Date    string
		Moon    riseset.RiseSet
		Sun     riseset.RiseSet
		IsToday bool
	}

	type mypar struct {
		Rows      []gridrow
		Lon       float64
		Lat       float64
		Zon       float64
		Year      int
		Month     int
		MonthName string
		PrevYear  int
		PrevMonth int
		NextYear  int
		NextMonth int
	}

	var Passme mypar
	Passme.Lat = Lat
	Passme.Lon = Lon
	Passme.Zon = Zon
	Passme.Year = year
	Passme.Month = month
	Passme.MonthName = time.Month(month).String()
	Passme.PrevYear = prevYear
	Passme.PrevMonth = prevMonth
	Passme.NextYear = nextYear
	Passme.NextMonth = nextMonth

	for day := 1; day <= lastDay; day++ {
		d := time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.UTC)
		dateStr := d.Format("02-01-2006")
		Passme.Rows = append(Passme.Rows, gridrow{
			Date:    dateStr,
			Moon:    riseset.Riseset(riseset.Moon, d, Lon, Lat, Zon),
			Sun:     riseset.Riseset(riseset.Sun, d, Lon, Lat, Zon),
			IsToday: dateStr == today,
		})
	}

	if err := templates.ExecuteTemplate(w, "calendar.html", &Passme); err != nil {
		slog.Error("Error executing calendar template", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

func handleIndex(w http.ResponseWriter, r *http.Request) {
	// ServeMux routes all unmatched paths to "/", so treat anything other than
	// the root as a 404.
	if r.URL.Path != "/" {
		handle404(w, r)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	data := struct{ GoogleMapsKey string }{GoogleMapsKey: getGoogleMapsKey()}

	if err := templates.ExecuteTemplate(w, "index.html", data); err != nil {
		slog.Error("Error executing index template", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

// timesResponse is the JSON shape returned by /gettimes. On success the
// riseset.RiseSet fields are populated; on error, Error is set and the
// rest are zero-valued.
type timesResponse struct {
	Rise        string `json:",omitempty"`
	Set         string `json:",omitempty"`
	AlwaysAbove bool   `json:",omitempty"`
	AlwaysBelow bool   `json:",omitempty"`
	Error       string `json:",omitempty"`
}

func gettimes(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	enc := json.NewEncoder(w)

	writeErr := func(msg string) {
		_ = enc.Encode(timesResponse{Error: msg})
	}

	a := r.URL.Query().Get("lon")
	b := r.URL.Query().Get("lat")
	c := r.URL.Query().Get("zon")
	if a == "" || b == "" || c == "" {
		writeErr("missing lon, lat, or zon parameter")
		return
	}
	lon, err := strconv.ParseFloat(a, 64)
	if err != nil || lon < -180 || lon > 180 {
		writeErr("invalid lon")
		return
	}
	lat, err := strconv.ParseFloat(b, 64)
	if err != nil || lat < -90 || lat > 90 {
		writeErr("invalid lat")
		return
	}
	zon, err := strconv.ParseFloat(c, 64)
	if err != nil || zon < -12 || zon > 14 {
		writeErr("invalid zon")
		return
	}

	// Shift UTC "now" by the client's timezone offset so the date portion
	// matches the client's local wall-clock date. riseset uses only the date.
	newdate := time.Now().UTC().Add(time.Hour * time.Duration(zon))
	rs := riseset.Riseset(riseset.Moon, newdate, lon, lat, zon)
	_ = enc.Encode(timesResponse{
		Rise:        rs.Rise,
		Set:         rs.Set,
		AlwaysAbove: rs.AlwaysAbove,
		AlwaysBelow: rs.AlwaysBelow,
	})
}

func handleArchive(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	data := struct{ Code string }{Code: risetBasSource}
	if err := templates.ExecuteTemplate(w, "archive.html", data); err != nil {
		slog.Error("Error executing archive template", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

func handleFavicon(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "favicon.ico")
}

func handle404(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusNotFound)
	if err := templates.ExecuteTemplate(w, "404.html", nil); err != nil {
		slog.Error("Error executing 404 template", "error", err)
	}
}
