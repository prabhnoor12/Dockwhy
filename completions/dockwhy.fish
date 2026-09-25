complete -c dockwhy -f

function __docker_containers
    docker ps -a --format '{{.Names}}' 2>/dev/null
end

function __docker_compose_projects
    docker compose ls --format '{{.Name}}' 2>/dev/null
end

complete -c dockwhy -l tail -d 'Number of log lines' -x -a '50 100 200 500'
complete -c dockwhy -l no-logs -d 'Skip log collection'
complete -c dockwhy -l timeout -d 'Timeout for Docker commands' -x -a '10s 30s 1m'
complete -c dockwhy -l events-since -d 'Event lookback window' -x -a '1h 24h 7d'
complete -c dockwhy -l no-events -d 'Skip event lookup'
complete -c dockwhy -l json -d 'Output JSON'
complete -c dockwhy -l project -d 'Docker Compose project' -x -a '(__docker_compose_projects)'
complete -c dockwhy -l trend -d 'Show crash patterns'
complete -c dockwhy -l smart-logs -d 'Extract error patterns'
complete -c dockwhy -l resources -d 'Resource recommendations'
complete -c dockwhy -l compare -d 'Compare containers' -x -a '(__docker_containers)'
complete -c dockwhy -l exit-code -d 'Look up exit code' -x
complete -c dockwhy -l rules -d 'Custom rules file' -r -F
complete -c dockwhy -l watch -d 'Continuous monitoring' -x -a '5s 30s 1m'
complete -c dockwhy -l report -d 'Generate report' -r -F
complete -c dockwhy -l output -d 'Integration format' -x -a 'pagerduty slack prometheus webhook'
complete -c dockwhy -l kube -d 'Kubernetes pod mode'
complete -c dockwhy -l kube-namespace -d 'Kubernetes namespace' -x
complete -c dockwhy -l kube-container -d 'Pod container name' -x
complete -c dockwhy -l version -d 'Print version'
complete -c dockwhy -l help -d 'Show help'

complete -c dockwhy -a '(__docker_containers)' -d 'Container name or ID'
