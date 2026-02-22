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
# 3. Create the server-side deploy script (runs as root via sudo)
#
#    Reads User/Group directly from the installed service file so
#    this script never needs to hardcode a username.
# ---------------------------------------------------------------
cat > /usr/local/bin/deploy-moon << 'DEPLOY_SCRIPT'
#!/bin/bash
# /usr/local/bin/deploy-moon
# Runs as root (via sudo) during GitHub Actions deployments.

set -e

DEPLOY_SRC="${1:-/tmp/moon-deploy}"
DEPLOY_DIR=/var/www/moon

# Read the service owner from the installed unit — no hardcoded username
SERVICE_USER=$(systemctl show moon --property=User --value)
SERVICE_GROUP=$(systemctl show moon --property=Group --value)

if [ -z "$SERVICE_USER" ]; then
    echo "[deploy] ERROR: Could not read User from moon.service"
    exit 1
fi

echo "[deploy] Installing binary to $DEPLOY_DIR/moon (owner: $SERVICE_USER:$SERVICE_GROUP)..."
cp "$DEPLOY_SRC/moon" "$DEPLOY_DIR/moon"
chmod +x "$DEPLOY_DIR/moon"

echo "[deploy] Updating web assets..."
cp "$DEPLOY_SRC/index.html"    "$DEPLOY_DIR/"
cp "$DEPLOY_SRC/about.html"    "$DEPLOY_DIR/"
cp "$DEPLOY_SRC/calendar.html" "$DEPLOY_DIR/"
cp -r "$DEPLOY_SRC/static/"    "$DEPLOY_DIR/"
chown -R "$SERVICE_USER:$SERVICE_GROUP" "$DEPLOY_DIR"

echo "[deploy] Restarting service..."
systemctl restart moon

echo "[deploy] Verifying service is active..."
sleep 2
if ! systemctl is-active --quiet moon; then
    echo "[deploy] ERROR: Service failed to start. Status:"
    systemctl status moon --no-pager --lines=30
    exit 1
fi

echo "[deploy] Cleaning up..."
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
# 5. Print next steps
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
