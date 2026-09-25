# dockwhy.nvim

Neovim plugin for diagnosing Docker containers with dockwhy.

## Requirements

- Neovim 0.7+
- dockwhy CLI installed and on `PATH`
- Docker CLI installed

## Installation

Using [lazy.nvim](https://github.com/folke/lazy.nvim):

```lua
{
    "prabhnoor12/dockwhy",
    dir = "ide-extensions/neovim",
    config = function()
        require("dockwhy").setup({
            keymaps = {
                diagnose = "<leader>dd",
                trend = "<leader>dt",
                resources = "<leader>dr",
            }
        })
    end
}
```

Using [packer.nvim](https://github.com/wbthomason/packer.nvim):

```lua
use {
    "prabhnoor12/dockwhy",
    dir = "ide-extensions/neovim",
    config = function()
        require("dockwhy").setup()
    end
}
```

## Usage

### Commands

- `:Dockwhy [container]` — Diagnose a container
- `:DockwhyTrend [container]` — Show crash trends
- `:DockwhyResources [container]` — Show resource recommendations
- `:DockwhyReport [container]` — Generate incident report

### Lua API

```lua
local dockwhy = require("dockwhy")

-- Diagnose a container
dockwhy.diagnose("my-container")

-- Show crash trends
dockwhy.trend("my-container")

-- Show resource recommendations
dockwhy.resources("my-container")

-- Generate incident report
dockwhy.report("my-container")
```

### Keymaps

Configure in your setup:

```lua
require("dockwhy").setup({
    keymaps = {
        diagnose = "<leader>dd",
        trend = "<leader>dt",
        resources = "<leader>dr",
    }
})
```

## License

Apache 2.0
