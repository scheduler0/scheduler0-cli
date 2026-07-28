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

1. **Sign in:**

```bash
scheduler0 login
```

This opens `app.scheduler0.com` in your browser, where you approve access using
your existing signed-in session. A **short-lived** credential is then stored at
`~/.scheduler0/config.json` and used for all subsequent commands. When it
expires, just run `scheduler0 login` again. To sign out, run `scheduler0 logout`.

The command actor for created/modified/deleted/archived fields is taken
automatically from your signed-in identity — there is no `--created-by` flag.

Useful flags:

```bash
# Point at a non-default environment (e.g. staging or self-hosted)
scheduler0 login \
  --app-url https://app.staging.scheduler0.com \
  --base-url https://api.staging.scheduler0.com
```

#### Headless / SSH sessions

The default flow opens a local browser and receives the authorization code via
a redirect to `127.0.0.1` — which only works when the browser and the CLI
share the same machine. Over a plain SSH session there's no local browser, and
even if you open the printed URL on your laptop instead, its browser can't
reach a loopback port on the remote host, so that redirect can never arrive.

For this case, `scheduler0 login` automatically detects a headless SSH session
(no local display) and falls back to a **device code** flow instead: it prints
a short code and a URL, you open that URL on *any* device (your laptop, your
phone — it doesn't need to reach the remote machine at all) and enter the
code, and the CLI polls until you approve it. Force it explicitly with
`--device` (or the equivalent `--no-browser`) if the auto-detection doesn't
trigger:

```bash
scheduler0 login --device
```

2. **Check cluster health:**

```bash
scheduler0 healthcheck
```

3. **List projects:**

```bash
scheduler0 projects list
```

## Commands

> **Note for Self-Hosted Users**: Account Management and Async Tasks Management APIs are designed for users who run Scheduler0 in their own infrastructure and need granular control over team access and resource usage. These features enable multi-tenant management and async task tracking in self-hosted deployments.

### Accounts

> **Self-Hosted Feature**: Account management is for self-hosted deployments that require multi-tenant access control and resource isolation.

```bash
# Create an account
scheduler0 accounts create --name "My Account"

# Get account details
scheduler0 accounts get <account-id>

# Get an account's token balance
scheduler0 accounts tokens get <account-id>

# Add tokens to an account's balance
scheduler0 accounts tokens add <account-id> --amount 1000

# Generate random AES-256 secret key(s) offline (no API call)
scheduler0 accounts generate-secret-key [--count 1]

# Re-encrypt stored secrets after rotating the server SecretKey (admin-scoped session)
scheduler0 accounts rotate-secret --old-secret-key <OLD_HEX_KEY>
```

### Projects

```bash
# List projects
scheduler0 projects list [--limit 10] [--offset 0] [--order-by date_created] [--order-direction desc]

# Get project details
scheduler0 projects get <project-id>

# Create a project
scheduler0 projects create --name "My Project" --description "Description"

# Update a project
scheduler0 projects update <project-id> --description "New description"

# Delete a project
scheduler0 projects delete <project-id>
```

### Jobs

```bash
# List jobs
scheduler0 jobs list [--project-id <id>] [--limit 10] [--offset 0] [--order-by date_created] [--order-direction desc]

# Get job details
scheduler0 jobs get <job-id>

# Create a job (--project-id is required)
scheduler0 jobs create \
  --project-id 123 \
  --timezone "UTC" \
  --spec "0 30 * * * *" \
  --data '{"key": "value"}'

# All jobs create flags:
#   --project-id <id>          Project ID (required)
#   --executor-id <id>         Executor ID
#   --data <string>            Job payload data
#   --spec <cron>              Cron specification
#   --start-date <RFC3339>     Start date
#   --end-date <RFC3339>       End date
#   --timezone <tz>            Timezone (default "UTC")
#   --timezone-offset <secs>   Timezone offset in seconds
#   --retry-max <n>            Maximum retries
#   --status <active|inactive> Job status (default "active")

# Update a job (same flags as create, minus the required --project-id)
scheduler0 jobs update <job-id> \
  --spec "0 0 * * * *" \
  --status "inactive"

# Delete a job
scheduler0 jobs delete <job-id>
```

### Executors

```bash
# List executors
scheduler0 executors list [--limit 10] [--offset 0] [--order-by date_created] [--order-direction desc]

# Get executor details
scheduler0 executors get <executor-id>

# Create a webhook executor
scheduler0 executors create \
  --name "webhook-executor" \
  --type "webhook_url" \
  --webhook-url "https://example.com/webhook" \
  --webhook-method "POST" \
  --webhook-secret "secret"

# Create a cloud function executor
scheduler0 executors create \
  --name "cloud-function" \
  --type "cloud_function" \
  --region "us-west-1" \
  --cloud-provider "aws" \
  --cloud-resource-url "https://example.com/function" \
  --cloud-api-key "key" \
  --cloud-api-secret "secret"

# Update an executor
scheduler0 executors update <executor-id> \
  --name "updated-name"

# Delete an executor
scheduler0 executors delete <executor-id>
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

# Classify a prompt with the intent guardrail only (no AI execution, no credits consumed)
scheduler0 prompt classify --prompt "Send weekly reports every Monday at 9 AM"
```

**Note**: The `prompt` command requires credits. Each prompt execution consumes 1 credit. If you have insufficient credits, you'll receive an error. `prompt classify` runs only the edge intent classifier and consumes no credits.

The `--timezone` flag is optional. When omitted, the AI assumes `UTC`. When set to an IANA name (e.g. `America/New_York`), the AI interprets relative phrases like *"9am tomorrow"* in that timezone and emits timestamps with the matching numeric offset. Invalid timezone strings are rejected by the API with `400 Bad Request`.

### AI Scheduling (schedule)

Turn a natural language prompt into scheduled jobs in one call. The server runs the
prompt pipeline (intent guardrail + generation), resolves or creates a project, picks
the best-matching executor (or uses a pinned/only one), and creates the jobs
synchronously. The `createdBy` field is taken automatically from your signed-in
identity (or `SCHEDULER0_ACTOR` in CI) — there is no `--created-by` flag. This endpoint
requires credits.

```bash
scheduler0 schedule \
  --prompt "Send weekly reports every Monday at 9 AM" \
  [--purposes "reporting,communication"] \
  [--events "weekly_cycle"] \
  [--recipients "team@example.com"] \
  [--channels "email"] \
  [--timezone "America/New_York"] \
  [--locale "en"] \
  [--project-id <id>] \
  [--executor-id <id>]
```

### AI Suggestions

Analyze conversations and compute optimal send times using the AI suggestions engine.
Because the request bodies are complex, they are read as JSON from a `--file` path, or
from stdin when `--file` is omitted or set to `-`.

```bash
# Analyze a conversation for suggestions and obligations
scheduler0 suggestions analyze --file conversation.json
cat conversation.json | scheduler0 suggestions analyze

# Recommend optimal future send times for a message
scheduler0 suggestions time --file sendtime.json
cat sendtime.json | scheduler0 suggestions time
```

### AI Settings & Models

Manage per-account AI provider settings, and view the catalog of approved models.

```bash
# Get the current account's AI provider settings
scheduler0 ai-settings get [--account-id <id>]

# Create or update the account's AI provider settings
scheduler0 ai-settings upsert \
  --provider openai \
  --model gpt-4o \
  --openai-api-key <key> \
  [--anthropic-api-key <key>] \
  [--bedrock-access-key-id <id>] \
  [--bedrock-secret-key <key>] \
  [--bedrock-region <region>] \
  [--account-id <id>]

# List the per-provider approved model catalog
scheduler0 ai-models
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

Credentials carry **scopes** (`read`, `write`, `execute`, `admin`) that control which
routes they can call, and they automatically expire (90 days by default; the
browser login flow mints much shorter-lived ones). The `admin` scope authorizes
account- and cluster-level operations and can only be granted by an operator or an
existing admin credential.

The `apiSecret` is returned **once** at creation time as `plaintextSecret` in the JSON response — store it immediately.

```bash
# List credentials (JSON by default; `--output table` shows status, scopes, expiry)
scheduler0 credentials list [--limit 10] [--offset 0] [--output json|table]

# Get credential details
scheduler0 credentials get <credential-id>

# Create a credential. --scopes accepts a comma-separated subset of
# read,write,execute,admin and defaults to read,write,execute.
# Pass --archived to create it in an archived state.
scheduler0 credentials create --scopes "read,write" [--archived]

# Update a credential
scheduler0 credentials update <credential-id> --archived true

# Archive a credential
scheduler0 credentials archive <credential-id>

# Delete a credential
scheduler0 credentials delete <credential-id>
```

#### Replacing an expiring credential

When a credential nears its 90-day expiry, create a replacement, roll out the new `apiKey`/`plaintextSecret` to your clients, then archive the old one:

```bash
scheduler0 credentials create --scopes "read,write,execute"
# Update your clients with the new key/secret printed above
scheduler0 credentials archive <old-credential-id>
```

#### Rotating the server's SecretKey (self-hosting only)

If `SecretKey` in your secrets source is compromised and replaced, re-encrypt all stored secrets (credential api secrets, executor cloud credentials, and AI provider keys) to match the new key:

```bash
# 1. Generate a new key (offline)
scheduler0 accounts generate-secret-key

# 2. Update SecretKey in your secrets source and reload/restart the server

# 3. Re-encrypt stored secrets, passing the previous key.
#    Requires an admin-scoped session (sign in as an operator with `scheduler0 login`).
scheduler0 accounts rotate-secret --old-secret-key <OLD_HEX_KEY>
```

The server decrypts every stored secret with the old key and re-encrypts it with the currently-loaded new key. Credentials keep working across rotation — the `api_secret` is verified by decrypting the stored value, and the `api_key` identifier is unchanged. The operation is idempotent: rows already on the new key are skipped, so if interrupted you can simply re-run it.

### Async Tasks

> **Self-Hosted Feature**: Async task management is for self-hosted deployments that need to track and monitor asynchronous operations like batch job creation.

```bash
# Get async task status (e.g., after creating a job)
scheduler0 async-tasks get <request-id>
```

### Local Executor

Run jobs directly on this machine. Register the machine once, then start the
long-running service that polls for assigned jobs and executes them locally.

```bash
# Register this machine as a local executor (saves the executor ID to config)
scheduler0 local-executor register \
  --name "my-machine" \
  --command "/path/to/handler.sh" \
  [--working-dir /path/to/dir]

# Start the local executor service
scheduler0 local-executor start [--poll-interval 1m]
```

### Backup & Restore

> **Self-Hosted Feature**: Database backup and restore require an admin-scoped session.

```bash
# Start an automatic timestamped backup
scheduler0 backup start

# Restore the database from a backup file (S3 object key or local path)
scheduler0 backup restore <file-name>
```

### Healthcheck

```bash
# Check cluster health (no authentication required)
scheduler0 healthcheck
```

## Authentication

The CLI authenticates with a **short-lived credential** obtained by signing in
through your browser:

```bash
scheduler0 login             # opens app.scheduler0.com to authorize the CLI
scheduler0 login --device    # headless/SSH-friendly device-code flow
scheduler0 logout            # clears the stored credential
```

`scheduler0 login` auto-detects a headless SSH session and switches to the
device-code flow on its own; see [Headless / SSH sessions](#headless--ssh-sessions).

Under the hood, requests are authenticated with the credential's API key/secret
and account id (`X-API-Key`, `X-Secret-Key`, `X-Account-ID` headers). The session
expires automatically; re-run `scheduler0 login` to refresh it.

Operator (admin) commands such as `accounts` management, `backup`/`restore`, and
`accounts rotate-secret` require a credential carrying the **`admin`** scope.

### CI / non-interactive authentication

`scheduler0 login` is interactive, so in CI authenticate with environment
variables instead — when set, they **take priority** over any on-disk session,
so no `~/.scheduler0/config.json` file is needed at all:

```bash
export SCHEDULER0_API_KEY=...        # required
export SCHEDULER0_API_SECRET=...     # required
export SCHEDULER0_ACCOUNT_ID=...     # required
export SCHEDULER0_BASE_URL=...       # optional, defaults to https://api.scheduler0.com
export SCHEDULER0_ACTOR=github-actions   # required for create/update/delete/archive commands
export SCHEDULER0_EXPIRES_AT=...     # optional RFC3339; omit and let the server enforce expiry
```

Mint the credential once interactively and store its output as CI secrets:

```bash
scheduler0 credentials create --scopes read,write,execute
```

## Configuration

The session is stored in `~/.scheduler0/config.json`, written by `scheduler0 login`:

```json
{
  "base_url": "https://api.scheduler0.com",
  "app_url": "https://app.scheduler0.com",
  "api_key": "…",
  "api_secret": "…",
  "account_id": "…",
  "clerk_user_id": "user_…",
  "expires_at": "2026-07-07T12:00:00Z",
  "scopes": ["read", "write", "execute"]
}
```

You can override the endpoint or account for a single command:

```bash
scheduler0 projects list \
  --base-url https://api.example.com \
  --account-id different-account
```

## Examples

### Complete Workflow

```bash
# 1. Sign in
scheduler0 login

# 2. Create a project
scheduler0 projects create --name "My Project"

# 3. Create an executor
scheduler0 executors create \
  --name "webhook" \
  --type "webhook_url" \
  --webhook-url "https://api.example.com/webhook" \
  --webhook-method "POST"

# 4. Create a job (or use AI prompt to generate job configurations)
scheduler0 jobs create \
  --project-id 1 \
  --timezone "UTC" \
  --spec "0 30 * * * *" \
  --data '{"message": "Hello"}'

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

