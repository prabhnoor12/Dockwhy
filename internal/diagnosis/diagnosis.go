package diagnosis

import (
	"fmt"
	"strings"

	"github.com/dockwhy/dockwhy/internal/docker"
)

type Result struct {
	Container     docker.Container `json:"container"`
	Reason        string           `json:"reason"`
	Summary       string           `json:"summary"`
	Severity      string           `json:"severity"`
	Confidence    string           `json:"confidence"`
	Evidence      []Evidence       `json:"evidence"`
	Advice        []string         `json:"advice"`
	Logs          string           `json:"recent_logs"`
	LogsTruncated bool             `json:"logs_truncated,omitempty"`
	LogError      string           `json:"log_error,omitempty"`
	Events        []docker.Event   `json:"events,omitempty"`
	EventsError   string           `json:"events_error,omitempty"`
}

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
	r := Result{
		Container: c, Logs: logs, LogsTruncated: logsTruncated, Events: events,
		Severity: "info", Confidence: "high",
	}
	if logErr != nil {
		r.LogError = logErr.Error()
	}
	if eventsErr != nil {
		r.EventsError = eventsErr.Error()
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

	switch {
	case c.State.OOMKilled:
		r.Severity, r.Reason, r.Summary, r.Confidence = "critical", "out of memory", "Docker killed the container after it exceeded its memory limit.", "high"
		r.Advice = append(r.Advice, "Reduce the application's memory use or raise the container memory limit.")
	case c.State.ExitCode == 137:
		r.Severity, r.Reason, r.Summary, r.Confidence = "critical", "forcefully killed (exit 137)", "The process received SIGKILL; this is commonly an out-of-memory kill, although an external kill can produce the same code.", "medium"
		r.Advice = append(r.Advice, "Check OOMKilled above and container/host events; reduce memory use or increase the limit if it was an OOM kill.")
	case c.State.ExitCode == 143:
		r.Reason, r.Summary, r.Confidence = "gracefully stopped (exit 143)", "The process received SIGTERM, usually from docker stop, a deployment, or an orchestrator.", "medium"
		r.Advice = append(r.Advice, "Check deployment and orchestrator events around the finished time.")
	case c.State.ExitCode == 126:
		r.Reason, r.Summary, r.Confidence = "command not executable (exit 126)", "The configured entrypoint or command was found but could not be executed.", "high"
		r.Advice = append(r.Advice, "Check executable permissions, the image architecture, and the container user.")
	case c.State.ExitCode == 127:
		r.Reason, r.Summary, r.Confidence = "command not found (exit 127)", "The configured entrypoint or command was not available in the image.", "high"
		r.Advice = append(r.Advice, "Verify the image entrypoint and command, including PATH and copied file names.")
	case c.State.ExitCode == 139:
		r.Severity, r.Reason, r.Summary, r.Confidence = "error", "segmentation fault (exit 139)", "The main process crashed with SIGSEGV.", "high"
		r.Advice = append(r.Advice, "Inspect the application logs and crash-dump tooling for the failing process.")
	case c.State.Error != "":
		r.Severity, r.Reason, r.Summary = "error", "Docker runtime error", "Docker reported an error while creating or running the container."
		r.Advice = append(r.Advice, "Use the Docker error and recent logs to correct the runtime or mount configuration.")
	case c.State.Restarting || c.State.Status == "restarting":
		r.Severity, r.Reason, r.Summary = "warning", "restart loop", "Docker is repeatedly restarting the container; the exit code above is the latest observed failure."
		r.Advice = append(r.Advice, "Inspect the earliest failed restart and consider temporarily disabling the restart policy while debugging.")
	case c.State.Running:
		r.Reason, r.Summary = "container is running", "The container has not stopped; the details describe its current state and latest health signals."
		r.Advice = append(r.Advice, "If this is unexpected, inspect health checks and restart events rather than treating it as a crash.")
	case c.State.Paused:
		r.Reason, r.Summary = "container is paused", "Docker has paused the container; its main process has not necessarily crashed."
		r.Advice = append(r.Advice, "Resume it with docker unpause if the pause was not intentional.")
	case c.State.Status == "created":
		r.Reason, r.Summary = "container has not started", "The container was created but its main process has not run yet."
		r.Advice = append(r.Advice, "Check the image entrypoint, mounts, and the command used to start it.")
	case c.State.ExitCode == 0:
		r.Reason, r.Summary = "clean exit", "The main process exited successfully with code 0; this was not an application crash."
		r.Advice = append(r.Advice, "If the container should stay alive, check whether its command is a short-lived job or worker startup script.")
	default:
		r.Reason, r.Summary, r.Confidence = "application exited", fmt.Sprintf("The main process exited with code %d.", c.State.ExitCode), "low"
		r.Advice = append(r.Advice, "Read the recent logs for the application's own shutdown or crash message.")
	}

	if c.State.Health != nil && c.State.Health.Status == "unhealthy" {
		add("health diagnosis", "health checks are failing")
		r.Advice = append(r.Advice, "Review the latest health-check output and confirm the endpoint, port, timeout, and startup grace period.")
		if c.State.Running && r.Reason == "container is running" {
			r.Severity, r.Reason, r.Summary = "warning", "unhealthy health check", "The container is running, but its health check is failing. Docker does not normally stop a container solely because it is unhealthy."
		}
	}

	if logsTruncated {
		add("logs", "output truncated by safety limit")
	}
	if c.RestartCount > 0 {
		r.Advice = append(r.Advice, fmt.Sprintf("The container has restarted %d time(s); inspect the restart loop's first failure, not just its latest state.", c.RestartCount))
	}
	if c.MemoryLimit > 0 {
		add("memory limit", formatBytes(c.MemoryLimit))
	} else {
		add("memory limit", "unlimited")
	}
	if c.MemoryReserved > 0 {
		add("memory reservation", formatBytes(c.MemoryReserved))
	}
	if c.MemorySwapLimit > 0 {
		add("memory+swap limit", formatBytes(c.MemorySwapLimit))
	} else if c.MemoryLimit > 0 && c.MemorySwapLimit == -1 {
		add("memory+swap limit", "unlimited")
	}
	if c.DiskLimit != "" {
		add("disk limit", c.DiskLimit)
	} else {
		add("disk limit", "not configured/reported")
	}
	if c.SizeRW > 0 {
		add("writable layer", formatBytes(c.SizeRW))
	}
	if c.SizeRootFS > 0 {
		add("root filesystem", formatBytes(c.SizeRootFS))
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
		} else {
			r.Reason, r.Summary, r.Severity, r.Confidence = "disk full", "The logs or Docker runtime error indicate that the container or host ran out of disk space.", "critical", "high"
			r.Advice = append(r.Advice, "Check host filesystem usage, Docker's data-root, volumes, and any storage quota.")
		}
	}
	return r
}

func formatBytes(value int64) string {
	units := []string{"B", "KiB", "MiB", "GiB", "TiB"}
	n := float64(value)
	i := 0
	for n >= 1024 && i < len(units)-1 {
		n /= 1024
		i++
	}
	if i == 0 {
		return fmt.Sprintf("%d %s", value, units[i])
	}
	return fmt.Sprintf("%.1f %s", n, units[i])
}
