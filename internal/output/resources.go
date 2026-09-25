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
)

// ResourceJSON writes the resource report as indented JSON.
func ResourceJSON(w io.Writer, report diagnosis.ResourceReport) error {
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	return encoder.Encode(report)
}

// ResourceText writes the resource report as human-readable text.
func ResourceText(w io.Writer, report diagnosis.ResourceReport) error {
	var b strings.Builder
	fmt.Fprintf(&b, "Resource analysis: %s\n", report.Container)
	fmt.Fprintf(&b, "%s\n", report.Summary)

	for _, rec := range report.Recommendations {
		fmt.Fprintln(&b)
		priority := strings.ToUpper(rec.Priority)
		fmt.Fprintf(&b, "  [%s] %s\n", priority, rec.Resource)
		fmt.Fprintf(&b, "    Current:  %s\n", rec.Current)
		if rec.Peak != "" {
			fmt.Fprintf(&b, "    Peak:     %s\n", rec.Peak)
		}
		fmt.Fprintf(&b, "    Usage:    %s\n", rec.Utilization)
		fmt.Fprintf(&b, "    Advice:   %s\n", rec.Suggestion)
	}

	_, err := io.WriteString(w, b.String())
	return err
}
