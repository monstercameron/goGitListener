#!/bin/bash

# Exit immediately if a command exits with a non-zero status
set -e

# Define variables
SERVICE_NAME="goGitListener"
BINARY_NAME="goGitListener"
SERVICE_FILE="/etc/systemd/system/$SERVICE_NAME.service"
INSTALL_DIR="/root/goGitListener"

# Check if running as root
if [ "$EUID" -ne 0 ]; then
    echo "Please run as root"
    exit 1
fi

# Stop the service if it's running
if systemctl is-active --quiet $SERVICE_NAME; then
    echo "Stopping $SERVICE_NAME service..."
    systemctl stop $SERVICE_NAME
fi

# Ensure the installation directory exists
if [ ! -d "$INSTALL_DIR" ]; then
    echo "Error: $INSTALL_DIR does not exist. Please ensure the directory is created and contains the necessary files."
    exit 1
fi

# Change to the installation directory
cd $INSTALL_DIR

# Build the Go binary
echo "Building the Go binary..."
go build -o $BINARY_NAME main.go

if [ $? -ne 0 ]; then
    echo "Error: Failed to build the Go binary."
    exit 1
fi

echo "Go binary built successfully."

# Create systemd service file
echo "Creating systemd service file..."
cat > $SERVICE_FILE <<EOL
[Unit]
Description=goGitListener - GitHub Webhook Listener
After=network.target

[Service]
ExecStart=$INSTALL_DIR/$BINARY_NAME
WorkingDirectory=$INSTALL_DIR
User=root
Group=root
Restart=always

[Install]
WantedBy=multi-user.target
EOL

# Reload systemd, enable and start the service
echo "Enabling and starting the service..."
systemctl daemon-reload
systemctl enable $SERVICE_NAME
systemctl start $SERVICE_NAME

echo "Installation complete. The goGitListener is now running as a service."
echo "You can check its status with: systemctl status $SERVICE_NAME"