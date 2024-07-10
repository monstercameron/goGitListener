#!/bin/bash

# Exit immediately if a command exits with a non-zero status
set -e

# Change to the GoGitListener directory
cd /root/goGitListener

# Pull the latest changes from the main branch
git pull origin main

# Rebuild the Go application
go build -o gogitlistener main.go

# Restart the GoGitListener service
systemctl restart gogitlistener

echo "GoGitListener updated and restarted successfully"