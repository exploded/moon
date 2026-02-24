# Moon Rise and Set Times

A web application that calculates and displays moon rise and set times for any location on Earth using Google Maps integration.

## Features

- Interactive Google Maps interface for location selection
- Automatic geolocation detection
- Smart timezone selector with auto-detection and 50+ timezones
- Real-time moon rise and set calculations
- Full month calendar view with sun and moon times

## Technology Stack

- **Backend**: Go 1.21+
- **Frontend**: Vanilla JavaScript, jQuery 3.7.1
- **Maps**: Google Maps JavaScript API with AdvancedMarkerElement
- **Calculations**: [riseset](https://github.com/exploded/riseset) library

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
   Use .env.example to create your own .env file

   ```env
   GOOGLE_MAPS_API_KEY=your_actual_api_key_here
   PROD=False
   PORT=8484
   ```

4. **Get a Google Maps API Key**
   - Go to [Google Cloud Console](https://console.cloud.google.com/)
   - Create a new project or select existing one
   - Enable "Maps JavaScript API"
   - Create credentials (API Key)
   - **IMPORTANT**: Restrict the API key (recommended for production):
     - **Application restrictions**: Set HTTP referrers
       - Add your domain: `https://yourdomain.com/*`
       - For local development: `http://localhost:8484/*`
     - **API restrictions**: Restrict key to "Maps JavaScript API" only
     - Set usage quotas to prevent abuse

   > **Security Note**: Google Maps JavaScript API keys are client-side visible by design. The security comes from properly configuring API key restrictions in Google Cloud Console, not from hiding the key.

### Windows Development

```cmd
build.bat
```

Or for quick restart (when already built):

```cmd
run.bat
```

### Cross-Compilation (Building for Linux from Windows)

**Build for Debian/Ubuntu (amd64):**
```cmd
set GOOS=linux
set GOARCH=amd64
go build -o moon
```

**Build for Windows (local):**
```cmd
set GOOS=windows
set GOARCH=amd64
go build -o moon.exe
```

## Testing

```bash
go test -v
go test -cover
```

## Deployment

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

## Configuration

### Environment Variables

| Variable              | Required | Default | Description                        |
|-----------------------|----------|---------|------------------------------------|
| `GOOGLE_MAPS_API_KEY` | Yes      | —       | Your Google Maps API key           |
| `PROD`                | No       | `False` | Set to `True` for production mode  |
| `PORT`                | No       | `8484`  | Port the server listens on         |

### Server Settings

- **Port**: 8484 (default)
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

## License

Copyright 2020–2026 James McHugh

Licensed under the Apache License, Version 2.0. See [LICENSE](http://www.apache.org/licenses/LICENSE-2.0).

## Credits

- Moon rise/set algorithm by [Keith Burnett](http://www.stargazing.net/kepler/moonrise.html)
- Background image: NASA/Goddard Space Flight Center Scientific Visualization
- Map integration: Google Maps JavaScript API
