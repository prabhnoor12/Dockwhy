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
	"strings"
)

// SmartLogResult holds the extracted relevant log sections.
type SmartLogResult struct {
	Sections []LogSection `json:"sections"`
	Summary  string       `json:"summary"`
}

// LogSection is a contiguous block of related log lines around a signal.
type LogSection struct {
	Kind    string   `json:"kind"`
	Lines   []string `json:"lines"`
	Context string   `json:"context,omitempty"`
}

// ExtractSmartLogs scans log text for known error patterns and returns the
// most relevant sections instead of an arbitrary tail.
func ExtractSmartLogs(logs string) SmartLogResult {
	if strings.TrimSpace(logs) == "" {
		return SmartLogResult{Summary: "no log output"}
	}

	lines := strings.Split(logs, "\n")
	var sections []LogSection
	used := make(map[int]bool)

	patterns := []struct {
		kind    string
		context string
		match   func(line string) bool
		expand  int
	}{
		{"panic", "Go panic trace", matchPanic, 10},
		{"stack_trace", "Java/JVM stack trace", matchJavaStack, 15},
		{"fatal", "Fatal error", matchFatal, 3},
		{"error", "Error message", matchError, 2},
		{"exception", "Exception", matchException, 5},
		{"oom", "Out of memory", matchOOM, 2},
		{"connection_refused", "Connection failure", matchConnectionRefused, 2},
		{"timeout", "Timeout", matchTimeout, 2},
		{"segfault", "Segmentation fault", matchSegfault, 2},
	}

	for _, p := range patterns {
		for i, line := range lines {
			if used[i] || !p.match(line) {
				continue
			}
			start := maxInt(0, i-p.expand)
			end := minInt(len(lines), i+p.expand+1)
			var sectionLines []string
			for j := start; j < end; j++ {
				sectionLines = append(sectionLines, lines[j])
				used[j] = true
			}
			sections = append(sections, LogSection{
				Kind:    p.kind,
				Lines:   sectionLines,
				Context: p.context,
			})
		}
	}

	if len(sections) == 0 {
		n := len(lines)
		if n > 20 {
			n = 20
		}
		var tail []string
		for i := len(lines) - n; i < len(lines); i++ {
			if strings.TrimSpace(lines[i]) != "" {
				tail = append(tail, lines[i])
			}
		}
		if len(tail) > 0 {
			sections = append(sections, LogSection{
				Kind:    "tail",
				Lines:   tail,
				Context: "last non-empty log lines (no error patterns detected)",
			})
		}
	}

	summary := smartLogSummary(sections)
	return SmartLogResult{Sections: sections, Summary: summary}
}

func smartLogSummary(sections []LogSection) string {
	if len(sections) == 0 {
		return "no relevant log sections found"
	}
	kinds := make(map[string]int)
	for _, s := range sections {
		kinds[s.Kind]++
	}
	var parts []string
	for kind, count := range kinds {
		parts = append(parts, formatCount(kind, count))
	}
	return "found: " + strings.Join(parts, ", ")
}

func formatCount(kind string, count int) string {
	if count == 1 {
		return "1 " + kind
	}
	return strings.Replace(string(rune('0'+count))+" "+kind+"s", string(rune('0'+count)), string(rune('0'+count)), 1)
}

func matchPanic(line string) bool {
	lower := strings.ToLower(line)
	return strings.Contains(lower, "panic:") || strings.Contains(lower, "goroutine") && strings.Contains(lower, "[running]")
}

func matchJavaStack(line string) bool {
	trimmed := strings.TrimSpace(line)
	return strings.HasPrefix(trimmed, "at ") || strings.Contains(line, "Exception in thread") || strings.Contains(line, "Caused by:")
}

func matchFatal(line string) bool {
	lower := strings.ToLower(line)
	return strings.Contains(lower, "fatal") || strings.Contains(lower, "fatalf")
}

func matchError(line string) bool {
	lower := strings.ToLower(line)
	return (strings.Contains(lower, "error") || strings.Contains(lower, "err:")) && !strings.Contains(lower, "error=")
}

func matchException(line string) bool {
	lower := strings.ToLower(line)
	return strings.Contains(lower, "exception") || strings.Contains(lower, "traceback")
}

func matchOOM(line string) bool {
	lower := strings.ToLower(line)
	return strings.Contains(lower, "out of memory") || strings.Contains(lower, "cannot allocate memory") || strings.Contains(lower, "oom")
}

func matchConnectionRefused(line string) bool {
	lower := strings.ToLower(line)
	return strings.Contains(lower, "connection refused") || strings.Contains(lower, "connection reset") || strings.Contains(lower, "no route to host")
}

func matchTimeout(line string) bool {
	lower := strings.ToLower(line)
	return strings.Contains(lower, "timeout") || strings.Contains(lower, "timed out") || strings.Contains(lower, "deadline exceeded")
}

func matchSegfault(line string) bool {
	lower := strings.ToLower(line)
	return strings.Contains(lower, "segfault") || strings.Contains(lower, "sigsegv") || strings.Contains(lower, "segmentation fault")
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
