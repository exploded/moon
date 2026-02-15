# 🌙 Moon Rise and Set Times

A web application that calculates and displays moon rise and set times for any location on Earth using Google Maps integration.

## Features

- 🗺️ Interactive Google Maps interface for location selection
- 🌍 Automatic geolocation detection
- 🕐 Smart timezone selector with auto-detection and 50+ timezones
- 🌙 Real-time moon rise and set calculations
- 📅 10-day calendar view with sun and moon times
- 📱 Fully responsive design for mobile and desktop
- 🔒 Security headers and CSP protection
- ⚡ Static asset caching for performance
- ♿ ARIA accessibility features

## Technology Stack

- **Backend**: Go 1.21+
- **Frontend**: Vanilla JavaScript, jQuery 3.7.1
- **Maps**: Google Maps JavaScript API with AdvancedMarkerElement
- **Calculations**: [riseset](https://github.com/exploded/riseset) library

## Prerequisites

- Go 1.21 or higher
- Google Maps API key with Maps JavaScript API enabled
- Git (for cloning the repository)

## Installation

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
   
   Create a `.env` file (use `.env.example` as template):
   ```bash
   cp .env.example .env
   ```
   
   Edit `.env` and add your Google Maps API key:
   ```env
   GOOGLE_MAPS_API_KEY=your_actual_api_key_here
   PROD=False
   ```

4. **Get a Google Maps API Key**
   - Go to [Google Cloud Console](https://console.cloud.google.com/)
   - Create a new project or select existing one
   - Enable "Maps JavaScript API"
   - Create credentials (API Key)
   - **IMPORTANT**: Restrict the API key (recommended for production):
     - **Application restrictions**: Set HTTP referrers
       - Add your domain: `https://yourdomain.com/*`
       - For local development: `http://localhost:8181/*`
     - **API restrictions**: Restrict key to "Maps JavaScript API" only
     - Set usage quotas to prevent abuse
   
   > **Security Note**: Google Maps JavaScript API keys are client-side visible by design. The security comes from properly configuring API key restrictions in Google Cloud Console, not from hiding the key.

## Running the Application

### Development

```bash
# Load environment variables and run
go run moon.go
```

The server will start on `http://localhost:8181`

### Windows Development

Use the provided batch files:

1. **First time setup:**
   ```cmd
   build.bat
   ```
   This will:
   - Build the application
   - Create `.env` from `.env.example` if missing
   - Prompt you to add your API key

2. **Edit `.env`** and replace the placeholder with your actual API key

3. **Run the application:**
   ```cmd
   build.bat
   ```
   or for quick restart (when already built):
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

**Build for Debian/Ubuntu (arm64 - Raspberry Pi, etc):**
```cmd
set GOOS=linux
set GOARCH=arm64
go build -o moon
```

**Build for Windows (local):**
```cmd
set GOOS=windows
set GOARCH=amd64
go build -o moon.exe
```

Or use PowerShell:
```powershell
$env:GOOS="linux"; $env:GOARCH="amd64"; go build -o moon
```

Then upload the `moon` binary to your Debian server.

### Production

```bash
# Build the binary
go build -o moon

# Set environment variables
export GOOGLE_MAPS_API_KEY="your_key_here"
export PROD="True"

# Run the server
./moon
```

### Using systemd (Linux)

1. Build and install the binary:
   ```bash
   # Build for Linux
   go build -o moon
   
   # Copy to system binary location
   sudo cp moon /usr/local/bin/moon
   sudo chmod +x /usr/local/bin/moon
   ```

2. Create environment file:
   ```bash
   sudo nano /usr/local/bin/moon-env
   ```
   Add:
   ```
   GOOGLE_MAPS_API_KEY=your_key_here
   PROD=True
   ```

3. Copy application files to working directory:
   ```bash
   sudo mkdir -p /var/www/moon
   sudo cp *.html /var/www/moon/
   sudo cp -r static /var/www/moon/
   sudo chown -R www-data:www-data /var/www/moon
   ```

4. Install and enable the service:
   ```bash
   sudo cp moon.service /etc/systemd/system/
   sudo systemctl daemon-reload
   sudo systemctl enable moon
   sudo systemctl start moon
   sudo systemctl status moon
   ```

### Debian/Ubuntu Production Deployment

#### 1. Deploy Application

```bash
# Create application directory
sudo mkdir -p /var/www/moon
sudo chown $USER:$USER /var/www/moon

# Clone or copy your application
cd /var/www/moon
# ... copy your files here ...

# Install dependencies
go mod download

# Build the application
go build -o moon

# Make executable
chmod +x moon
```

#### 2. Build and Install Binary

```bash
# Build for Linux (if cross-compiling from Windows, see Cross-Compilation section)
cd /var/www/moon
go build -o moon

# Install binary to system location
sudo cp moon /usr/local/bin/moon
sudo chmod +x /usr/local/bin/moon
```

#### 3. Create Environment File

```bash
sudo nano /usr/local/bin/moon-env
```

Add:
```env
GOOGLE_MAPS_API_KEY=your_actual_api_key_here
PROD=True
```

```bash
sudo chmod 600 /usr/local/bin/moon-env
```

#### 4. Install and Start systemd Service

```bash
# Copy service file
sudo cp moon.service /etc/systemd/system/

# Set proper ownership for working directory
sudo chown -R www-data:www-data /var/www/moon

# Reload systemd
sudo systemctl daemon-reload

# Enable and start
sudo systemctl enable moon
sudo systemctl start moon

# Check status
sudo systemctl status moon

# View logs
sudo journalctl -u moon -f
```

#### 5. Configure Nginx Reverse Proxy (Optional)

Install Nginx:
```bash
sudo apt install nginx -y
```

Create site configuration:
```bash
sudo nano /etc/nginx/sites-available/moon
```

```nginx
server {
    listen 80;
    server_name your-domain.com;

    location / {
        proxy_pass http://localhost:8181;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection 'upgrade';
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        proxy_cache_bypass $http_upgrade;
    }
}
```

Enable site:
```bash
sudo ln -s /etc/nginx/sites-available/moon /etc/nginx/sites-enabled/
sudo nginx -t
sudo systemctl restart nginx
```

#### 6. Setup SSL with Let's Encrypt (Optional)

```bash
# Install certbot
sudo apt install certbot python3-certbot-nginx -y

# Obtain certificate
sudo certbot --nginx -d your-domain.com

# Auto-renewal is configured automatically
# Test renewal:
sudo certbot renew --dry-run
```

#### 7. Firewall Configuration

```bash
# If using UFW
sudo ufw allow 'Nginx Full'
sudo ufw allow OpenSSH
sudo ufw enable
sudo ufw status
```

## Testing

Run the test suite:
```bash
go test -v
```

Run with coverage:
```bash
go test -cover
```

## Project Structure

```
moon/
├── moon.go              # Main server application
├── moon_test.go         # Unit tests
├── index.html           # Home page
├── about.html           # About page
├── calendar.html        # Calendar view
├── static/
│   ├── styles.css       # Global styles
│   ├── script.js        # Client-side JavaScript
│   └── moon.jpg         # Background image
├── go.mod               # Go module dependencies
├── .env.example         # Environment variables template
├── .gitignore           # Git ignore rules
├── moon.service         # systemd service file
└── README.md            # This file
```

## API Endpoints

- `GET /` - Home page with map interface
- `GET /about` - About page
- `GET /calendar` - Calendar view (query params: lat, lon, zon)
- `GET /gettimes` - JSON API for moon times (query params: lat, lon, zon)
- `GET /static/*` - Static assets (cached for 7 days)

## Configuration

### Environment Variables

- `GOOGLE_MAPS_API_KEY` - Your Google Maps API key (required)
- `PROD` - Set to "True" for production mode (default: False)

### Server Settings

- **Port**: 8181 (default)
- **Read Timeout**: 5 seconds
- **Write Timeout**: 5 seconds
- **Idle Timeout**: 120 seconds
- **Shutdown Grace Period**: 5 seconds

## Security Features

- X-Content-Type-Options: nosniff
- X-Frame-Options: DENY
- X-XSS-Protection: 1; mode=block
- Referrer-Policy: strict-origin-when-cross-origin
- Content-Security-Policy with strict resource restrictions
- Input validation for latitude, longitude, and timezone
- Graceful shutdown on SIGTERM/SIGINT
- API key passed via server-side template rendering (not exposed via endpoint)
- Encourages Google Maps API key restrictions (HTTP referrers, API restrictions, quotas)

## Browser Support

- Chrome/Edge (latest)
- Firefox (latest)
- Safari (latest)
- Mobile browsers (iOS Safari, Chrome Mobile)

## Known Limitations

- Calendar view shows 10 days from current date
- Requires JavaScript enabled
- Geolocation requires HTTPS in production

## License

Copyright 2020, 2023, 2026 James McHugh

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.

## Credits

- Moon rise/set algorithm by [Keith Burnett](http://www.stargazing.net/kepler/moonrise.html)
- Background image: NASA/Goddard Space Flight Center Scientific Visualization
- Map integration: Google Maps JavaScript API
