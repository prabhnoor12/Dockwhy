package docker

import "time"

// Container is the small, stable view of Docker inspect data used by the
// diagnosis package. Keeping Docker's wire format out of the rest of the app
// makes the reporting and diagnosis logic straightforward to test.
type Container struct {
	ID              string            `json:"id"`
	Name            string            `json:"name"`
	Image           string            `json:"image"`
	Created         string            `json:"created"`
	Entrypoint      []string          `json:"entrypoint,omitempty"`
	Command         []string          `json:"command,omitempty"`
	WorkingDir      string            `json:"working_dir,omitempty"`
	User            string            `json:"user,omitempty"`
	StopSignal      string            `json:"stop_signal,omitempty"`
	LogDriver       string            `json:"log_driver,omitempty"`
	Mounts          []Mount           `json:"mounts,omitempty"`
	ComposeProject  string            `json:"compose_project,omitempty"`
	ComposeService  string            `json:"compose_service,omitempty"`
	State           State             `json:"state"`
	RestartCount    int               `json:"restart_count"`
	RestartPolicy   string            `json:"restart_policy"`
	MemoryLimit     int64             `json:"memory_limit_bytes"`
	MemorySwapLimit int64             `json:"memory_swap_limit_bytes"`
	MemoryReserved  int64             `json:"memory_reservation_bytes"`
	NanoCPUs        int64             `json:"nano_cpus"`
	CPUQuota        int64             `json:"cpu_quota"`
	CPUPeriod       int64             `json:"cpu_period"`
	PidsLimit       int64             `json:"pids_limit"`
	DiskReadOnly    bool              `json:"disk_read_only"`
	DiskLimit       string            `json:"disk_limit,omitempty"`
	SizeRW          int64             `json:"writable_layer_bytes"`
	SizeRootFS      int64             `json:"root_filesystem_bytes"`
	Labels          map[string]string `json:"labels,omitempty"`
}

// Stats is a point-in-time resource usage snapshot from docker stats.
type Stats struct {
	CPUPercent       float64 `json:"cpu_percent"`
	MemoryUsageBytes int64   `json:"memory_usage_bytes"`
	MemoryLimitBytes int64   `json:"memory_limit_bytes"`
	MemoryPercent    float64 `json:"memory_percent"`
	NetworkRxBytes   int64   `json:"network_rx_bytes"`
	NetworkTxBytes   int64   `json:"network_tx_bytes"`
	BlockReadBytes   int64   `json:"block_read_bytes"`
	BlockWriteBytes  int64   `json:"block_write_bytes"`
	PidsCurrent      int64   `json:"pids_current"`
}

// Mount describes a single mount point on a container.
type Mount struct {
	Type        string `json:"type"`
	Name        string `json:"name,omitempty"`
	Source      string `json:"source,omitempty"`
	Destination string `json:"destination"`
	Mode        string `json:"mode,omitempty"`
	RW          bool   `json:"rw"`
	Propagation string `json:"propagation,omitempty"`
}

// State is the runtime state of a container as reported by docker inspect.
type State struct {
	Status     string  `json:"status"`
	Running    bool    `json:"running"`
	Paused     bool    `json:"paused"`
	Restarting bool    `json:"restarting"`
	OOMKilled  bool    `json:"oom_killed"`
	ExitCode   int     `json:"exit_code"`
	Error      string  `json:"error,omitempty"`
	StartedAt  string  `json:"started_at,omitempty"`
	FinishedAt string  `json:"finished_at,omitempty"`
	Health     *Health `json:"health,omitempty"`
}

// Health is the health-check state of a container.
type Health struct {
	Status        string        `json:"status"`
	FailingStreak int           `json:"failing_streak"`
	Log           []HealthCheck `json:"log,omitempty"`
}

// HealthCheck is a single health-check execution record.
type HealthCheck struct {
	Start    string `json:"start"`
	End      string `json:"end"`
	ExitCode int    `json:"exit_code"`
	Output   string `json:"output"`
}

// LogOutput holds the text and truncation status of container logs.
type LogOutput struct {
	Text      string `json:"-"`
	Truncated bool   `json:"-"`
}

// Event is a Docker lifecycle event for a container.
type Event struct {
	TimeNano int64             `json:"time_nano"`
	Action   string            `json:"action"`
	Actor    string            `json:"actor"`
	Attrs    map[string]string `json:"attributes,omitempty"`
}

// ContainerSummary is a lightweight container reference returned by project listing.
type ContainerSummary struct {
	ID             string `json:"id"`
	Name           string `json:"name"`
	ComposeService string `json:"compose_service,omitempty"`
	Status         string `json:"status"`
}

// Client abstracts Docker CLI access so the diagnosis and output packages
// can be tested without a running Docker daemon.
type Client interface {
	Inspect(name string) (Container, error)
	Logs(name string, tail int) (LogOutput, error)
	Events(containerID string, since time.Duration) ([]Event, error)
	Stats(name string) (Stats, error)
	ListByProject(project string) ([]ContainerSummary, error)
}
