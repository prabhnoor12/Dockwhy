# dockwhy

`dockwhy` explains why a Docker container stopped:

```text
dockwhy my-api
```

Check the installed binary with `dockwhy --version`.

It inspects the container exit code, OOM-kill flag, health-check state, restart count, resource limits, writable-layer size, Docker lifecycle events, and the last 50 log lines. Use `--tail N` to change the number of log lines, `--json` for automation, or `--no-events` when event access is unavailable.

## Install and run

```text
go install github.com/dockwhy/dockwhy/cmd/dockwhy@latest
dockwhy my-api
```

Each Docker command has a 10-second timeout by default. Configure it with `--timeout 30s`; use `--events-since 2h` to change the lifecycle-event lookback window.

Available flags:

```text
--tail N          show up to N recent log lines (maximum 5000)
--no-logs         skip log collection and output when logs are sensitive
--timeout D       timeout for each Docker command, for example 30s
--events-since D  lifecycle-event lookback, for example 2h
--no-events       skip lifecycle events when Docker event access is unavailable
--json            emit stable machine-readable JSON
```

The Docker CLI must be installed and connected to the daemon. Docker contexts and normal Docker environment variables are respected.

For running containers, dockwhy also reports a point-in-time resource snapshot from `docker stats`. If the container belongs to Docker Compose, the report identifies its project and service and suggests a service-level log command.

## Install a release

Tagged releases publish archives for Linux, macOS, and Windows, with SHA-256 checksums, on the GitHub Releases page. Download the archive matching your operating system and CPU architecture, extract `dockwhy` (or `dockwhy.exe`), and place it on your `PATH`.

To publish a release from the repository, create and push a semantic-version tag:

```text
git tag v0.1.0
git push origin v0.1.0
```

The release workflow runs GoReleaser and creates the cross-platform archives automatically.

## Project structure

```text
cmd/dockwhy/          executable entrypoint
internal/cli/         flags and command orchestration
internal/docker/      Docker CLI adapter and normalized models
internal/diagnosis/   explainable diagnosis rules
internal/output/      human and JSON reporters
```
