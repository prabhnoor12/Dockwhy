# dockwhy.vim

Vim plugin for diagnosing Docker containers with dockwhy.

## Requirements

- Vim 8+ or Neovim
- dockwhy CLI installed and on `PATH`
- Docker CLI installed

## Installation

Using [vim-plug](https://github.com/junegunn/vim-plug):

```vim
Plug 'prabhnoor12/dockwhy', { 'rtp': 'ide-extensions/vim' }
```

Using [Vundle](https://github.com/VundleVim/Vundle.vim):

```vim
Plugin 'prabhnoor12/dockwhy', { 'rtp': 'ide-extensions/vim' }
```

Manual installation:

```bash
cp -r ide-extensions/vim/plugin/* ~/.vim/plugin/
```

## Usage

### Commands

- `:Dockwhy [container]` — Diagnose a container
- `:DockwhyTrend [container]` — Show crash trends
- `:DockwhyResources [container]` — Show resource recommendations
- `:DockwhyReport [container]` — Generate incident report

### Key Mappings

Add to your `.vimrc`:

```vim
nmap <leader>dd <Plug>DockwhyDiagnose
nmap <leader>dt <Plug>DockwhyTrend
nmap <leader>dr <Plug>DockwhyResources
nmap <leader>dr <Plug>DockwhyReport
```

Disable default mappings:

```vim
let g:dockwhy_no_mappings = 1
```

## License

Apache 2.0
