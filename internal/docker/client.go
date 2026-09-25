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
	ctx             context.Context
	runOverride     func(maxBytes int, args ...string) (commandOutput, error)
}

func NewCLIClient(ctx context.Context, timeout time.Duration) *CLIClient {
	if timeout <= 0 {
		timeout = 10 * time.Second
	}
	if ctx == nil {
		ctx = context.Background()
	}
	return &CLIClient{Binary: "docker", Timeout: timeout, MaxInspectBytes: 16 << 20, MaxLogBytes: 2 << 20, ctx: ctx}
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

func (c *CLIClient) Stats(name string) (Stats, error) {
	out, err := c.run(1<<20, "stats", "--no-stream", "--format", "{{json .}}", name)
	if err != nil {
		return Stats{}, dockerCommandError("stats", name, out, err)
	}
	line := strings.TrimSpace(out.stdout)
	if line == "" {
		return Stats{}, fmt.Errorf("decode docker stats: empty output")
	}
	var raw statsRecord
	if err := json.Unmarshal([]byte(line), &raw); err != nil {
		return Stats{}, fmt.Errorf("decode docker stats: %w", err)
	}
	stats, err := raw.stats()
	if err != nil {
		return Stats{}, fmt.Errorf("decode docker stats: %w", err)
	}
	return stats, nil
}

func (c *CLIClient) ListByProject(project string) ([]ContainerSummary, error) {
	out, err := c.run(4<<20, "ps", "-a", "--filter", "label=com.docker.compose.project="+project, "--format", "{{json .}}")
	if err != nil {
		return nil, dockerCommandError("ps", project, out, err)
	}
	var summaries []ContainerSummary
	for _, line := range strings.Split(strings.TrimSpace(out.stdout), "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		var raw struct {
			ID     string `json:"ID"`
			Names  string `json:"Names"`
			Status string `json:"Status"`
			Labels string `json:"Labels"`
		}
		if err := json.Unmarshal([]byte(line), &raw); err != nil {
			return nil, fmt.Errorf("decode docker ps output: %w", err)
		}
		summary := ContainerSummary{
			ID:     raw.ID,
			Name:   raw.Names,
			Status: raw.Status,
		}
		for _, label := range strings.Split(raw.Labels, ",") {
			if strings.HasPrefix(label, "com.docker.compose.service=") {
				summary.ComposeService = strings.TrimPrefix(label, "com.docker.compose.service=")
			}
		}
		summaries = append(summaries, summary)
	}
	return summaries, nil
}

type commandOutput struct {
	stdout    string
	stderr    string
	truncated bool
}

func (c *CLIClient) run(maxBytes int, args ...string) (commandOutput, error) {
	if c.runOverride != nil {
		return c.runOverride(maxBytes, args...)
	}
	if maxBytes <= 0 {
		maxBytes = 1 << 20
	}
	timeout := c.Timeout
	if timeout <= 0 {
		timeout = 10 * time.Second
	}
	ctx, cancel := context.WithTimeout(c.ctx, timeout)
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
		return fmt.Errorf("docker %s %q timed out: %w", action, name, err)
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
		return fmt.Errorf("docker CLI is not installed or not on PATH: %w", err)
	}
	return fmt.Errorf("docker %s %q failed: %s: %w", action, name, detail, err)
}

type inspectContainer struct {
	ID           string            `json:"Id"`
	Name         string            `json:"Name"`
	Created      string            `json:"Created"`
	Mounts       []inspectMount    `json:"Mounts"`
	RestartCount int               `json:"RestartCount"`
	Config       inspectConfig     `json:"Config"`
	State        inspectState      `json:"State"`
	Host         inspectHostConfig `json:"HostConfig"`
	SizeRW       int64             `json:"SizeRw"`
	SizeRoot     int64             `json:"SizeRootFs"`
}

type inspectConfig struct {
	Image      string            `json:"Image"`
	Labels     map[string]string `json:"Labels"`
	Entrypoint []string          `json:"Entrypoint"`
	Cmd        []string          `json:"Cmd"`
	WorkingDir string            `json:"WorkingDir"`
	User       string            `json:"User"`
	StopSignal string            `json:"StopSignal"`
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
	LogConfig         struct {
		Type string `json:"Type"`
	} `json:"LogConfig"`
}

type inspectMount struct {
	Type        string `json:"Type"`
	Name        string `json:"Name"`
	Source      string `json:"Source"`
	Destination string `json:"Destination"`
	Mode        string `json:"Mode"`
	RW          bool   `json:"RW"`
	Propagation string `json:"Propagation"`
}

type eventRecord struct {
	Action   string `json:"Action"`
	TimeNano int64  `json:"timeNano"`
	Actor    struct {
		ID         string            `json:"ID"`
		Attributes map[string]string `json:"Attributes"`
	} `json:"Actor"`
}

type statsRecord struct {
	CPUPercent string `json:"CPUPerc"`
	MemUsage   string `json:"MemUsage"`
	MemPercent string `json:"MemPerc"`
	NetIO      string `json:"NetIO"`
	BlockIO    string `json:"BlockIO"`
	PIDs       string `json:"PIDs"`
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
		Entrypoint: append([]string(nil), r.Config.Entrypoint...), Command: append([]string(nil), r.Config.Cmd...),
		WorkingDir: r.Config.WorkingDir, User: r.Config.User, StopSignal: r.Config.StopSignal,
		LogDriver:      r.Host.LogConfig.Type,
		ComposeProject: r.Config.Labels["com.docker.compose.project"],
		ComposeService: r.Config.Labels["com.docker.compose.service"],
		State: State{Status: r.State.Status, Running: r.State.Running, Paused: r.State.Paused,
			Restarting: r.State.Restarting, OOMKilled: r.State.OOMKilled, ExitCode: r.State.ExitCode,
			Error: r.State.Error, StartedAt: r.State.StartedAt, FinishedAt: r.State.FinishedAt, Health: health},
		RestartCount: r.RestartCount, RestartPolicy: r.Host.RestartPolicy.Name,
		MemoryLimit: r.Host.Memory, MemorySwapLimit: r.Host.MemorySwap, MemoryReserved: r.Host.MemoryReservation,
		NanoCPUs: r.Host.NanoCPUs, CPUQuota: r.Host.CPUQuota, CPUPeriod: r.Host.CPUPeriod, PidsLimit: r.Host.PidsLimit,
		DiskReadOnly: r.Host.ReadonlyRootfs, DiskLimit: diskLimit, SizeRW: r.SizeRW, SizeRootFS: r.SizeRoot,
		Mounts: normalizeMounts(r.Mounts), Labels: r.Config.Labels,
	}
}

func (r statsRecord) stats() (Stats, error) {
	cpu, err := parsePercent(r.CPUPercent)
	if err != nil {
		return Stats{}, fmt.Errorf("CPU percentage %q: %w", r.CPUPercent, err)
	}
	memoryUsage, memoryLimit, err := parseBytePair(r.MemUsage)
	if err != nil {
		return Stats{}, fmt.Errorf("memory usage %q: %w", r.MemUsage, err)
	}
	memoryPercent, err := parsePercent(r.MemPercent)
	if err != nil {
		return Stats{}, fmt.Errorf("memory percentage %q: %w", r.MemPercent, err)
	}
	networkRx, networkTx, err := parseBytePair(r.NetIO)
	if err != nil {
		return Stats{}, fmt.Errorf("network I/O %q: %w", r.NetIO, err)
	}
	blockRead, blockWrite, err := parseBytePair(r.BlockIO)
	if err != nil {
		return Stats{}, fmt.Errorf("block I/O %q: %w", r.BlockIO, err)
	}
	pids := int64(0)
	if strings.TrimSpace(r.PIDs) != "" && strings.TrimSpace(r.PIDs) != "N/A" {
		pids, err = strconv.ParseInt(strings.TrimSpace(r.PIDs), 10, 64)
		if err != nil {
			return Stats{}, fmt.Errorf("PIDs %q: %w", r.PIDs, err)
		}
	}
	return Stats{
		CPUPercent: cpu, MemoryUsageBytes: memoryUsage, MemoryLimitBytes: memoryLimit,
		MemoryPercent: memoryPercent, NetworkRxBytes: networkRx, NetworkTxBytes: networkTx,
		BlockReadBytes: blockRead, BlockWriteBytes: blockWrite, PidsCurrent: pids,
	}, nil
}

func parsePercent(value string) (float64, error) {
	value = strings.TrimSpace(strings.TrimSuffix(value, "%"))
	value = strings.Replace(value, ",", ".", 1)
	return strconv.ParseFloat(value, 64)
}

func parseBytePair(value string) (int64, int64, error) {
	parts := strings.SplitN(value, " / ", 2)
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("expected two byte values")
	}
	first, err := parseBytes(parts[0])
	if err != nil {
		return 0, 0, err
	}
	second, err := parseBytes(parts[1])
	if err != nil {
		return 0, 0, err
	}
	return first, second, nil
}

func parseBytes(value string) (int64, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0, fmt.Errorf("empty byte value")
	}
	units := []struct {
		suffix string
		factor float64
	}{
		{"KiB", 1 << 10}, {"MiB", 1 << 20}, {"GiB", 1 << 30}, {"TiB", 1 << 40},
		{"kB", 1e3}, {"MB", 1e6}, {"GB", 1e9}, {"TB", 1e12}, {"B", 1},
	}
	for _, unit := range units {
		if strings.HasSuffix(value, unit.suffix) {
			number := strings.TrimSpace(strings.TrimSuffix(value, unit.suffix))
			number = strings.Replace(number, ",", ".", 1)
			parsed, err := strconv.ParseFloat(number, 64)
			if err != nil {
				return 0, err
			}
			return int64(parsed * unit.factor), nil
		}
	}
	return 0, fmt.Errorf("unknown byte unit")
}

func normalizeMounts(raw []inspectMount) []Mount {
	if len(raw) == 0 {
		return nil
	}
	mounts := make([]Mount, 0, len(raw))
	for _, mount := range raw {
		mounts = append(mounts, Mount{
			Type: mount.Type, Name: mount.Name, Source: mount.Source,
			Destination: mount.Destination, Mode: mount.Mode, RW: mount.RW,
			Propagation: mount.Propagation,
		})
	}
	return mounts
}
