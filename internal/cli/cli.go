package cli

import (
	"flag"
	"fmt"
	"io"
	"os"
	"sync"
	"time"

	"github.com/dockwhy/dockwhy/internal/diagnosis"
	"github.com/dockwhy/dockwhy/internal/docker"
	"github.com/dockwhy/dockwhy/internal/output"
	"github.com/dockwhy/dockwhy/internal/version"
)

func Run() {
	client := docker.NewCLIClient()
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr, client))
}

func run(args []string, stdout, stderr io.Writer, client docker.Client) int {
	flags := flag.NewFlagSet("dockwhy", flag.ContinueOnError)
	flags.SetOutput(stderr)
	tail := flags.Int("tail", 50, "number of recent log lines to show")
	jsonOutput := flags.Bool("json", false, "print machine-readable JSON")
	noLogs := flags.Bool("no-logs", false, "skip container log lookup and output")
	showVersion := flags.Bool("version", false, "print the dockwhy version")
	timeout := flags.Duration("timeout", 10*time.Second, "maximum time for each Docker command")
	eventsSince := flags.Duration("events-since", 24*time.Hour, "look back this far for Docker lifecycle events")
	noEvents := flags.Bool("no-events", false, "skip Docker lifecycle event lookup")
	flags.Usage = func() {
		fmt.Fprintln(stderr, "dockwhy - explain why a Docker container stopped")
		fmt.Fprintln(stderr, "\nUsage:\n  dockwhy [flags] CONTAINER")
		flags.PrintDefaults()
	}
	if err := flags.Parse(args); err != nil {
		return 2
	}
	if *showVersion {
		fmt.Fprintln(stdout, version.Version)
		return 0
	}
	if flags.NArg() != 1 {
		flags.Usage()
		return 2
	}
	if *tail < 0 {
		fmt.Fprintln(stderr, "-tail must be zero or greater")
		return 2
	}
	if *tail > 5000 {
		fmt.Fprintln(stderr, "-tail must not exceed 5000")
		return 2
	}
	if *timeout <= 0 || *eventsSince <= 0 {
		fmt.Fprintln(stderr, "-timeout and -events-since must be greater than zero")
		return 2
	}
	if configurable, ok := client.(interface{ SetTimeout(time.Duration) }); ok {
		configurable.SetTimeout(*timeout)
	}

	containerName := flags.Arg(0)
	c, err := client.Inspect(containerName)
	if err != nil {
		fmt.Fprintf(stderr, "dockwhy: %s\n", err)
		return 1
	}
	var logOutput docker.LogOutput
	var logErr error
	var events []docker.Event
	var eventsErr error
	var stats *docker.Stats
	var statsErr error
	var wait sync.WaitGroup
	if !*noLogs {
		wait.Add(1)
		go func() {
			defer wait.Done()
			logOutput, logErr = client.Logs(containerName, *tail)
		}()
	}
	if !*noEvents {
		wait.Add(1)
		go func() {
			defer wait.Done()
			events, eventsErr = client.Events(c.ID, *eventsSince)
		}()
	}
	if c.State.Running {
		wait.Add(1)
		go func() {
			defer wait.Done()
			value, err := client.Stats(containerName)
			if err != nil {
				statsErr = err
				return
			}
			stats = &value
		}()
	}
	wait.Wait()
	result := diagnosis.AnalyzeDetailed(c, logOutput.Text, logOutput.Truncated, events, logErr, eventsErr)
	result.LogsSkipped = *noLogs
	result.Stats = stats
	if statsErr != nil {
		result.StatsError = statsErr.Error()
	}
	if *jsonOutput {
		err = output.JSON(stdout, result)
	} else {
		err = output.Text(stdout, result)
	}
	if err != nil {
		fmt.Fprintf(stderr, "dockwhy: write output: %s\n", err)
		return 1
	}
	return 0
}
