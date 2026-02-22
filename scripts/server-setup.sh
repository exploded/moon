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
# 3. Create sudoers entry (least privilege)
# ---------------------------------------------------------------
SUDOERS_FILE="/etc/sudoers.d/moon-deploy"

cat > "$SUDOERS_FILE" << 'EOF'
# Allow the deploy user to install the moon app without a password
deploy ALL=(ALL) NOPASSWD: \
    /bin/cp /tmp/moon-deploy/moon /usr/local/bin/moon, \
    /bin/chmod +x /usr/local/bin/moon, \
    /bin/cp /tmp/moon-deploy/index.html /var/www/moon/, \
    /bin/cp /tmp/moon-deploy/about.html /var/www/moon/, \
    /bin/cp /tmp/moon-deploy/calendar.html /var/www/moon/, \
    /bin/cp -r /tmp/moon-deploy/static/ /var/www/moon/, \
    /bin/chown -R www-data\:www-data /var/www/moon, \
    /usr/bin/systemctl restart moon, \
    /usr/bin/systemctl is-active moon
EOF

chmod 440 "$SUDOERS_FILE"
# Validate the file
visudo -c -f "$SUDOERS_FILE"
echo "[ok] sudoers entry created at $SUDOERS_FILE"

# ---------------------------------------------------------------
# 4. Ensure /var/www/moon exists and is owned correctly
# ---------------------------------------------------------------
mkdir -p /var/www/moon
chown -R www-data:www-data /var/www/moon
echo "[ok] /var/www/moon ready"

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
