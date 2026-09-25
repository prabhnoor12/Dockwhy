package cli

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/prabhnoor12/dockwhy/internal/diagnosis"
	"github.com/prabhnoor12/dockwhy/internal/docker"
	"github.com/prabhnoor12/dockwhy/internal/kube"
	"github.com/prabhnoor12/dockwhy/internal/output"
	"github.com/prabhnoor12/dockwhy/internal/version"
)

const defaultTimeout = 10 * time.Second

func lookupFlagValue(args []string, name string) string {
	for i, arg := range args {
		var value string
		switch {
		case arg == "--"+name || arg == "-"+name:
			if i+1 < len(args) {
				value = args[i+1]
			}
		case strings.HasPrefix(arg, "--"+name+"="):
			value = strings.TrimPrefix(arg, "--"+name+"=")
		case strings.HasPrefix(arg, "-"+name+"="):
			value = strings.TrimPrefix(arg, "-"+name+"=")
		default:
			continue
		}
		return value
	}
	return ""
}

func Run() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	timeout := defaultTimeout
	if v := lookupFlagValue(os.Args[1:], "timeout"); v != "" {
		if d, err := time.ParseDuration(v); err == nil && d > 0 {
			timeout = d
		}
	}
	client := docker.NewCLIClient(ctx, timeout)
	kubeClient := kube.NewKubeCLI(ctx, timeout)
	os.Exit(run(ctx, os.Args[1:], os.Stdout, os.Stderr, client, kubeClient))
}

func run(ctx context.Context, args []string, stdout, stderr io.Writer, client docker.Client, kubeClient kube.Client) int {
	flags := flag.NewFlagSet("dockwhy", flag.ContinueOnError)
	flags.SetOutput(stderr)
	tail := flags.Int("tail", 50, "number of recent log lines to show")
	jsonOutput := flags.Bool("json", false, "print machine-readable JSON")
	noLogs := flags.Bool("no-logs", false, "skip container log lookup and output")
	showVersion := flags.Bool("version", false, "print the dockwhy version")
	timeout := flags.Duration("timeout", 10*time.Second, "maximum time for each Docker command")
	eventsSince := flags.Duration("events-since", 24*time.Hour, "look back this far for Docker lifecycle events")
	noEvents := flags.Bool("no-events", false, "skip Docker lifecycle event lookup")
	project := flags.String("project", "", "diagnose all containers in a Docker Compose project")
	trend := flags.Bool("trend", false, "show crash patterns across restart history")
	smartLogs := flags.Bool("smart-logs", false, "extract error patterns from logs instead of showing raw tail")
	resources := flags.Bool("resources", false, "show resource sizing recommendations")
	report := flags.String("report", "", "write a Markdown incident report to this file path (use - for stdout)")
	outputFormat := flags.String("output", "", "output format for integrations: pagerduty, slack, prometheus, webhook")
	kubeMode := flags.Bool("kube", false, "treat the container argument as a Kubernetes pod name")
	kubeNamespace := flags.String("kube-namespace", "", "Kubernetes namespace (default: auto-detect or \"default\")")
	kubeContainer := flags.String("kube-container", "", "specific container name in a multi-container pod")
	compare := flags.String("compare", "", "compare configuration with another container to find drift")
	exitCodeLookup := flags.Int("exit-code", -1, "look up information about a specific exit code")
	rulesFile := flags.String("rules", "", "path to a JSON file with custom diagnosis rules")
	watch := flags.Duration("watch", 0, "continuously diagnose at this interval (e.g. 5s, 1m); 0 disables")
	flags.Usage = func() {
		fmt.Fprintln(stderr, "dockwhy - explain why a Docker container stopped")
		fmt.Fprintln(stderr, "\nUsage:")
		fmt.Fprintln(stderr, "  dockwhy [flags] CONTAINER")
		fmt.Fprintln(stderr, "  dockwhy --project PROJECT")
		fmt.Fprintln(stderr, "\nFlags:")
		flags.PrintDefaults()
	}
	if err := flags.Parse(args); err != nil {
		return 2
	}
	if *showVersion {
		fmt.Fprintln(stdout, version.Version)
		return 0
	}
	if *exitCodeLookup >= 0 {
		return runExitCodeLookup(*exitCodeLookup, *jsonOutput, stdout)
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
	if *watch < 0 {
		fmt.Fprintln(stderr, "-watch must be zero or greater")
		return 2
	}
	switch *outputFormat {
	case "", "pagerduty", "slack", "prometheus", "webhook":
	default:
		fmt.Fprintf(stderr, "unsupported --output format %q (valid: pagerduty, slack, prometheus, webhook)\n", *outputFormat)
		return 2
	}

	if *project != "" {
		return runProject(*project, *jsonOutput, *noLogs, *noEvents, *tail, *eventsSince, stdout, stderr, client)
	}

	if flags.NArg() != 1 {
		flags.Usage()
		return 2
	}

	if *trend {
		return runTrend(flags.Arg(0), *jsonOutput, *eventsSince, stdout, stderr, client)
	}

	if *compare != "" {
		return runCompare(flags.Arg(0), *compare, *jsonOutput, stdout, stderr, client)
	}

	containerArg := flags.Arg(0)
	var kubePodInfo *kube.PodInfo
	if *kubeMode {
		if kubeClient == nil {
			fmt.Fprintln(stderr, "dockwhy: --kube requires a Kubernetes client")
			return 1
		}
		ns := *kubeNamespace
		if ns == "" {
			ns = kubeClient.DetectNamespace()
		}
		podInfo, err := kubeClient.GetPod(ns, containerArg)
		if err != nil {
			fmt.Fprintf(stderr, "dockwhy: %s\n", err)
			return 1
		}
		kubePodInfo = &podInfo
		resolvedID, err := kubeClient.ResolveContainer(containerArg, *kubeContainer, ns)
		if err != nil {
			fmt.Fprintf(stderr, "dockwhy: %s\n", err)
			return 1
		}
		containerArg = resolvedID
		if !*jsonOutput && *outputFormat == "" {
			output.KubeContext(stdout, podInfo)
			fmt.Fprintln(stdout)
		}
	}

	if *watch > 0 {
		return runWatch(ctx, flags.Arg(0), *watch, *jsonOutput, *noLogs, *noEvents, *smartLogs, *resources, *report, *outputFormat, *rulesFile, *tail, *eventsSince, stdout, stderr, client)
	}

	return runSingle(containerArg, *jsonOutput, *noLogs, *noEvents, *smartLogs, *resources, *report, *outputFormat, *rulesFile, *tail, *eventsSince, stdout, stderr, client, kubePodInfo)
}

func runSingle(containerName string, jsonOutput, noLogs, noEvents, smartLogs, resources bool, reportPath, outputFormat, rulesPath string, tail int, eventsSince time.Duration, stdout, stderr io.Writer, client docker.Client, podInfo *kube.PodInfo) int {
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
	if !noLogs {
		wait.Add(1)
		go func() {
			defer wait.Done()
			logOutput, logErr = client.Logs(containerName, tail)
		}()
	}
	if !noEvents {
		wait.Add(1)
		go func() {
			defer wait.Done()
			events, eventsErr = client.Events(c.ID, eventsSince)
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
	result.LogsSkipped = noLogs
	result.Stats = stats
	if statsErr != nil {
		result.StatsError = statsErr.Error()
	}
	if rulesPath != "" {
		rules, rulesErr := diagnosis.LoadRules(rulesPath)
		if rulesErr != nil {
			fmt.Fprintf(stderr, "dockwhy: %s\n", rulesErr)
			return 1
		}
		customFindings := diagnosis.ApplyRules(rules, c, logOutput.Text)
		result.Findings = append(result.Findings, customFindings...)
		customAdvice := diagnosis.ApplyRulesAdvice(rules, c, logOutput.Text)
		result.Advice = append(result.Advice, customAdvice...)
	}
	if jsonOutput {
		if podInfo != nil {
			err = output.JSONWithKube(stdout, result, *podInfo)
		} else {
			err = output.JSON(stdout, result)
		}
	} else {
		switch outputFormat {
		case "pagerduty":
			err = output.PagerDuty(stdout, result)
		case "slack":
			err = output.Slack(stdout, result)
		case "prometheus":
			err = output.Prometheus(stdout, result)
		case "webhook":
			err = output.Webhook(stdout, result)
		default:
			err = output.Text(stdout, result)
		}
	}
	if err != nil {
		fmt.Fprintf(stderr, "dockwhy: write output: %s\n", err)
		return 1
	}
	if smartLogs && !noLogs {
		smartResult := diagnosis.ExtractSmartLogs(logOutput.Text)
		fmt.Fprintln(stdout)
		if jsonOutput {
			err = output.SmartLogJSON(stdout, smartResult)
		} else {
			err = output.SmartLogText(stdout, smartResult)
		}
		if err != nil {
			fmt.Fprintf(stderr, "dockwhy: write smart log output: %s\n", err)
			return 1
		}
	}
	if resources {
		resourceReport := diagnosis.AnalyzeResources(containerName, stats, c)
		fmt.Fprintln(stdout)
		if jsonOutput {
			err = output.ResourceJSON(stdout, resourceReport)
		} else {
			err = output.ResourceText(stdout, resourceReport)
		}
		if err != nil {
			fmt.Fprintf(stderr, "dockwhy: write resource output: %s\n", err)
			return 1
		}
	}
	if reportPath != "" {
		var reportWriter io.Writer
		if reportPath == "-" {
			reportWriter = stdout
		} else {
			f, fileErr := os.Create(reportPath)
			if fileErr != nil {
				fmt.Fprintf(stderr, "dockwhy: create report file: %s\n", fileErr)
				return 1
			}
			defer f.Close()
			reportWriter = f
		}
		if reportErr := output.ReportMarkdown(reportWriter, result); reportErr != nil {
			fmt.Fprintf(stderr, "dockwhy: write report: %s\n", reportErr)
			return 1
		}
		if reportPath != "-" {
			fmt.Fprintf(stderr, "dockwhy: report written to %s\n", reportPath)
		}
	}
	return 0
}

func runProject(project string, jsonOutput, noLogs, noEvents bool, tail int, eventsSince time.Duration, stdout, stderr io.Writer, client docker.Client) int {
	summaries, err := client.ListByProject(project)
	if err != nil {
		fmt.Fprintf(stderr, "dockwhy: %s\n", err)
		return 1
	}
	if len(summaries) == 0 {
		fmt.Fprintf(stderr, "dockwhy: no containers found for project %q\n", project)
		return 1
	}

	var diagnoses []diagnosis.ContainerDiagnosis
	allEvents := make(map[string][]docker.Event)
	var mu sync.Mutex
	var wait sync.WaitGroup

	for _, summary := range summaries {
		wait.Add(1)
		go func(name string) {
			defer wait.Done()
			c, err := client.Inspect(name)
			if err != nil {
				return
			}
			var logOutput docker.LogOutput
			var logErr error
			var events []docker.Event
			var eventsErr error
			var inner sync.WaitGroup
			if !noLogs {
				inner.Add(1)
				go func() {
					defer inner.Done()
					logOutput, logErr = client.Logs(name, tail)
				}()
			}
			if !noEvents {
				inner.Add(1)
				go func() {
					defer inner.Done()
					events, eventsErr = client.Events(c.ID, eventsSince)
				}()
			}
			inner.Wait()

			result := diagnosis.AnalyzeDetailed(c, logOutput.Text, logOutput.Truncated, events, logErr, eventsErr)
			result.LogsSkipped = noLogs

			mu.Lock()
			diagnoses = append(diagnoses, diagnosis.ContainerDiagnosis{Container: c, Result: result})
			if len(events) > 0 {
				allEvents[c.Name] = events
			}
			mu.Unlock()
		}(summary.Name)
	}
	wait.Wait()

	projectResult := diagnosis.AnalyzeProject(project, diagnoses, allEvents)

	if jsonOutput {
		err = output.ProjectJSON(stdout, projectResult)
	} else {
		err = output.ProjectText(stdout, projectResult)
	}
	if err != nil {
		fmt.Fprintf(stderr, "dockwhy: write output: %s\n", err)
		return 1
	}
	return 0
}

func runTrend(containerName string, jsonOutput bool, eventsSince time.Duration, stdout, stderr io.Writer, client docker.Client) int {
	c, err := client.Inspect(containerName)
	if err != nil {
		fmt.Fprintf(stderr, "dockwhy: %s\n", err)
		return 1
	}
	events, eventsErr := client.Events(c.ID, eventsSince)
	if eventsErr != nil {
		fmt.Fprintf(stderr, "dockwhy: %s\n", eventsErr)
		return 1
	}
	trendResult := diagnosis.AnalyzeTrend(containerName, events, c)
	if jsonOutput {
		err = output.TrendJSON(stdout, trendResult)
	} else {
		err = output.TrendText(stdout, trendResult)
	}
	if err != nil {
		fmt.Fprintf(stderr, "dockwhy: write output: %s\n", err)
		return 1
	}
	return 0
}

func runCompare(containerA, containerB string, jsonOutput bool, stdout, stderr io.Writer, client docker.Client) int {
	a, err := client.Inspect(containerA)
	if err != nil {
		fmt.Fprintf(stderr, "dockwhy: inspect %s: %s\n", containerA, err)
		return 1
	}
	b, err := client.Inspect(containerB)
	if err != nil {
		fmt.Fprintf(stderr, "dockwhy: inspect %s: %s\n", containerB, err)
		return 1
	}
	result := diagnosis.CompareContainers(a, b)
	if jsonOutput {
		err = output.CompareJSON(stdout, result)
	} else {
		err = output.CompareText(stdout, result)
	}
	if err != nil {
		fmt.Fprintf(stderr, "dockwhy: write output: %s\n", err)
		return 1
	}
	return 0
}

func runExitCodeLookup(code int, jsonOutput bool, stdout io.Writer) int {
	info := diagnosis.LookupExitCode(code)
	var err error
	if jsonOutput {
		err = output.ExitCodeJSON(stdout, info)
	} else {
		err = output.ExitCodeText(stdout, info)
	}
	if err != nil {
		return 1
	}
	return 0
}

func runWatch(ctx context.Context, containerName string, interval time.Duration, jsonOutput, noLogs, noEvents, smartLogs, resources bool, reportPath, outputFormat, rulesPath string, tail int, eventsSince time.Duration, stdout, stderr io.Writer, client docker.Client) int {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	runOnce := func() {
		if !jsonOutput {
			fmt.Fprintf(stdout, "\n=== %s ===\n", time.Now().UTC().Format(time.RFC3339))
		}
		runSingle(containerName, jsonOutput, noLogs, noEvents, smartLogs, resources, reportPath, outputFormat, rulesPath, tail, eventsSince, stdout, stderr, client, nil)
	}

	runOnce()
	for {
		select {
		case <-ticker.C:
			runOnce()
		case <-ctx.Done():
			return 0
		}
	}
}
