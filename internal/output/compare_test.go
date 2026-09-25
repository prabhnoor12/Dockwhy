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
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/prabhnoor12/dockwhy/internal/diagnosis"
)

func TestCompareText(t *testing.T) {
	result := diagnosis.CompareResult{
		ContainerA:    "api-v1",
		ContainerB:    "api-v2",
		CriticalCount: 1,
		WarningCount:  1,
		Summary:       "Found 2 differences",
		Diffs: []diagnosis.ConfigDiff{
			{Field: "image", Before: "api:v1", After: "api:v2", Category: diagnosis.DiffCritical, Note: "different image"},
			{Field: "user", Before: "root", After: "app", Category: diagnosis.DiffWarning},
		},
	}
	var buf bytes.Buffer
	if err := CompareText(&buf, result); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if !strings.Contains(out, "api-v1 vs api-v2") {
		t.Error("missing container names")
	}
	if !strings.Contains(out, "[CRITICAL]") {
		t.Error("missing critical tag")
	}
	if !strings.Contains(out, "[WARNING]") {
		t.Error("missing warning tag")
	}
	if !strings.Contains(out, "api:v1") {
		t.Error("missing before value")
	}
	if !strings.Contains(out, "api:v2") {
		t.Error("missing after value")
	}
	if !strings.Contains(out, "different image") {
		t.Error("missing note")
	}
}

func TestCompareTextNoDiffs(t *testing.T) {
	result := diagnosis.CompareResult{
		ContainerA: "a",
		ContainerB: "b",
		Summary:    "No differences",
	}
	var buf bytes.Buffer
	if err := CompareText(&buf, result); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "No differences found") {
		t.Error("should indicate no differences")
	}
}

func TestCompareJSON(t *testing.T) {
	result := diagnosis.CompareResult{
		ContainerA:    "a",
		ContainerB:    "b",
		CriticalCount: 1,
		Diffs: []diagnosis.ConfigDiff{
			{Field: "image", Before: "x", After: "y", Category: diagnosis.DiffCritical},
		},
	}
	var buf bytes.Buffer
	if err := CompareJSON(&buf, result); err != nil {
		t.Fatal(err)
	}
	var decoded diagnosis.CompareResult
	if err := json.Unmarshal(buf.Bytes(), &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded.ContainerA != "a" || decoded.ContainerB != "b" {
		t.Error("JSON round-trip mismatch")
	}
	if len(decoded.Diffs) != 1 {
		t.Error("expected 1 diff in JSON")
	}
}
