# dockwhy

`dockwhy` explains why a Docker container stopped:

```text
dockwhy my-api
```

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
--timeout D       timeout for each Docker command, for example 30s
--events-since D  lifecycle-event lookback, for example 2h
--no-events       skip lifecycle events when Docker event access is unavailable
--json            emit stable machine-readable JSON
```

The Docker CLI must be installed and connected to the daemon. Docker contexts and normal Docker environment variables are respected.

## Project structure

```text
cmd/dockwhy/          executable entrypoint
internal/cli/         flags and command orchestration
internal/docker/      Docker CLI adapter and normalized models
internal/diagnosis/   explainable diagnosis rules
internal/output/      human and JSON reporters
```
