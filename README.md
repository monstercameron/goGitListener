# GoGitListener

## Table of Contents
1. [Introduction](#introduction)
2. [Features](#features)
3. [Prerequisites](#prerequisites)
4. [Installation](#installation)
5. [Configuration](#configuration)
6. [Usage](#usage)
7. [Metrics](#metrics)
8. [How It Works](#how-it-works)
9. [Security Considerations](#security-considerations)
10. [Troubleshooting](#troubleshooting)
11. [Contributing](#contributing)
12. [License](#license)

## Introduction

GoGitListener is a Go application designed to automate actions in response to GitHub push events. It's particularly useful for managing multiple projects on a single server, such as a Digital Ocean VPS. When a push event occurs in a GitHub repository, this listener receives the webhook, verifies it, and executes a specified script for the corresponding project.

## Features

- Handles multiple projects with different configurations
- Securely verifies webhook signatures
- Executes custom scripts for each project on push events
- Configurable through a JSON file
- Lightweight and easy to set up
- Logs webhook requests for monitoring and debugging
- Provides a `/metrics` endpoint to view logs

## Prerequisites

- Go 1.16 or later
- A server with a public IP address (e.g., a Digital Ocean VPS)
- Basic knowledge of Go, GitHub, and server administration
- Git installed on the server

## Installation

### Automated Installation

An installation script is provided to easily set up GoGitListener as a systemd service.

1. Ensure you have root access and Go is installed on your system.
2. Place `install.sh`, `main.go`, and `config.json` in the same directory.
3. Make the script executable:
   ```
   chmod +x install.sh
   ```
4. Run the installation script:
   ```
   sudo ./install.sh
   ```
5. The script will build the application, create a systemd service, and start it.
6. You can check the status of the service with:
   ```
   systemctl status gogitlistener
   ```

GoGitListener will now start automatically on system boot.

### Manual Installation

If you prefer to install manually:

1. Clone this repository or copy the Go script to your server:
   ```
   git clone https://github.com/yourusername/gogitlistener.git
   cd gogitlistener
   ```

2. Build the Go application:
   ```
   go build -o gogitlistener main.go
   ```

3. Make the built application executable:
   ```
   chmod +x gogitlistener
   ```

## Configuration

1. Create a `config.json` file in the same directory as the executable:
   ```json
   {
     "project1": {
       "secret": "your_webhook_secret_for_project1",
       "path": "/path/to/project1"
     },
     "project2": {
       "secret": "your_webhook_secret_for_project2",
       "path": "/path/to/project2"
     }
   }
   ```

2. For each project, create a `cd.sh` script in the `scripts` directory within the project path. For example:
   ```
   /path/to/project1/scripts/cd.sh
   /path/to/project2/scripts/cd.sh
   ```

   This script will be executed when a push event is received. For the GoGitListener project itself, you can use the following `cd.sh` script:

   ```bash
   #!/bin/bash
   cd /path/to/gogitlistener
   git pull origin main
   go build -o gogitlistener main.go
   sudo systemctl restart gogitlistener
   ```

   Make sure to replace `/path/to/gogitlistener` with the actual path to your GoGitListener directory.

3. Make sure the `cd.sh` scripts are executable:
   ```
   chmod +x /path/to/project1/scripts/cd.sh
   chmod +x /path/to/project2/scripts/cd.sh
   ```

## Usage

1. Start GoGitListener:
   ```
   ./gogitlistener
   ```

2. Set up GitHub webhooks for each project:
   - Go to your GitHub repository
   - Navigate to Settings > Webhooks > Add webhook
   - Set Payload URL to `http://your_server_ip:8080/webhook?project=project1` (replace `project1` with your project name)
   - Set Content type to `application/json`
   - Set Secret to the corresponding secret in your `config.json`
   - Choose "Just the push event" for events to trigger this webhook

3. The listener will now receive webhooks and execute the corresponding `scripts/cd.sh` script for each push event.

## Metrics

GoGitListener provides a `/metrics` endpoint to view the logs of webhook requests. To access the metrics:

1. Ensure GoGitListener is running.
2. Open a web browser or use a tool like `curl` to access:
   ```
   http://your_server_ip:8080/metrics
   ```
3. You will see the contents of the `logs/log.log` file, which contains detailed information about each webhook request received.

Note: In a production environment, you should secure this endpoint to prevent unauthorized access to potentially sensitive information.

## How It Works

1. When a push event occurs on GitHub, it sends a POST request to the specified webhook URL.
2. GoGitListener receives this request and extracts the project name from the URL query parameters.
3. It loads the project configuration from `config.json` and verifies the webhook signature using the project's secret.
4. If the signature is valid, it executes the `scripts/cd.sh` script in the project's specified path.
5. The script typically pulls the latest changes, rebuilds the application if necessary, and restarts the service.
6. All webhook requests are logged to `logs/log.log` for monitoring and debugging purposes.

## Security Considerations

- Use HTTPS instead of HTTP for webhook URLs in production.
- Keep your webhook secrets secure and don't share them.
- Regularly update your Go installation and dependencies.
- Implement proper firewall rules on your server.
- Limit the permissions of the user running GoGitListener.
- Secure the `/metrics` endpoint in production environments.

## Troubleshooting

- **Webhook not triggering**: Check GitHub webhook delivery logs and ensure your server is accessible.
- **Invalid signature errors**: Verify that the secrets in `config.json` match those set in GitHub.
- **Script not executing**: Check file permissions and paths in `config.json`. Ensure `scripts/cd.sh` exists in the project directory.
- **Errors in script execution**: Review your `cd.sh` scripts and check the listener's log output.
- **Service not restarting**: Ensure the user running GoGitListener has permission to restart the service.

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## License

This project is licensed under the MIT License - see the LICENSE file for details.