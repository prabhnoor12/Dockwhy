package docker

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

var errFakeCommand = errors.New("fake command failed")

func TestCLIClientAdapterParsesDockerCommands(t *testing.T) {
	client := NewCLIClient(context.Background(), 10*time.Second)
	client.runOverride = func(_ int, args ...string) (commandOutput, error) {
		switch args[0] {
		case "inspect":
			return commandOutput{stdout: `[{"Id":"container-id","Name":"/api","Created":"2026-08-29T09:00:00Z","RestartCount":0,"Config":{"Image":"api:latest","Labels":{"com.docker.compose.project":"shop","com.docker.compose.service":"api"},"Entrypoint":["/bin/sh"],"Cmd":["./start.sh"]},"State":{"Status":"running","Running":true,"ExitCode":0},"HostConfig":{"LogConfig":{"Type":"json-file"}}}]`}, nil
		case "logs":
			return commandOutput{stdout: "2026-08-29T10:00:00Z application started\n"}, nil
		case "events":
			return commandOutput{stdout: `{"Action":"start","timeNano":1756461600000000000,"Actor":{"ID":"container-id","Attributes":{}}}
{"Action":"die","timeNano":1756461605000000000,"Actor":{"ID":"container-id","Attributes":{"exitCode":"1"}}}
`}, nil
		case "stats":
			return commandOutput{stdout: `{"CPUPerc":"5.00%","MemUsage":"128MiB / 1GiB","MemPerc":"12.50%","NetIO":"1MB / 2MB","BlockIO":"3MB / 4MB","PIDs":"4"}`}, nil
		default:
			return commandOutput{}, nil
		}
	}

	container, err := client.Inspect("api")
	if err != nil {
		t.Fatal(err)
	}
	if container.Name != "api" || container.LogDriver != "json-file" || container.ComposeService != "api" {
		t.Fatalf("unexpected container: %#v", container)
	}

	logs, err := client.Logs("api", 25)
	if err != nil || logs.Text != "2026-08-29T10:00:00Z application started" {
		t.Fatalf("unexpected logs: %#v, error=%v", logs, err)
	}

	events, err := client.Events(container.ID, time.Hour)
	if err != nil || len(events) != 2 || events[0].Action != "start" || events[1].Action != "die" {
		t.Fatalf("unexpected events: %#v, error=%v", events, err)
	}

	stats, err := client.Stats("api")
	if err != nil || stats.MemoryUsageBytes != 128*1024*1024 || stats.PidsCurrent != 4 {
		t.Fatalf("unexpected stats: %#v, error=%v", stats, err)
	}
}

func TestCLIClientAdapterReportsCommandErrors(t *testing.T) {
	client := NewCLIClient(context.Background(), 10*time.Second)
	client.runOverride = func(_ int, args ...string) (commandOutput, error) {
		return commandOutput{stderr: strings.Join(args, " ") + " failed"}, errFakeCommand
	}
	if _, err := client.Inspect("api"); err == nil || !strings.Contains(err.Error(), "docker inspect") {
		t.Fatalf("unexpected inspect error: %v", err)
	}
}
