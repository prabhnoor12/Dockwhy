# Dockwhy for JetBrains IDEs

Integrate dockwhy with IntelliJ IDEA, WebStorm, PyCharm, GoLand, and other JetBrains IDEs.

## Setup as External Tool

1. Open **Settings/Preferences** → **Tools** → **External Tools**
2. Click **+** to add a new tool
3. Configure the following:

### Diagnose Container

- **Name**: `Dockwhy: Diagnose Container`
- **Program**: `dockwhy`
- **Arguments**: `--json $SelectedText$`
- **Working directory**: `$ProjectFileDir$`

### Diagnose with Smart Logs

- **Name**: `Dockwhy: Smart Logs`
- **Program**: `dockwhy`
- **Arguments**: `--smart-logs --tail 100 $SelectedText$`
- **Working directory**: `$ProjectFileDir$`

### Show Crash Trends

- **Name**: `Dockwhy: Crash Trends`
- **Program**: `dockwhy`
- **Arguments**: `--trend $SelectedText$`
- **Working directory**: `$ProjectFileDir$`

### Generate Incident Report

- **Name**: `Dockwhy: Generate Report`
- **Program**: `dockwhy`
- **Arguments**: `--report $ProjectFileDir$/incident-report.md $SelectedText$`
- **Working directory**: `$ProjectFileDir$`

## Usage

1. Select a container name or ID in your editor
2. Right-click → **External Tools** → **Dockwhy: Diagnose Container**
3. Or use **Tools** → **External Tools** → **Dockwhy** from the menu

The diagnosis output appears in the **Run** tool window.

## Docker Tool Window Integration

If you have the Docker plugin installed:

1. Open the **Docker** tool window
2. Right-click a container
3. Select **Dockwhy: Diagnose** (after configuring the external tool)

## Keyboard Shortcuts

Assign custom shortcuts in **Settings/Preferences** → **Keymap**:

1. Search for "Dockwhy"
2. Right-click the action → **Add Keyboard Shortcut**
3. Recommended: `Ctrl+Alt+D` (Windows/Linux) or `Cmd+Alt+D` (macOS)

## Requirements

- dockwhy CLI installed and on `PATH`
- Docker CLI installed and connected

## License

Apache 2.0
