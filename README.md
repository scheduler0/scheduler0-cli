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

The CLI supports two authentication methods:

### API Key Authentication (Managed/Hosted)
For managed or hosted Scheduler0 instances, you'll be prompted for:
- Base URL (e.g., `https://api.scheduler0.com`)
- API Key
- API Secret
- Account ID

Or provide via flags:
```bash
scheduler0 init \
  --base-url https://api.scheduler0.com \
  --api-key your-api-key \
  --api-secret your-api-secret \
  --account-id your-account-id
```

### Basic Authentication (Self-Hosted)
For self-hosted Scheduler0 instances, you'll be prompted for:
- Base URL (e.g., `http://localhost:7070`)
- Username (set during infrastructure setup)
- Password (set during infrastructure setup)

Or provide via flags:
```bash
scheduler0 init \
  --base-url http://localhost:7070 \
  --username your-username \
  --password your-password \
  --auth-type basic
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

### AI-Powered Job Creation

```bash
# Create job configurations from a natural language prompt
scheduler0 prompt \
  --prompt "Send weekly reports every Monday at 9 AM" \
  --purposes "reporting,communication" \
  --events "weekly_cycle" \
  --recipients "team@example.com,manager@example.com" \
  --channels "email" \
  --timezone "America/New_York"

# Simple prompt (only required field)
scheduler0 prompt --prompt "Follow up 2 days after the demo"
```

**Note**: This endpoint requires credits. Each prompt execution consumes 1 credit. If you have insufficient credits, you'll receive an error.

The `--timezone` flag is optional. When omitted, the AI assumes `UTC`. When set to an IANA name (e.g. `America/New_York`), the AI interprets relative phrases like *"9am tomorrow"* in that timezone and emits timestamps with the matching numeric offset. Invalid timezone strings are rejected by the API with `400 Bad Request`.

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

Credentials carry **scopes** (`read`, `write`, `execute`) that control which routes
they can call, and they automatically expire **90 days** after creation.

The `apiSecret` is returned **once** at creation time as `plaintextSecret` in the JSON response — store it immediately.

```bash
# List credentials (JSON by default; `--output table` shows status, scopes, expiry)
scheduler0 credentials list [--limit 10] [--offset 0] [--output json|table]

# Get credential details
scheduler0 credentials get <credential-id>

# Create a credential. --scopes accepts a comma-separated subset of read,write,execute
# and defaults to all three.
scheduler0 credentials create \
  --created-by "user-id" \
  --scopes "read,write"

# Update a credential
scheduler0 credentials update <credential-id> --archived true --modified-by "user-id"

# Archive a credential
scheduler0 credentials archive <credential-id> --archived-by "user-id"

# Delete a credential
scheduler0 credentials delete <credential-id> --deleted-by "user-id"
```

#### Replacing an expiring credential

When a credential nears its 90-day expiry, create a replacement, roll out the new `apiKey`/`plaintextSecret` to your clients, then archive the old one:

```bash
scheduler0 credentials create --created-by "user-id" --scopes "read,write,execute"
# Update your clients with the new key/secret printed above
scheduler0 credentials archive <old-credential-id> --archived-by "user-id"
```

#### Rotating the server's SecretKey (self-hosting only)

If `SecretKey` in your secrets source is compromised and replaced, re-encrypt all stored credentials to match the new key:

```bash
# Requires basic auth (configure auth_type=basic or use --username/--password)
scheduler0 credentials rotate-secret
```

Update `SecretKey` in your secrets source **first**. The command saves the old key as a savepoint, reloads the new key, re-encrypts all credentials in batches, and removes the savepoint on completion. If interrupted, re-run to resume.

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

## Authentication

Scheduler0 CLI supports two authentication methods:

### 1. API Key Authentication (Managed/Hosted)
For managed or hosted Scheduler0 instances, use API Key + Secret authentication:
- `X-API-Key`: Your API key
- `X-Secret-Key`: Your API secret  
- `X-Account-ID`: Your account ID

### 2. Basic Authentication (Self-Hosted)
For self-hosted Scheduler0 instances, use username and password set during infrastructure setup:
- Username and password configured during Scheduler0 setup
- Uses HTTP Basic Authentication
- No account ID required (single-tenant)

The CLI will automatically detect which authentication method to use based on your configuration.

## Configuration

Credentials are stored in `~/.scheduler0/config.json`:

**API Key Authentication:**
```json
{
  "base_url": "https://api.scheduler0.com",
  "api_key": "your-api-key",
  "api_secret": "your-api-secret",
  "account_id": "your-account-id",
  "auth_type": "api_key"
}
```

**Basic Authentication (Self-Hosted):**
```json
{
  "base_url": "http://localhost:7070",
  "username": "your-username",
  "password": "your-password",
  "auth_type": "basic"
}
```

You can override the saved configuration using flags:

**API Key Auth:**
```bash
scheduler0 projects list \
  --base-url https://api.example.com \
  --api-key different-key \
  --api-secret different-secret \
  --account-id different-account
```

**Basic Auth:**
```bash
scheduler0 projects list \
  --base-url http://localhost:7070 \
  --username admin \
  --password secret
```

## Environment Variables

You can also use environment variables (though using `init` is recommended):

**For API Key Authentication:**
- `SCHEDULER0_BASE_URL`
- `SCHEDULER0_API_KEY`
- `SCHEDULER0_API_SECRET`
- `SCHEDULER0_ACCOUNT_ID`

**For Basic Authentication (Self-Hosted):**
- `SCHEDULER0_BASE_URL`
- `SCHEDULER0_USERNAME`
- `SCHEDULER0_PASSWORD`

## Examples

### Complete Workflow (API Key Authentication)

```bash
# 1. Initialize with API key authentication
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

# 4. Create a job (or use AI prompt to generate job configurations)
scheduler0 jobs create \
  --project-id 1 \
  --timezone "UTC" \
  --spec "0 30 * * * *" \
  --data '{"message": "Hello"}' \
  --created-by "user-123"

# Alternative: Use AI to generate job configurations from natural language
scheduler0 prompt \
  --prompt "Send weekly reports every Monday at 9 AM" \
  --purposes "reporting" \
  --timezone "America/New_York"

# 5. List executions
scheduler0 executions \
  --start-date "2024-01-01T00:00:00Z" \
  --end-date "2024-12-31T23:59:59Z"
```

### Self-Hosted Workflow (Basic Authentication)

```bash
# 1. Initialize with basic authentication for self-hosted instance
scheduler0 init \
  --base-url http://localhost:7070 \
  --username admin \
  --password your-password \
  --auth-type basic

# 2. Check cluster health
scheduler0 healthcheck

# 3. Create a project (no account ID needed for self-hosted)
scheduler0 projects create --name "My Project" --created-by "admin"

# 4. Create and manage jobs
scheduler0 jobs create \
  --project-id 1 \
  --timezone "UTC" \
  --spec "0 30 * * * *" \
  --data '{"message": "Hello"}' \
  --created-by "admin"
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

