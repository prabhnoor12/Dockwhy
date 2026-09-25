// Copyright 2026 Prabhnoor12
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package diagnosis

import (
	"fmt"

	"github.com/prabhnoor12/dockwhy/internal/docker"
	"github.com/prabhnoor12/dockwhy/internal/format"
)

// ResourceRecommendation is a single sizing suggestion.
type ResourceRecommendation struct {
	Resource    string `json:"resource"`
	Current     string `json:"current"`
	Peak        string `json:"peak,omitempty"`
	Utilization string `json:"utilization"`
	Suggestion  string `json:"suggestion"`
	Priority    string `json:"priority"`
}

// ResourceReport holds all resource recommendations for a container.
type ResourceReport struct {
	Container      string                   `json:"container"`
	Recommendations []ResourceRecommendation `json:"recommendations"`
	Summary        string                   `json:"summary"`
}

// AnalyzeResources produces sizing recommendations from a stats snapshot
// and the container's configured limits.
func AnalyzeResources(containerName string, stats *docker.Stats, container docker.Container) ResourceReport {
	rr := ResourceReport{Container: containerName}
	if stats == nil {
		rr.Summary = "no resource snapshot available (container may be stopped)"
		return rr
	}

	if container.MemoryLimit > 0 {
		memLimit := container.MemoryLimit
		memUsage := stats.MemoryUsageBytes
		pct := float64(memUsage) / float64(memLimit) * 100
		rec := ResourceRecommendation{
			Resource:    "memory",
			Current:     format.Bytes(memLimit) + " limit",
			Peak:        format.Bytes(memUsage) + " usage",
			Utilization: fmt.Sprintf("%.0f%%", pct),
		}
		switch {
		case pct >= 90:
			rec.Priority = "critical"
			rec.Suggestion = fmt.Sprintf("Memory utilization is %.0f%%. Increase limit to %s or optimize allocation to avoid OOM kills.", pct, format.Bytes(memLimit*2))
		case pct >= 75:
			rec.Priority = "warning"
			rec.Suggestion = fmt.Sprintf("Memory utilization is %.0f%%. Consider increasing limit to %s for headroom.", pct, format.Bytes(memLimit*3/2))
		case pct < 20 && memLimit > 64*1024*1024:
			rec.Priority = "info"
			rec.Suggestion = fmt.Sprintf("Memory utilization is only %.0f%%. Limit could be reduced to %s to free host resources.", pct, format.Bytes(memLimit/2))
		default:
			rec.Priority = "ok"
			rec.Suggestion = fmt.Sprintf("Memory utilization is %.0f%%. Current limit is appropriate.", pct)
		}
		rr.Recommendations = append(rr.Recommendations, rec)
	} else {
		rr.Recommendations = append(rr.Recommendations, ResourceRecommendation{
			Resource:   "memory",
			Current:    "no limit",
			Peak:       format.Bytes(stats.MemoryUsageBytes) + " usage",
			Priority:   "warning",
			Suggestion: fmt.Sprintf("No memory limit set. Container is using %s. Set a limit to prevent uncontrolled growth.", format.Bytes(stats.MemoryUsageBytes)),
		})
	}

	if container.NanoCPUs > 0 {
		cpus := float64(container.NanoCPUs) / 1e9
		cpuPct := stats.CPUPercent
		rec := ResourceRecommendation{
			Resource:    "cpu",
			Current:     fmt.Sprintf("%.2f CPUs", cpus),
			Utilization: fmt.Sprintf("%.1f%%", cpuPct),
		}
		effectivePct := cpuPct / (cpus * 100) * 100
		switch {
		case effectivePct >= 90:
			rec.Priority = "critical"
			rec.Suggestion = fmt.Sprintf("CPU utilization is %.1f%% of %.2f CPUs. Increase allocation or optimize.", cpuPct, cpus)
		case effectivePct >= 75:
			rec.Priority = "warning"
			rec.Suggestion = fmt.Sprintf("CPU utilization is %.1f%% of %.2f CPUs. Consider increasing to %.2f CPUs.", cpuPct, cpus, cpus*1.5)
		default:
			rec.Priority = "ok"
			rec.Suggestion = fmt.Sprintf("CPU utilization is %.1f%% of %.2f CPUs. Current allocation is appropriate.", cpuPct, cpus)
		}
		rr.Recommendations = append(rr.Recommendations, rec)
	}

	if container.PidsLimit > 0 && stats.PidsCurrent > 0 {
		pct := float64(stats.PidsCurrent) / float64(container.PidsLimit) * 100
		rec := ResourceRecommendation{
			Resource:    "pids",
			Current:     fmt.Sprintf("%d limit", container.PidsLimit),
			Peak:        fmt.Sprintf("%d current", stats.PidsCurrent),
			Utilization: fmt.Sprintf("%.0f%%", pct),
		}
		if pct >= 80 {
			rec.Priority = "warning"
			rec.Suggestion = fmt.Sprintf("PID utilization is %.0f%%. Approaching limit; increase PidsLimit if needed.", pct)
		} else {
			rec.Priority = "ok"
			rec.Suggestion = fmt.Sprintf("PID utilization is %.0f%%. Current limit is appropriate.", pct)
		}
		rr.Recommendations = append(rr.Recommendations, rec)
	}

	rr.Summary = resourceSummary(rr.Recommendations)
	return rr
}

func resourceSummary(recs []ResourceRecommendation) string {
	critical := 0
	warning := 0
	for _, r := range recs {
		switch r.Priority {
		case "critical":
			critical++
		case "warning":
			warning++
		}
	}
	if critical > 0 {
		return fmt.Sprintf("%d critical resource issue(s) require attention", critical)
	}
	if warning > 0 {
		return fmt.Sprintf("%d resource warning(s) — consider adjusting limits", warning)
	}
	return "resource limits are appropriate for current usage"
}
