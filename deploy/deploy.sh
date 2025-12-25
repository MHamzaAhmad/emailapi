#!/bin/bash
set -e

# Directory where the app is cloned
APP_DIR="/home/muhammadusama_mofx/emailapi"

# Navigate to app dir
cd "$APP_DIR" || exit

echo "Pulling latest changes..."
git pull origin main

echo "Installing dependencies..."
# Assuming we are in the root of the monorepo
cd apps/api
go mod download

echo "Building application..."
go build -o emailapi main.go

echo "Restarting service..."
sudo systemctl restart emailapi

echo "Deployment successful!"
