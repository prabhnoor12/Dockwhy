# Dockwhy for VS Code

Diagnose why Docker containers stopped directly from Visual Studio Code.

## Features

- **Container Tree View** — Browse stopped and running containers in the sidebar
- **One-Click Diagnosis** — Right-click any container to diagnose it instantly
- **Rich Webview Results** — View diagnosis results with syntax highlighting, severity badges, and actionable recommendations
- **Docker Compose Support** — Diagnose entire projects with a single command
- **Crash Trend Analysis** — Identify crash loops and patterns
- **Resource Recommendations** — Get memory and CPU sizing suggestions
- **Incident Reports** — Generate Markdown postmortem documents
- **Configurable** — Customize executable path, log tail size, and timeouts

## Requirements

- [dockwhy CLI](https://github.com/prabhnoor12/dockwhy) must be installed and available on your `PATH`
- Docker CLI must be installed and connected to a daemon

## Installation

1. Install the dockwhy CLI:
   ```bash
   go install github.com/prabhnoor12/dockwhy/cmd/dockwhy@latest
   ```

2. Install this extension from the VS Code Marketplace (or build from source):
   ```bash
   cd ide-extensions/vscode
   npm install
   npm run package
   code --install-extension dockwhy-0.1.0.vsix
   ```

## Usage

### Command Palette

Open the Command Palette (`Ctrl+Shift+P` / `Cmd+Shift+P`) and type "Dockwhy":

- **Dockwhy: Diagnose Container** — Diagnose a specific container by name or ID
- **Dockwhy: Diagnose Docker Compose Project** — Diagnose all containers in a project
- **Dockwhy: Show Crash Trends** — Analyze crash patterns
- **Dockwhy: Show Resource Recommendations** — Get sizing suggestions
- **Dockwhy: Generate Incident Report** — Create a Markdown postmortem

### Sidebar

The Dockwhy sidebar shows two views:

- **Stopped Containers** — Containers that have exited (most likely to need diagnosis)
- **Running Containers** — Currently active containers

Right-click any container and select **Diagnose with Dockwhy** to run a diagnosis.

### Context Menu

Right-click in the editor or terminal to access Dockwhy commands.

## Configuration

Open VS Code settings and search for "dockwhy":

| Setting | Default | Description |
|---------|---------|-------------|
| `dockwhy.executablePath` | `dockwhy` | Path to the dockwhy executable |
| `dockwhy.defaultTail` | `50` | Number of log lines to show |
| `dockwhy.defaultTimeout` | `10s` | Timeout for Docker commands |
| `dockwhy.autoRefresh` | `false` | Automatically refresh container list |
| `dockwhy.refreshInterval` | `30` | Auto-refresh interval in seconds |

## Development

```bash
cd ide-extensions/vscode
npm install
npm run watch  # Watch for changes
```

Press `F5` in VS Code to open a new Extension Development Host window.

## Building

```bash
npm run compile
npm run package
```

This creates a `.vsix` file that can be installed with `code --install-extension`.

## License

Apache 2.0 — see [LICENSE](../../LICENSE) for details.
