# IDE Extensions and Integrations

Editor integrations for dockwhy, bringing Docker container diagnostics directly to your development environment.

## Available Extensions

### VS Code
Full-featured extension with sidebar, commands, and rich webview panels.

**Location**: `vscode/`

**Features**:
- Container tree view (stopped and running)
- One-click diagnosis from context menu
- Rich HTML diagnosis reports
- Docker Compose project diagnosis
- Crash trend analysis
- Resource recommendations
- Incident report generation
- Configurable settings

**Installation**:
```bash
cd vscode
npm install
npm run package
code --install-extension dockwhy-0.1.0.vsix
```

### JetBrains IDEs
External tools configuration for IntelliJ, WebStorm, PyCharm, GoLand, etc.

**Location**: `jetbrains/`

**Features**:
- External tools integration
- Context menu actions
- Customizable keyboard shortcuts
- Docker tool window integration

**Setup**: See `jetbrains/README.md` for configuration instructions.

### Neovim
Lua plugin for modern Neovim (0.7+).

**Location**: `neovim/`

**Features**:
- Commands: `:Dockwhy`, `:DockwhyTrend`, `:DockwhyResources`, `:DockwhyReport`
- Lua API for scripting
- Configurable keymaps
- Buffer-based output

**Installation** (lazy.nvim):
```lua
{
    "prabhnoor12/dockwhy",
    dir = "ide-extensions/neovim",
    config = function()
        require("dockwhy").setup()
    end
}
```

### Vim
Vimscript plugin for Vim 8+.

**Location**: `vim/`

**Features**:
- Commands: `:Dockwhy`, `:DockwhyTrend`, `:DockwhyResources`, `:DockwhyReport`
- Plug mappings for custom keybindings
- Lightweight and fast

**Installation** (vim-plug):
```vim
Plug 'prabhnoor12/dockwhy', { 'rtp': 'ide-extensions/vim' }
```

## Shell Completions

Tab completion for container names, flags, and output formats.

**Location**: `completions/` (at project root)

### Bash
```bash
# Add to ~/.bashrc
source /path/to/dockwhy/completions/dockwhy.bash
```

Or install system-wide:
```bash
sudo cp completions/dockwhy.bash /etc/bash_completion.d/dockwhy
```

### Zsh
```bash
# Add to ~/.zshrc
fpath=(/path/to/dockwhy/completions $fpath)
autoload -U compinit && compinit
```

Or install system-wide:
```bash
sudo cp completions/dockwhy.zsh /usr/share/zsh/site-functions/_dockwhy
```

### Fish
```bash
# Install to fish completions directory
cp completions/dockwhy.fish ~/.config/fish/completions/dockwhy.fish
```

## Requirements

All extensions require:
- `dockwhy` CLI installed and on `PATH`
- Docker CLI installed and connected to a daemon
- (Kubernetes features) `kubectl` configured

## Development

### VS Code Extension
```bash
cd vscode
npm install
npm run watch  # Watch mode
# Press F5 in VS Code to launch Extension Development Host
```

### Testing
Each extension has its own testing approach. See individual READMEs for details.

## Contributing

Contributions welcome! Please follow the main project's [CONTRIBUTING.md](../CONTRIBUTING.md) guidelines.

## License

Apache 2.0 — see [LICENSE](../LICENSE) for details.
