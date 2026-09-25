package diagnosis

import (
	"fmt"
	"sort"
	"strings"

	"github.com/prabhnoor12/dockwhy/internal/docker"
	"github.com/prabhnoor12/dockwhy/internal/format"
)

// Result is the complete diagnosis for a container, combining the primary
// explanation, ranked alternative findings, supporting evidence, and advice.
type Result struct {
	Container     docker.Container `json:"container"`
	Findings      []Finding        `json:"findings"`
	Stats         *docker.Stats    `json:"stats,omitempty"`
	StatsError    string           `json:"stats_error,omitempty"`
	Reason        string           `json:"reason"`
	Summary       string           `json:"summary"`
	Severity      string           `json:"severity"`
	Confidence    string           `json:"confidence"`
	Evidence      []Evidence       `json:"evidence"`
	Advice        []string         `json:"advice"`
	Logs          string           `json:"recent_logs"`
	LogsSkipped   bool             `json:"logs_skipped,omitempty"`
	LogsTruncated bool             `json:"logs_truncated,omitempty"`
	LogError      string           `json:"log_error,omitempty"`
	Events        []docker.Event   `json:"events,omitempty"`
	EventsError   string           `json:"events_error,omitempty"`
}

// Finding is one explanation supported by a distinct set of evidence. The
// first finding is the primary diagnosis; later findings are secondary
// signals that may also help explain the container's state.
type Finding struct {
	Rank       int        `json:"rank"`
	Reason     string     `json:"reason"`
	Summary    string     `json:"summary"`
	Severity   string     `json:"severity"`
	Confidence string     `json:"confidence"`
	Evidence   []Evidence `json:"evidence,omitempty"`
	Advice     []string   `json:"advice,omitempty"`
}

// Evidence is a single named data point that supports a finding.
type Evidence struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// Analyze preserves the small API used by callers that do not collect events.
func Analyze(c docker.Container, logs string, logErr error) Result {
	return AnalyzeDetailed(c, logs, false, nil, logErr, nil)
}

// AnalyzeDetailed produces an explainable diagnosis. Docker flags and exit
// codes are treated as facts; text-based signals are either secondary or
// marked with lower confidence.
func AnalyzeDetailed(c docker.Container, logs string, logsTruncated bool, events []docker.Event, logErr, eventsErr error) Result {
	orderedEvents := append([]docker.Event(nil), events...)
	sort.SliceStable(orderedEvents, func(i, j int) bool {
		return orderedEvents[i].TimeNano < orderedEvents[j].TimeNano
	})
	r := Result{
		Container: c, Logs: logs, LogsTruncated: logsTruncated, Events: orderedEvents,
		Severity: "info", Confidence: "high",
	}
	if logErr != nil {
		r.LogError = logErr.Error()
	}
	if eventsErr != nil {
		r.EventsError = eventsErr.Error()
	}

	type candidate struct {
		finding Finding
		score   int
		order   int
	}
	candidates := make([]candidate, 0, 4)
	order := 0
	addFinding := func(score int, reason, summary, severity, confidence string, evidence []Evidence, advice ...string) {
		order++
		candidates = append(candidates, candidate{
			finding: Finding{
				Reason: reason, Summary: summary, Severity: severity,
				Confidence: confidence, Evidence: evidence, Advice: advice,
			},
			score: score,
			order: order,
		})
	}
	fact := func(name, value string) Evidence {
		return Evidence{Name: name, Value: value}
	}

	add := func(name, value string) {
		if strings.TrimSpace(value) != "" {
			r.Evidence = append(r.Evidence, Evidence{Name: name, Value: value})
		}
	}
	add("status", c.State.Status)
	add("exit code", fmt.Sprint(c.State.ExitCode))
	add("started", c.State.StartedAt)
	add("finished", c.State.FinishedAt)
	add("restart policy", c.RestartPolicy)
	add("restart count", fmt.Sprint(c.RestartCount))
	if c.State.Error != "" {
		add("Docker error", c.State.Error)
	}
	if c.State.OOMKilled {
		add("OOM killed", "true")
	}
	if c.State.Health != nil {
		add("health", fmt.Sprintf("%s (failing streak: %d)", c.State.Health.Status, c.State.Health.FailingStreak))
		if len(c.State.Health.Log) > 0 {
			latest := c.State.Health.Log[len(c.State.Health.Log)-1]
			add("latest health check", fmt.Sprintf("exit %d: %s", latest.ExitCode, strings.TrimSpace(latest.Output)))
		}
	}
	if len(c.Entrypoint) > 0 {
		add("entrypoint", formatCommand(c.Entrypoint))
	}
	if len(c.Command) > 0 {
		add("command", formatCommand(c.Command))
	}
	if c.WorkingDir != "" {
		add("working directory", c.WorkingDir)
	}
	if c.User != "" {
		add("user", c.User)
	}
	if c.StopSignal != "" {
		add("stop signal", c.StopSignal)
	}
	if c.LogDriver != "" {
		add("logging driver", c.LogDriver)
	}
	if c.ComposeProject != "" {
		add("Compose project", c.ComposeProject)
	}
	if c.ComposeService != "" {
		add("Compose service", c.ComposeService)
	}
	for _, mount := range c.Mounts {
		if mount.Destination == "" {
			continue
		}
		mode := "ro"
		if mount.RW {
			mode = "rw"
		}
		source := mount.Source
		if source == "" {
			source = mount.Name
		}
		if source == "" {
			source = mount.Type
		}
		add("mount", fmt.Sprintf("%s -> %s (%s)", source, mount.Destination, mode))
	}

	switch {
	case c.State.OOMKilled:
		addFinding(100, "out of memory", "Docker killed the container after it exceeded its memory limit.", "critical", "high",
			[]Evidence{fact("OOM killed", "true"), fact("exit code", fmt.Sprint(c.State.ExitCode))},
			"Reduce the application's memory use or raise the container memory limit.")
	case c.State.ExitCode == 137:
		addFinding(95, "forcefully killed (exit 137)", "The process received SIGKILL; this is commonly an out-of-memory kill, although an external kill can produce the same code.", "critical", "medium",
			[]Evidence{fact("exit code", "137"), fact("OOM killed", fmt.Sprint(c.State.OOMKilled))},
			"Check OOMKilled above and container/host events; reduce memory use or increase the limit if it was an OOM kill.")
	case c.State.ExitCode == 143:
		addFinding(85, "gracefully stopped (exit 143)", "The process received SIGTERM, usually from docker stop, a deployment, or an orchestrator.", "info", "medium",
			[]Evidence{fact("exit code", "143")},
			"Check deployment and orchestrator events around the finished time.")
	case c.State.ExitCode == 126:
		addFinding(90, "command not executable (exit 126)", "The configured entrypoint or command was found but could not be executed.", "error", "high",
			[]Evidence{fact("exit code", "126")},
			"Check executable permissions, the image architecture, and the container user.")
	case c.State.ExitCode == 127:
		addFinding(90, "command not found (exit 127)", "The configured entrypoint or command was not available in the image.", "error", "high",
			[]Evidence{fact("exit code", "127")},
			"Verify the image entrypoint and command, including PATH and copied file names.")
	case c.State.ExitCode == 139:
		addFinding(90, "segmentation fault (exit 139)", "The main process crashed with SIGSEGV.", "error", "high",
			[]Evidence{fact("exit code", "139")},
			"Inspect the application logs and crash-dump tooling for the failing process.")
	case c.State.Error != "":
		addFinding(80, "Docker runtime error", "Docker reported an error while creating or running the container.", "error", "high",
			[]Evidence{fact("Docker error", c.State.Error)},
			"Use the Docker error and recent logs to correct the runtime or mount configuration.")
	case c.State.Restarting || c.State.Status == "restarting":
		addFinding(75, "restart loop", "Docker is repeatedly restarting the container; the exit code above is the latest observed failure.", "warning", "high",
			[]Evidence{fact("status", c.State.Status), fact("restart count", fmt.Sprint(c.RestartCount))},
			"Inspect the earliest failed restart and consider temporarily disabling the restart policy while debugging.")
	case c.State.Running:
		addFinding(40, "container is running", "The container has not stopped; the details describe its current state and latest health signals.", "info", "high",
			[]Evidence{fact("running", "true")},
			"If this is unexpected, inspect health checks and restart events rather than treating it as a crash.")
	case c.State.Paused:
		addFinding(60, "container is paused", "Docker has paused the container; its main process has not necessarily crashed.", "warning", "high",
			[]Evidence{fact("paused", "true")},
			"Resume it with docker unpause if the pause was not intentional.")
	case c.State.Status == "created":
		addFinding(60, "container has not started", "The container was created but its main process has not run yet.", "info", "high",
			[]Evidence{fact("status", "created")},
			"Check the image entrypoint, mounts, and the command used to start it.")
	case c.State.ExitCode == 0:
		addFinding(70, "clean exit", "The main process exited successfully with code 0; this was not an application crash.", "info", "high",
			[]Evidence{fact("exit code", "0")},
			"If the container should stay alive, check whether its command is a short-lived job or worker startup script.")
	default:
		addFinding(30, "application exited", fmt.Sprintf("The main process exited with code %d.", c.State.ExitCode), "info", "low",
			[]Evidence{fact("exit code", fmt.Sprint(c.State.ExitCode))},
			"Read the recent logs for the application's own shutdown or crash message.")
	}

	if c.State.Health != nil && c.State.Health.Status == "unhealthy" {
		add("health diagnosis", "health checks are failing")
		score := 55
		if c.State.Running {
			score = 95
		}
		addFinding(score, "unhealthy health check", "The container is running, but its health check is failing. Docker does not normally stop a container solely because it is unhealthy.", "warning", "high",
			[]Evidence{fact("health", fmt.Sprintf("%s (failing streak: %d)", c.State.Health.Status, c.State.Health.FailingStreak))},
			"Review the latest health-check output and confirm the endpoint, port, timeout, and startup grace period.")
	}

	if logsTruncated {
		add("logs", "output truncated by safety limit")
	}
	if c.RestartCount > 0 {
		r.Advice = append(r.Advice, fmt.Sprintf("The container has restarted %d time(s); inspect the restart loop's first failure, not just its latest state.", c.RestartCount))
	}
	if c.MemoryLimit > 0 {
		add("memory limit", format.Bytes(c.MemoryLimit))
	} else {
		add("memory limit", "unlimited")
	}
	if c.MemoryReserved > 0 {
		add("memory reservation", format.Bytes(c.MemoryReserved))
	}
	if c.MemorySwapLimit > 0 {
		add("memory+swap limit", format.Bytes(c.MemorySwapLimit))
	} else if c.MemoryLimit > 0 && c.MemorySwapLimit == -1 {
		add("memory+swap limit", "unlimited")
	}
	if c.DiskLimit != "" {
		add("disk limit", c.DiskLimit)
	} else {
		add("disk limit", "not configured/reported")
	}
	if c.SizeRW > 0 {
		add("writable layer", format.Bytes(c.SizeRW))
	}
	if c.SizeRootFS > 0 {
		add("root filesystem", format.Bytes(c.SizeRootFS))
	}
	if c.DiskReadOnly {
		add("root filesystem", "read-only")
	}
	if c.NanoCPUs > 0 {
		add("CPU limit", fmt.Sprintf("%.2f CPUs", float64(c.NanoCPUs)/1e9))
	}
	if c.CPUQuota > 0 && c.CPUPeriod > 0 {
		add("CPU quota", fmt.Sprintf("%d/%d microseconds", c.CPUQuota, c.CPUPeriod))
	}
	if c.PidsLimit > 0 {
		add("process limit", fmt.Sprintf("%d PIDs", c.PidsLimit))
	}

	logText := strings.ToLower(logs)
	diskSignal := strings.Contains(logText, "no space left on device") || strings.Contains(strings.ToLower(c.State.Error), "no space left on device")
	if diskSignal {
		if c.State.OOMKilled || c.State.ExitCode == 137 {
			add("disk signal", "no space left on device (secondary signal; OOM evidence remains primary)")
			addFinding(20, "disk full (secondary signal)", "The logs or Docker runtime error also indicate that storage may have been exhausted.", "warning", "low",
				[]Evidence{fact("disk signal", "no space left on device")},
				"Check host filesystem usage, Docker's data-root, volumes, and any storage quota.")
		} else {
			addFinding(100, "disk full", "The logs or Docker runtime error indicate that the container or host ran out of disk space.", "critical", "high",
				[]Evidence{fact("disk signal", "no space left on device")},
				"Check host filesystem usage, Docker's data-root, volumes, and any storage quota.")
		}
	}

	sort.SliceStable(candidates, func(i, j int) bool {
		if candidates[i].score != candidates[j].score {
			return candidates[i].score > candidates[j].score
		}
		return candidates[i].order < candidates[j].order
	})
	for i, candidate := range candidates {
		candidate.finding.Rank = i + 1
		r.Findings = append(r.Findings, candidate.finding)
	}
	if len(r.Findings) > 0 {
		primary := r.Findings[0]
		r.Severity, r.Reason, r.Summary, r.Confidence = primary.Severity, primary.Reason, primary.Summary, primary.Confidence
		for _, finding := range r.Findings {
			for _, advice := range finding.Advice {
				if !containsString(r.Advice, advice) {
					r.Advice = append(r.Advice, advice)
				}
			}
		}
	}
	if c.RestartCount > 0 {
		advice := fmt.Sprintf("The container has restarted %d time(s); inspect the restart loop's first failure, not just its latest state.", c.RestartCount)
		r.Advice = append(r.Advice, advice)
	}
	if c.ComposeService != "" {
		advice := fmt.Sprintf("This container belongs to Compose service %q; use `docker compose logs %s` to inspect the service logs.", c.ComposeService, c.ComposeService)
		if !containsString(r.Advice, advice) {
			r.Advice = append(r.Advice, advice)
		}
	}
	return r
}

func containsString(values []string, wanted string) bool {
	for _, value := range values {
		if value == wanted {
			return true
		}
	}
	return false
}

func formatCommand(command []string) string {
	quoted := make([]string, 0, len(command))
	for _, part := range command {
		quoted = append(quoted, fmt.Sprintf("%q", part))
	}
	return strings.Join(quoted, " ")
}
