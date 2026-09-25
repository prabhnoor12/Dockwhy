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

	"github.com/prabhnoor12/dockwhy/internal/docker"
)

func TestMatchesRuleContainerName(t *testing.T) {
	c := docker.Container{Name: "my-api"}
	if !matchesRule(RuleMatch{ContainerName: "my-api"}, c, "") {
		t.Error("should match exact name")
	}
	if matchesRule(RuleMatch{ContainerName: "other"}, c, "") {
		t.Error("should not match different name")
	}
}

func TestMatchesRuleContainerGlob(t *testing.T) {
	c := docker.Container{Name: "shop-api-1"}
	if !matchesRule(RuleMatch{ContainerGlob: "shop-*"}, c, "") {
		t.Error("should match prefix glob")
	}
	if !matchesRule(RuleMatch{ContainerGlob: "*-api-*"}, c, "") {
		t.Error("should match middle glob")
	}
	if !matchesRule(RuleMatch{ContainerGlob: "*"}, c, "") {
		t.Error("wildcard should match everything")
	}
	if matchesRule(RuleMatch{ContainerGlob: "web-*"}, c, "") {
		t.Error("should not match wrong prefix")
	}
	if !matchesRule(RuleMatch{ContainerGlob: "shop-api-?"}, c, "") {
		t.Error("should match single-char wildcard")
	}
	if !matchesRule(RuleMatch{ContainerGlob: "[a-z]*"}, c, "") {
		t.Error("should match character class")
	}
	if matchesRule(RuleMatch{ContainerGlob: "[0-9]*"}, c, "") {
		t.Error("should not match wrong character class")
	}
}

func TestMatchesRuleLabel(t *testing.T) {
	c := docker.Container{
		Name:   "test",
		Labels: map[string]string{"app": "api", "env": "prod"},
	}
	if !matchesRule(RuleMatch{Label: "app=api"}, c, "") {
		t.Error("should match label with value")
	}
	if !matchesRule(RuleMatch{Label: "app"}, c, "") {
		t.Error("should match label existence")
	}
	if matchesRule(RuleMatch{Label: "app=web"}, c, "") {
		t.Error("should not match wrong label value")
	}
	if matchesRule(RuleMatch{Label: "missing"}, c, "") {
		t.Error("should not match missing label")
	}
}

func TestMatchesRuleImage(t *testing.T) {
	c := docker.Container{Name: "test", Image: "api:v2"}
	if !matchesRule(RuleMatch{Image: "api:v2"}, c, "") {
		t.Error("should match image")
	}
	if matchesRule(RuleMatch{Image: "api:v1"}, c, "") {
		t.Error("should not match wrong image")
	}
}

func TestMatchesRuleLogPattern(t *testing.T) {
	c := docker.Container{Name: "test"}
	if !matchesRule(RuleMatch{LogPattern: "connection refused"}, c, "error: connection refused to database") {
		t.Error("should match log pattern")
	}
	if matchesRule(RuleMatch{LogPattern: "timeout"}, c, "error: connection refused") {
		t.Error("should not match absent pattern")
	}
}

func TestMatchesRuleExitCode(t *testing.T) {
	c := docker.Container{Name: "test", State: docker.State{ExitCode: 137}}
	code137 := 137
	code1 := 1
	if !matchesRule(RuleMatch{ExitCode: &code137}, c, "") {
		t.Error("should match exit code")
	}
	if matchesRule(RuleMatch{ExitCode: &code1}, c, "") {
		t.Error("should not match wrong exit code")
	}
}

func TestMatchesRuleOOMKilled(t *testing.T) {
	c := docker.Container{Name: "test", State: docker.State{OOMKilled: true}}
	trueVal := true
	falseVal := false
	if !matchesRule(RuleMatch{OOMKilled: &trueVal}, c, "") {
		t.Error("should match OOM killed")
	}
	if matchesRule(RuleMatch{OOMKilled: &falseVal}, c, "") {
		t.Error("should not match when not OOM killed")
	}
}

func TestApplyRules(t *testing.T) {
	rules := []Rule{
		{
			Name:     "db-connection",
			Reason:   "Database connection failure detected",
			Severity: "critical",
			Match:    RuleMatch{LogPattern: "connection refused"},
			Advice:   []string{"Check database is running", "Verify network connectivity"},
		},
		{
			Name:  "api-prefix",
			Reason: "API container detected",
			Match: RuleMatch{ContainerGlob: "api-*"},
		},
	}
	c := docker.Container{Name: "api-v2", State: docker.State{ExitCode: 1}}
	logs := "error: connection refused to database"
	findings := ApplyRules(rules, c, logs)
	if len(findings) != 2 {
		t.Fatalf("expected 2 findings, got %d", len(findings))
	}
	if findings[0].Reason != "Database connection failure detected" {
		t.Errorf("first finding = %q", findings[0].Reason)
	}
	if findings[0].Confidence != "custom" {
		t.Errorf("confidence should be 'custom': %q", findings[0].Confidence)
	}
}

func TestApplyRulesAdvice(t *testing.T) {
	rules := []Rule{
		{
			Name:   "oom-rule",
			Reason: "OOM detected",
			Match:  RuleMatch{OOMKilled: boolPtr(true)},
			Advice: []string{"Increase memory limit", "Check for leaks"},
		},
	}
	c := docker.Container{Name: "test", State: docker.State{OOMKilled: true}}
	advice := ApplyRulesAdvice(rules, c, "")
	if len(advice) != 2 {
		t.Fatalf("expected 2 advice items, got %d", len(advice))
	}
}

func TestLoadRules(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "rules.json")
	content := `{
		"rules": [
			{
				"name": "test-rule",
				"reason": "Test reason",
				"match": {"container_name": "my-app"},
				"severity": "error",
				"advice": ["fix it"]
			}
		]
	}`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	rules, err := LoadRules(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(rules) != 1 {
		t.Fatalf("expected 1 rule, got %d", len(rules))
	}
	if rules[0].Name != "test-rule" {
		t.Errorf("name = %q", rules[0].Name)
	}
}

func TestLoadRulesMissingName(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "rules.json")
	content := `{"rules": [{"reason": "no name"}]}`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	_, err := LoadRules(path)
	if err == nil {
		t.Fatal("expected error for missing name")
	}
}

func TestLoadRulesMissingReason(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "rules.json")
	content := `{"rules": [{"name": "no-reason"}]}`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	_, err := LoadRules(path)
	if err == nil {
		t.Fatal("expected error for missing reason")
	}
}

func boolPtr(b bool) *bool {
	return &b
}
