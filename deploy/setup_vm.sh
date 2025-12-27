#!/bin/bash
set -e

# Update system
echo "Updating system..."
sudo apt-get update && sudo apt-get upgrade -y

# Install essential tools
echo "Installing essential tools..."
sudo apt-get install -y git make build-essential curl vim

# Install Redis
echo "Installing Redis..."
sudo apt-get install -y redis-server
sudo systemctl enable redis-server
sudo systemctl start redis-server

# Install Envoy Proxy (works on Ubuntu and Debian including Bookworm)
echo "Installing Envoy..."
ENVOY_VERSION="1.36.4"

# Download Envoy static binary directly from GitHub releases
# This method works on any Linux distro (Ubuntu, Debian Bookworm, etc.)
wget -q "https://github.com/envoyproxy/envoy/releases/download/v${ENVOY_VERSION}/envoy-${ENVOY_VERSION}-linux-x86_64" -O /tmp/envoy
chmod +x /tmp/envoy
sudo mv /tmp/envoy /usr/local/bin/envoy

# Verify installation
envoy --version

# Create envoy user and directories
sudo useradd --system --no-create-home --shell /bin/false envoy || true
sudo mkdir -p /etc/envoy /var/log/envoy
sudo chown envoy:envoy /var/log/envoy

# Install Certbot (standalone mode, no nginx plugin needed)
echo "Installing Certbot..."
sudo apt-get install -y certbot

# Install Go (latest stable)
echo "Installing Go..."
GO_VERSION="1.25.5"
wget "https://go.dev/dl/go${GO_VERSION}.linux-amd64.tar.gz"
sudo rm -rf /usr/local/go
sudo tar -C /usr/local -xzf "go${GO_VERSION}.linux-amd64.tar.gz"
rm "go${GO_VERSION}.linux-amd64.tar.gz"

# Add Go to PATH for the current session and permanently
export PATH=$PATH:/usr/local/go/bin
echo 'export PATH=$PATH:/usr/local/go/bin' | sudo tee -a /etc/profile

# Create app directory if it doesn't exist
sudo mkdir -p /opt/emailapi
sudo chown -R $USER:$USER /opt/emailapi

# Apply performance tuning (BBRv3, TCP buffers, etc.)
echo "Applying performance tuning..."
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
sudo cp "${SCRIPT_DIR}/sysctl-performance.conf" /etc/sysctl.d/99-emailapi-performance.conf
sudo sysctl -p /etc/sysctl.d/99-emailapi-performance.conf

# Verify BBR is enabled (use full path for minimal installs where /usr/sbin isn't in PATH)
echo "Verifying BBR congestion control..."
if /usr/sbin/sysctl net.ipv4.tcp_congestion_control | grep -q bbr; then
    echo "✓ BBR congestion control is active"
else
    echo "⚠ BBR may not be available on this kernel"
fi

echo "Setup complete! Please reboot or run 'source /etc/profile' to update PATH."
