package cli

import (
	"flag"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/dockwhy/dockwhy/internal/diagnosis"
	"github.com/dockwhy/dockwhy/internal/docker"
	"github.com/dockwhy/dockwhy/internal/output"
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
	logOutput, logErr := client.Logs(containerName, *tail)
	var events []docker.Event
	var eventsErr error
	if !*noEvents {
		events, eventsErr = client.Events(c.ID, *eventsSince)
	}
	result := diagnosis.AnalyzeDetailed(c, logOutput.Text, logOutput.Truncated, events, logErr, eventsErr)
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
