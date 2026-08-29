package docker

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

// CLIClient talks to Docker using the same CLI users use interactively. It
// also works with Docker contexts and DOCKER_HOST without extra configuration.
type CLIClient struct {
	Binary          string
	Timeout         time.Duration
	MaxInspectBytes int
	MaxLogBytes     int
}

func NewCLIClient() *CLIClient {
	return &CLIClient{Binary: "docker", Timeout: 10 * time.Second, MaxInspectBytes: 16 << 20, MaxLogBytes: 2 << 20}
}

func (c *CLIClient) SetTimeout(timeout time.Duration) {
	if timeout > 0 {
		c.Timeout = timeout
	}
}

func (c *CLIClient) Inspect(name string) (Container, error) {
	var raw inspectContainer
	out, err := c.run(c.MaxInspectBytes, "inspect", "--type", "container", "--size", name)
	if err != nil {
		return Container{}, dockerCommandError("inspect", name, out, err)
	}
	if out.truncated {
		return Container{}, fmt.Errorf("docker inspect output exceeded the %d-byte safety limit", c.MaxInspectBytes)
	}
	var items []inspectContainer
	if err := json.Unmarshal([]byte(out.stdout), &items); err != nil {
		return Container{}, fmt.Errorf("decode docker inspect output: %w", err)
	}
	if len(items) == 0 {
		return Container{}, fmt.Errorf("container %q was not found", name)
	}
	raw = items[0]
	return raw.container(), nil
}

func (c *CLIClient) Logs(name string, tail int) (LogOutput, error) {
	out, err := c.run(c.MaxLogBytes, "logs", "--tail", strconv.Itoa(tail), "--timestamps", name)
	if err != nil {
		return LogOutput{Text: strings.TrimSpace(joinOutput(out)), Truncated: out.truncated}, dockerCommandError("logs", name, out, err)
	}
	return LogOutput{Text: strings.TrimSpace(joinOutput(out)), Truncated: out.truncated}, nil
}

func (c *CLIClient) Events(containerID string, since time.Duration) ([]Event, error) {
	if since <= 0 {
		since = 24 * time.Hour
	}
	now := time.Now().UTC()
	out, err := c.run(2<<20, "events", "--since", now.Add(-since).Format(time.RFC3339Nano), "--until", now.Format(time.RFC3339Nano), "--filter", "container="+containerID, "--format", "{{json .}}")
	if err != nil {
		return nil, dockerCommandError("events", containerID, out, err)
	}
	var events []Event
	for _, line := range strings.Split(strings.TrimSpace(out.stdout), "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		var raw eventRecord
		if err := json.Unmarshal([]byte(line), &raw); err != nil {
			return nil, fmt.Errorf("decode docker event: %w", err)
		}
		events = append(events, Event{TimeNano: raw.TimeNano, Action: raw.Action, Actor: raw.Actor.ID, Attrs: raw.Actor.Attributes})
	}
	return events, nil
}

type commandOutput struct {
	stdout    string
	stderr    string
	truncated bool
}

func (c *CLIClient) run(maxBytes int, args ...string) (commandOutput, error) {
	if maxBytes <= 0 {
		maxBytes = 1 << 20
	}
	timeout := c.Timeout
	if timeout <= 0 {
		timeout = 10 * time.Second
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, c.Binary, args...)
	var stdout, stderr limitedBuffer
	stdout.limit, stderr.limit = maxBytes, 1<<20
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	if ctx.Err() != nil {
		return commandOutput{stdout: stdout.String(), stderr: stderr.String(), truncated: stdout.truncated || stderr.truncated}, ctx.Err()
	}
	return commandOutput{stdout: stdout.String(), stderr: stderr.String(), truncated: stdout.truncated || stderr.truncated}, err
}

func dockerCommandError(action, name string, output commandOutput, err error) error {
	if errors.Is(err, context.DeadlineExceeded) {
		return fmt.Errorf("docker %s %q timed out", action, name)
	}
	detail := strings.TrimSpace(output.stderr)
	if detail == "" {
		detail = strings.TrimSpace(output.stdout)
	}
	if detail == "" {
		detail = err.Error()
	}
	if output.truncated {
		detail += " [docker output truncated]"
	}
	if errors.Is(err, exec.ErrNotFound) {
		return fmt.Errorf("docker CLI is not installed or not on PATH")
	}
	return fmt.Errorf("docker %s %q failed: %s", action, name, detail)
}

type inspectContainer struct {
	ID           string            `json:"Id"`
	Name         string            `json:"Name"`
	Created      string            `json:"Created"`
	RestartCount int               `json:"RestartCount"`
	Config       inspectConfig     `json:"Config"`
	State        inspectState      `json:"State"`
	Host         inspectHostConfig `json:"HostConfig"`
	SizeRW       int64             `json:"SizeRw"`
	SizeRoot     int64             `json:"SizeRootFs"`
}

type inspectConfig struct {
	Image  string            `json:"Image"`
	Labels map[string]string `json:"Labels"`
}

type inspectState struct {
	Status     string         `json:"Status"`
	Running    bool           `json:"Running"`
	Paused     bool           `json:"Paused"`
	Restarting bool           `json:"Restarting"`
	OOMKilled  bool           `json:"OOMKilled"`
	ExitCode   int            `json:"ExitCode"`
	Error      string         `json:"Error"`
	StartedAt  string         `json:"StartedAt"`
	FinishedAt string         `json:"FinishedAt"`
	Health     *inspectHealth `json:"Health"`
}

type inspectHealth struct {
	Status        string        `json:"Status"`
	FailingStreak int           `json:"FailingStreak"`
	Log           []HealthCheck `json:"Log"`
}

type inspectHostConfig struct {
	RestartPolicy struct {
		Name string `json:"Name"`
	} `json:"RestartPolicy"`
	Memory            int64             `json:"Memory"`
	MemorySwap        int64             `json:"MemorySwap"`
	MemoryReservation int64             `json:"MemoryReservation"`
	NanoCPUs          int64             `json:"NanoCpus"`
	CPUQuota          int64             `json:"CpuQuota"`
	CPUPeriod         int64             `json:"CpuPeriod"`
	PidsLimit         int64             `json:"PidsLimit"`
	ReadonlyRootfs    bool              `json:"ReadonlyRootfs"`
	StorageOpt        map[string]string `json:"StorageOpt"`
}

type eventRecord struct {
	Action   string `json:"Action"`
	TimeNano int64  `json:"timeNano"`
	Actor    struct {
		ID         string            `json:"ID"`
		Attributes map[string]string `json:"Attributes"`
	} `json:"Actor"`
}

type limitedBuffer struct {
	bytes.Buffer
	limit     int
	truncated bool
}

func (b *limitedBuffer) Write(p []byte) (int, error) {
	remaining := b.limit - b.Len()
	if remaining <= 0 {
		b.truncated = true
		return len(p), nil
	}
	if len(p) > remaining {
		_, _ = b.Buffer.Write(p[:remaining])
		b.truncated = true
		return len(p), nil
	}
	return b.Buffer.Write(p)
}

func joinOutput(output commandOutput) string {
	if output.stderr == "" {
		return output.stdout
	}
	if output.stdout == "" {
		return output.stderr
	}
	return output.stdout + "\n" + output.stderr
}

func (r inspectContainer) container() Container {
	diskLimit := ""
	if r.Host.StorageOpt != nil {
		diskLimit = r.Host.StorageOpt["size"]
	}
	var health *Health
	if r.State.Health != nil {
		health = &Health{
			Status: r.State.Health.Status, FailingStreak: r.State.Health.FailingStreak,
			Log: r.State.Health.Log,
		}
	}
	return Container{
		ID: r.ID, Name: strings.TrimPrefix(r.Name, "/"), Image: r.Config.Image, Created: r.Created,
		State: State{Status: r.State.Status, Running: r.State.Running, Paused: r.State.Paused,
			Restarting: r.State.Restarting, OOMKilled: r.State.OOMKilled, ExitCode: r.State.ExitCode,
			Error: r.State.Error, StartedAt: r.State.StartedAt, FinishedAt: r.State.FinishedAt, Health: health},
		RestartCount: r.RestartCount, RestartPolicy: r.Host.RestartPolicy.Name,
		MemoryLimit: r.Host.Memory, MemorySwapLimit: r.Host.MemorySwap, MemoryReserved: r.Host.MemoryReservation,
		NanoCPUs: r.Host.NanoCPUs, CPUQuota: r.Host.CPUQuota, CPUPeriod: r.Host.CPUPeriod, PidsLimit: r.Host.PidsLimit,
		DiskReadOnly: r.Host.ReadonlyRootfs, DiskLimit: diskLimit, SizeRW: r.SizeRW, SizeRootFS: r.SizeRoot,
		Labels: r.Config.Labels,
	}
}
