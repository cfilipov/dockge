// Command mock-docker masquerades as the `docker` CLI. When placed first on
// PATH, exec.Command("docker", ...) in handlers resolves to this binary.
//
// It produces Docker Compose v2-style progress output (ANSI spinners,
// checkmarks, elapsed time) and communicates state changes to the fake
// Docker daemon via POST /_mock/state/{stack} over the Unix socket
// specified by DOCKER_HOST.
package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	osexec "os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

// ANSI escape sequences for Docker Compose v2 style output.
const (
	ansiGreen   = "\033[32m"
	ansiReset   = "\033[0m"
	ansiHideCur = "\033[?25l"
	ansiShowCur = "\033[?25h"
	ansiCurUp   = "\033[A"
	ansiEraseLn = "\033[2K"
)

// spinnerFrames matches the Braille spinner used by Docker Compose v2.
var spinnerFrames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

func main() {
	args := os.Args[1:]
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "mock-docker: no command specified")
		os.Exit(1)
	}

	// Route: "compose ...", "exec ...", "image ...", "start/stop/restart ..."
	switch args[0] {
	case "compose":
		handleCompose(args[1:])
	case "exec":
		handleExec(args[1:])
	case "image":
		handleImage(args[1:])
	case "start", "stop", "restart":
		handleContainerAction(args[0], args[1:])
	default:
		fmt.Fprintf(os.Stderr, "[mock-docker] unsupported command: %s\n", args[0])
		os.Exit(0)
	}
}

func handleCompose(args []string) {
	// Skip --env-file and -p/--project-name flags to get to the subcommand
	var envArgs []string
	projectName := "" // set by -p flag
	subcmdIdx := 0
	for subcmdIdx < len(args) {
		if args[subcmdIdx] == "--env-file" && subcmdIdx+1 < len(args) {
			envArgs = append(envArgs, args[subcmdIdx], args[subcmdIdx+1])
			subcmdIdx += 2
			continue
		}
		if (args[subcmdIdx] == "-p" || args[subcmdIdx] == "--project-name") && subcmdIdx+1 < len(args) {
			projectName = args[subcmdIdx+1]
			subcmdIdx += 2
			continue
		}
		break
	}

	if subcmdIdx >= len(args) {
		fmt.Fprintln(os.Stderr, "mock-docker: no compose subcommand")
		os.Exit(1)
	}

	subcmd := args[subcmdIdx]
	restArgs := args[subcmdIdx+1:]

	// Determine stack name: prefer -p flag, fall back to working directory
	stackName := projectName
	if stackName == "" {
		stackName = filepath.Base(mustGetwd())
	}

	switch subcmd {
	case "up":
		allServices := getServicesFromCompose()
		svc := findServiceArg(restArgs)
		isWholeStack := svc == ""
		forceRecreate := hasFlag(restArgs, "--force-recreate")
		services := allServices
		if svc != "" {
			services = []string{svc}
		}
		composeUp(stackName, services, isWholeStack, forceRecreate)
	case "stop":
		services := getServicesFromCompose()
		svc := findServiceArg(restArgs)
		if svc != "" {
			services = []string{svc}
		}
		composeStop(stackName, services, svc == "")
	case "down":
		services := getServicesFromCompose()
		removeVolumes := hasFlag(restArgs, "-v") || hasFlag(restArgs, "--volumes")
		composeDown(stackName, services, removeVolumes)
	case "restart":
		svc := findServiceArg(restArgs)
		isWholeStack := svc == ""
		services := getServicesFromCompose()
		if svc != "" {
			services = []string{svc}
		}
		composeRestart(stackName, services, isWholeStack)
	case "pull":
		services := getServicesFromCompose()
		svc := findServiceArg(restArgs)
		if svc != "" {
			services = []string{svc}
		}
		composePull(services)
	case "pause":
		services := getServicesFromCompose()
		composePause(stackName, services)
	case "unpause":
		services := getServicesFromCompose()
		composeUnpause(stackName, services)
	case "config":
		composeConfig()
	case "exec":
		composeExec(restArgs)
	case "logs":
		composeLogs(restArgs)
	default:
		fmt.Fprintf(os.Stderr, "[mock-docker] unsupported compose command: %s\n", subcmd)
	}
}

func handleExec(args []string) {
	// docker exec [-it] <container> <shell> [args...]
	// Skip -i, -t, -it flags to find the container name and shell.
	var rest []string
	for i := 0; i < len(args); i++ {
		a := args[i]
		if a == "-i" || a == "-t" || a == "-it" || a == "-ti" {
			continue
		}
		// Combined flags like -itu, etc
		if strings.HasPrefix(a, "-") && !strings.HasPrefix(a, "--") {
			continue
		}
		rest = append(rest, a)
	}
	if len(rest) < 2 {
		fmt.Fprintln(os.Stderr, "[mock-docker] exec: container and command required")
		os.Exit(1)
	}
	shell := rest[1]
	execShell(shell)
}

func handleImage(args []string) {
	if len(args) >= 1 && args[0] == "prune" {
		fmt.Println("Total reclaimed space: 0B")
	}
}

// handleContainerAction handles plain docker start/stop/restart <container>.
// Output mirrors the real docker CLI: prints the container name on success.
func handleContainerAction(action string, args []string) {
	if len(args) == 0 {
		fmt.Fprintf(os.Stderr, "[mock-docker] %s: container name required\n", action)
		os.Exit(1)
	}
	containerName := args[0]
	fmt.Println(containerName)
}

// --- Exec Helper ---

// execShell replaces the current process with a shell using syscall.Exec.
// This is how real `docker exec` works — the docker process replaces itself
// with the target command, so only one process owns the PTY. Using
// os/exec.Command.Run() would fork a child while mock-docker stays alive,
// causing both processes to share the PTY and produce a double prompt.
func execShell(shell string) {
	shellPath, err := osexec.LookPath(shell)
	if err != nil {
		shellPath, err = osexec.LookPath("sh")
		if err != nil {
			fmt.Fprintln(os.Stderr, "[mock-docker] exec: no shell available")
			os.Exit(1)
		}
	}

	// Replace this process with the shell — no child process, clean PTY ownership.
	err = syscall.Exec(shellPath, []string{shell}, os.Environ())
	// Only reached if exec fails
	fmt.Fprintf(os.Stderr, "[mock-docker] exec: %v\n", err)
	os.Exit(1)
}

// --- Compose Commands ---

func composeUp(stackName string, services []string, isWholeStack, forceRecreate bool) {
	var tasks []progressTask
	if isWholeStack {
		tasks = append(tasks, progressTask{
			name: fmt.Sprintf("Network %s_default", stackName), action: "Creating", done: "Created",
		})
	}
	for _, svc := range services {
		if forceRecreate {
			tasks = append(tasks,
				progressTask{name: fmt.Sprintf("Container %s-%s-1", stackName, svc), action: "Recreating", done: "Recreated"},
				progressTask{name: fmt.Sprintf("Container %s-%s-1", stackName, svc), action: "Starting", done: "Started"},
			)
		} else {
			tasks = append(tasks,
				progressTask{name: fmt.Sprintf("Container %s-%s-1", stackName, svc), action: "Creating", done: "Created"},
				progressTask{name: fmt.Sprintf("Container %s-%s-1", stackName, svc), action: "Starting", done: "Started"},
			)
		}
	}

	renderProgress(os.Stdout, "Running", tasks)
	// When force-recreating, transition through exited→running so the daemon
	// publishes die+start events for log banner injection.
	if forceRecreate {
		if isWholeStack {
			setMockState(stackName, "exited")
		} else {
			for _, svc := range services {
				setMockServiceState(stackName, svc, "exited")
			}
		}
	}
	if isWholeStack {
		setMockState(stackName, "running")
	} else {
		for _, svc := range services {
			setMockServiceState(stackName, svc, "running")
		}
	}
}

func composeStop(stackName string, services []string, isWholeStack bool) {
	var tasks []progressTask
	for _, svc := range services {
		tasks = append(tasks, progressTask{
			name: fmt.Sprintf("Container %s-%s-1", stackName, svc), action: "Stopping", done: "Stopped",
		})
	}

	renderProgress(os.Stdout, "Stopping", tasks)
	if isWholeStack {
		setMockState(stackName, "exited")
	} else {
		for _, svc := range services {
			setMockServiceState(stackName, svc, "exited")
		}
	}
}

func composeDown(stackName string, services []string, removeVolumes bool) {
	var tasks []progressTask
	for _, svc := range services {
		tasks = append(tasks,
			progressTask{name: fmt.Sprintf("Container %s-%s-1", stackName, svc), action: "Stopping", done: "Stopped"},
			progressTask{name: fmt.Sprintf("Container %s-%s-1", stackName, svc), action: "Removing", done: "Removed"},
		)
	}
	if removeVolumes {
		// Add volume removal tasks for common volume names
		for _, svc := range services {
			tasks = append(tasks, progressTask{
				name: fmt.Sprintf("Volume %s_%s-data", stackName, svc), action: "Removing", done: "Removed",
			})
		}
	}
	tasks = append(tasks, progressTask{
		name: fmt.Sprintf("Network %s_default", stackName), action: "Removing", done: "Removed",
	})

	renderProgress(os.Stdout, "Running", tasks)
	deleteMockState(stackName)
}

func composeRestart(stackName string, services []string, isWholeStack bool) {
	var tasks []progressTask
	for _, svc := range services {
		tasks = append(tasks, progressTask{
			name: fmt.Sprintf("Container %s-%s-1", stackName, svc), action: "Restarting", done: "Started",
		})
	}

	renderProgress(os.Stdout, "Restarting", tasks)
	// Transition through exited→running so the daemon publishes die+start events.
	// Without this, setting "running" on an already-running container is a no-op.
	if isWholeStack {
		setMockState(stackName, "exited")
		setMockState(stackName, "running")
	} else {
		for _, svc := range services {
			setMockServiceState(stackName, svc, "exited")
			setMockServiceState(stackName, svc, "running")
		}
	}
}

func composePull(services []string) {
	var tasks []progressTask
	for _, svc := range services {
		tasks = append(tasks, progressTask{
			name: svc, action: "Pulling", done: "Pulled",
		})
	}

	renderProgress(os.Stdout, "Pulling", tasks)
}

func composeExec(args []string) {
	// docker compose exec [flags] <service> <command> [args...]
	// Skip flags (-T, -it, --user, etc.) to find service and command.
	var rest []string
	for i := 0; i < len(args); i++ {
		a := args[i]
		if strings.HasPrefix(a, "-") {
			// Skip flags that take a value
			if a == "--user" || a == "-u" || a == "--env" || a == "-e" || a == "--workdir" || a == "-w" {
				i++ // skip next arg (the value)
			}
			continue
		}
		rest = append(rest, a)
	}
	if len(rest) < 2 {
		fmt.Fprintln(os.Stderr, "[mock-docker] compose exec: service and command required")
		os.Exit(1)
	}
	shell := rest[1]
	execShell(shell)
}

func composePause(stackName string, services []string) {
	var tasks []progressTask
	for _, svc := range services {
		tasks = append(tasks, progressTask{
			name: fmt.Sprintf("Container %s-%s-1", stackName, svc), action: "Pausing", done: "Paused",
		})
	}
	renderProgress(os.Stdout, "Pausing", tasks)
	setMockState(stackName, "paused")
}

func composeUnpause(stackName string, services []string) {
	var tasks []progressTask
	for _, svc := range services {
		tasks = append(tasks, progressTask{
			name: fmt.Sprintf("Container %s-%s-1", stackName, svc), action: "Unpausing", done: "Unpaused",
		})
	}
	renderProgress(os.Stdout, "Unpausing", tasks)
	setMockState(stackName, "running")
}

func composeConfig() {
	// Validate compose file exists and has services section
	composeFile := findComposeFile()
	if composeFile == "" {
		fmt.Fprintln(os.Stderr, "no configuration file provided: not found")
		os.Exit(1)
	}

	f, err := os.Open(composeFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "no configuration file provided: not found\n")
		os.Exit(1)
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	hasServices := false
	for scanner.Scan() {
		if strings.TrimSpace(scanner.Text()) == "services:" {
			hasServices = true
			break
		}
	}
	if !hasServices {
		fmt.Fprintln(os.Stderr, "services must be a mapping")
		os.Exit(1)
	}
}

func composeLogs(args []string) {
	services := getServicesFromCompose()
	if len(services) == 0 {
		return
	}

	stackName := filepath.Base(mustGetwd())

	// logColors mirrors docker compose's service name color palette.
	logColors := []string{
		"\033[36m", "\033[33m", "\033[32m", "\033[35m", "\033[34m",
		"\033[96m", "\033[93m", "\033[92m", "\033[95m", "\033[94m",
	}

	maxLen := 0
	for _, svc := range services {
		if len(svc) > maxLen {
			maxLen = len(svc)
		}
	}

	var buf bytes.Buffer
	for i, svc := range services {
		color := logColors[i%len(logColors)]
		padded := fmt.Sprintf("%-*s", maxLen, svc)
		prefix := color + padded + " | " + "\033[0m"

		lines := fetchServiceStartupLogs(stackName, svc)
		for _, line := range lines {
			fmt.Fprintf(&buf, "%s%s\n", prefix, line)
		}
	}
	os.Stdout.Write(buf.Bytes())

	// If -f/--follow, block until killed
	if hasFlag(args, "-f") || hasFlag(args, "--follow") {
		select {} // block forever (parent will kill us)
	}
}

// mockLogsResponse matches the JSON from /_mock/logs/{stack}/{service}.
type mockLogsResponse struct {
	Startup   []string `json:"startup"`
	Heartbeat []string `json:"heartbeat"`
	Interval  string   `json:"interval"`
	Shutdown  []string `json:"shutdown"`
}

// fetchServiceStartupLogs fetches the resolved startup log lines from the FakeDaemon.
// Falls back to generic output if the daemon is unavailable.
func fetchServiceStartupLogs(stackName, service string) []string {
	dockerHost := os.Getenv("DOCKER_HOST")
	if dockerHost == "" || !strings.HasPrefix(dockerHost, "unix://") {
		return []string{"[INFO] " + service + " started"}
	}
	socketPath := strings.TrimPrefix(dockerHost, "unix://")

	client := &http.Client{
		Transport: &http.Transport{
			DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
				return net.DialTimeout("unix", socketPath, 2*time.Second)
			},
		},
		Timeout: 5 * time.Second,
	}

	url := fmt.Sprintf("http://docker/_mock/logs/%s/%s", stackName, service)
	resp, err := client.Get(url)
	if err != nil {
		return []string{"[INFO] " + service + " started"}
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return []string{"[INFO] " + service + " started"}
	}

	var logs mockLogsResponse
	if err := json.NewDecoder(resp.Body).Decode(&logs); err != nil {
		return []string{"[INFO] " + service + " started"}
	}

	if len(logs.Startup) > 0 {
		return logs.Startup
	}
	return []string{"[INFO] " + service + " started"}
}

// --- Mock State Communication ---

func setMockState(stackName, status string) {
	body, _ := json.Marshal(map[string]string{"status": status})
	mockHTTP("POST", "/_mock/state/"+stackName, bytes.NewReader(body))
}

func setMockServiceState(stackName, service, status string) {
	body, _ := json.Marshal(map[string]string{"status": status})
	mockHTTP("POST", "/_mock/state/"+stackName+"/"+service, bytes.NewReader(body))
}

func deleteMockState(stackName string) {
	mockHTTP("DELETE", "/_mock/state/"+stackName, nil)
}

func mockHTTP(method, path string, body io.Reader) {
	dockerHost := os.Getenv("DOCKER_HOST")
	if dockerHost == "" || !strings.HasPrefix(dockerHost, "unix://") {
		return // not in mock mode, ignore
	}
	socketPath := strings.TrimPrefix(dockerHost, "unix://")

	client := &http.Client{
		Transport: &http.Transport{
			DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
				return net.DialTimeout("unix", socketPath, 2*time.Second)
			},
		},
		Timeout: 5 * time.Second,
	}

	req, err := http.NewRequest(method, "http://docker"+path, body)
	if err != nil {
		return
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := client.Do(req)
	if err != nil {
		return
	}
	resp.Body.Close()
}

// --- Progress Renderer ---

type progressTask struct {
	name   string
	action string
	done   string
}

func renderProgress(w io.Writer, verb string, tasks []progressTask) {
	n := len(tasks)
	if n == 0 {
		return
	}

	const framesPerTask = 3
	const delay = 50 * time.Millisecond

	taskStart := make([]time.Time, n)
	taskElapsed := make([]time.Duration, n)

	fmt.Fprint(w, ansiHideCur)

	// Draw initial frame
	writeHeader(w, verb, 0, n)
	spinIdx := 0
	for i := range tasks {
		writeTaskPending(w, tasks[i], spinIdx)
	}

	// Animate: complete tasks one at a time
	for completed := 0; completed < n; completed++ {
		taskStart[completed] = time.Now()

		for frame := 0; frame < framesPerTask; frame++ {
			time.Sleep(delay)
			spinIdx++
			moveCursorUp(w, n+1)
			writeHeader(w, verb, completed, n)
			for i := range tasks {
				if i < completed {
					writeTaskDone(w, tasks[i], taskElapsed[i])
				} else {
					writeTaskPending(w, tasks[i], spinIdx)
				}
			}
		}

		taskElapsed[completed] = time.Since(taskStart[completed])
		time.Sleep(delay)
		moveCursorUp(w, n+1)
		writeHeader(w, verb, completed+1, n)
		for i := range tasks {
			if i <= completed {
				writeTaskDone(w, tasks[i], taskElapsed[i])
			} else {
				writeTaskPending(w, tasks[i], spinIdx)
			}
		}
	}

	fmt.Fprint(w, ansiShowCur)
}

func writeHeader(w io.Writer, verb string, completed, total int) {
	fmt.Fprintf(w, "\r%s %s[+]%s %s %d/%d\r\n",
		ansiEraseLn, ansiGreen, ansiReset, verb, completed, total)
}

func writeTaskPending(w io.Writer, t progressTask, spinIdx int) {
	frame := spinnerFrames[spinIdx%len(spinnerFrames)]
	fmt.Fprintf(w, "\r%s %s %s  %s\r\n", ansiEraseLn, frame, t.name, t.action)
}

func writeTaskDone(w io.Writer, t progressTask, elapsed time.Duration) {
	fmt.Fprintf(w, "\r%s %s✔%s %s  %-12s %.1fs\r\n",
		ansiEraseLn, ansiGreen, ansiReset, t.name, t.done, elapsed.Seconds())
}

func moveCursorUp(w io.Writer, lines int) {
	for i := 0; i < lines; i++ {
		fmt.Fprint(w, ansiCurUp)
	}
}

// --- Helpers ---

func getServicesFromCompose() []string {
	composeFile := findComposeFile()
	if composeFile == "" {
		return nil
	}

	f, err := os.Open(composeFile)
	if err != nil {
		return nil
	}
	defer f.Close()

	var services []string
	inServices := false
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimRight(line, " \t")
		if trimmed == "services:" {
			inServices = true
			continue
		}
		if !inServices {
			continue
		}
		if len(trimmed) > 0 && trimmed[0] != ' ' && trimmed[0] != '#' {
			break
		}
		if len(line) > 2 && line[0] == ' ' && line[1] == ' ' && line[2] != ' ' && strings.HasSuffix(trimmed, ":") {
			svc := strings.TrimSpace(strings.TrimSuffix(trimmed, ":"))
			services = append(services, svc)
		}
	}
	return services
}

func findComposeFile() string {
	for _, name := range []string{"compose.yaml", "docker-compose.yaml", "docker-compose.yml", "compose.yml"} {
		if _, err := os.Stat(name); err == nil {
			return name
		}
	}
	return ""
}

func findServiceArg(args []string) string {
	for _, a := range args {
		if !strings.HasPrefix(a, "-") {
			return a
		}
	}
	return ""
}

func hasFlag(args []string, flag string) bool {
	for _, a := range args {
		if a == flag {
			return true
		}
	}
	return false
}

func mustGetwd() string {
	dir, err := os.Getwd()
	if err != nil {
		fmt.Fprintln(os.Stderr, "mock-docker: cannot get working directory:", err)
		os.Exit(1)
	}
	return dir
}
