# `ffly` — FunctionFly Developer CLI

The `ffly` CLI is the primary developer interface for FunctionFly.

## Install and upgrade

- **Install script (Linux/macOS):**  
  `curl -fsSL https://raw.githubusercontent.com/functionfly/functionfly/main/scripts/install.sh | bash`
- **Homebrew:** `brew tap functionfly/tap && brew install ffly` (when tap is configured)
- **From source:** `go build -o bin/ffly ./cmd/ffly` (binary at `bin/ffly`)

Upgrade: run the install script again with `VERSION=latest`, or `brew upgrade ffly` / `scoop update ffly` / `choco upgrade ffly`. Or run `ffly self-update` to print instructions.

See [packaging/README.md](../../packaging/README.md) for Windows and release artifacts.

## Quick Start

```bash
ffly login                   # Authenticate
ffly init my-function        # Scaffold a new function
cd my-function
ffly dev                     # Run locally at http://localhost:8787
ffly publish                 # Publish to the global registry
ffly test                    # Test the deployed function
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
| `FFLY_DEV_EMAIL` / `FFLY_DEV_PASSWORD` | Dev login (with `ffly login --dev`) |
| `FFLY_DEV_LOGIN=1` | Force dev email/password login |
| `FFLY_TOKEN` | Bearer token (overrides stored credentials) |
| `FFLY_TELEMETRY` | Set to `0`, `false`, or `no` to disable telemetry |
| `FFLY_CONFIG` | Path to config file (overrides `~/.functionfly/config.yaml`) |

- View current config: `ffly config` or `ffly config view`
- Reset to defaults: `ffly config reset`

Credentials (after login) are stored in `~/.functionfly/credentials.json`.

## Commands

| Command | Description |
|---------|-------------|
| `ffly login` | OAuth login (GitHub or Google) |
| `ffly whoami` | Show current user |
| `ffly logout` | Clear credentials |
| `ffly config` | View or reset global config |
| `ffly self-update` | Print upgrade instructions |
| `ffly init <name>` | Scaffold a new function project |
| `ffly dev` | Run locally with hot reload |
| `ffly publish` | Publish to registry |
| `ffly publish --build` | Build then publish |
| `ffly test` | Test deployed function |
| `ffly update patch` | Bump function version (patch/minor/major) |
| `ffly stats` | View usage statistics |
| `ffly logs` | View recent logs |
| `ffly logs --follow` | Stream live logs |
| `ffly rollback` | Roll back to previous version |
| `ffly env list/set/get/unset` | Manage environment variables |
| `ffly secrets list/set/unset` | Manage secrets |
| `ffly completion bash/zsh/fish/powershell` | Shell completion |

## JSON Output

All commands support `--json` for CI/CD:

```bash
ffly publish --json
ffly stats --json
ffly whoami --json
```
