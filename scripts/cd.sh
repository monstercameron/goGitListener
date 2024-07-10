#!/bin/bash

# Exit immediately if a command exits with a non-zero status
set -e

# Define variables
SERVICE_NAME="gogitlistener"
INSTALL_DIR="/root/$SERVICE_NAME"
BINARY_NAME="gogitlistener"

# Change to the GoGitListener directory
cd $INSTALL_DIR

# Stop the service
echo "Stopping $SERVICE_NAME service..."
systemctl stop $SERVICE_NAME

# Pull the latest changes from the main branch
echo "Pulling latest changes from GitHub..."
git pull origin main

# Rebuild the Go application
echo "Rebuilding the application..."
go build -o $BINARY_NAME main.go

# Start the service
echo "Starting $SERVICE_NAME service..."
systemctl start $SERVICE_NAME

echo "GoGitListener updated and restarted successfully"