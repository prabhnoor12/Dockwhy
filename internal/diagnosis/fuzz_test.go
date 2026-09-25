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
	"os"
	"path/filepath"
	"testing"
)

func FuzzLookupExitCode(f *testing.F) {
	f.Add(0)
	f.Add(1)
	f.Add(126)
	f.Add(127)
	f.Add(137)
	f.Add(139)
	f.Add(143)
	f.Add(255)
	f.Add(-1)
	f.Add(256)
	f.Add(130)

	f.Fuzz(func(t *testing.T, code int) {
		info := LookupExitCode(code)
		if info.Code != code {
			t.Errorf("LookupExitCode(%d).Code = %d", code, info.Code)
		}
		if info.Name == "" {
			t.Errorf("LookupExitCode(%d).Name is empty", code)
		}
		if info.Description == "" {
			t.Errorf("LookupExitCode(%d).Description is empty", code)
		}
		if info.Category == "" {
			t.Errorf("LookupExitCode(%d).Category is empty", code)
		}
	})
}

func FuzzExtractSmartLogs(f *testing.F) {
	f.Add("panic: runtime error\ngoroutine 1 [running]:\n")
	f.Add("FATAL: something broke\n")
	f.Add("out of memory: Kill process\n")
	f.Add("normal log line\n")
	f.Add("")
	f.Add("connection refused\n")
	f.Add("dial tcp 127.0.0.1:5432: connection refused\n")
	f.Add("java.lang.NullPointerException\n\tat com.example.Main\n")
	f.Add("segmentation fault (core dumped)\n")

	f.Fuzz(func(t *testing.T, logs string) {
		result := ExtractSmartLogs(logs)
		if result.Summary == "" {
			t.Error("ExtractSmartLogs returned empty summary")
		}
		for _, section := range result.Sections {
			if section.Kind == "" {
				t.Error("section has empty kind")
			}
			if len(section.Lines) == 0 {
				t.Error("section has no lines")
			}
		}
	})
}

func FuzzGlobMatch(f *testing.F) {
	f.Add("*", "anything")
	f.Add("shop-*", "shop-api-1")
	f.Add("*-api-*", "shop-api-1")
	f.Add("exact", "exact")
	f.Add("?", "a")
	f.Add("[abc]", "b")
	f.Add("", "")
	f.Add("[", "invalid")

	f.Fuzz(func(t *testing.T, pattern, s string) {
		_ = globMatch(pattern, s)
	})
}

func FuzzLoadRulesValidation(f *testing.F) {
	f.Add(`{"rules":[{"name":"x","reason":"y","match":{}}]}`)
	f.Add(`{"rules":[]}`)
	f.Add(`{}`)
	f.Add(`invalid json`)
	f.Add(`{"rules":[{"name":"","reason":"y"}]}`)
	f.Add(`{"rules":[{"name":"x"}]}`)
	f.Add(``)
	f.Add(`{"rules":[{"name":"x","reason":"y","match":{"container_glob":"*-api-*"}}]}`)

	f.Fuzz(func(t *testing.T, jsonStr string) {
		dir := t.TempDir()
		p := filepath.Join(dir, "rules.json")
		if err := os.WriteFile(p, []byte(jsonStr), 0644); err != nil {
			t.Fatal(err)
		}
		_, _ = LoadRules(p)
	})
}
