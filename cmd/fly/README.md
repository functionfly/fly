# `fly` — FunctionFly Developer CLI

The `fly` CLI is the primary developer interface for FunctionFly.

## Install and upgrade

- **Install script (Linux/macOS):**  
  `curl -fsSL https://raw.githubusercontent.com/functionfly/functionfly/main/scripts/install.sh | bash`
- **Homebrew:** `brew tap functionfly/tap && brew install fly` (when tap is configured)
- **From source:** `go build -o bin/fly ./cmd/fly` (binary at `bin/fly`)

Upgrade: run the install script again with `VERSION=latest`, or `brew upgrade fly` / `scoop update fly` / `choco upgrade fly`. Or run `fly self-update` to print instructions.

See [packaging/README.md](../../packaging/README.md) for Windows and release artifacts.

## Quick Start

```bash
fly login                    # Authenticate
fly init my-function         # Scaffold a new function
cd my-function
fly dev                      # Run locally at http://localhost:8787
fly publish                  # Publish to the global registry
fly test                     # Test the deployed function
```

## Configuration

Precedence (highest first):

1. **Environment variables** (`FFLY_*`)
2. **Global config file** `~/.functionfly/config.yaml`
3. **Defaults**

| Variable | Description |
|----------|-------------|
| `FFLY_API_URL` | API base URL (e.g. `https://api.functionfly.com` or `http://localhost:8080`) |
| `FFLY_API_TIMEOUT` | Request timeout (e.g. `30s`) |
| `FFLY_DEV_EMAIL` / `FFLY_DEV_PASSWORD` | Dev login (with `fly login --dev`) |
| `FFLY_DEV_LOGIN=1` | Force dev email/password login |
| `FFLY_TOKEN` | Bearer token (overrides stored credentials) |
| `FFLY_TELEMETRY` | Set to `0`, `false`, or `no` to disable telemetry |
| `FFLY_CONFIG` | Path to config file (overrides `~/.functionfly/config.yaml`) |

- View current config: `fly config` or `fly config view`
- Reset to defaults: `fly config reset`

Credentials (after login) are stored in `~/.functionfly/credentials.json`.

## Commands

| Command | Description |
|---------|-------------|
| `fly login` | OAuth login (GitHub or Google) |
| `fly whoami` | Show current user |
| `fly logout` | Clear credentials |
| `fly config` | View or reset global config |
| `fly self-update` | Print upgrade instructions |
| `fly init <name>` | Scaffold a new function project |
| `fly dev` | Run locally with hot reload |
| `fly publish` | Publish to registry |
| `fly publish --build` | Build then publish |
| `fly test` | Test deployed function |
| `fly update patch` | Bump function version (patch/minor/major) |
| `fly stats` | View usage statistics |
| `fly logs` | View recent logs |
| `fly logs --follow` | Stream live logs |
| `fly rollback` | Roll back to previous version |
| `fly env list/set/get/unset` | Manage environment variables |
| `fly secrets list/set/unset` | Manage secrets |
| `fly completion bash/zsh/fish/powershell` | Shell completion |

## JSON Output

All commands support `--json` for CI/CD:

```bash
fly publish --json
fly stats --json
fly whoami --json
```
