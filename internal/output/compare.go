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

	"github.com/prabhnoor12/dockwhy/internal/diagnosis"
)

// CompareText writes a human-readable comparison report.
func CompareText(w io.Writer, result diagnosis.CompareResult) error {
	fmt.Fprintf(w, "Configuration Diff: %s vs %s\n", result.ContainerA, result.ContainerB)
	fmt.Fprintf(w, "%s\n\n", result.Summary)

	if len(result.Diffs) == 0 {
		fmt.Fprintln(w, "No differences found.")
		return nil
	}

	for _, d := range result.Diffs {
		tag := ""
		switch d.Category {
		case diagnosis.DiffCritical:
			tag = "[CRITICAL]"
		case diagnosis.DiffWarning:
			tag = "[WARNING] "
		case diagnosis.DiffInfo:
			tag = "[info]    "
		}
		fmt.Fprintf(w, "  %s %s\n", tag, d.Field)
		fmt.Fprintf(w, "    before: %s\n", d.Before)
		fmt.Fprintf(w, "    after:  %s\n", d.After)
		if d.Note != "" {
			fmt.Fprintf(w, "    note:   %s\n", d.Note)
		}
		fmt.Fprintln(w)
	}
	return nil
}

// CompareJSON writes the comparison as JSON.
func CompareJSON(w io.Writer, result diagnosis.CompareResult) error {
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	return encoder.Encode(result)
}
