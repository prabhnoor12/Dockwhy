" dockwhy.vim - Vim plugin for Docker container diagnostics
" Requires: dockwhy CLI installed and on PATH

if exists('g:loaded_dockwhy')
    finish
endif
let g:loaded_dockwhy = 1

function! s:notify(msg)
    echohl WarningMsg
    echo "[Dockwhy] " . a:msg
    echohl None
endfunction

function! s:run_dockwhy(args)
    let l:cmd = "dockwhy " . join(a:args, " ")
    let l:output = system(l:cmd)
    if v:shell_error
        call s:notify("Error: " . l:output)
        return ""
    endif
    return l:output
endfunction

function! dockwhy#diagnose(...)
    let l:container = a:0 > 0 ? a:1 : input("Container name or ID: ")
    if l:container == ""
        return
    endif

    call s:notify("Diagnosing " . l:container . "...")
    let l:output = s:run_dockwhy(["--json", l:container])
    if l:output != ""
        new
        put =l:output
        1delete
        setlocal filetype=json
        setlocal buftype=nofile
        setlocal bufhidden=wipe
        execute "file dockwhy://" . l:container
    endif
endfunction

function! dockwhy#trend(...)
    let l:container = a:0 > 0 ? a:1 : input("Container name or ID: ")
    if l:container == ""
        return
    endif

    call s:notify("Analyzing trends for " . l:container . "...")
    let l:output = s:run_dockwhy(["--trend", l:container])
    if l:output != ""
        new
        put =l:output
        1delete
        setlocal buftype=nofile
        setlocal bufhidden=wipe
        execute "file dockwhy-trend://" . l:container
    endif
endfunction

function! dockwhy#resources(...)
    let l:container = a:0 > 0 ? a:1 : input("Container name or ID: ")
    if l:container == ""
        return
    endif

    call s:notify("Analyzing resources for " . l:container . "...")
    let l:output = s:run_dockwhy(["--resources", l:container])
    if l:output != ""
        new
        put =l:output
        1delete
        setlocal filetype=json
        setlocal buftype=nofile
        setlocal bufhidden=wipe
        execute "file dockwhy-resources://" . l:container
    endif
endfunction

function! dockwhy#report(...)
    let l:container = a:0 > 0 ? a:1 : input("Container name or ID: ")
    if l:container == ""
        return
    endif

    let l:output_file = input("Report file path: ", l:container . "-report.md")
    if l:output_file == ""
        return
    endif

    call s:notify("Generating report for " . l:container . "...")
    let l:output = s:run_dockwhy(["--report", l:output_file, l:container])
    call s:notify("Report saved to " . l:output_file)
endfunction

command! -nargs=? Dockwhy call dockwhy#diagnose(<f-args>)
command! -nargs=? DockwhyTrend call dockwhy#trend(<f-args>)
command! -nargs=? DockwhyResources call dockwhy#resources(<f-args>)
command! -nargs=? DockwhyReport call dockwhy#report(<f-args>)

if !exists('g:dockwhy_no_mappings') || !g:dockwhy_no_mappings
    nnoremap <silent> <Plug>DockwhyDiagnose :call dockwhy#diagnose()<CR>
    nnoremap <silent> <Plug>DockwhyTrend :call dockwhy#trend()<CR>
    nnoremap <silent> <Plug>DockwhyResources :call dockwhy#resources()<CR>
    nnoremap <silent> <Plug>DockwhyReport :call dockwhy#report()<CR>
endif
