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

// ExitCodeInfo provides detailed information about a process exit code.
type ExitCodeInfo struct {
	Code        int      `json:"code"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Causes      []string `json:"causes"`
	Fixes       []string `json:"fixes"`
	Signal      string   `json:"signal,omitempty"`
	Category    string   `json:"category"`
}

// LookupExitCode returns detailed information about an exit code.
func LookupExitCode(code int) ExitCodeInfo {
	if info, ok := exitCodeDB[code]; ok {
		return info
	}
	if code > 128 && code <= 255 {
		signalNum := code - 128
		if name, ok := signalNames[signalNum]; ok {
			return ExitCodeInfo{
				Code:        code,
				Name:        name,
				Description: "Process was killed by " + name,
				Causes:      []string{"External signal sent to the process", "OOM killer, watchdog, or orchestrator intervention"},
				Fixes:       []string{"Check system logs (dmesg) for OOM kills", "Review orchestrator kill policies", "Check resource limits"},
				Signal:      name,
				Category:    "signal",
			}
		}
	}
	return ExitCodeInfo{
		Code:        code,
		Name:        "unknown",
		Description: "Application-specific exit code",
		Causes:      []string{"Check application documentation for this exit code", "Review application logs for error messages"},
		Fixes:       []string{"Consult the application's exit code documentation", "Check application logs around the time of exit"},
		Category:    "application",
	}
}

// AllExitCodes returns all exit codes in the knowledge base.
func AllExitCodes() []ExitCodeInfo {
	codes := make([]ExitCodeInfo, 0, len(exitCodeDB))
	for _, info := range exitCodeDB {
		codes = append(codes, info)
	}
	return codes
}

var signalNames = map[int]string{
	1:  "SIGHUP",
	2:  "SIGINT",
	3:  "SIGQUIT",
	6:  "SIGABRT",
	9:  "SIGKILL",
	11: "SIGSEGV",
	15: "SIGTERM",
	19: "SIGSTOP",
	24: "SIGXCPU",
	25: "SIGXFSZ",
}

var exitCodeDB = map[int]ExitCodeInfo{
	0: {
		Code:        0,
		Name:        "success",
		Description: "Process exited normally and successfully",
		Causes:      []string{"Application completed its work", "Graceful shutdown via SIGTERM handler"},
		Fixes:       []string{"No action needed if this was expected", "If unexpected, check why the container stopped instead of staying running"},
		Category:    "success",
	},
	1: {
		Code:        1,
		Name:        "general_error",
		Description: "General application error",
		Causes:      []string{"Unhandled exception or error in application code", "Missing configuration or environment variables", "Failed to connect to a dependency"},
		Fixes:       []string{"Check application logs for error details", "Verify environment variables and configuration", "Check connectivity to dependencies (database, cache, etc.)"},
		Category:    "application",
	},
	2: {
		Code:        2,
		Name:        "misuse_of_command",
		Description: "Misuse of shell command or missing argument",
		Causes:      []string{"Invalid command-line arguments", "Missing required arguments", "Syntax error in command"},
		Fixes:       []string{"Check the command and arguments in the Dockerfile or docker-compose.yml", "Verify the entrypoint and cmd configuration"},
		Category:    "configuration",
	},
	126: {
		Code:        126,
		Name:        "command_not_executable",
		Description: "Command found but not executable",
		Causes:      []string{"File permissions do not allow execution", "Script missing shebang line (#!/bin/sh)", "Binary compiled for wrong architecture"},
		Fixes:       []string{"Run chmod +x on the entrypoint script", "Add a shebang line to scripts", "Verify the binary matches the container architecture (amd64 vs arm64)"},
		Category:    "configuration",
	},
	127: {
		Code:        127,
		Name:        "command_not_found",
		Description: "Command or executable not found",
		Causes:      []string{"Executable not installed in the image", "PATH does not include the executable location", "Typo in the command name"},
		Fixes:       []string{"Install the required package in the Dockerfile", "Use the full path to the executable", "Check the entrypoint/cmd for typos", "Verify the base image contains the required tools"},
		Category:    "configuration",
	},
	128: {
		Code:        128,
		Name:        "invalid_exit_code",
		Description: "Invalid exit code argument",
		Causes:      []string{"exit called with a non-integer argument"},
		Fixes:       []string{"Check application code for invalid exit() calls"},
		Category:    "application",
	},
	134: {
		Code:        134,
		Name:        "sigabrt",
		Description: "Process aborted (SIGABRT, signal 6)",
		Causes:      []string{"Application detected a fatal error (assertion failure)", "Memory corruption detected", "C++ std::terminate called", "glibc detected heap corruption"},
		Fixes:       []string{"Check application logs for assertion failures", "Run with increased logging/debug mode", "Check for memory corruption with valgrind", "Review recent code changes for null pointer or buffer issues"},
		Signal:      "SIGABRT",
		Category:    "signal",
	},
	137: {
		Code:        137,
		Name:        "sigkill",
		Description: "Process killed forcefully (SIGKILL, signal 9)",
		Causes:      []string{"OOM killer terminated the process (most common)", "docker stop --kill-after timeout expired", "Kubernetes killed the pod (eviction or OOM)", "External process sent SIGKILL"},
		Fixes:       []string{"Increase container memory limit", "Check for memory leaks in the application", "Review dmesg for OOM killer messages", "Check Kubernetes pod events for eviction reasons", "Profile memory usage under load"},
		Signal:      "SIGKILL",
		Category:    "signal",
	},
	139: {
		Code:        139,
		Name:        "sigsegv",
		Description: "Segmentation fault (SIGSEGV, signal 11)",
		Causes:      []string{"Null pointer dereference", "Buffer overflow", "Accessing freed memory", "Stack overflow", "Native library bug (CGo, JNI)"},
		Fixes:       []string{"Check application logs for crash traces", "Update native dependencies/libraries", "Run with address sanitizer in development", "Review recent changes to native code or CGo bindings"},
		Signal:      "SIGSEGV",
		Category:    "signal",
	},
	143: {
		Code:        143,
		Name:        "sigterm",
		Description: "Process terminated gracefully (SIGTERM, signal 15)",
		Causes:      []string{"docker stop sent SIGTERM", "Kubernetes sent SIGTERM for pod termination", "Orchestrator scaling down", "User manually stopped the container"},
		Fixes:       []string{"This is normal during deployments and scaling", "If unexpected, check who/what stopped the container", "Ensure the app handles SIGTERM for graceful shutdown"},
		Signal:      "SIGTERM",
		Category:    "signal",
	},
	141: {
		Code:        141,
		Name:        "sigpipe",
		Description: "Broken pipe (SIGPIPE, signal 13)",
		Causes:      []string{"Writing to a pipe whose read end is closed", "Network connection closed by remote end"},
		Fixes:       []string{"Handle EPIPE errors in the application", "Check if downstream services are restarting"},
		Signal:      "SIGPIPE",
		Category:    "signal",
	},
	255: {
		Code:        255,
		Name:        "exit_status_out_of_range",
		Description: "Exit status out of range or SSH connection error",
		Causes:      []string{"Application exited with a value > 255 (wrapped around)", "SSH connection failure (common in SSH-based tools)"},
		Fixes:       []string{"Check application exit code handling", "If SSH-related, verify credentials and connectivity"},
		Category:    "application",
	},
}
