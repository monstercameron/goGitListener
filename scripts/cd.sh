#!/bin/bash

# Exit immediately if a command exits with a non-zero status
set -e

# Define variables
SERVICE_NAME="goGitListener"
INSTALL_DIR="/root/goGitListener"
BINARY_NAME="goGitListener"

# Change to the goGitListener directory
cd $INSTALL_DIR

# Stop the service
echo "Stopping $SERVICE_NAME service..."
systemctl stop $SERVICE_NAME

# Pull the latest changes from the main branch
echo "Pulling latest changes from GitHub..."
git pull

# Rebuild the Go application
echo "Rebuilding the application..."
/usr/local/go/bin/go build -o $BINARY_NAME

# Start the service
echo "Starting $SERVICE_NAME service..."
systemctl start $SERVICE_NAME

echo "goGitListener updated and restarted successfully"