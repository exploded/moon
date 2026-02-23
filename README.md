# Moon Rise and Set Times

A web application that calculates and displays moon rise and set times for any location on Earth using Google Maps integration.

## Features

- Interactive Google Maps interface for location selection
- Automatic geolocation detection
- Smart timezone selector with auto-detection and 50+ timezones
- Real-time moon rise and set calculations
- 10-day calendar view with sun and moon times
- Fully responsive design for mobile and desktop
- Security headers and CSP protection
- Static asset caching for performance
- ARIA accessibility features

## Technology Stack

- **Backend**: Go 1.21+
- **Frontend**: Vanilla JavaScript, jQuery 3.7.1
- **Maps**: Google Maps JavaScript API with AdvancedMarkerElement
- **Calculations**: [riseset](https://github.com/exploded/riseset) library

## Prerequisites

- Go 1.21 or higher
- Google Maps API key with Maps JavaScript API enabled

## Local Development

1. **Clone the repository**
   ```bash
   git clone https://github.com/exploded/moon.git
   cd moon
   ```

2. **Install dependencies**
   ```bash
   go mod download
   ```

3. **Set up environment variables**
   ```bash
   cp .env.example .env
   ```
   Edit `.env` and set your values:
   ```env
   GOOGLE_MAPS_API_KEY=your_actual_api_key_here
   PROD=False
   PORT=8484
   ```

4. **Run the server**
   ```bash
   go run moon.go
   ```
   The server starts on `http://localhost:8484`

### Google Maps API Key

- Go to [Google Cloud Console](https://console.cloud.google.com/)
- Enable "Maps JavaScript API" and create an API key
- Restrict the key to HTTP referrers (`http://localhost:8484/*` for dev, your domain for prod) and to "Maps JavaScript API" only

> **Note**: Google Maps JavaScript API keys are client-side visible by design. Security comes from configuring key restrictions in Google Cloud Console.

## CI/CD Deployment (GitHub Actions → Linode)

Every push to `master` automatically runs tests, builds a Linux binary, and deploys it to your Linode server via SSH.

### How it works

1. GitHub Actions runs `go test`
2. If tests pass, it cross-compiles a Linux amd64 binary
3. It SCPs the binary and web assets to the server
4. It SSHs in, installs the files, and restarts the systemd service

### One-time server setup

Run the provided setup script **once** on your Linode server:

```bash
sudo bash scripts/server-setup.sh
```

This creates a `deploy` user with SSH keys and the minimal sudoers permissions needed for deployment, then prints the secrets to add to GitHub.

### GitHub repository secrets

**Settings → Secrets and variables → Actions → New repository secret**

| Secret name      | Value                                       |
|------------------|---------------------------------------------|
| `DEPLOY_HOST`    | Your Linode public IP or hostname           |
| `DEPLOY_USER`    | `deploy`                                    |
| `DEPLOY_SSH_KEY` | The private key printed by the setup script |
| `DEPLOY_PORT`    | Your SSH port (only if not port 22)         |

### Triggering a deployment

```bash
git push origin master
```

View deployment logs at `https://github.com/exploded/moon/actions`

## Testing

```bash
go test -v
go test -cover
```

## Project Structure

```
moon/
├── moon.go                          # Main server application
├── moon_test.go                     # Unit tests
├── index.html                       # Home page
├── about.html                       # About page
├── calendar.html                    # Calendar view
├── static/
│   ├── styles.css                   # Global styles
│   ├── script.js                    # Client-side JavaScript
│   └── moon.jpg                     # Background image
├── .github/
│   └── workflows/
│       └── deploy.yml               # GitHub Actions CI/CD pipeline
├── scripts/
│   └── server-setup.sh              # One-time Linode server setup
├── go.mod                           # Go module dependencies
├── .env.example                     # Environment variables template
├── .gitignore                       # Git ignore rules
├── moon.service                     # systemd service file
└── README.md                        # This file
```

## API Endpoints

- `GET /` - Home page with map interface
- `GET /about` - About page
- `GET /calendar` - Calendar view (query params: lat, lon, zon)
- `GET /gettimes` - JSON API for moon times (query params: lat, lon, zon)
- `GET /static/*` - Static assets (cached 7 days)

## Configuration

### Environment Variables

| Variable              | Required | Default | Description                        |
|-----------------------|----------|---------|------------------------------------|
| `GOOGLE_MAPS_API_KEY` | Yes      | —       | Your Google Maps API key           |
| `PROD`                | No       | `False` | Set to `True` for production mode  |
| `PORT`                | No       | `8484`  | Port the server listens on         |

### Server Settings

- **Read Timeout**: 5 seconds
- **Write Timeout**: 5 seconds
- **Idle Timeout**: 120 seconds
- **Shutdown Grace Period**: 5 seconds

## Security Features

- X-Content-Type-Options, X-Frame-Options, X-XSS-Protection headers
- Referrer-Policy: strict-origin-when-cross-origin
- Content-Security-Policy with strict resource restrictions
- Input validation for latitude, longitude, and timezone
- Graceful shutdown on SIGTERM/SIGINT
- API key injected via server-side template rendering (not exposed via endpoint)

## Browser Support

Chrome/Edge, Firefox, Safari (all latest), iOS Safari, Chrome Mobile.

## Known Limitations

- Calendar view shows 10 days from current date
- Requires JavaScript enabled
- Geolocation requires HTTPS in production

## License

Copyright 2020, 2023, 2026 James McHugh

Licensed under the Apache License, Version 2.0. See [LICENSE](http://www.apache.org/licenses/LICENSE-2.0).

## Credits

- Moon rise/set algorithm by [Keith Burnett](http://www.stargazing.net/kepler/moonrise.html)
- Background image: NASA/Goddard Space Flight Center Scientific Visualization
- Map integration: Google Maps JavaScript API
