package diagnosis

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/prabhnoor12/dockwhy/internal/docker"
)

// Rule is a custom diagnosis rule loaded from a rules file.
type Rule struct {
	Name        string   `json:"name"`
	Description string   `json:"description,omitempty"`
	Match       RuleMatch `json:"match"`
	Severity    string   `json:"severity,omitempty"`
	Reason      string   `json:"reason"`
	Advice      []string `json:"advice,omitempty"`
}

// RuleMatch defines when a rule applies.
type RuleMatch struct {
	ContainerName string   `json:"container_name,omitempty"`
	ContainerGlob string   `json:"container_glob,omitempty"`
	Label         string   `json:"label,omitempty"`
	Image         string   `json:"image,omitempty"`
	LogPattern    string   `json:"log_pattern,omitempty"`
	ExitCode      *int     `json:"exit_code,omitempty"`
	OOMKilled     *bool    `json:"oom_killed,omitempty"`
}

// RulesFile is the top-level structure of a rules file.
type RulesFile struct {
	Rules []Rule `json:"rules"`
}

// LoadRules reads and parses a rules file from disk.
func LoadRules(path string) ([]Rule, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read rules file: %w", err)
	}
	var rf RulesFile
	if err := json.Unmarshal(data, &rf); err != nil {
		return nil, fmt.Errorf("parse rules file: %w", err)
	}
	for i, rule := range rf.Rules {
		if rule.Name == "" {
			return nil, fmt.Errorf("rule %d is missing a name", i)
		}
		if rule.Reason == "" {
			return nil, fmt.Errorf("rule %q is missing a reason", rule.Name)
		}
	}
	return rf.Rules, nil
}

// ApplyRules evaluates custom rules against a container and its logs,
// returning any additional findings.
func ApplyRules(rules []Rule, c docker.Container, logs string) []Finding {
	var findings []Finding
	for _, rule := range rules {
		if !matchesRule(rule.Match, c, logs) {
			continue
		}
		severity := rule.Severity
		if severity == "" {
			severity = "warning"
		}
		findings = append(findings, Finding{
			Reason:     rule.Reason,
			Summary:    rule.Description,
			Severity:   severity,
			Confidence: "custom",
			Evidence:   ruleEvidence(rule, c),
			Advice:     rule.Advice,
		})
	}
	return findings
}

// ApplyRulesAdvice returns custom advice from matching rules.
func ApplyRulesAdvice(rules []Rule, c docker.Container, logs string) []string {
	var advice []string
	for _, rule := range rules {
		if !matchesRule(rule.Match, c, logs) {
			continue
		}
		advice = append(advice, rule.Advice...)
	}
	return advice
}

func matchesRule(match RuleMatch, c docker.Container, logs string) bool {
	if match.ContainerName != "" && c.Name != match.ContainerName {
		return false
	}
	if match.ContainerGlob != "" {
		if !globMatch(match.ContainerGlob, c.Name) {
			return false
		}
	}
	if match.Label != "" {
		parts := strings.SplitN(match.Label, "=", 2)
		key := parts[0]
		val, ok := c.Labels[key]
		if !ok {
			return false
		}
		if len(parts) == 2 && val != parts[1] {
			return false
		}
	}
	if match.Image != "" && c.Image != match.Image {
		return false
	}
	if match.LogPattern != "" && !strings.Contains(logs, match.LogPattern) {
		return false
	}
	if match.ExitCode != nil && c.State.ExitCode != *match.ExitCode {
		return false
	}
	if match.OOMKilled != nil && c.State.OOMKilled != *match.OOMKilled {
		return false
	}
	return true
}

func globMatch(pattern, s string) bool {
	if pattern == "*" {
		return true
	}
	if !strings.Contains(pattern, "*") {
		return pattern == s
	}
	parts := strings.Split(pattern, "*")
	if len(parts) == 2 {
		return strings.HasPrefix(s, parts[0]) && strings.HasSuffix(s, parts[1]) && len(s) >= len(parts[0])+len(parts[1])
	}
	if strings.HasPrefix(pattern, "*") && strings.HasSuffix(pattern, "*") {
		inner := pattern[1 : len(pattern)-1]
		return strings.Contains(s, inner)
	}
	if strings.HasPrefix(pattern, "*") {
		return strings.HasSuffix(s, pattern[1:])
	}
	if strings.HasSuffix(pattern, "*") {
		return strings.HasPrefix(s, pattern[:len(pattern)-1])
	}
	return false
}

func ruleEvidence(rule Rule, c docker.Container) []Evidence {
	var evidence []Evidence
	evidence = append(evidence, Evidence{Name: "rule", Value: rule.Name})
	if rule.Match.ContainerName != "" {
		evidence = append(evidence, Evidence{Name: "matched_container", Value: rule.Match.ContainerName})
	}
	if rule.Match.ContainerGlob != "" {
		evidence = append(evidence, Evidence{Name: "matched_glob", Value: rule.Match.ContainerGlob})
	}
	if rule.Match.Label != "" {
		evidence = append(evidence, Evidence{Name: "matched_label", Value: rule.Match.Label})
	}
	if rule.Match.Image != "" {
		evidence = append(evidence, Evidence{Name: "matched_image", Value: rule.Match.Image})
	}
	if rule.Match.LogPattern != "" {
		evidence = append(evidence, Evidence{Name: "matched_log_pattern", Value: rule.Match.LogPattern})
	}
	return evidence
}
