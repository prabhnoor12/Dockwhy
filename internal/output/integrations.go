package output

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/prabhnoor12/dockwhy/internal/diagnosis"
)

// PagerDutyPayload is the event payload for PagerDuty's Events API v2.
type PagerDutyPayload struct {
	RoutingKey  string            `json:"routing_key,omitempty"`
	EventAction string            `json:"event_action"`
	Payload     PagerDutyPayload2 `json:"payload"`
}

type PagerDutyPayload2 struct {
	Summary   string `json:"summary"`
	Severity  string `json:"severity"`
	Source    string `json:"source"`
	Component string `json:"component"`
	Group     string `json:"group,omitempty"`
	Class     string `json:"class"`
}

// PagerDuty writes a PagerDuty Events API v2 payload.
func PagerDuty(w io.Writer, result diagnosis.Result) error {
	severity := "info"
	switch result.Severity {
	case "critical":
		severity = "critical"
	case "error":
		severity = "error"
	case "warning":
		severity = "warning"
	}
	payload := PagerDutyPayload{
		EventAction: "trigger",
		Payload: PagerDutyPayload2{
			Summary:   fmt.Sprintf("[%s] %s: %s", strings.ToUpper(result.Severity), result.Container.Name, result.Reason),
			Severity:  severity,
			Source:    result.Container.Name,
			Component: result.Container.ComposeService,
			Group:     result.Container.ComposeProject,
			Class:     result.Reason,
		},
	}
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	return encoder.Encode(payload)
}

// SlackPayload is the payload for a Slack incoming webhook.
type SlackPayload struct {
	Text        string            `json:"text"`
	Attachments []SlackAttachment `json:"attachments"`
}

type SlackAttachment struct {
	Color  string       `json:"color"`
	Title  string       `json:"title"`
	Text   string       `json:"text"`
	Fields []SlackField `json:"fields"`
}

type SlackField struct {
	Title string `json:"title"`
	Value string `json:"value"`
	Short bool   `json:"short"`
}

// Slack writes a Slack incoming webhook payload.
func Slack(w io.Writer, result diagnosis.Result) error {
	color := "#36a64f"
	switch result.Severity {
	case "critical":
		color = "#e01e5a"
	case "error":
		color = "#e89216"
	case "warning":
		color = "#f2c744"
	}
	payload := SlackPayload{
		Text: fmt.Sprintf(":warning: Container *%s* stopped — %s", result.Container.Name, result.Reason),
		Attachments: []SlackAttachment{
			{
				Color: color,
				Title: fmt.Sprintf("Diagnosis: %s", result.Reason),
				Text:  result.Summary,
				Fields: []SlackField{
					{Title: "Severity", Value: result.Severity, Short: true},
					{Title: "Confidence", Value: result.Confidence, Short: true},
					{Title: "Exit Code", Value: fmt.Sprintf("%d", result.Container.State.ExitCode), Short: true},
					{Title: "Image", Value: result.Container.Image, Short: true},
				},
			},
		},
	}
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	return encoder.Encode(payload)
}

// Prometheus writes the diagnosis as Prometheus text exposition format metrics.
func Prometheus(w io.Writer, result diagnosis.Result) error {
	var b strings.Builder
	c := result.Container

	b.WriteString("# HELP dockwhy_exit_code The exit code of the container.\n")
	b.WriteString("# TYPE dockwhy_exit_code gauge\n")
	fmt.Fprintf(&b, "dockwhy_exit_code{container=%q,image=%q,status=%q} %d\n", c.Name, c.Image, c.State.Status, c.State.ExitCode)

	b.WriteString("# HELP dockwhy_oom_killed Whether the container was OOM killed.\n")
	b.WriteString("# TYPE dockwhy_oom_killed gauge\n")
	oom := 0
	if c.State.OOMKilled {
		oom = 1
	}
	fmt.Fprintf(&b, "dockwhy_oom_killed{container=%q} %d\n", c.Name, oom)

	b.WriteString("# HELP dockwhy_restart_count The number of times the container has restarted.\n")
	b.WriteString("# TYPE dockwhy_restart_count gauge\n")
	fmt.Fprintf(&b, "dockwhy_restart_count{container=%q} %d\n", c.Name, c.RestartCount)

	b.WriteString("# HELP dockwhy_severity The severity of the diagnosis (0=info, 1=warning, 2=error, 3=critical).\n")
	b.WriteString("# TYPE dockwhy_severity gauge\n")
	fmt.Fprintf(&b, "dockwhy_severity{container=%q,reason=%q} %d\n", c.Name, result.Reason, severityInt(result.Severity))

	if result.Stats != nil {
		s := result.Stats
		b.WriteString("# HELP dockwhy_cpu_percent CPU usage percentage.\n")
		b.WriteString("# TYPE dockwhy_cpu_percent gauge\n")
		fmt.Fprintf(&b, "dockwhy_cpu_percent{container=%q} %.2f\n", c.Name, s.CPUPercent)
		b.WriteString("# HELP dockwhy_memory_usage_bytes Memory usage in bytes.\n")
		b.WriteString("# TYPE dockwhy_memory_usage_bytes gauge\n")
		fmt.Fprintf(&b, "dockwhy_memory_usage_bytes{container=%q} %d\n", c.Name, s.MemoryUsageBytes)
	}

	_, err := io.WriteString(w, b.String())
	return err
}

func severityInt(s string) int {
	switch s {
	case "critical":
		return 3
	case "error":
		return 2
	case "warning":
		return 1
	default:
		return 0
	}
}

// WebhookPayload is a generic webhook payload.
type WebhookPayload struct {
	Source     string             `json:"source"`
	Timestamp  string             `json:"timestamp"`
	Severity   string             `json:"severity"`
	Reason     string             `json:"reason"`
	Summary    string             `json:"summary"`
	Container  WebhookContainer   `json:"container"`
	Advice     []string           `json:"advice,omitempty"`
}

type WebhookContainer struct {
	Name       string `json:"name"`
	ID         string `json:"id"`
	Image      string `json:"image"`
	Status     string `json:"status"`
	ExitCode   int    `json:"exit_code"`
	OOMKilled  bool   `json:"oom_killed"`
}

// Webhook writes a generic JSON webhook payload.
func Webhook(w io.Writer, result diagnosis.Result) error {
	payload := WebhookPayload{
		Source:    "dockwhy",
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Severity:  result.Severity,
		Reason:   result.Reason,
		Summary:  result.Summary,
		Container: WebhookContainer{
			Name:      result.Container.Name,
			ID:        result.Container.ID,
			Image:     result.Container.Image,
			Status:    result.Container.State.Status,
			ExitCode:  result.Container.State.ExitCode,
			OOMKilled: result.Container.State.OOMKilled,
		},
		Advice: result.Advice,
	}
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	return encoder.Encode(payload)
}
