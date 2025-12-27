#!/bin/bash
# Certbot renewal hook for Envoy
# Place this file at: /etc/letsencrypt/renewal-hooks/deploy/envoy-reload.sh
# Make executable: chmod +x /etc/letsencrypt/renewal-hooks/deploy/envoy-reload.sh
#
# Envoy will reload TLS certificates on SIGHUP (sent via systemctl reload)

set -e

echo "Certificate renewed, reloading Envoy..."

# Reload Envoy to pick up new certificates
# SIGHUP triggers a hot restart which reloads config and certs
if systemctl is-active --quiet envoy; then
    systemctl reload envoy
    echo "Envoy reloaded successfully"
else
    echo "Envoy is not running, skipping reload"
fi
