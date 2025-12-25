#!/bin/bash
set -e

# Update system
echo "Updating system..."
sudo apt-get update && sudo apt-get upgrade -y

# Install essential tools
echo "Installing essential tools..."
sudo apt-get install -y git make build-essential curl

# Install Redis
echo "Installing Redis..."
sudo apt-get install -y redis-server
sudo systemctl enable redis-server
sudo systemctl start redis-server

# Install Nginx
echo "Installing Nginx..."
sudo apt-get install -y nginx
sudo systemctl enable nginx
sudo systemctl start nginx

# Install Certbot
echo "Installing Certbot..."
sudo apt-get install -y certbot python3-certbot-nginx

# Install Go (latest stable)
echo "Installing Go..."
GO_VERSION="1.25.0" # Update this to the desired version
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

echo "Setup complete! Please reboot or run 'source /etc/profile' to update PATH."
