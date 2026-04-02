# ffly — FunctionFly CLI

> Go from idea to global API in under 60 seconds.

`ffly` is the official command-line tool for [FunctionFly](https://functionfly.com). Write a function, publish it, and call it from anywhere — no infra required.

## Install

**macOS / Linux (one-liner)**
```bash
curl -fsSL https://raw.githubusercontent.com/functionfly/fly/main/scripts/install.sh | bash
```

**Homebrew**
```bash
brew tap functionfly/tap
brew install ffly
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
ffly login

# Scaffold a new function
ffly init slugify

# Run it locally
cd slugify && ffly dev

# Test it
ffly test

# Publish to the registry
ffly publish

# Deploy to production
ffly deploy --env production
```

Your function is now live at `https://api.functionfly.com/fx/<you>/slugify`.

---

## Commands

### Core

| Command | Description |
|---------|-------------|
| `ffly login` | Authenticate with FunctionFly |
| `ffly logout` | Clear stored credentials |
| `ffly whoami` | Show the current authenticated user |
| `ffly init <name>` | Scaffold a new function project |
| `ffly dev` | Run the function locally with hot reload |
| `ffly test` | Run tests against the local runtime |
| `ffly publish` | Publish the function to the registry |
| `ffly publish-batch` | Batch publish multiple functions |
| `ffly update` | Bump the function version |
| `ffly rollback` | Roll back to a previous version |

### Deployment

| Command | Description |
|---------|-------------|
| `ffly deploy` | Publish and promote to staging/production or start canary |
| `ffly canary` | Manage canary deployments (`start`, `status`, `promote`, `rollback`, `cancel`, `history`) |
| `ffly health` | Check deployed function health (supports `--watch`) |
| `ffly logs` | Stream live execution logs |
| `ffly stats` | View invocation stats |

### Configuration & Secrets

| Command | Description |
|---------|-------------|
| `ffly config` | Show/edit CLI configuration |
| `ffly env` | Manage environment variables (`list`, `set`, `get`, `unset`) |
| `ffly secrets` | Manage secrets (`list`, `set`, `unset`) |
| `ffly schedule` | Manage scheduled function executions (`set`, `list`, `get`, `remove`, `presets`, `trigger`) |

### Advanced

| Command | Description |
|---------|-------------|
| `ffly compile` | Compile functions to various formats (`python`, `rust`) |
| `ffly flypy` | FlyPy — Deterministic Python Compiler (`build`, `deploy`, `local`) |
| `ffly dre` | DRE (Deterministic Reliable Execution) and FXCERT operations |
| `ffly backend` | Manage execution backends (`add`, `list`, `remove`) |
| `ffly manifest ensure-descriptions` | Add descriptions to functionfly.jsonc files |

### Utilities

| Command | Description |
|---------|-------------|
| `ffly completion` | Generate shell completions (`bash`, `zsh`, `fish`, `powershell`) |
| `ffly doctor` | Run environment diagnostics |
| `ffly self-update` | Update the CLI itself |
| `ffly changelog` | Show the CLI changelog |

### Global flags

All commands support:
- `--debug` — Enable full debug output
- `--verbose` / `-v` — Enable verbose API calls
- `--trace` — Enable HTTP trace with request/response bodies
- `--format` / `-o` — Output format: `table`, `json` (default: `table`)
- `--version` — Show CLI version

---

## Configuration

The CLI reads config from (in order of precedence):

1. Environment variables (prefix `FFLY_`)
2. `functionfly.jsonc` in the current directory
3. `~/.functionfly/config.yaml`

### Environment variables

| Variable | Default | Description |
|----------|---------|-------------|
| `FFLY_API_URL` | `https://api.functionfly.com` | API endpoint |
| `FFLY_TOKEN` | — | Auth token (skips `ffly login`) |
| `FFLY_CONFIG` | `~/.functionfly/config.yaml` | Config file path |

### functionfly.jsonc

The function manifest uses JSONC format (JSON with comments):

```jsonc
{
  "name": "my-function",           // required, lowercase + hyphens
  "version": "1.0.0",              // required, semver
  "runtime": "node20",             // node18, node20, python3.11, deno, bun, rust, browser-wasm
  "entry": "index.ts",             // optional, auto-detected
  "public": true,                  // default: true
  "deterministic": false,          // default: false
  "cache_ttl": 3600,               // default: 3600, max: 86400
  "timeout_ms": 5000,              // default: 5000, max: 30000
  "memory_mb": 128,                // 128, 256, 512, or 1024
  "description": "My function",    // max 500 chars
  "dependencies": {},              // npm/python deps
  "env": {},                       // runtime env vars
  "typeCheck": true,               // TypeScript type checking
  "tsConfig": "tsconfig.json",     // custom tsconfig path
  "strictMode": false,             // strict TypeScript
  "skipTypeCheck": false           // skip type checking
}
```

---

## Development

**Requirements:** Go ≥ 1.24

```bash
# Clone
git clone https://github.com/functionfly/fly.git
cd fly

# Build
go build -o ffly ./cmd/ffly

# Test
go test ./...

# Test with race detector
go test -race ./...

# Integration tests (requires Rust toolchain with wasm32-wasi target)
go test -tags integration ./...

# Lint
golangci-lint run

# Run locally (no install)
./ffly --help
```

### Project structure

```
cmd/fly/           # CLI entry point and commands
internal/
├── bundler/       # TypeScript/JS/Python/WASM bundling
├── cli/           # HTTP client and config
├── credentials/   # Credential persistence (OS keychain)
├── flypy/         # FlyPy Python-to-Wasm compiler
├── manifest/      # functionfly.jsonc parser
├── testing/       # Test runner and validator
├── version/       # Build version injection
└── watcher/       # File watcher for hot reload
scripts/           # Install scripts (bash, PowerShell)
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

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.

---

## License

[Apache 2.0](LICENSE)
