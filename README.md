<div align="center">
  <img src="logo.png" alt="Scheduler0 CLI" width="200"/>
  <br />
  <h1>Scheduler0 CLI</h1>
</div>

A command-line interface for interacting with the Scheduler0 API. Built with Go for native installation on Windows, macOS, and Linux.

## Installation

### From Source

```bash
git clone https://github.com/scheduler0/scheduler0-cli.git
cd scheduler0-cli
go build -o scheduler0
```

### Binary Releases

Download pre-built binaries from the [Releases](https://github.com/scheduler0/scheduler0-cli/releases) page.

#### macOS

```bash
# Using Homebrew (when available)
brew install scheduler0/scheduler0-cli
```

#### Linux

```bash
# Download and install
wget https://github.com/scheduler0/scheduler0-cli/releases/latest/download/scheduler0-linux-amd64
chmod +x scheduler0-linux-amd64
sudo mv scheduler0-linux-amd64 /usr/local/bin/scheduler0
```

#### Windows

Download `scheduler0-windows-amd64.exe` from releases and add it to your PATH.

## Quick Start

1. **Initialize credentials:**

```bash
scheduler0 init
```

This will prompt you for:
- Base URL (e.g., `https://api.scheduler0.com`)
- API Key
- API Secret
- Account ID

Alternatively, you can provide these via flags:

```bash
scheduler0 init \
  --base-url https://api.scheduler0.com \
  --api-key your-api-key \
  --api-secret your-api-secret \
  --account-id your-account-id
```

Credentials are saved to `~/.scheduler0/config.json` and will be used for all subsequent commands.

2. **Check cluster health:**

```bash
scheduler0 healthcheck
```

3. **List projects:**

```bash
scheduler0 projects list
```

## Commands

> **Note for Self-Hosted Users**: Account Management, Feature Management, and Async Tasks Management APIs are designed for users who run Scheduler0 in their own infrastructure and need granular control over team access and resource usage. These features enable multi-tenant management, feature flags, and async task tracking in self-hosted deployments.

### Accounts

> **Self-Hosted Feature**: Account management is for self-hosted deployments that require multi-tenant access control and resource isolation.

```bash
# Create an account
scheduler0 accounts create --name "My Account"

# Get account details
scheduler0 accounts get <account-id>
```
<｜tool▁calls▁begin｜><｜tool▁call▁begin｜>
codebase_search

### Projects

```bash
# List projects
scheduler0 projects list [--limit 10] [--offset 0] [--order-by date_created] [--order-direction desc]

# Get project details
scheduler0 projects get <project-id>

# Create a project
scheduler0 projects create --name "My Project" --description "Description" --created-by "user-id"

# Update a project
scheduler0 projects update <project-id> --description "New description" --modified-by "user-id"

# Delete a project
scheduler0 projects delete <project-id> --deleted-by "user-id"
```

### Jobs

```bash
# List jobs
scheduler0 jobs list [--project-id <id>] [--limit 10] [--offset 0]

# Get job details
scheduler0 jobs get <job-id>

# Create a job
scheduler0 jobs create \
  --project-id 123 \
  --timezone "UTC" \
  --spec "0 30 * * * *" \
  --data '{"key": "value"}' \
  --created-by "user-id"

# Update a job
scheduler0 jobs update <job-id> \
  --spec "0 0 * * * *" \
  --status "inactive" \
  --modified-by "user-id"

# Delete a job
scheduler0 jobs delete <job-id> --deleted-by "user-id"
```

### Executors

```bash
# List executors
scheduler0 executors list [--limit 10] [--offset 0]

# Get executor details
scheduler0 executors get <executor-id>

# Create a webhook executor
scheduler0 executors create \
  --name "webhook-executor" \
  --type "webhook_url" \
  --webhook-url "https://example.com/webhook" \
  --webhook-method "POST" \
  --webhook-secret "secret" \
  --created-by "user-id"

# Create a cloud function executor
scheduler0 executors create \
  --name "cloud-function" \
  --type "cloud_function" \
  --region "us-west-1" \
  --cloud-provider "aws" \
  --cloud-resource-url "https://example.com/function" \
  --cloud-api-key "key" \
  --cloud-api-secret "secret" \
  --created-by "user-id"

# Update an executor
scheduler0 executors update <executor-id> \
  --name "updated-name" \
  --modified-by "user-id"

# Delete an executor
scheduler0 executors delete <executor-id> --deleted-by "user-id"
```

### Executions

```bash
# List executions
scheduler0 executions \
  --start-date "2024-01-01T00:00:00Z" \
  --end-date "2024-12-31T23:59:59Z" \
  [--project-id <id>] \
  [--job-id <id>] \
  [--limit 10] \
  [--offset 0]
```

### Credentials

```bash
# List credentials
scheduler0 credentials list [--limit 10] [--offset 0]

# Get credential details
scheduler0 credentials get <credential-id>

# Create a credential
scheduler0 credentials create --created-by "user-id"

# Update a credential
scheduler0 credentials update <credential-id> --archived true --modified-by "user-id"

# Archive a credential
scheduler0 credentials archive <credential-id> --archived-by "user-id"

# Delete a credential
scheduler0 credentials delete <credential-id> --deleted-by "user-id"
```

### Async Tasks

> **Self-Hosted Feature**: Async task management is for self-hosted deployments that need to track and monitor asynchronous operations like batch job creation.

```bash
# Get async task status (e.g., after creating a job)
scheduler0 async-tasks get <request-id>
```

### Feature Management

> **Self-Hosted Feature**: Feature management allows self-hosted users to enable or disable specific capabilities for accounts, providing granular control over resource usage and feature access. Features can be managed through the account endpoints (see Accounts section above).

### Healthcheck

```bash
# Check cluster health (no authentication required)
scheduler0 healthcheck
```

## Configuration

Credentials are stored in `~/.scheduler0/config.json`:

```json
{
  "base_url": "https://api.scheduler0.com",
  "api_key": "your-api-key",
  "api_secret": "your-api-secret",
  "account_id": "your-account-id"
}
```

You can override the saved configuration using flags:

```bash
scheduler0 projects list \
  --base-url https://api.example.com \
  --api-key different-key \
  --api-secret different-secret \
  --account-id different-account
```

## Environment Variables

You can also use environment variables (though using `init` is recommended):

- `SCHEDULER0_BASE_URL`
- `SCHEDULER0_API_KEY`
- `SCHEDULER0_API_SECRET`
- `SCHEDULER0_ACCOUNT_ID`

## Examples

### Complete Workflow

```bash
# 1. Initialize
scheduler0 init

# 2. Create a project
scheduler0 projects create --name "My Project" --created-by "user-123"

# 3. Create an executor
scheduler0 executors create \
  --name "webhook" \
  --type "webhook_url" \
  --webhook-url "https://api.example.com/webhook" \
  --webhook-method "POST" \
  --created-by "user-123"

# 4. Create a job
scheduler0 jobs create \
  --project-id 1 \
  --timezone "UTC" \
  --spec "0 30 * * * *" \
  --data '{"message": "Hello"}' \
  --created-by "user-123"

# 5. List executions
scheduler0 executions \
  --start-date "2024-01-01T00:00:00Z" \
  --end-date "2024-12-31T23:59:59Z"
```

## Building from Source

### Prerequisites

- Go 1.23 or later

### Build

```bash
go build -o scheduler0
```

### Cross-compilation

```bash
# Linux
GOOS=linux GOARCH=amd64 go build -o scheduler0-linux-amd64

# macOS
GOOS=darwin GOARCH=amd64 go build -o scheduler0-darwin-amd64
GOOS=darwin GOARCH=arm64 go build -o scheduler0-darwin-arm64

# Windows
GOOS=windows GOARCH=amd64 go build -o scheduler0-windows-amd64.exe
```

## License

This project is licensed under the MIT License - see the LICENSE file for details.

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

