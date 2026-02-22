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
# 3. Create the server-side deploy script (run as root via sudo)
# ---------------------------------------------------------------
cat > /usr/local/bin/deploy-moon << 'DEPLOY_SCRIPT'
#!/bin/bash
# /usr/local/bin/deploy-moon
# Runs as root (via sudo) during GitHub Actions deployments.

set -e

DEPLOY_SRC="${1:-/tmp/moon-deploy}"

echo "[deploy] Installing binary..."
cp "$DEPLOY_SRC/moon" /usr/local/bin/moon
chmod +x /usr/local/bin/moon

echo "[deploy] Installing service file..."
cp "$DEPLOY_SRC/moon.service" /etc/systemd/system/moon.service

echo "[deploy] Updating web assets..."
mkdir -p /var/www/moon
cp "$DEPLOY_SRC/index.html"    /var/www/moon/
cp "$DEPLOY_SRC/about.html"    /var/www/moon/
cp "$DEPLOY_SRC/calendar.html" /var/www/moon/
cp -r "$DEPLOY_SRC/static/"    /var/www/moon/
chown -R www-data:www-data /var/www/moon

# Create environment file if missing — edit it to add your API key
if [ ! -f /usr/local/bin/moon-env ]; then
    echo "[deploy] Creating empty environment file at /usr/local/bin/moon-env"
    cat > /usr/local/bin/moon-env << 'EOF'
GOOGLE_MAPS_API_KEY=REPLACE_WITH_YOUR_KEY
PROD=True
EOF
    echo "[deploy] WARNING: Edit /usr/local/bin/moon-env and set GOOGLE_MAPS_API_KEY"
fi

echo "[deploy] Reloading systemd and enabling service..."
systemctl daemon-reload
systemctl enable moon

echo "[deploy] Restarting service..."
systemctl restart moon

echo "[deploy] Verifying service is active..."
sleep 2
if ! systemctl is-active --quiet moon; then
    echo "[deploy] ERROR: Service failed to start. Status:"
    systemctl status moon --no-pager --lines=30
    exit 1
fi

echo "[deploy] Cleaning up staging directory..."
rm -rf "$DEPLOY_SRC"

echo "[deploy] Done — moon is running."
DEPLOY_SCRIPT

chmod +x /usr/local/bin/deploy-moon
echo "[ok] Created /usr/local/bin/deploy-moon"

# ---------------------------------------------------------------
# 4. Configure sudoers — only allow the one deploy script
# ---------------------------------------------------------------
SUDOERS_FILE="/etc/sudoers.d/moon-deploy"

cat > "$SUDOERS_FILE" << 'EOF'
# Allow the deploy user to run the moon deployment script as root
deploy ALL=(ALL) NOPASSWD: /usr/local/bin/deploy-moon
EOF

chmod 440 "$SUDOERS_FILE"
visudo -c -f "$SUDOERS_FILE"
echo "[ok] sudoers entry created at $SUDOERS_FILE"

# ---------------------------------------------------------------
# 5. Ensure /var/www/moon exists
# ---------------------------------------------------------------
mkdir -p /var/www/moon
chown -R www-data:www-data /var/www/moon
echo "[ok] /var/www/moon ready"

# ---------------------------------------------------------------
# 6. Remind about the environment file
# ---------------------------------------------------------------
if [ ! -f /usr/local/bin/moon-env ]; then
    cat > /usr/local/bin/moon-env << 'EOF'
GOOGLE_MAPS_API_KEY=REPLACE_WITH_YOUR_KEY
PROD=True
EOF
    echo "[ok] Created placeholder /usr/local/bin/moon-env"
fi

# ---------------------------------------------------------------
# 7. Print next steps
# ---------------------------------------------------------------
echo ""
echo "=== IMPORTANT: Edit the environment file before first deploy ==="
echo ""
echo "  sudo nano /usr/local/bin/moon-env"
echo ""
echo "  Set GOOGLE_MAPS_API_KEY to your real key."
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
