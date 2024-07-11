#!/bin/bash

# Exit immediately if a command exits with a non-zero status
set -e

# Define variables
SERVICE_NAME="goGitListener"
INSTALL_DIR="/root/goGitListener"
BINARY_NAME="goGitListener"
RESTART_FLAG="/tmp/restart_required"

# Change to the goGitListener directory
cd $INSTALL_DIR

# Pull the latest changes from the main branch
echo "Pulling latest changes from GitHub..."
git pull

# Rebuild the Go application
echo "Rebuilding the application..."
/usr/local/go/bin/go build -o $BINARY_NAME

# Create the restart flag file
echo "Creating restart flag..."
touch $RESTART_FLAG

echo "goGitListener updated successfully. It will restart after processing the current webhook."