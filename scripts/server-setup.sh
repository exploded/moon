#!/bin/bash
# server-setup.sh
#
# One-time setup script to prepare your Linode Debian server for automated
# deployments from GitHub Actions.
#
# Run as root or with sudo:
#   sudo bash scripts/server-setup.sh
#
# After running, follow the printed instructions to add the SSH public key
# to your GitHub repository secrets.

set -e

DEPLOY_USER="deploy"

echo "=== Moon App - Server Deployment Setup ==="
echo ""

# ---------------------------------------------------------------
# 1. Create deploy user
# ---------------------------------------------------------------
if id "$DEPLOY_USER" &>/dev/null; then
    echo "[ok] User '$DEPLOY_USER' already exists"
else
    useradd -m -s /bin/bash "$DEPLOY_USER"
    echo "[ok] Created user '$DEPLOY_USER'"
fi

# ---------------------------------------------------------------
# 2. Generate SSH key pair for GitHub Actions
# ---------------------------------------------------------------
KEY_DIR="/home/$DEPLOY_USER/.ssh"
KEY_FILE="$KEY_DIR/github_actions"

mkdir -p "$KEY_DIR"
chmod 700 "$KEY_DIR"

if [ ! -f "$KEY_FILE" ]; then
    ssh-keygen -t ed25519 -f "$KEY_FILE" -N "" -C "github-actions-moon-deploy"
    echo "[ok] Generated SSH key pair at $KEY_FILE"
else
    echo "[ok] SSH key already exists at $KEY_FILE"
fi

# Authorise the key for the deploy user
if ! grep -qF "$(cat "$KEY_FILE.pub")" "$KEY_DIR/authorized_keys" 2>/dev/null; then
    cat "$KEY_FILE.pub" >> "$KEY_DIR/authorized_keys"
    echo "[ok] Public key added to authorized_keys"
fi

chmod 600 "$KEY_DIR/authorized_keys"
chown -R "$DEPLOY_USER:$DEPLOY_USER" "$KEY_DIR"

# ---------------------------------------------------------------
# 3. Create application directory
# ---------------------------------------------------------------
APP_DIR="/var/www/moon"
if [ -d "$APP_DIR" ]; then
    echo "[ok] Application directory $APP_DIR already exists"
else
    mkdir -p "$APP_DIR"
    chown www-data:www-data "$APP_DIR"
    echo "[ok] Created application directory $APP_DIR"
fi

# ---------------------------------------------------------------
# 4. Create .env template
# ---------------------------------------------------------------
ENV_FILE="$APP_DIR/.env"
if [ -f "$ENV_FILE" ]; then
    echo "[ok] .env file already exists at $ENV_FILE (not overwriting)"
else
    cat > "$ENV_FILE" << 'ENV_TEMPLATE'
# Google Maps API Configuration
GOOGLE_MAPS_API_KEY=your_google_maps_api_key_here

# Production flag
PROD=True

# Port the server listens on (default: 8484)
PORT=8484

# Monitor portal log shipping (optional)
MONITOR_URL=
MONITOR_API_KEY=
ENV_TEMPLATE
    chown www-data:www-data "$ENV_FILE"
    chmod 600 "$ENV_FILE"
    echo "[ok] Created .env template at $ENV_FILE (edit with real values)"
fi

# ---------------------------------------------------------------
# 5. Create systemd service
# ---------------------------------------------------------------
SERVICE_FILE="/etc/systemd/system/moon.service"
cat > "$SERVICE_FILE" << 'SERVICE'
[Unit]
Description=Moon Rise and Set Times
After=network.target

[Service]
Type=simple
User=www-data
Group=www-data
WorkingDirectory=/var/www/moon
EnvironmentFile=/var/www/moon/.env
ExecStart=/var/www/moon/moon
Restart=on-failure
RestartSec=5

# Security hardening
NoNewPrivileges=true
PrivateTmp=true
ProtectSystem=full
ProtectHome=true

[Install]
WantedBy=multi-user.target
SERVICE

systemctl daemon-reload
echo "[ok] Created systemd service at $SERVICE_FILE"

# ---------------------------------------------------------------
# 6. Install the server-side deploy script (runs as root via sudo).
#    Download from the repo so this file remains the single source of truth;
#    subsequent deployments self-update via the script's own bundle-diff logic.
# ---------------------------------------------------------------
DEPLOY_SCRIPT_URL="https://raw.githubusercontent.com/exploded/moon/master/scripts/deploy-moon"
if ! curl -fsSL "$DEPLOY_SCRIPT_URL" -o /usr/local/bin/deploy-moon; then
    echo "[error] Failed to download deploy-moon from $DEPLOY_SCRIPT_URL"
    exit 1
fi
chmod +x /usr/local/bin/deploy-moon
echo "[ok] Installed /usr/local/bin/deploy-moon from $DEPLOY_SCRIPT_URL"

# ---------------------------------------------------------------
# 7. Configure sudoers — only allow the one deploy script
# ---------------------------------------------------------------
SUDOERS_FILE="/etc/sudoers.d/moon-deploy"

cat > "$SUDOERS_FILE" << 'EOF'
# Allow the deploy user to run the moon deployment script as root
deploy ALL=(ALL) NOPASSWD: /usr/local/bin/deploy-moon
# Allow stopping the moon service directly (used by the GitHub Actions workflow)
deploy ALL=(ALL) NOPASSWD: /usr/bin/systemctl stop moon
EOF

chmod 440 "$SUDOERS_FILE"
visudo -c -f "$SUDOERS_FILE"
echo "[ok] sudoers entry created at $SUDOERS_FILE"

# ---------------------------------------------------------------
# 8. Print next steps
# ---------------------------------------------------------------
echo ""
echo "=== Setup complete. Add these secrets to your GitHub repository: ==="
echo ""
echo "Go to: GitHub repo → Settings → Secrets and variables → Actions"
echo ""
echo "Secret name     : DEPLOY_HOST"
echo "Secret value    : $(hostname -I | awk '{print $1}')  (your server's public IP)"
echo ""
echo "Secret name     : DEPLOY_USER"
echo "Secret value    : $DEPLOY_USER"
echo ""
echo "Secret name     : DEPLOY_SSH_KEY"
echo "Secret value    : (paste the private key below)"
echo ""
echo "---BEGIN PRIVATE KEY (copy everything including the dashes)---"
cat "$KEY_FILE"
echo "---END PRIVATE KEY---"
echo ""
echo "Optional secret : DEPLOY_PORT  (only if SSH is not on port 22)"
echo ""
echo "After adding secrets, push to master to trigger your first deployment."
