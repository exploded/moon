# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Build & Run

```bash
# Windows local dev (build + run, loads .env automatically):
build.bat

# Quick restart (skip rebuild):
run.bat

# Build only:
go build -o moon.exe

# Linux production binary (required for Linode deploy — must be static):
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o moon .
```

Requires a `.env` file with `GOOGLE_MAPS_API_KEY`. See `.env.example`.

## Tests

```bash
go test -v ./...
```

Tests use `httptest` against handlers directly. Templates must be present at `templates/*.html` relative to the working directory (parsed via `template.ParseGlob("templates/*.html")` at init).

## Deployment

Push to `master` triggers GitHub Actions: runs tests, builds static Linux binary, SCPs to Linode, runs `deploy-moon` script. No manual deploy steps needed.

The deploy script (`scripts/deploy-moon`) stops the service, replaces the binary (must `rm -f` first to avoid "text file busy"), copies web assets, restarts. It has self-update logic.

## Architecture

Single-file Go server (`moon.go`) with no framework. All handlers, middleware, and `main()` live in one file.

**Request flow:** `main()` → `makeHTTPServer()` → `http.ServeMux` with middleware chain: `requestLogger(securityHeaders(mux))`. Static assets get an additional `cacheStaticAssets` wrapper.

**Routes:**
- `/` — home page with Google Maps, geolocation, moon rise/set display
- `/about` — static about page
- `/calendar` — full-month table of sun/moon rise/set times; server-rendered with `year`/`month`/`lat`/`lon`/`zon` query params
- `/gettimes` — JSON API returning `riseset.RiseSet` for given `lon`/`lat`/`zon`
- `/archive` — local mirror of Keith Burnett's (now-offline) moonrise algorithm page, with the original QBASIC source listing embedded inline in a collapsible code block
- `/static/` — CSS, JS, background image

**Templates:** Go `html/template` files under `templates/` (`index.html`, `about.html`, `calendar.html`, `404.html`, `archive.html`). Parsed once at init via `template.ParseGlob("templates/*.html")`; init panics on parse failure. `templates/riset.bas` is Keith Burnett's QBASIC source — not a template; read at init into a package var and injected into `archive.html` as `{{.Code}}`. Google Maps API key is injected server-side into `index.html` (used only in the Maps script URL, not exposed via a JS global).

**Key dependency:** `github.com/exploded/riseset` — calculates rise/set times. Pinned to a pseudo-version commit hash in `go.mod`. Update with `go get github.com/exploded/riseset@<commit>`.

**Frontend:** Vanilla JS (no jQuery). `static/script.js` handles Google Maps (AdvancedMarkerElement), geolocation, timezone auto-detection, and `fetch` calls to `/gettimes`. The `updateCalLink()` function keeps the calendar link in sync with current lat/lon/zon. Event handlers are attached via `addEventListener` in a `DOMContentLoaded` block (no inline `on*` attributes, so CSP forbids `'unsafe-inline'` for `script-src`).

**riseset API caveat:** Always check `AlwaysAbove`/`AlwaysBelow` before displaying `Rise`/`Set` values. Rise/Set are empty strings when the moon never rises or never sets.
