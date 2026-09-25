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

//go:build integration

package integration

import (
	"os/exec"
	"strings"
	"testing"
)

func requireDocker(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("docker"); err != nil {
		t.Skip("docker CLI not found on PATH")
	}
	if err := exec.Command("docker", "info").Run(); err != nil {
		t.Skip("docker daemon not reachable: ", err)
	}
}

func requireDockwhy(t *testing.T) string {
	t.Helper()
	path, err := exec.LookPath("dockwhy")
	if err != nil {
		t.Skip("dockwhy binary not found on PATH; build with: go build -o dockwhy ./cmd/dockwhy")
	}
	return path
}

func TestIntegrationVersion(t *testing.T) {
	bin := requireDockwhy(t)
	out, err := exec.Command(bin, "--version").CombinedOutput()
	if err != nil {
		t.Fatalf("dockwhy --version failed: %v\n%s", err, out)
	}
	if strings.TrimSpace(string(out)) == "" {
		t.Fatal("version output is empty")
	}
}

func TestIntegrationExitCodeLookup(t *testing.T) {
	bin := requireDockwhy(t)
	out, err := exec.Command(bin, "--exit-code", "137", "--json").CombinedOutput()
	if err != nil {
		t.Fatalf("dockwhy --exit-code failed: %v\n%s", err, out)
	}
	if !strings.Contains(string(out), "sigkill") && !strings.Contains(string(out), "SIGKILL") {
		t.Fatalf("expected SIGKILL in output: %s", out)
	}
}

func TestIntegrationDiagnoseStoppedContainer(t *testing.T) {
	requireDocker(t)
	bin := requireDockwhy(t)

	name := "dockwhy-test-stopped"
	exec.Command("docker", "rm", "-f", name).Run()

	create := exec.Command("docker", "run", "--name", name, "alpine:latest", "sh", "-c", "echo 'test error' && exit 1")
	create.CombinedOutput()
	defer exec.Command("docker", "rm", "-f", name).Run()

	out, err := exec.Command(bin, "--json", name).CombinedOutput()
	if err != nil {
		t.Fatalf("dockwhy failed: %v\n%s", err, out)
	}
	if !strings.Contains(string(out), `"reason"`) {
		t.Fatalf("expected JSON with reason: %s", out)
	}
}

func TestIntegrationDiagnoseRunningContainer(t *testing.T) {
	requireDocker(t)
	bin := requireDockwhy(t)

	name := "dockwhy-test-running"
	exec.Command("docker", "rm", "-f", name).Run()

	run := exec.Command("docker", "run", "-d", "--name", name, "alpine:latest", "sleep", "60")
	if out, err := run.CombinedOutput(); err != nil {
		t.Fatalf("failed to start container: %v\n%s", err, out)
	}
	defer exec.Command("docker", "rm", "-f", name).Run()

	out, err := exec.Command(bin, "--json", name).CombinedOutput()
	if err != nil {
		t.Fatalf("dockwhy failed: %v\n%s", err, out)
	}
	if !strings.Contains(string(out), `"running"`) {
		t.Fatalf("expected running state in output: %s", out)
	}
}
