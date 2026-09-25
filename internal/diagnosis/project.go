package diagnosis

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/prabhnoor12/dockwhy/internal/docker"
)

// ProjectResult is the diagnosis of an entire Compose project.
type ProjectResult struct {
	Project     string              `json:"project"`
	Containers  []ContainerDiagnosis `json:"containers"`
	RootCause   *ContainerDiagnosis  `json:"root_cause,omitempty"`
	Summary     string              `json:"summary"`
	Timeline    []TimelineEntry     `json:"timeline"`
}

// ContainerDiagnosis pairs a container with its diagnosis result.
type ContainerDiagnosis struct {
	Container docker.Container `json:"container"`
	Result    Result           `json:"result"`
}

// TimelineEntry is a single event in the project-wide timeline.
type TimelineEntry struct {
	TimeNano  int64  `json:"time_nano"`
	Container string `json:"container"`
	Service   string `json:"service,omitempty"`
	Event     string `json:"event"`
	Detail    string `json:"detail,omitempty"`
}

// AnalyzeProject diagnoses all containers in a Compose project and identifies
// the most likely root cause by correlating failure times and dependency signals.
func AnalyzeProject(project string, diagnoses []ContainerDiagnosis, allEvents map[string][]docker.Event) ProjectResult {
	pr := ProjectResult{Project: project, Containers: diagnoses}

	for containerName, events := range allEvents {
		for _, event := range events {
			entry := TimelineEntry{
				TimeNano:  event.TimeNano,
				Container: containerName,
				Event:     event.Action,
			}
			if service := composeServiceFor(diagnoses, containerName); service != "" {
				entry.Service = service
			}
			if code, ok := event.Attrs["exitCode"]; ok {
				entry.Detail = "exit " + code
			}
			pr.Timeline = append(pr.Timeline, entry)
		}
	}
	sort.SliceStable(pr.Timeline, func(i, j int) bool {
		return pr.Timeline[i].TimeNano < pr.Timeline[j].TimeNano
	})

	pr.RootCause = identifyRootCause(diagnoses, pr.Timeline)
	pr.Summary = projectSummary(pr)
	return pr
}

func composeServiceFor(diagnoses []ContainerDiagnosis, name string) string {
	for _, d := range diagnoses {
		if d.Container.Name == name {
			return d.Container.ComposeService
		}
	}
	return ""
}

func identifyRootCause(diagnoses []ContainerDiagnosis, timeline []TimelineEntry) *ContainerDiagnosis {
	var failed []ContainerDiagnosis
	for _, d := range diagnoses {
		if d.Container.State.ExitCode != 0 || d.Container.State.OOMKilled || !d.Container.State.Running {
			if d.Container.State.Status != "running" && d.Container.State.Status != "created" {
				failed = append(failed, d)
			}
		}
	}
	if len(failed) == 0 {
		return nil
	}

	earliest := earliestFailure(failed, timeline)
	if earliest != nil {
		return earliest
	}

	sort.SliceStable(failed, func(i, j int) bool {
		return severityRank(failed[i].Result.Severity) > severityRank(failed[j].Result.Severity)
	})
	return &failed[0]
}

func earliestFailure(failed []ContainerDiagnosis, timeline []TimelineEntry) *ContainerDiagnosis {
	failureTimes := make(map[string]int64)
	for _, entry := range timeline {
		if entry.Event == "die" || entry.Event == "kill" || entry.Event == "oom" {
			if _, exists := failureTimes[entry.Container]; !exists {
				failureTimes[entry.Container] = entry.TimeNano
			}
		}
	}
	if len(failureTimes) == 0 {
		return nil
	}

	var earliest *ContainerDiagnosis
	var earliestTime int64
	for i := range failed {
		t, ok := failureTimes[failed[i].Container.Name]
		if !ok {
			if ft := parseTime(failed[i].Container.State.FinishedAt); ft > 0 {
				t = ft
			} else {
				continue
			}
		}
		if earliest == nil || t < earliestTime {
			earliest = &failed[i]
			earliestTime = t
		}
	}
	return earliest
}

func parseTime(s string) int64 {
	t, err := time.Parse(time.RFC3339Nano, s)
	if err != nil {
		return 0
	}
	return t.UnixNano()
}

func severityRank(severity string) int {
	switch severity {
	case "critical":
		return 4
	case "error":
		return 3
	case "warning":
		return 2
	case "info":
		return 1
	default:
		return 0
	}
}

func projectSummary(pr ProjectResult) string {
	total := len(pr.Containers)
	running := 0
	stopped := 0
	unhealthy := 0
	for _, d := range pr.Containers {
		switch {
		case d.Container.State.Running && (d.Container.State.Health == nil || d.Container.State.Health.Status != "unhealthy"):
			running++
		case d.Container.State.Health != nil && d.Container.State.Health.Status == "unhealthy":
			unhealthy++
			stopped++
		default:
			stopped++
		}
	}

	var parts []string
	parts = append(parts, fmt.Sprintf("%d container(s) in project %q", total, pr.Project))
	if running > 0 {
		parts = append(parts, fmt.Sprintf("%d running", running))
	}
	if stopped-running > 0 {
		parts = append(parts, fmt.Sprintf("%d stopped/unhealthy", stopped))
	}
	if unhealthy > 0 {
		parts = append(parts, fmt.Sprintf("%d unhealthy", unhealthy))
	}
	if pr.RootCause != nil {
		parts = append(parts, fmt.Sprintf("root cause: %s (%s)", pr.RootCause.Container.Name, pr.RootCause.Result.Reason))
	}
	return strings.Join(parts, ", ")
}
