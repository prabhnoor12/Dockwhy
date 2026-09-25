package diagnosis

import (
	"fmt"

	"github.com/prabhnoor12/dockwhy/internal/docker"
)

// DiffCategory groups a configuration difference by importance.
type DiffCategory string

const (
	DiffCritical DiffCategory = "critical"
	DiffWarning  DiffCategory = "warning"
	DiffInfo     DiffCategory = "info"
)

// ConfigDiff is a single configuration difference between two containers.
type ConfigDiff struct {
	Field    string       `json:"field"`
	Before   string       `json:"before"`
	After    string       `json:"after"`
	Category DiffCategory `json:"category"`
	Note     string       `json:"note,omitempty"`
}

// CompareResult summarizes the differences between two containers.
type CompareResult struct {
	ContainerA   string       `json:"container_a"`
	ContainerB   string       `json:"container_b"`
	Diffs        []ConfigDiff `json:"diffs"`
	Summary      string       `json:"summary"`
	CriticalCount int         `json:"critical_count"`
	WarningCount  int         `json:"warning_count"`
}

// CompareContainers compares two container configurations and returns the
// differences, categorized by how likely they are to cause failures.
func CompareContainers(a, b docker.Container) CompareResult {
	cr := CompareResult{
		ContainerA: a.Name,
		ContainerB: b.Name,
	}

	cr.addDiff("image", a.Image, b.Image, DiffCritical, "different image may have different behavior or bugs")
	cr.addDiff("entrypoint", joinStr(a.Entrypoint), joinStr(b.Entrypoint), DiffCritical, "different entrypoint changes what the container runs")
	cr.addDiff("command", joinStr(a.Command), joinStr(b.Command), DiffWarning, "different command may change runtime behavior")
	cr.addDiff("working_dir", a.WorkingDir, b.WorkingDir, DiffWarning, "different working directory may affect relative paths")
	cr.addDiff("user", a.User, b.User, DiffWarning, "different user may cause permission issues")
	cr.addDiff("stop_signal", a.StopSignal, b.StopSignal, DiffInfo, "")
	cr.addDiff("restart_policy", a.RestartPolicy, b.RestartPolicy, DiffWarning, "different restart policy affects recovery behavior")
	cr.addDiff("read_only_rootfs", fmt.Sprintf("%t", a.DiskReadOnly), fmt.Sprintf("%t", b.DiskReadOnly), DiffWarning, "read-only rootfs prevents writes")

	cr.addIntDiff("memory_limit", a.MemoryLimit, b.MemoryLimit, DiffCritical, "different memory limits may cause OOM")
	cr.addIntDiff("memory_swap_limit", a.MemorySwapLimit, b.MemorySwapLimit, DiffWarning, "")
	cr.addIntDiff("memory_reservation", a.MemoryReserved, b.MemoryReserved, DiffInfo, "")
	cr.addIntDiff("nano_cpus", a.NanoCPUs, b.NanoCPUs, DiffWarning, "different CPU limits may cause throttling")
	cr.addIntDiff("cpu_quota", a.CPUQuota, b.CPUQuota, DiffWarning, "")
	cr.addIntDiff("cpu_period", a.CPUPeriod, b.CPUPeriod, DiffInfo, "")
	cr.addIntDiff("pids_limit", a.PidsLimit, b.PidsLimit, DiffWarning, "different PID limits may kill processes")

	cr.addIntDiff("restart_count", int64(a.RestartCount), int64(b.RestartCount), DiffInfo, "")

	aMounts := summarizeMounts(a.Mounts)
	bMounts := summarizeMounts(b.Mounts)
	cr.addDiff("mounts", aMounts, bMounts, DiffWarning, "different mounts change available data")

	cr.addDiff("log_driver", a.LogDriver, b.LogDriver, DiffInfo, "")

	criticalCount := 0
	warningCount := 0
	for _, d := range cr.Diffs {
		switch d.Category {
		case DiffCritical:
			criticalCount++
		case DiffWarning:
			warningCount++
		}
	}
	cr.CriticalCount = criticalCount
	cr.WarningCount = warningCount

	if len(cr.Diffs) == 0 {
		cr.Summary = fmt.Sprintf("No configuration differences found between %s and %s", a.Name, b.Name)
	} else {
		cr.Summary = fmt.Sprintf("Found %d difference(s) between %s and %s (%d critical, %d warning)",
			len(cr.Diffs), a.Name, b.Name, criticalCount, warningCount)
	}

	return cr
}

func (cr *CompareResult) addDiff(field, a, b string, category DiffCategory, note string) {
	if a == b {
		return
	}
	if a == "" {
		a = "(not set)"
	}
	if b == "" {
		b = "(not set)"
	}
	cr.Diffs = append(cr.Diffs, ConfigDiff{
		Field:    field,
		Before:   a,
		After:    b,
		Category: category,
		Note:     note,
	})
}

func (cr *CompareResult) addIntDiff(field string, a, b int64, category DiffCategory, note string) {
	if a == b {
		return
	}
	cr.Diffs = append(cr.Diffs, ConfigDiff{
		Field:    field,
		Before:   fmt.Sprintf("%d", a),
		After:    fmt.Sprintf("%d", b),
		Category: category,
		Note:     note,
	})
}

func joinStr(ss []string) string {
	if len(ss) == 0 {
		return ""
	}
	result := ss[0]
	for _, s := range ss[1:] {
		result += " " + s
	}
	return result
}

func summarizeMounts(mounts []docker.Mount) string {
	if len(mounts) == 0 {
		return "(none)"
	}
	result := ""
	for i, m := range mounts {
		if i > 0 {
			result += ", "
		}
		result += m.Source + ":" + m.Destination
	}
	return result
}
