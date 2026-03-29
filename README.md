# fly — FunctionFly CLI

> Go from idea to global API in under 60 seconds.

`fly` is the official command-line tool for [FunctionFly](https://functionfly.com). Write a function, publish it, and call it from anywhere — no infra required.

## Install

**macOS / Linux (one-liner)**
```bash
curl -fsSL https://raw.githubusercontent.com/functionfly/fly/main/scripts/install.sh | bash
```

**Homebrew**
```bash
brew tap functionfly/tap
brew install fly
```

**Windows (PowerShell)**
```powershell
iwr -useb https://raw.githubusercontent.com/functionfly/fly/main/scripts/install.ps1 | iex
```

**Download directly** — see [Releases](https://github.com/functionfly/fly/releases)

---

## Quick start

```bash
# Log in
fly login

# Scaffold a new function
fly init slugify

# Run it locally
cd slugify && fly dev

# Test it
fly test

# Publish to the registry
fly publish
```

Your function is now live at `https://api.functionfly.com/<you>/slugify`.

---

## Commands

| Command | Description |
|---------|-------------|
| `fly login` | Authenticate with FunctionFly |
| `fly logout` | Clear stored credentials |
| `fly init <name>` | Scaffold a new function |
| `fly dev` | Run the function locally with hot reload |
| `fly test` | Run tests against the local runtime |
| `fly publish` | Publish the function to the registry |
| `fly rollback` | Roll back to a previous version |
| `fly logs` | Stream live logs |
| `fly stats` | View invocation stats |
| `fly update` | Bump the function version |
| `fly config` | Show/edit CLI configuration |
| `fly whoami` | Show the current authenticated user |
| `fly completion` | Generate shell completions |

---

## Configuration

The CLI reads config from (in order of precedence):

1. Environment variables (prefix `FFLY_`)
2. `functionfly.jsonc` in the current directory
3. `~/.functionfly/config.yaml`

Key env vars:

| Variable | Default | Description |
|----------|---------|-------------|
| `FFLY_API_URL` | `https://api.functionfly.com` | API endpoint |
| `FFLY_TOKEN` | — | Auth token (skips `fly login`) |
| `FFLY_CONFIG` | `~/.functionfly/config.yaml` | Config file path |

---

## Development

**Requirements:** Go ≥ 1.25

```bash
# Clone
git clone https://github.com/functionfly/fly.git
cd fly

# Build
go build -o fly ./cmd/fly

# Test
go test ./...

# Run locally (no install)
./fly --help
```

### Release process

Tags trigger the GoReleaser workflow automatically:

```bash
git tag v1.2.3
git push origin v1.2.3
```

GoReleaser produces:
- Linux/macOS/Windows binaries (amd64 + arm64)
- `.deb`, `.rpm`, `.apk` packages
- Homebrew formula update
- GitHub Release with changelog

---

## License

[Apache 2.0](LICENSE)
