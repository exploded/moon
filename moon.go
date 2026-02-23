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
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/exploded/riseset"
)

// Template cache
var templates *template.Template

// Initialize templates at startup
func init() {
	var err error
	templates, err = template.ParseGlob("*.html")
	if err != nil {
		log.Printf("Warning: Error parsing templates: %v", err)
	}
}

// Get Google Maps API key from environment variable
func getGoogleMapsKey() string {
	key := os.Getenv("GOOGLE_MAPS_API_KEY")
	if key == "" {
		log.Println("WARNING: GOOGLE_MAPS_API_KEY not set in environment")
	}
	return key
}

// Add request logging
func requestLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("%s %s %s", r.Method, r.RequestURI, time.Since(start))
	})
}

// Add security headers to all responses
func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("X-XSS-Protection", "1; mode=block")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		// CSP allows Google Maps with WebAssembly and all required resources
		w.Header().Set("Content-Security-Policy", "default-src 'self'; script-src 'self' 'unsafe-inline' 'unsafe-eval' blob: https://maps.googleapis.com https://code.jquery.com; style-src 'self' 'unsafe-inline' https://fonts.googleapis.com; img-src 'self' data: blob: https://*.googleapis.com https://*.gstatic.com; font-src 'self' https://fonts.gstatic.com; connect-src 'self' data: https://maps.googleapis.com https://*.gstatic.com; worker-src blob:")
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

func makeServerFromMux(mux *http.ServeMux) *http.Server {
	// set timeouts so that a slow or malicious client doesn't
	// hold resources forever
	return &http.Server{
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 5 * time.Second,
		IdleTimeout:  120 * time.Second,
		Handler:      requestLogger(securityHeaders(mux)),
	}
}

func makeHTTPServer() *http.Server {
	mux := &http.ServeMux{}
	mux.HandleFunc("/", handleIndex)
	mux.HandleFunc("/about", about)
	mux.HandleFunc("/gettimes", gettimes)
	mux.HandleFunc("/calendar", calendar)
	mux.HandleFunc("/favicon.ico", handleFavicon)
	path, _ := os.Getwd()
	log.Printf("Working directory: %s", path)
	fileServer := http.FileServer(http.Dir(path + "/static"))
	mux.Handle("/static/", http.StripPrefix("/static/", cacheStaticAssets(fileServer)))
	// 404 handler for all other routes
	mux.HandleFunc("/404", handle404)
	return makeServerFromMux(mux)
}

func main() {
	var flgProduction bool
	if os.Getenv("PROD") == "True" {
		flgProduction = true
	} else {
		flgProduction = false
	}

	httpPort := os.Getenv("PORT")
	if httpPort == "" {
		httpPort = "8484"
	}
	httpPort = ":" + httpPort

	log.Printf("Production: %v", flgProduction)
	log.Printf("HTTP Port: %s", httpPort)

	httpSrv := makeHTTPServer()
	httpSrv.Addr = httpPort

	// Start server in goroutine
	go func() {
		log.Printf("Starting HTTP server on %s", httpSrv.Addr)
		if err := httpSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("httpSrv.ListenAndServe() failed: %v", err)
		}
	}()

	// Wait for interrupt signal for graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down server...")

	// Give outstanding requests 5 seconds to complete
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := httpSrv.Shutdown(ctx); err != nil {
		log.Printf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exited")
}

func about(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if templates != nil {
		if err := templates.ExecuteTemplate(w, "about.html", nil); err != nil {
			log.Printf("Error executing about template: %v", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		}
	} else {
		// Fallback to parsing on demand
		t, err := template.ParseFiles("about.html")
		if err != nil {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			log.Printf("Error parsing about template: %v", err)
			return
		}
		if err := t.Execute(w, nil); err != nil {
			log.Printf("Error executing about template: %v", err)
		}
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

	type gridrow struct {
		Date string
		Moon riseset.RiseSet
		Sun  riseset.RiseSet
	}

	var arow gridrow
	var newdate time.Time

	type mypar struct {
		Rows []gridrow
		Lon  float64
		Lat  float64
		Zon  float64
	}

	var Passme mypar
	Passme.Lat = Lat
	Passme.Lon = Lon
	Passme.Zon = Zon

	var zondur time.Duration = time.Hour * time.Duration(Zon)
	newdate = time.Now().Add(zondur)

	for i := 0; i < 10; i++ {
		newdate = newdate.AddDate(0, 0, 1)
		arow.Date = newdate.Format("02-01-2006")
		arow.Moon = riseset.Riseset(riseset.Moon, newdate, Lon, Lat, Zon)
		arow.Sun = riseset.Riseset(riseset.Sun, newdate, Lon, Lat, Zon)
		Passme.Rows = append(Passme.Rows, arow)
	}

	if templates != nil {
		if err := templates.ExecuteTemplate(w, "calendar.html", &Passme); err != nil {
			log.Printf("Error executing calendar template: %v", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		}
	} else {
		// Fallback to parsing on demand
		t, err := template.ParseFiles("calendar.html")
		if err != nil {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			log.Printf("Error parsing calendar template: %v", err)
			return
		}
		if err := t.Execute(w, &Passme); err != nil {
			log.Printf("Error executing calendar template: %v", err)
		}
	}
}

func handleIndex(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	
	type IndexData struct {
		GoogleMapsKey string
	}
	
	data := IndexData{
		GoogleMapsKey: getGoogleMapsKey(),
	}
	
	if templates != nil {
		if err := templates.ExecuteTemplate(w, "index.html", data); err != nil {
			log.Printf("Error executing index template: %v", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		}
	} else {
		// Fallback to parsing on demand
		t, err := template.ParseFiles("index.html")
		if err != nil {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			log.Printf("Error parsing index template: %v", err)
			return
		}
		if err := t.Execute(w, data); err != nil {
			log.Printf("Error executing index template: %v", err)
		}
	}
}

func gettimes(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	a := r.URL.Query().Get("lon")
	b := r.URL.Query().Get("lat")
	c := r.URL.Query().Get("zon")
	var mydata riseset.RiseSet
	if a == "" || b == "" || c == "" {
		mydata = riseset.RiseSet{Rise: "error", Set: "error"}
	} else {
		lon, err := strconv.ParseFloat(a, 64)
		if err != nil || lon < -180 || lon > 180 {
			mydata = riseset.RiseSet{Rise: "error", Set: "error"}
			json.NewEncoder(w).Encode(mydata)
			return
		}
		lat, err := strconv.ParseFloat(b, 64)
		if err != nil || lat < -90 || lat > 90 {
			mydata = riseset.RiseSet{Rise: "error", Set: "error"}
			json.NewEncoder(w).Encode(mydata)
			return
		}
		zon, err := strconv.ParseFloat(c, 64)
		if err != nil || zon < -12 || zon > 14 {
			mydata = riseset.RiseSet{Rise: "error", Set: "error"}
			json.NewEncoder(w).Encode(mydata)
			return
		}
		var zondur time.Duration
		var newdate time.Time
		zondur = time.Hour * time.Duration(zon)
		newdate = time.Now().Add(zondur)
		mydata = riseset.Riseset(riseset.Moon, newdate, lon, lat, zon)
	}
	json.NewEncoder(w).Encode(mydata)
}



func handleFavicon(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "favicon.ico")
}

func handle404(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusNotFound)
	fmt.Fprint(w, `<!DOCTYPE html>
<html lang="en">
<head>
	<meta charset="utf-8">
	<meta name="viewport" content="width=device-width, initial-scale=1">
	<title>404 - Page Not Found</title>
	<link rel="stylesheet" href="/static/styles.css">
</head>
<body>
	<div class="container">
		<header>
			<div class="header-row">
				<h1 class="header-title">🌙 Moon Rise and Set Times</h1>
				<div class="spacer"></div>
				<nav class="nav">
					<a class="nav-link" href="/"><i class="material-icons" style="font-size: 18px;">home</i> Home</a>
					<a class="nav-link" href="/about">About</a>
					<a class="nav-link" href="/calendar">Calendar</a>
				</nav>
			</div>
		</header>
		<main>
			<div class="page-content">
				<div class="card">
					<div class="card-content" style="text-align: center; padding: 60px 24px;">
						<h2 style="color: white; margin: 0 0 16px 0;">404 - Page Not Found</h2>
						<p style="color: rgba(255,255,255,0.9); margin: 0 0 24px 0;">The page you're looking for doesn't exist.</p>
						<a href="/" class="btn">Go to Home</a>
					</div>
				</div>
			</div>
		</main>
	</div>
</body>
</html>`)
}
