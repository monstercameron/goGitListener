#!/bin/bash

# Exit immediately if a command exits with a non-zero status
set -e

# Define variables
SERVICE_NAME="gogitlistener"
GO_FILE="main.go"
BINARY_NAME="gogitlistener"
SERVICE_FILE="/etc/systemd/system/$SERVICE_NAME.service"
INSTALL_DIR="/root/$SERVICE_NAME"

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

# Create installation directory if it doesn't exist
mkdir -p $INSTALL_DIR

# Copy main.go and config.json to installation directory
cp $GO_FILE $INSTALL_DIR/
cp config.json $INSTALL_DIR/

# Create systemd service file
echo "Creating systemd service file..."
cat > $SERVICE_FILE <<EOL
[Unit]
Description=GoGitListener - GitHub Webhook Listener
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

echo "Installation complete. The GoGitListener is now running as a service."
echo "You can check its status with: systemctl status $SERVICE_NAME"