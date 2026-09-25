# dockwhy

[![CI](https://github.com/prabhnoor12/dockwhy/actions/workflows/ci.yml/badge.svg)](https://github.com/prabhnoor12/dockwhy/actions/workflows/ci.yml)
[![Latest release](https://img.shields.io/github/v/release/prabhnoor12/dockwhy)](https://github.com/prabhnoor12/dockwhy/releases/latest)
[![License](https://img.shields.io/badge/license-Apache--2.0-blue.svg)](LICENSE)

`dockwhy` explains why a Docker container stopped. It combines the container's exit status, Docker runtime state, health checks, restart history, resource limits, lifecycle events, and recent logs into one readable diagnosis.

```text
dockwhy my-api
```

It can also produce JSON for scripts and CI systems:

```bash
dockwhy --json my-api > diagnosis.json
```

## Contents

- [Features](#features)
- [Requirements](#requirements)
- [Installation](#installation)
- [Usage](#usage)
- [Options](#options)
- [Analysis Modes](#analysis-modes)
- [Kubernetes Support](#kubernetes-support)
- [Output Formats and Integrations](#output-formats-and-integrations)
- [Custom Diagnosis Rules](#custom-diagnosis-rules)
- [Docker Access and Security](#docker-access-and-security)
- [Exit Codes](#exit-codes)
- [IDE Extensions and Editor Integration](#ide-extensions-and-editor-integration)
- [Development](#development)
- [Releases](#releases)
- [Contributing](#contributing)
- [License](#license)

## Features

- **Single-container diagnosis** — combines exit status, runtime state, health checks, restart history, resource limits, lifecycle events, and logs into ranked findings with severity, confidence, evidence, and advice
- **Docker Compose project-wide diagnosis** — diagnose all containers in a stack concurrently
- **Crash trend analysis** — detect crash loops, group by exit code, show patterns over time
- **Smart log extraction** — identify panics, fatal errors, OOM, segfaults, connection failures, Java stack traces, and timeouts
- **Resource sizing recommendations** — memory, CPU, and PID limit analysis from `docker stats`
- **Container configuration drift detection** — compare two containers and highlight critical differences
- **Exit code knowledge base** — look up causes and fixes for any exit code
- **Custom diagnosis rules** — load JSON rule files for org-specific patterns
- **Continuous watch mode** — diagnose at a fixed interval for real-time monitoring
- **Kubernetes pod awareness** — resolve pod names to container IDs via `kubectl`
- **Markdown incident reports** — generate full postmortem documents
- **Integration outputs** — PagerDuty, Slack, Prometheus, and generic webhook formats
- **Zero external dependencies** — built entirely on the Go standard library
- **Cross-platform** — Linux, macOS, and Windows (amd64 and arm64)

## Requirements

- The Docker CLI must be installed and available on `PATH`.
- The Docker CLI must be connected to a reachable daemon.
- The current user must have permission to inspect the container and read its logs.

`dockwhy` uses the same Docker CLI configuration as the rest of your system. Docker contexts and standard Docker environment variables such as `DOCKER_HOST` are respected.

Check the connection before troubleshooting:

```bash
docker info
```

## Installation

### Install with Go

If Go is installed, install the latest version directly from this repository:

```bash
go install github.com/prabhnoor12/dockwhy/cmd/dockwhy@latest
```

Make sure Go's binary directory is on your `PATH`, then verify the installation:

```bash
dockwhy --version
```

### Install a release binary

Download the archive for your operating system and CPU architecture from the [GitHub Releases page](https://github.com/prabhnoor12/dockwhy/releases/latest). Extract `dockwhy` (or `dockwhy.exe` on Windows) and place it somewhere on your `PATH`.

Release archives are currently provided for:

- Linux: `amd64`, `arm64`
- macOS: `amd64`, `arm64`
- Windows: `amd64`, `arm64`

Verify the downloaded archive with the SHA-256 values in `checksums.txt` before installing it.

### Install with Homebrew

After adding the project tap, install the latest release on macOS or Linux:

```bash
brew tap prabhnoor12/tap
brew install dockwhy
```

The fully qualified form is also supported:

```bash
brew install prabhnoor12/tap/dockwhy
```

### Install with Scoop

On Windows, add the project bucket once and install `dockwhy`:

```powershell
scoop bucket add dockwhy https://github.com/prabhnoor12/scoop-bucket
scoop install dockwhy
```

### Package-manager setup for maintainers

The Homebrew and Scoop commands above require these public repositories to exist:

- `prabhnoor12/homebrew-tap`, containing the generated Homebrew formula
- `prabhnoor12/scoop-bucket`, containing the generated Scoop manifest

The release workflow publishes updates to both repositories through GoReleaser. Configure these repository secrets in `prabhnoor12/dockwhy`:

- `HOMEBREW_TAP_TOKEN`: a GitHub token with write access to `prabhnoor12/homebrew-tap`
- `SCOOP_BUCKET_TOKEN`: a GitHub token with write access to `prabhnoor12/scoop-bucket`

The regular `GITHUB_TOKEN` only publishes the GitHub Release for this repository; it cannot normally write to the separate tap and bucket repositories. Until these secrets are configured, the package-manager publishing steps are skipped and regular GitHub Releases continue to work.

### Build from source

```bash
git clone https://github.com/prabhnoor12/dockwhy.git
cd dockwhy
go build -o dockwhy ./cmd/dockwhy
```

## Usage

Diagnose a stopped or running container by name or ID:

```bash
dockwhy my-api
dockwhy 0123456789ab
```

### Basic examples

```bash
# Include more recent log lines
dockwhy --tail 200 my-api

# Look further back in the Docker event history
dockwhy --events-since 2h my-api

# Avoid collecting logs when they may contain sensitive data
dockwhy --no-logs my-api

# Skip events when Docker event access is unavailable
dockwhy --no-events my-api

# Produce machine-readable output
dockwhy --json my-api
```

### Advanced examples

```bash
# Diagnose all containers in a Docker Compose project
dockwhy --project my-stack

# Show crash patterns across restart history
dockwhy --trend my-api

# Extract error patterns instead of raw log tail
dockwhy --smart-logs my-api

# Get resource sizing recommendations
dockwhy --resources my-api

# Compare two containers for configuration drift
dockwhy --compare my-api-old my-api

# Look up an exit code
dockwhy --exit-code 137

# Load custom diagnosis rules
dockwhy --rules ./my-rules.json my-api

# Continuously monitor at 30-second intervals
dockwhy --watch 30s my-api

# Generate a Markdown incident report
dockwhy --report incident.md my-api

# Output in PagerDuty format
dockwhy --output pagerduty my-api

# Diagnose a Kubernetes pod
dockwhy --kube my-pod --kube-namespace production
```

For running containers, the report includes a point-in-time resource snapshot from `docker stats`. For Docker Compose containers, it identifies the project and service and suggests a service-level log command.

## Options

```text
--tail N             show up to N recent log lines (maximum 5000; default 50)
--no-logs            skip log collection and output
--timeout D          timeout for each Docker command (default 10s)
--events-since D     lifecycle-event lookback window (default 24h)
--no-events          skip Docker lifecycle event lookup
--json               emit machine-readable JSON
--project NAME       diagnose all containers in a Docker Compose project
--trend              show crash patterns across restart history
--smart-logs         extract error patterns from logs instead of raw tail
--resources          show resource sizing recommendations from docker stats
--compare CONTAINER  compare configuration with another container to find drift
--exit-code N        look up information about a specific exit code
--rules FILE         path to a JSON file with custom diagnosis rules
--watch D            continuously diagnose at this interval (e.g. 5s, 1m); 0 disables
--report FILE        write a Markdown incident report to this file path (use - for stdout)
--output FORMAT      output format for integrations: pagerduty, slack, prometheus, webhook
--kube               treat the container argument as a Kubernetes pod name
--kube-namespace NS  Kubernetes namespace (default: auto-detect or "default")
--kube-container NS  specific container name in a multi-container pod
--version            print the dockwhy version
```

Each Docker command has its own timeout. Increase it for slow remote daemons:

```bash
dockwhy --timeout 30s my-api
```

## Analysis Modes

### Docker Compose project-wide diagnosis

Diagnose all containers in a Docker Compose stack concurrently:

```bash
dockwhy --project my-stack
```

This discovers all containers belonging to the Compose project, diagnoses each one, and produces a combined report showing which services are healthy and which have issues.

### Crash trend analysis

Detect crash loops and patterns across restart history:

```bash
dockwhy --trend my-api
```

Groups restarts by exit code, shows frequency over time, and identifies whether the container is in a crash loop.

### Smart log extraction

Instead of showing a raw log tail, extract meaningful error patterns:

```bash
dockwhy --smart-logs my-api
```

Identifies:
- Panics (Go, Python, Node.js, Ruby)
- Fatal errors and unhandled exceptions
- Out-of-memory (OOM) kills
- Segmentation faults
- Connection failures and timeouts
- Java stack traces
- Database errors

### Resource sizing recommendations

Analyze container resource usage and get sizing recommendations:

```bash
dockwhy --resources my-api
```

Provides recommendations for:
- Memory limits based on current usage
- CPU limits based on current usage
- PID limit analysis

### Container configuration drift detection

Compare two containers to find configuration differences:

```bash
dockwhy --compare my-api-old my-api
```

Highlights differences in environment variables, resource limits, volume mounts, network settings, and other configuration that could explain behavioral changes.

### Exit code lookup

Look up detailed information about any exit code:

```bash
dockwhy --exit-code 137
```

Returns the meaning, common causes, and suggested fixes for the exit code. This is also integrated into diagnosis output automatically.

### Continuous watch mode

Monitor a container continuously at a fixed interval:

```bash
dockwhy --watch 30s my-api
```

Re-runs the diagnosis every 30 seconds. Useful for real-time monitoring during incident response.

## Kubernetes Support

`dockwhy` can diagnose containers running in Kubernetes by resolving pod names to container IDs via `kubectl`:

```bash
# Diagnose a container in a pod
dockwhy --kube my-pod

# Specify the namespace
dockwhy --kube my-pod --kube-namespace production

# Diagnose a specific container in a multi-container pod
dockwhy --kube my-pod --kube-container sidecar
```

Features:
- Automatic namespace detection from downward API or `--kube-namespace`
- Multi-container pod support with `--kube-container`
- Pod metadata enrichment in output (node, phase, QoS class, labels, container statuses)

Kubernetes support requires `kubectl` to be installed and configured with access to the cluster.

## Output Formats and Integrations

### Human-readable text (default)

```bash
dockwhy my-api
```

### Machine-readable JSON

```bash
dockwhy --json my-api
```

### Markdown incident report

Generate a full postmortem document:

```bash
dockwhy --report incident.md my-api
dockwhy --report - my-api  # write to stdout
```

### Integration output formats

Produce payloads for external systems:

```bash
# PagerDuty Events API v2
dockwhy --output pagerduty my-api

# Slack incoming webhook
dockwhy --output slack my-api

# Prometheus text exposition metrics
dockwhy --output prometheus my-api

# Generic JSON webhook
dockwhy --output webhook my-api
```

These outputs can be piped directly to the respective APIs or used in automation scripts.

## Custom Diagnosis Rules

Load custom JSON rule files to add organization-specific patterns and advice:

```bash
dockwhy --rules ./my-rules.json my-api
```

Rule file format:

```json
[
  {
    "name": "custom-oom",
    "description": "Custom OOM detection for Java apps",
    "condition": {
      "exit_code": 137,
      "log_contains": "java.lang.OutOfMemoryError"
    },
    "severity": "critical",
    "advice": "Increase JVM heap size with -Xmx or add more memory to the container"
  }
]
```

Rules are evaluated alongside built-in diagnostics and appear in the output with their custom severity and advice.

## Docker Access and Security

`dockwhy` reads Docker metadata, events, statistics, and logs; it does not modify or restart containers. Access to the Docker daemon is still sensitive. On many systems, access to the Docker socket is effectively equivalent to root access on the host.

When logs may contain credentials, tokens, or personal data, use `--no-logs`. When using a remote Docker daemon, configure a secure Docker context or authenticated TLS connection rather than exposing an unauthenticated TCP socket.

If Docker reports a permission error:

1. Run `docker info` as the same user that will run `dockwhy`.
2. Check the selected context with `docker context show`.
3. Check `DOCKER_HOST` and other Docker environment variables.
4. Use `--no-events` if the daemon permits container inspection but denies event access.

## Exit Codes

- `0`: diagnosis completed and output was written.
- `1`: Docker could not be queried or output could not be written.
- `2`: invalid command-line arguments or missing container argument.

## IDE Extensions and Editor Integration

Use dockwhy directly from your favorite editor:

### VS Code
Full-featured extension with sidebar, commands, and rich webview panels.

```bash
cd ide-extensions/vscode
npm install
npm run package
code --install-extension dockwhy-0.1.0.vsix
```

Features: container tree view, one-click diagnosis, Docker Compose support, crash trends, resource recommendations, incident reports.

### Neovim
Lua plugin for modern Neovim (0.7+):

```lua
-- lazy.nvim
{ "prabhnoor12/dockwhy", dir = "ide-extensions/neovim", config = function() require("dockwhy").setup() end }
```

Commands: `:Dockwhy`, `:DockwhyTrend`, `:DockwhyResources`, `:DockwhyReport`

### Vim
Vimscript plugin for Vim 8+:

```vim
" vim-plug
Plug 'prabhnoor12/dockwhy', { 'rtp': 'ide-extensions/vim' }
```

### JetBrains IDEs
External tools configuration for IntelliJ, WebStorm, PyCharm, GoLand, etc. See `ide-extensions/jetbrains/README.md` for setup instructions.

### Shell Completions
Tab completion for container names, flags, and output formats:

**Bash**:
```bash
source completions/dockwhy.bash  # Add to ~/.bashrc
```

**Zsh**:
```bash
fpath=(completions $fpath)  # Add to ~/.zshrc
autoload -U compinit && compinit
```

**Fish**:
```bash
cp completions/dockwhy.fish ~/.config/fish/completions/
```

See `ide-extensions/README.md` for detailed documentation.

## Development

The repository is organized as follows:

```text
cmd/dockwhy/          executable entrypoint
internal/cli/         flags and command orchestration
internal/diagnosis/   explainable diagnosis rules
internal/docker/      Docker CLI adapter and normalized models
internal/kube/        Kubernetes CLI adapter and pod types
internal/output/      human, JSON, and integration reporters
internal/format/      byte formatting utilities
internal/version/     version variable (set by ldflags at build time)
```

Run the checks locally:

```bash
make build    # compile
make test     # run all tests with race detector
make lint     # run golangci-lint
make vet      # run go vet
make test-coverage  # run tests with coverage report
```

Or directly with Go commands:

```bash
go test -race ./...
go vet ./...
go build ./...
```

Integration tests require a running Docker daemon:

```bash
go test -tags integration ./internal/integration/ -v
```

## Releases

Releases are built by GoReleaser and published to [GitHub Releases](https://github.com/prabhnoor12/dockwhy/releases). To publish a release, create and push a semantic-version tag:

```bash
git tag v0.1.0
git push origin v0.1.0
```

The release workflow creates cross-platform archives, a SHA-256 checksum file, Homebrew formula, and Scoop manifest.

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for development setup, code style guidelines, and the pull request process.

## License

`dockwhy` is licensed under the [Apache License 2.0](LICENSE).
