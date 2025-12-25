#!/bin/bash
set -e

# Directory where the app is cloned
APP_DIR="/opt/emailapi"

# Navigate to app dir
cd "$APP_DIR" || exit

echo "Pulling latest changes..."
git pull origin main

echo "Installing dependencies..."
# Assuming we are in the root of the monorepo
cd apps/api
/usr/local/go/bin/go mod download

echo "Building application..."
/usr/local/go/bin/go build -o emailapi main.go

echo "Restarting service..."
sudo systemctl restart emailapi

echo "Deployment successful!"
