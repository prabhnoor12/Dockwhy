# Contributing to dockwhy

Thanks for your interest in contributing! This document covers everything you need to get started.

## Getting started

```bash
git clone https://github.com/prabhnoor12/dockwhy.git
cd dockwhy
make build    # verify it compiles
make test     # run all tests
make lint     # run the linter
make vet      # run go vet
```

Requirements: Go 1.23 or later. No external dependencies — the project uses only the Go standard library.

## Development workflow

1. Create a branch: `feature/your-feature`, `fix/your-fix`, or `docs/your-change`
2. Make your changes
3. Run the full check list:
   ```bash
   go test -race ./...
   go vet ./...
   golangci-lint run ./...
   ```
4. Commit and push
5. Open a pull request

## Code style

- **stdlib only** — no new dependencies without discussion in an issue first
- **Standard `flag` package** for CLI; we are not migrating to Cobra
- **Internal packages** under `internal/` are not part of the public API
- **Error messages**: lowercase start, no trailing punctuation (Go convention)
- **Exported types and functions** must have doc comments
- **Table-driven tests** preferred (see `diagnosis_test.go`, `rules_test.go` for examples)
- **No unnecessary comments** — code should be self-documenting; comments explain *why*, not *what*

## Project layout

| Package | Purpose |
|---------|---------|
| `cmd/dockwhy/` | Executable entrypoint (delegates to `cli.Run()`) |
| `internal/cli/` | Flag parsing, command orchestration, signal handling, watch/project/trend/compare modes |
| `internal/diagnosis/` | Core analysis engine: `AnalyzeDetailed`, exit codes, rules, smart logs, resources, trends, compare, project |
| `internal/docker/` | Docker CLI adapter (`CLIClient`), normalized types (`Container`, `Stats`, `Event`) |
| `internal/kube/` | Kubernetes CLI adapter (`KubeCLI`), pod types |
| `internal/output/` | All reporters: text, JSON, PagerDuty, Slack, Prometheus, webhook, Markdown report, and more |
| `internal/format/` | Byte formatting utilities |
| `internal/version/` | Version variable (set by ldflags at build time) |

## Testing

- All new code must have tests
- Use fake/mock clients (see `fakeClient` in `cli_test.go`) rather than requiring Docker
- Fuzz tests are welcome for parsing functions (see `fuzz_test.go`)
- Integration tests require a running Docker daemon and the `integration` build tag:
  ```bash
  go test -tags integration ./internal/integration/ -v
  ```

## Pull request process

- One commit per logical change, or squash before merging
- CI must pass (race detector, linter, vet, multi-OS matrix)
- Update `README.md` if flags or behavior change
- Update `CHANGELOG.md` for user-visible changes

## Reporting bugs

Open an issue on GitHub with:
- dockwhy version (`dockwhy --version`)
- Docker version (`docker version`)
- Operating system
- Full command you ran
- Full output (text or JSON)
- What you expected instead

## License

Apache 2.0 — by contributing, you agree your contributions are licensed under the same terms.
