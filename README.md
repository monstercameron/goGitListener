# goGitListener

## Table of Contents
1. [Introduction](#introduction)
2. [Features](#features)
3. [Prerequisites](#prerequisites)
4. [Installation](#installation)
5. [Configuration](#configuration)
6. [Usage](#usage)
7. [Webhook Handling](#webhook-handling)
8. [Metrics](#metrics)
9. [Logging](#logging)
10. [Updating](#updating)
11. [Troubleshooting](#troubleshooting)
12. [Contributing](#contributing)
13. [License](#license)

## Introduction

goGitListener is a Go application designed to listen for GitHub webhooks and execute custom scripts based on push events. It's particularly useful for automating deployment processes or running specific tasks when changes are pushed to a GitHub repository.

## Features

- Handles multiple projects with different configurations
- Securely verifies webhook signatures
- Executes custom scripts for each project on push events
- Configurable through a JSON file
- Detailed logging for easy debugging
- Provides a `/metrics` endpoint to view logs
- Graceful update process that doesn't interrupt webhook processing

## Prerequisites

- Go 1.16 or later
- Git
- A server with a public IP address (e.g., a DigitalOcean droplet)
- Basic knowledge of Go, Git, and server administration
- Systemd for service management (typically pre-installed on most Linux distributions)

## Installation

1. Clone the repository:
   ```
   git clone https://github.com/yourusername/goGitListener.git
   cd goGitListener
   ```

2. Build the application:
   ```
   go build -o goGitListener
   ```

3. Set up the systemd service:
   ```
   sudo nano /etc/systemd/system/goGitListener.service
   ```
   
   Add the following content (adjust paths as necessary):
   ```
   [Unit]
   Description=goGitListener - GitHub Webhook Listener
   After=network.target

   [Service]
   ExecStart=/root/goGitListener/goGitListener
   WorkingDirectory=/root/goGitListener
   User=root
   Group=root
   Restart=always

   [Install]
   WantedBy=multi-user.target
   ```

4. Enable and start the service:
   ```
   sudo systemctl enable goGitListener
   sudo systemctl start goGitListener
   ```

## Configuration

1. Create a `config.json` file in the same directory as the executable:
   ```json
   {
     "projectName": {
       "secret": "your_github_webhook_secret",
       "path": "/path/to/project"
     }
   }
   ```

2. For each project, create a `cd.sh` script in the `scripts` directory within the project path:
   ```bash
   #!/bin/bash
   set -e
   
   cd /path/to/project
   git pull
   # Add any other commands you want to run after a push
   
   touch /tmp/restart_required
   ```

3. Make sure the `cd.sh` scripts are executable:
   ```
   chmod +x /path/to/project/scripts/cd.sh
   ```

## Usage

1. Set up GitHub webhooks for each project:
   - Go to your GitHub repository
   - Navigate to Settings > Webhooks > Add webhook
   - Set Payload URL to `http://your_server_ip:3002/webhook?project=projectName`
   - Set Content type to `application/json`
   - Set Secret to the corresponding secret in your `config.json`
   - Choose "Just the push event" for events to trigger this webhook

2. The listener will now receive webhooks and execute the corresponding `scripts/cd.sh` script for each push event.

## Webhook Handling

When a webhook is received:
1. The application verifies the webhook signature
2. It executes the appropriate `cd.sh` script for the project
3. If the script creates a `/tmp/restart_required` file, the application will gracefully restart after processing the webhook

## Metrics

Access the `/metrics` endpoint to view logs:
```
http://your_server_ip:3002/metrics
```

## Logging

Logs are stored in the `logs/log.log` file. Each webhook request and processing step is logged for easy debugging.

## Updating

To update the goGitListener itself:

1. Push changes to the goGitListener repository
2. The application will pull the latest changes, rebuild itself, and gracefully restart

## Troubleshooting

- Check the logs at `logs/log.log` for detailed information about each webhook request and processing step
- Ensure that the `cd.sh` scripts have the correct permissions and are executable
- Verify that the GitHub webhook secrets in `config.json` match those set in GitHub

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## License

This project is licensed under the MIT License - see the LICENSE file for details.