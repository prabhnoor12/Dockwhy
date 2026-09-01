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

## Requirements

- The Docker CLI must be installed and available on `PATH`.
- The Docker CLI must be connected to a reachable daemon.
- The current user must have permission to inspect the container and read its logs.

`dockwhy` uses the same Docker CLI configuration as the rest of your system. Docker contexts and standard Docker environment variables such as `DOCKER_HOST` are respected.

Check the connection before troubleshooting:

```bash
docker info
```

## Install

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

Useful examples:

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

For running containers, the report includes a point-in-time resource snapshot from `docker stats`. For Docker Compose containers, it identifies the project and service and suggests a service-level log command.

## Options

```text
--tail N          show up to N recent log lines (maximum 5000; default 50)
--no-logs         skip log collection and output
--timeout D       timeout for each Docker command (default 10s)
--events-since D  lifecycle-event lookback window (default 24h)
--no-events       skip Docker lifecycle event lookup
--json            emit machine-readable JSON
--version         print the dockwhy version
```

Each Docker command has its own timeout. Increase it for slow remote daemons, for example:

```bash
dockwhy --timeout 30s my-api
```

## Docker access and security

`dockwhy` reads Docker metadata, events, statistics, and logs; it does not modify or restart containers. Access to the Docker daemon is still sensitive. On many systems, access to the Docker socket is effectively equivalent to root access on the host.

When logs may contain credentials, tokens, or personal data, use `--no-logs`. When using a remote Docker daemon, configure a secure Docker context or authenticated TLS connection rather than exposing an unauthenticated TCP socket.

If Docker reports a permission error:

1. Run `docker info` as the same user that will run `dockwhy`.
2. Check the selected context with `docker context show`.
3. Check `DOCKER_HOST` and other Docker environment variables.
4. Use `--no-events` if the daemon permits container inspection but denies event access.

## Exit codes

- `0`: diagnosis completed and output was written.
- `1`: Docker could not be queried or output could not be written.
- `2`: invalid command-line arguments or missing container argument.

## Development

The repository is organized as follows:

```text
cmd/dockwhy/          executable entrypoint
internal/cli/         flags and command orchestration
internal/docker/      Docker CLI adapter and normalized models
internal/diagnosis/   explainable diagnosis rules
internal/output/      human and JSON reporters
```

Run the checks locally:

```bash
go test ./...
go vet ./...
go build ./...
```

## Releases

Releases are built by GoReleaser and published to [GitHub Releases](https://github.com/prabhnoor12/dockwhy/releases). To publish a release, create and push a semantic-version tag:

```bash
git tag v0.1.0
git push origin v0.1.0
```

The release workflow creates cross-platform archives and a SHA-256 checksum file.

## License

`dockwhy` is licensed under the [Apache License 2.0](LICENSE).
