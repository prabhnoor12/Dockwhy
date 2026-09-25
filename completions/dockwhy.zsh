#compdef dockwhy

_dockwhy() {
    local -a opts
    opts=(
        '--tail[Number of log lines]:lines:'
        '--no-logs[Skip log collection]'
        '--timeout[Timeout for Docker commands]:duration:'
        '--events-since[Lookback window for events]:duration:'
        '--no-events[Skip event lookup]'
        '--json[Output machine-readable JSON]'
        '--project[Diagnose Docker Compose project]:project:_docker_compose_projects'
        '--trend[Show crash patterns]'
        '--smart-logs[Extract error patterns]'
        '--resources[Show resource recommendations]'
        '--compare[Compare with another container]:container:_docker_containers'
        '--exit-code[Look up exit code]:code:'
        '--rules[Custom rules file]:file:_files'
        '--watch[Continuous monitoring]:interval:'
        '--report[Generate incident report]:file:_files'
        '--output[Integration output format]:format:(pagerduty slack prometheus webhook)'
        '--kube[Treat argument as Kubernetes pod]'
        '--kube-namespace[Kubernetes namespace]:namespace:'
        '--kube-container[Container in multi-container pod]:container:'
        '--version[Print version]'
        '--help[Show help]'
    )

    _arguments -s $opts '*:container:_docker_containers_stopped'
}

_docker_containers_stopped() {
    local -a containers
    containers=(${(f)"$(docker ps -a --format '{{.Names}}\t{{.Status}}' 2>/dev/null)"})
    _describe -t containers 'stopped containers' containers
}

_docker_containers() {
    local -a containers
    containers=(${(f)"$(docker ps -a --format '{{.Names}}' 2>/dev/null)"})
    _describe -t containers 'containers' containers
}

_docker_compose_projects() {
    local -a projects
    projects=(${(f)"$(docker compose ls --format '{{.Name}}' 2>/dev/null)"})
    _describe -t projects 'compose projects' projects
}

_dockwhy "$@"
