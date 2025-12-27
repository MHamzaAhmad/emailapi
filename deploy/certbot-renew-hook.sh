#!/bin/bash
# Certbot renewal hook for Envoy
# Place this file at: /etc/letsencrypt/renewal-hooks/deploy/envoy-reload.sh
# Make executable: chmod +x /etc/letsencrypt/renewal-hooks/deploy/envoy-reload.sh
#
# Envoy will reload TLS certificates on SIGHUP (sent via systemctl reload)

set -e

echo "Certificate renewed, fixing permissions..."

# Fix permissions for new private key files
DOMAIN="api.simpleemailapi.dev"
chgrp envoy /etc/letsencrypt/archive/${DOMAIN}/privkey*.pem
chmod 640 /etc/letsencrypt/archive/${DOMAIN}/privkey*.pem

echo "Reloading Envoy..."

# Reload Envoy to pick up new certificates
# SIGHUP triggers a hot restart which reloads config and certs
if systemctl is-active --quiet envoy; then
    systemctl reload envoy
    echo "Envoy reloaded successfully"
else
    echo "Envoy is not running, skipping reload"
fi
