# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.1.0] - 2026-09-25

### Added

#### Core diagnosis
- Single-container diagnosis combining exit status, Docker runtime state, health checks, restart history, resource limits, lifecycle events, and recent logs
- JSON output mode (`--json`) for scripts and CI pipelines
- Human-readable text output with severity, confidence, evidence, and actionable advice

#### Analysis modes
- Docker Compose project-wide diagnosis (`--project`) — diagnose all containers in a stack concurrently
- Crash trend analysis (`--trend`) — detect crash loops, group by exit code, show patterns over time
- Smart log extraction (`--smart-logs`) — identify panics, fatal errors, OOM, segfaults, connection failures, Java stack traces, and timeouts
- Resource sizing recommendations (`--resources`) — memory, CPU, and PID limit analysis from `docker stats`
- Container configuration drift detection (`--compare`) — compare two containers and highlight critical differences
- Exit code knowledge base (`--exit-code`) — look up causes and fixes for any exit code, integrated into diagnosis output
- Custom diagnosis rules (`--rules`) — load JSON rule files for org-specific patterns and advice
- Continuous watch mode (`--watch`) — diagnose at a fixed interval for real-time monitoring

#### Kubernetes support
- Kubernetes pod awareness (`--kube`) — resolve pod names to container IDs via `kubectl`
- Namespace auto-detection from downward API or `--kube-namespace`
- Multi-container pod support (`--kube-container`)
- Pod metadata enrichment in output (node, phase, QoS, labels, container statuses)

#### Output and integrations
- Markdown incident report generation (`--report`) — full postmortem document
- PagerDuty Events API v2 payload (`--output pagerduty`)
- Slack incoming webhook payload (`--output slack`)
- Prometheus text exposition metrics (`--output prometheus`)
- Generic JSON webhook (`--output webhook`)

#### Distribution
- Cross-platform binary releases (Linux, macOS, Windows; amd64 and arm64) via GoReleaser
- Homebrew tap (`prabhnoor12/homebrew-tap`)
- Scoop bucket (`prabhnoor12/scoop-bucket`)
- Docker container image
- Shell completions (bash, zsh, fish)
- Manual page (`dockwhy(1)`)

#### Project hardening
- CI pipeline with race detector, golangci-lint, coverage reporting, and multi-OS matrix
- Dependabot for automated dependency updates
- Security policy with private vulnerability reporting
- Fuzz tests for parsing functions
- Benchmarks for core analysis functions
- Integration test suite (build-tagged, requires Docker daemon)
- Zero external dependencies — stdlib only
