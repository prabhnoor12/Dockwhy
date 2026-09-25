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
	"testing"
)

func TestExtractSmartLogsFindsPanic(t *testing.T) {
	logs := "2026-09-25 starting up\npanic: runtime error: index out of range [5] with length 3\ngoroutine 1 [running]:\nmain.processItems(0xc0001)\n  /app/main.go:42\n"
	result := ExtractSmartLogs(logs)
	if len(result.Sections) == 0 {
		t.Fatal("expected at least one section")
	}
	found := false
	for _, s := range result.Sections {
		if s.Kind == "panic" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected panic section, got: %#v", result.Sections)
	}
}

func TestExtractSmartLogsFindsFatal(t *testing.T) {
	logs := "starting\nFATAL: configuration file not found\nshutting down\n"
	result := ExtractSmartLogs(logs)
	found := false
	for _, s := range result.Sections {
		if s.Kind == "fatal" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("expected fatal section")
	}
}

func TestExtractSmartLogsFindsOOM(t *testing.T) {
	logs := "allocating memory\nout of memory: Kill process 1234\n"
	result := ExtractSmartLogs(logs)
	found := false
	for _, s := range result.Sections {
		if s.Kind == "oom" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("expected oom section")
	}
}

func TestExtractSmartLogsFallsBackToTail(t *testing.T) {
	logs := "normal log line 1\nnormal log line 2\nnormal log line 3\n"
	result := ExtractSmartLogs(logs)
	if len(result.Sections) != 1 || result.Sections[0].Kind != "tail" {
		t.Fatalf("expected tail fallback, got: %#v", result.Sections)
	}
}

func TestExtractSmartLogsEmptyInput(t *testing.T) {
	result := ExtractSmartLogs("")
	if result.Summary != "no log output" {
		t.Fatalf("summary = %q", result.Summary)
	}
}

func TestExtractSmartLogsFindsConnectionRefused(t *testing.T) {
	logs := "connecting to database\ndial tcp 127.0.0.1:5432: connection refused\nretrying...\n"
	result := ExtractSmartLogs(logs)
	found := false
	for _, s := range result.Sections {
		if s.Kind == "connection_refused" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("expected connection_refused section")
	}
}

func TestExtractSmartLogsSummary(t *testing.T) {
	logs := "panic: something broke\nFATAL: shutting down\n"
	result := ExtractSmartLogs(logs)
	if !strings.Contains(result.Summary, "found:") {
		t.Fatalf("expected summary with findings: %q", result.Summary)
	}
}
