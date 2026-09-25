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

package output

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/prabhnoor12/dockwhy/internal/diagnosis"
	"github.com/prabhnoor12/dockwhy/internal/format"
)

// ProjectJSON writes the project diagnosis as indented JSON.
func ProjectJSON(w io.Writer, result diagnosis.ProjectResult) error {
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	return encoder.Encode(result)
}

// ProjectText writes the project diagnosis as human-readable text.
func ProjectText(w io.Writer, result diagnosis.ProjectResult) error {
	var b strings.Builder
	fmt.Fprintf(&b, "Project: %s\n", result.Project)
	fmt.Fprintf(&b, "%s\n", result.Summary)

	if result.RootCause != nil {
		fmt.Fprintf(&b, "\nRoot cause: %s\n", result.RootCause.Container.Name)
		fmt.Fprintf(&b, "  Reason: %s\n", result.RootCause.Result.Reason)
		fmt.Fprintf(&b, "  %s\n", result.RootCause.Result.Summary)
		if len(result.RootCause.Result.Advice) > 0 {
			fmt.Fprintf(&b, "  Fix: %s\n", result.RootCause.Result.Advice[0])
		}
	}

	fmt.Fprintln(&b, "\nContainers:")
	for _, d := range result.Containers {
		c := d.Container
		status := c.State.Status
		if c.State.Health != nil && c.State.Health.Status == "unhealthy" {
			status += " (unhealthy)"
		}
		marker := "  "
		if result.RootCause != nil && result.RootCause.Container.Name == c.Name {
			marker = "->"
		}
		fmt.Fprintf(&b, "  %s %-20s %-12s exit=%d", marker, c.Name, status, c.State.ExitCode)
		if c.ComposeService != "" {
			fmt.Fprintf(&b, "  service=%s", c.ComposeService)
		}
		if c.MemoryLimit > 0 {
			fmt.Fprintf(&b, "  mem_limit=%s", format.Bytes(c.MemoryLimit))
		}
		fmt.Fprintln(&b)
	}

	if len(result.Timeline) > 0 {
		fmt.Fprintln(&b, "\nTimeline:")
		for _, entry := range result.Timeline {
			service := ""
			if entry.Service != "" {
				service = fmt.Sprintf("[%s] ", entry.Service)
			}
			detail := ""
			if entry.Detail != "" {
				detail = " " + entry.Detail
			}
			fmt.Fprintf(&b, "  %-25s %s%s: %s%s\n",
				formatEventTime(entry.TimeNano), service, entry.Container,
				describeEvent(entry.Event), detail)
		}
	}

	_, err := io.WriteString(w, b.String())
	return err
}
