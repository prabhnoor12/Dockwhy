_dockwhy() {
    local cur prev opts containers
    COMPREPLY=()
    cur="${COMP_WORDS[COMP_CWORD]}"
    prev="${COMP_WORDS[COMP_CWORD-1]}"

    opts="--tail --no-logs --timeout --events-since --no-events --json --project --trend --smart-logs --resources --compare --exit-code --rules --watch --report --output --kube --kube-namespace --kube-container --version --help"

    if [[ ${cur} == -* ]]; then
        COMPREPLY=( $(compgen -W "${opts}" -- "${cur}") )
        return 0
    fi

    if [[ ${prev} == "--output" ]]; then
        COMPREPLY=( $(compgen -W "pagerduty slack prometheus webhook" -- "${cur}") )
        return 0
    fi

    if [[ ${prev} == "--project" || ${prev} == "--compare" || ${prev} == "--rules" || ${prev} == "--report" ]]; then
        COMPREPLY=( $(compgen -f -- "${cur}") )
        return 0
    fi

    # Complete container names
    containers=$(docker ps -a --format '{{.Names}}' 2>/dev/null)
    COMPREPLY=( $(compgen -W "${containers}" -- "${cur}") )
    return 0
}

complete -F _dockwhy dockwhy
