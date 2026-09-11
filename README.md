<div align="center">
  <img src="logo.png" alt="Scheduler0 CLI" width="200"/>
  <br />
  <h1>Scheduler0 CLI</h1>
</div>

`scheduler0` is the command-line client for the [Scheduler0](https://scheduler0.com) API. It is a single static Go binary for macOS, Linux and Windows.

## Installation

### Install script (macOS / Linux)

```bash
curl -fsSL https://raw.githubusercontent.com/scheduler0/scheduler0-cli/main/install.sh | bash
```

The script detects your OS (`darwin`/`linux`) and architecture (`amd64`/`arm64`), downloads the matching release archive plus `checksums.txt`, verifies the SHA-256 checksum, extracts the binary and installs it to `/usr/local/bin` (using `sudo` when needed, falling back to `~/.local/bin` if `sudo` is unavailable). On macOS it also clears the Gatekeeper quarantine flag.

Options:

```bash
# Pin a version (first argument or VERSION env var; leading "v" optional)
curl -fsSL https://raw.githubusercontent.com/scheduler0/scheduler0-cli/main/install.sh | bash -s -- 1.0.1
VERSION=1.0.1 bash install.sh

# Install somewhere else
INSTALL_DIR="$HOME/bin" bash install.sh
```

It needs `curl` or `wget`, `tar`, and `shasum` or `sha256sum` (checksum verification is skipped with a warning if neither is present).

### Release archives

Download from [GitHub Releases](https://github.com/scheduler0/scheduler0-cli/releases). Each release ships:

| OS | Arch | Asset |
|----|------|-------|
| Linux | amd64, arm64 | `scheduler0_<version>_linux_<arch>.tar.gz` |
| macOS | amd64, arm64 | `scheduler0_<version>_darwin_<arch>.tar.gz` |
| Windows | amd64 | `scheduler0_<version>_windows_amd64.zip` |

plus `checksums.txt` (SHA-256). Archives contain the `scheduler0` binary (`scheduler0.exe` on Windows), `LICENSE` and this README. Extract the binary and put it on your `PATH`.

### `go install`

```bash
go install github.com/scheduler0/scheduler0-cli@latest
```

Requires Go 1.26+. Binaries built this way report `scheduler0 version dev` because the version is only injected by the release build.

### From source

```bash
git clone https://github.com/scheduler0/scheduler0-cli.git
cd scheduler0-cli
make build              # -> bin/scheduler0 (VERSION=x.y.z make build to stamp a version)
make install            # copies bin/scheduler0 to /usr/local/bin (uses sudo)
make build-all          # cross-compile into bin/ for linux/darwin (amd64, arm64) and windows/amd64
```

### Docker

A multi-arch (amd64/arm64) image is published to the public ECR gallery on every release, tagged `latest` and with the release version: [gallery.ecr.aws/p2b2u7y5/scheduler0-cli](https://gallery.ecr.aws/p2b2u7y5/scheduler0-cli). `scheduler0` is the entrypoint, so pass the subcommand as the container arguments:

```bash
docker pull public.ecr.aws/p2b2u7y5/scheduler0-cli:latest   # or :1.0.1
docker run --rm \
  -e SCHEDULER0_API_KEY -e SCHEDULER0_API_SECRET -e SCHEDULER0_ACCOUNT_ID -e SCHEDULER0_ACTOR \
  public.ecr.aws/p2b2u7y5/scheduler0-cli:latest projects list
```

To build the same image locally from the repository `Dockerfile`:

```bash
docker build -t scheduler0-cli .
```

Inside a container there is no browser, so authenticate with environment variables (see [CI / non-interactive authentication](#ci--non-interactive-authentication)) or mount a `~/.scheduler0` directory produced by `scheduler0 login --device`.

## Quick start

```bash
scheduler0 login              # sign in through app.scheduler0.com
scheduler0 healthcheck        # cluster leader + Raft stats (no auth needed)
scheduler0 projects list
```

## Authentication

### `scheduler0 login`

`login` obtains a short-lived API credential by authorizing the CLI in the Scheduler0 web app, then stores it in `~/.scheduler0/config.json`.

Two flows are implemented:

- **Browser (default).** The CLI starts a loopback HTTP listener on a random `127.0.0.1` port, generates a PKCE verifier/challenge and a CSRF `state`, opens `<app-url>/cli/authorize?...` in your default browser (or prints the URL if it can't), and waits up to 5 minutes for the redirect back to `http://127.0.0.1:<port>/callback`. It then exchanges the one-time code at `<app-url>/cli/token` for the credential.
- **Device code.** The CLI requests a code from `<app-url>/cli/device/code`, prints a one-time user code and a verification URL (plus a "complete" URL that pre-fills the code), and polls `<app-url>/cli/token` at the server-provided interval (at least 3 s) until you approve it on any device, or the code expires.

The device flow is used automatically when the CLI detects a headless SSH session (`SSH_CONNECTION`/`SSH_TTY`/`SSH_CLIENT` set and, on Linux, no `DISPLAY`/`WAYLAND_DISPLAY`). Force it with `--device` (alias `--no-browser`).

```bash
scheduler0 login
scheduler0 login --device
scheduler0 login --app-url https://app.staging.scheduler0.com --base-url https://api.staging.scheduler0.com
```

Flags: `--app-url` (web app used for login, default `https://app.scheduler0.com`), `--base-url` (API endpoint, default `https://api.scheduler0.com`), `--device` / `--no-browser`. Endpoint resolution is flag > value already in the config file > default.

On success the CLI prints the account id and the session expiry. The token response provides the API key/secret, account id, your user id and the credential's scopes. When the session expires, run `scheduler0 login` again.

### `scheduler0 logout`

Clears the stored credential (API key/secret, account id, expiry, user id, scopes) from the config file. Endpoints and any local executor registration are kept. The remote credential is not revoked; it expires on its own (archive or delete it with `scheduler0 credentials archive|delete` if you need to revoke it immediately).

### How requests are authenticated

Every request (except `healthcheck`) sends `X-API-Key`, `X-Secret-Key` and `X-Account-ID`. The `createdBy` / `modifiedBy` / `deletedBy` / `archivedBy` fields the API requires are filled automatically from the signed-in user id (or `SCHEDULER0_ACTOR` in CI); there is no `--created-by` flag. Commands that hit admin-only routes (`accounts`, `cluster`, `backup`, `accounts rotate-secret`) need a credential carrying the `admin` scope; the API returns `403 credential missing required scope: admin` otherwise.

### CI / non-interactive authentication

If any of `SCHEDULER0_API_KEY`, `SCHEDULER0_API_SECRET` or `SCHEDULER0_ACCOUNT_ID` is set, the CLI uses the environment instead of the config file (all three must then be set, or the command fails with `SCHEDULER0_API_KEY, SCHEDULER0_API_SECRET, and SCHEDULER0_ACCOUNT_ID must all be set`).

| Variable | Purpose |
|----------|---------|
| `SCHEDULER0_API_KEY` | credential API key (required) |
| `SCHEDULER0_API_SECRET` | credential secret (required) |
| `SCHEDULER0_ACCOUNT_ID` | account id (required) |
| `SCHEDULER0_BASE_URL` | API endpoint, default `https://api.scheduler0.com` |
| `SCHEDULER0_ACTOR` | value recorded as `createdBy`/`modifiedBy`/`deletedBy`/`archivedBy`; required for every create/update/delete/archive command (`SCHEDULER0_ACTOR must be set to perform this action` otherwise) |
| `SCHEDULER0_EXPIRES_AT` | optional RFC3339 expiry; if unset the session is treated as valid and the server enforces expiry (401) |

Mint a long-lived credential once from an interactive session and store its output as CI secrets:

```bash
scheduler0 credentials create --scopes read,write,execute
```

## Configuration

`~/.scheduler0/config.json` is written by `scheduler0 login` and `scheduler0 local-executor register` (mode `0600`):

```json
{
  "base_url": "https://api.scheduler0.com",
  "app_url": "https://app.scheduler0.com",
  "api_key": "…",
  "api_secret": "…",
  "account_id": "…",
  "expires_at": "2026-07-07T12:00:00Z",
  "clerk_user_id": "user_…",
  "scopes": ["read", "write", "execute"],
  "local_executor": {
    "id": "42",
    "name": "my-machine",
    "command": "/path/to/handler.sh",
    "working_dir": "/path/to/dir"
  }
}
```

A file-based session is considered expired 30 seconds before `expires_at` (or immediately if `expires_at` is missing) and the CLI asks you to run `scheduler0 login`.

Precedence for the endpoint and account, highest first:

1. Global flags `--base-url` and `--account-id` (available on every command; several commands also accept a local `--account-id`).
2. `SCHEDULER0_*` environment variables (when a CI session is present).
3. `~/.scheduler0/config.json`.

Note that the API validates the credential against `X-Account-ID`, so `--account-id` only works with an account the credential actually belongs to.

The local executor also keeps a SQLite database at `~/.scheduler0/local-executor.db` (job cache and unreported executions).

## Output, exit codes, completion

- Read/create/update commands print the API's JSON response pretty-printed (usually the `{"success": ..., "data": ...}` envelope; `prompt`, `prompt classify` and `schedule` print the unwrapped result). Delete/archive commands and a few admin commands print a one-line confirmation. `credentials list --output table` is the only tabular output.
- Exit code is `0` on success and `1` on any error (usage errors, API errors, expired session). API errors are surfaced as `API error: <server response body>`.
- `scheduler0 --version` prints the version; `scheduler0 completion bash|zsh|fish|powershell` generates shell completions (`scheduler0 completion <shell> --help` shows how to install them).

## Command reference

`projects|jobs|executors|credentials list` accept `--limit` (default 10; the API rejects values above 100), `--offset` (default 0), `--order-by` (default `date_created`) and `--order-direction asc|desc` (default `desc`).

### Projects

```bash
scheduler0 projects list
scheduler0 projects get <project-id>
scheduler0 projects create --name "My Project" --description "What it is for"   # both required
scheduler0 projects update <project-id> --description "New description"          # only the description can change
scheduler0 projects delete <project-id>                                          # also deletes the project's jobs
```

`--order-by` accepts `id|name|description|date_created|account_id`.

### Jobs

```bash
scheduler0 jobs list [--project-id <id>]
scheduler0 jobs get <job-id>

scheduler0 jobs create \
  --project-id 123 \                 # required
  --executor-id 7 \
  --spec "0 30 * * * *" \            # cron spec; omit it for a one-time job that fires at --start-date
  --data '{"key": "value"}' \
  --timezone "UTC" \                 # default UTC
  [--timezone-offset <seconds>] [--start-date <RFC3339>] [--end-date <RFC3339>] \
  [--retry-max <n>] [--status active|inactive]

scheduler0 jobs update <job-id> --spec "0 0 * * * *" --status inactive   # same flags as create, none required
scheduler0 jobs delete <job-id>
```

`--spec` is a six-field cron expression with a leading seconds field (`sec min hour dom month dow`; `@hourly` / `@every 1h30m` also work). Always write all six fields: a five-field expression is accepted but read as `sec min hour dom month`, so `0 9 * * 1` means minute 9 of every hour in January, not Monday 09:00.

`jobs create` is asynchronous: the API answers `202` and `data` is a request id. Poll it with `scheduler0 async-tasks get <request-id>`.

### Executors

```bash
scheduler0 executors list
scheduler0 executors get <executor-id>

# Webhook executor
scheduler0 executors create \
  --name "webhook-executor" --type webhook_url \
  --webhook-url "https://example.com/webhook" --webhook-method POST \
  [--webhook-secret "secret"] [--description "..."] [--tags email,sales] [--payload-aggregation]

# Cloud function executor (all five cloud flags are required by the API)
scheduler0 executors create \
  --name "cloud-function" --type cloud_function \
  --cloud-provider aws --region us-west-1 \
  --cloud-resource-url "https://example.com/function" \
  --cloud-api-key "key" --cloud-api-secret "secret"

# Update replaces the whole definition: --name and --type are required and
# the type-specific fields must be passed again.
scheduler0 executors update <executor-id> \
  --name "renamed" --type webhook_url \
  --webhook-url "https://example.com/webhook" --webhook-method POST

scheduler0 executors delete <executor-id>

# Fire a synthetic job through the executor right now (no job/execution is created).
# Not supported for local executors.
scheduler0 executors test-invoke <executor-id> [--age 24h] [--execution-time <RFC3339>]
```

`--description` and `--tags` are what `scheduler0 schedule` uses to match an executor to a prompt. `--webhook-method` accepts `GET|POST|PUT|DELETE` (default `POST` on create). Secrets (`cloudApiKey`, `cloudApiSecret`, `webhookSecret`) are returned only in the create response. `--order-by` accepts `id|date_created|date_modified|created_by|modified_by|deleted_by`.

### Local executor

Run jobs on this machine with a command you control. Registration creates an executor of type `local` (`POST /local-executors`) and saves its id to `~/.scheduler0/config.json`; then `start` runs the long-lived poller. One local executor per machine; re-running `register` overwrites the previous one.

```bash
scheduler0 local-executor register --name "my-machine" --command "/path/to/handler.sh" [--working-dir /path/to/dir]
scheduler0 local-executor start [--poll-interval 1m]
```

What `start` does:

- Polls `GET /local-executors/{id}/jobs` every `--poll-interval` (default `1m`) and refreshes the executor's `command`/`workingDir` from `GET /executors/{id}` each time, so changes made in the web app take effect without restarting.
- Caches jobs in `~/.scheduler0/local-executor.db` and keeps scheduling from that cache while the API is unreachable.
- Computes each job's next run from its cron `spec`, `timezone`, `startDate`/`endDate` (a job without a `spec` runs once at `startDate`), and runs the command at that time.
- Runs the command **without a shell**: the string is split on whitespace, so no quoting, pipes or globbing. Wrap anything complex in a script (see `helloworld.sh`).
- Passes the job to the command **both** on stdin (the raw `data` string) and via environment variables `SCHEDULER0_JOB_ID`, `SCHEDULER0_JOB_DATA`, `SCHEDULER0_JOB_SPEC`, `SCHEDULER0_EXEC_UNIQUE_ID`. A zero exit code is reported as success, anything else as failed; stdout/stderr are logged to the terminal.
- Reports executions in batches (up to 200 rows per request) to `POST /local-executors/{id}/executions` every 5 minutes and on shutdown (`SIGINT`/`SIGTERM`). Rows that could not be reported are kept and retried on the next attempt.

Assign jobs to it with `scheduler0 jobs create --executor-id <id> ...`. Logs go to stderr with a `[local-executor]` prefix.

### Executions

```bash
scheduler0 executions [--start-date <RFC3339>] [--end-date <RFC3339>] [--project-id <id>] [--job-id <id>] [--limit 10] [--offset 0]
scheduler0 executions analytics --start-date 2024-01-01 --start-time 09:00   # UTC; per-minute counts
scheduler0 executions totals                                                 # scheduled/success/failed totals
scheduler0 executions cleanup-old-logs --account-id <id> --retention-months 6  # deletes older logs; execute scope
```

Execution `state` is `0` scheduled, `1` success, `2` failed.

### Credentials

Credentials carry `scopes` (`read`, `write`, `execute`, `admin`) and expire (90 days by default; `login` mints shorter-lived ones). `admin` can only be granted by an admin credential.

```bash
scheduler0 credentials list [--output json|table]
scheduler0 credentials get <credential-id>
scheduler0 credentials create [--scopes read,write] [--archived]   # default scopes: read,write,execute
scheduler0 credentials archive <credential-id>
scheduler0 credentials delete <credential-id>
```

`create` is the **only** time the secret is returned, as `plaintextSecret`; `apiKey` is the stable identifier. `--order-by` accepts `id|date_created|date_modified|created_by|modified_by|deleted_by|expires_at`.

There is no rotate command. To replace a credential: `credentials create`, roll the new `apiKey`/`plaintextSecret` out to clients, then `credentials archive <old-id>`.

`scheduler0 credentials update <id> [--archived]` sends `PUT /credentials/{id}`. Only the archived flag (and `modifiedBy`, filled from the signed-in user) can change; `apiKey`, `apiSecret`, `scopes` and `expiresAt` are fixed at creation and the server rejects attempts to change the key or secret with `400`. Running it without `--archived` un-archives the credential. `credentials archive` is the direct archive endpoint and is preferred for that purpose.

### Async tasks

```bash
scheduler0 async-tasks get <request-id>    # request id returned by `jobs create`; state 0 not started, 1 in progress, 2 success, 3 failed
```

### AI

```bash
# Generate job configurations from a prompt (nothing is created; consumes credits)
scheduler0 prompt --prompt "Send weekly reports every Monday at 9 AM" \
  [--purposes reporting,communication] [--events weekly_cycle] \
  [--recipients team@example.com] [--channels email] \
  [--timezone America/New_York] [--locale en]

# Run only the intent guardrail (no model call, no credits)
scheduler0 prompt classify --prompt "Send weekly reports every Monday at 9 AM" [--locale en]

# Generate AND create the jobs in one call (resolves/creates a project, picks an executor; consumes credits)
scheduler0 schedule --prompt "Send weekly reports every Monday at 9 AM" \
  [--purposes ...] [--events ...] [--recipients ...] [--channels ...] \
  [--timezone America/New_York] [--locale en] [--project-id <id>] [--executor-id <id>]

# Suggestions engine; request body is JSON from --file or stdin
scheduler0 suggestions analyze --file conversation.json
cat sendtime.json | scheduler0 suggestions time

# Account AI provider settings and model catalog
scheduler0 ai-settings get [--account-id <id>]
scheduler0 ai-settings upsert \
  --active-models '[{"provider":"openai","model":"gpt-4.1-mini"},{"provider":"anthropic","model":"claude-sonnet-4-5"}]' \
  [--openai-api-key ...] [--anthropic-api-key ...] [--openrouter-api-key ...] \
  [--bedrock-access-key-id ...] [--bedrock-secret-key ...] [--bedrock-region ...]
scheduler0 ai-models            # same as `scheduler0 ai-settings models`

# Prompt-request audit log
scheduler0 ai-prompt-requests [--provider openai] [--model ...] [--status ...] [--search ...] \
  [--start-date <RFC3339>] [--end-date <RFC3339>] [--order ASC|DESC] [--limit 25] [--offset 0]
```

`prompt` limits (checked locally and by the API): prompt ≤ 160 characters; `--purposes`, `--events`, `--recipients`, `--channels` ≤ 5 items of ≤ 36 characters each. `--timezone` is an IANA name (default `UTC`). When the intent guardrail rejects a prompt (HTTP 422) the CLI prints the classification (`decision`, `reason`) and exits `1`. Other AI errors you may see: `429` monthly quota exhausted, `402` platform AI credits exhausted, `409` (`schedule`) no executor could be matched, `503` (`classify`) classifier not configured.

`ai-settings upsert` returns only an acknowledgement; run `ai-settings get` to read back the stored settings (keys are masked).

### Features

```bash
scheduler0 features    # GET /features: list every feature the server knows about
```

### Accounts (admin scope; self-hosting)

```bash
scheduler0 accounts create --name "My Account"
scheduler0 accounts get <account-id>
scheduler0 accounts update <account-id> --name "Renamed"
scheduler0 accounts execution-count get <account-id>
scheduler0 accounts execution-count increase <account-id> --count 100
scheduler0 accounts ai-usage <account-id>
scheduler0 accounts tokens get <account-id>
scheduler0 accounts tokens add <account-id> --amount 1000
scheduler0 accounts feature add <account-id> --feature-id <id>
scheduler0 accounts feature remove <account-id> --feature-id <id>
scheduler0 accounts features-all add <account-id>
scheduler0 accounts features-all remove <account-id>

# Server SecretKey rotation
scheduler0 accounts generate-secret-key [--count 1]                 # offline; prints random 64-hex AES-256 keys
scheduler0 accounts rotate-secret --old-secret-key <OLD_HEX_KEY>    # POST /account/rotate-secret
```

For API-key callers, `<account-id>` must be the account the credential belongs to (`403` otherwise).

Rotation procedure: generate a new key, update `SecretKey` in the server's secrets source and reload/restart the server, then run `rotate-secret` with the **previous** key. The server decrypts every stored secret (credential secrets, executor cloud credentials, AI provider keys) with the old key and re-encrypts it with the loaded one, and prints how many rows were re-encrypted. Existing API keys keep working.

### Cluster (admin scope; self-hosting)

```bash
scheduler0 cluster list-nodes
scheduler0 cluster add-node --node-id <id> --node-address <raft-addr> --client-address <api-addr>
scheduler0 cluster remove-node --node-id <id>
scheduler0 cluster promote-node --node-id <id>
scheduler0 cluster demote-node --node-id <id>
scheduler0 cluster transfer-leadership
scheduler0 cluster force-rebuild --seed-node-id <id>
scheduler0 cluster add-self
scheduler0 cluster remove-self
scheduler0 cluster reset-raft            # destructive: clears local Raft state on the target node and exits it
scheduler0 cluster dump schedule-queue | job-executions-cache | job-queues | job-queue-versions
```

The `dump` endpoints are debug endpoints; their output shape is not a stable contract.

### Backup and restore (admin scope; self-hosting)

```bash
scheduler0 backup start                  # POST /cluster/backup -> 202 {status, requestId}
scheduler0 backup restore <file-name>    # POST /cluster/restore; S3 object key when S3 is configured, else a local path
```

### Healthcheck

```bash
scheduler0 healthcheck [--base-url https://api.example.com]
```

No authentication is sent. The base URL comes from the session/config, or from `--base-url` when you are not signed in.

## Troubleshooting

| Message | Cause / fix |
|---------|-------------|
| `not signed in: run 'scheduler0 login', or set SCHEDULER0_API_KEY/SCHEDULER0_API_SECRET/SCHEDULER0_ACCOUNT_ID for CI` | No `~/.scheduler0/config.json` and no env credential. |
| `session expired or missing: run 'scheduler0 login'` | `expires_at` in the config is in the past (or missing). Sign in again. |
| `SCHEDULER0_API_KEY, SCHEDULER0_API_SECRET, and SCHEDULER0_ACCOUNT_ID must all be set` | Only some of the CI variables are set. |
| `SCHEDULER0_ACTOR must be set to perform this action` | A create/update/delete/archive command was run with an env credential but no actor. |
| `no signed-in user found: run 'scheduler0 login'` | The config has a credential but no user id (e.g. hand-written). Sign in again. |
| `login timed out after 5m0s` | The browser redirect never arrived. Retry, or use `scheduler0 login --device` on remote machines. |
| `device code expired before it was approved` | Approve the code sooner, or re-run `scheduler0 login --device`. |
| `API error: {"success":false,"data":"credential missing required scope: admin"}` | The command needs an admin credential (accounts, cluster, backup, rotate-secret). |
| `API error: ...` with HTTP 401 | Key/secret/account mismatch, or the credential expired or was archived. |
| `failed to list ...: API error ...` after `--limit 150` | `projects|jobs|executors|credentials list` reject `limit` above 100. |
| `prompt not accepted by intent guardrail (decision=reject)` | The classifier rejected the prompt; the printed `reason` explains why. |
| `no local executor registered on this machine` | Run `scheduler0 local-executor register` first. |
| `base URL is required. Use --base-url flag or run 'scheduler0 login'` | `healthcheck` needs an endpoint when there is no config. |

## License

MIT — see `LICENSE`.
