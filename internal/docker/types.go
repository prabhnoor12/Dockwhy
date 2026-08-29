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

type Health struct {
	Status        string        `json:"status"`
	FailingStreak int           `json:"failing_streak"`
	Log           []HealthCheck `json:"log,omitempty"`
}

type HealthCheck struct {
	Start    string `json:"start"`
	End      string `json:"end"`
	ExitCode int    `json:"exit_code"`
	Output   string `json:"output"`
}

type LogOutput struct {
	Text      string `json:"-"`
	Truncated bool   `json:"-"`
}

type Event struct {
	TimeNano int64             `json:"time_nano"`
	Action   string            `json:"action"`
	Actor    string            `json:"actor"`
	Attrs    map[string]string `json:"attributes,omitempty"`
}

type Client interface {
	Inspect(name string) (Container, error)
	Logs(name string, tail int) (LogOutput, error)
	Events(containerID string, since time.Duration) ([]Event, error)
}
