package handlers

import (
	"context"
	"encoding/binary"
	"log/slog"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/coder/websocket"

	"github.com/cfilipov/dockge/internal/compose"
	"github.com/cfilipov/dockge/internal/models"
	"github.com/cfilipov/dockge/internal/terminal"
)

var termWSIDCounter uint64

func nextTermWSID() string {
	return "t" + strconv.FormatUint(atomic.AddUint64(&termWSIDCounter, 1), 10)
}

// HandleTerminalWS handles dedicated WebSocket connections for terminal streams.
// Route: /ws/terminal/{type}?token=JWT&stack=...&service=...&container=...&shell=...
//
// Terminal types:
//   - combined             — combined logs for a stack
//   - container-log        — single service log (by stack+service)
//   - container-log-by-name — single container log (by container name)
//   - exec                 — interactive shell (by stack+service)
//   - exec-by-name         — interactive shell (by container name)
//   - console              — main server console
//   - compose              — compose action output viewer (by stack)
//   - container-action     — standalone container action output viewer (by container)
func (app *App) HandleTerminalWS(w http.ResponseWriter, r *http.Request) {
	// Authenticate via ?token= query parameter (skip in --no-auth mode)
	if !app.NoAuth {
		token := r.URL.Query().Get("token")
		if token == "" {
			http.Error(w, "token required", http.StatusUnauthorized)
			return
		}
		_, err := models.VerifyJWT(token, app.JWTSecret)
		if err != nil {
			http.Error(w, "invalid token", http.StatusUnauthorized)
			return
		}
	}

	// Parse terminal type from URL path: /ws/terminal/{type}
	path := strings.TrimPrefix(r.URL.Path, "/ws/terminal/")
	termType := strings.TrimRight(path, "/")
	if termType == "" {
		http.Error(w, "terminal type required", http.StatusBadRequest)
		return
	}

	// Parse query parameters
	q := r.URL.Query()
	stackName := q.Get("stack")
	serviceName := q.Get("service")
	containerName := q.Get("container")
	shell := q.Get("shell")
	if shell == "" {
		shell = "bash"
	}

	// Accept WebSocket upgrade
	ws, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		InsecureSkipVerify: true,
	})
	if err != nil {
		slog.Error("terminal ws accept", "err", err)
		return
	}

	connID := nextTermWSID()
	slog.Debug("terminal ws connected", "type", termType, "id", connID)

	// Dispatch to type-specific handler
	switch termType {
	case "combined":
		if stackName == "" {
			writeErrorAndClose(ws, "stack parameter required")
			return
		}
		app.handleTerminalWSCombined(ws, connID, stackName)

	case "container-log":
		if stackName == "" || serviceName == "" {
			writeErrorAndClose(ws, "stack and service parameters required")
			return
		}
		app.handleTerminalWSContainerLog(ws, connID, stackName, serviceName)

	case "container-log-by-name":
		if containerName == "" {
			writeErrorAndClose(ws, "container parameter required")
			return
		}
		app.handleTerminalWSContainerLogByName(ws, connID, containerName)

	case "exec":
		if stackName == "" || serviceName == "" {
			writeErrorAndClose(ws, "stack and service parameters required")
			return
		}
		app.handleTerminalWSExec(ws, connID, stackName, serviceName, shell)

	case "exec-by-name":
		if containerName == "" {
			writeErrorAndClose(ws, "container parameter required")
			return
		}
		app.handleTerminalWSExecByName(ws, connID, containerName, shell)

	case "console":
		app.handleTerminalWSConsole(ws, connID, shell)

	case "compose":
		if stackName == "" {
			writeErrorAndClose(ws, "stack parameter required")
			return
		}
		app.handleTerminalWSCompose(ws, connID, stackName)

	case "container-action":
		if containerName == "" {
			writeErrorAndClose(ws, "container parameter required")
			return
		}
		app.handleTerminalWSContainerAction(ws, connID, containerName)

	default:
		writeErrorAndClose(ws, "unknown terminal type: "+termType)
	}
}

// writeErrorAndClose writes an error message as text to the terminal and closes.
func writeErrorAndClose(ws *websocket.Conn, msg string) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ws.Write(ctx, websocket.MessageBinary, []byte("\r\n[Error] "+msg+"\r\n"))
	ws.Close(websocket.StatusPolicyViolation, msg)
}

// writeBinary sends a binary frame.
func writeBinary(ws *websocket.Conn, data []byte) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return ws.Write(ctx, websocket.MessageBinary, data)
}

// terminalBinaryWriter creates a terminal.WriteFunc that sends binary WS frames.
func terminalBinaryWriter(ws *websocket.Conn) terminal.WriteFunc {
	return func(data string) {
		writeBinary(ws, []byte(data))
	}
}

// handleTerminalWSCombined streams combined logs for a stack.
func (app *App) handleTerminalWSCombined(ws *websocket.Conn, connID, stackName string) {
	termName := "combined-" + stackName

	// Get or create combined log terminal
	term := app.Terms.Get(termName)
	if term == nil {
		term = app.startCombinedLogs(termName, stackName)
	}
	if term == nil {
		writeErrorAndClose(ws, "failed to start combined logs")
		return
	}

	// Atomic join: register writer and get buffer
	buf := term.JoinAndGetBuffer(connID, terminalBinaryWriter(ws))
	if buf != "" {
		writeBinary(ws, []byte(buf))
	}

	// Block on read pump — log terminals are display-only, so we just
	// wait for the client to close the connection.
	app.terminalWSReadPump(ws, connID, termName, false)
}

// handleTerminalWSContainerLog streams logs for a single service.
func (app *App) handleTerminalWSContainerLog(ws *websocket.Conn, connID, stackName, serviceName string) {
	termName := "container-log-" + serviceName

	// Recreate: fresh log stream, carry over writers
	term := app.Terms.Recreate(termName, terminal.TypePipe)

	ctx, cancel := context.WithCancel(context.Background())
	term.SetCancel(cancel)

	go app.runContainerLogLoop(ctx, term, termName, stackName, serviceName)

	term.AddWriter(connID, terminalBinaryWriter(ws))
	buf := term.Buffer()
	if buf != "" {
		writeBinary(ws, []byte(buf))
	}

	app.terminalWSReadPump(ws, connID, termName, false)
}

// handleTerminalWSContainerLogByName streams logs for a container by name.
func (app *App) handleTerminalWSContainerLogByName(ws *websocket.Conn, connID, containerName string) {
	termName := "container-log-by-name-" + containerName
	term := app.Terms.Recreate(termName, terminal.TypePipe)

	ctx, cancel := context.WithCancel(context.Background())
	term.SetCancel(cancel)

	go app.runContainerLogByNameLoop(ctx, term, termName, containerName)

	term.AddWriter(connID, terminalBinaryWriter(ws))
	buf := term.Buffer()
	if buf != "" {
		writeBinary(ws, []byte(buf))
	}

	app.terminalWSReadPump(ws, connID, termName, false)
}

// handleTerminalWSExec starts an interactive docker compose exec session.
func (app *App) handleTerminalWSExec(ws *websocket.Conn, connID, stackName, serviceName, shell string) {
	termName := "container-exec-" + stackName + "-" + serviceName + "-0"

	term := app.Terms.Recreate(termName, terminal.TypePTY)
	term.AddWriter(connID, terminalBinaryWriter(ws))

	dir := filepath.Join(app.StacksDir, stackName)
	execArgs := []string{"compose"}
	execArgs = append(execArgs, compose.GlobalEnvArgs(app.StacksDir, stackName)...)
	execArgs = append(execArgs, "exec", serviceName, shell)
	cmd := exec.Command("docker", execArgs...)
	cmd.Dir = dir
	cmd.Env = os.Environ()

	if err := term.StartPTY(cmd); err != nil {
		slog.Error("terminal ws exec start", "err", err, "stack", stackName, "service", serviceName)
		app.Terms.Remove(termName)
		writeErrorAndClose(ws, "failed to start terminal: "+err.Error())
		return
	}

	term.OnExit(func() {
		app.Terms.RemoveAfter(termName, 30*time.Second)
	})

	// Send buffer replay
	buf := term.Buffer()
	if buf != "" {
		writeBinary(ws, []byte(buf))
	}

	app.terminalWSReadPump(ws, connID, termName, true)
}

// handleTerminalWSExecByName starts an interactive docker exec session by container name.
func (app *App) handleTerminalWSExecByName(ws *websocket.Conn, connID, containerName, shell string) {
	termName := "container-exec-by-name-" + containerName

	// Check if already running
	existing := app.Terms.Get(termName)
	if existing != nil && existing.IsRunning() {
		existing.AddWriter(connID, terminalBinaryWriter(ws))
		buf := existing.Buffer()
		if buf != "" {
			writeBinary(ws, []byte(buf))
		}
		app.terminalWSReadPump(ws, connID, termName, true)
		return
	}

	term := app.Terms.Recreate(termName, terminal.TypePTY)
	term.AddWriter(connID, terminalBinaryWriter(ws))

	cmd := exec.Command("docker", "exec", "-it", containerName, shell)
	cmd.Env = os.Environ()

	if err := term.StartPTY(cmd); err != nil {
		slog.Error("terminal ws exec-by-name start", "err", err, "container", containerName)
		app.Terms.Remove(termName)
		writeErrorAndClose(ws, "failed to start terminal: "+err.Error())
		return
	}

	term.OnExit(func() {
		app.Terms.RemoveAfter(termName, 30*time.Second)
	})

	buf := term.Buffer()
	if buf != "" {
		writeBinary(ws, []byte(buf))
	}

	app.terminalWSReadPump(ws, connID, termName, true)
}

// handleTerminalWSCompose joins a compose action terminal (read-only viewer).
// The terminal is created by the compose action handler; this viewer joins
// to receive buffered output and live updates.
func (app *App) handleTerminalWSCompose(ws *websocket.Conn, connID, stackName string) {
	termName := "compose-" + stackName
	term := app.Terms.GetOrCreate(termName)
	buf := term.JoinAndGetBuffer(connID, terminalBinaryWriter(ws))
	if buf != "" {
		writeBinary(ws, []byte(buf))
	}
	app.terminalWSReadPump(ws, connID, termName, false)
}

// handleTerminalWSContainerAction joins a standalone container action terminal (read-only viewer).
func (app *App) handleTerminalWSContainerAction(ws *websocket.Conn, connID, containerName string) {
	termName := "container-" + containerName
	term := app.Terms.GetOrCreate(termName)
	buf := term.JoinAndGetBuffer(connID, terminalBinaryWriter(ws))
	if buf != "" {
		writeBinary(ws, []byte(buf))
	}
	app.terminalWSReadPump(ws, connID, termName, false)
}

// handleTerminalWSConsole starts a bash/sh console terminal.
func (app *App) handleTerminalWSConsole(ws *websocket.Conn, connID, shell string) {
	termName := "console"

	// Check if already running
	existing := app.Terms.Get(termName)
	if existing != nil && existing.IsRunning() {
		existing.AddWriter(connID, terminalBinaryWriter(ws))
		buf := existing.Buffer()
		if buf != "" {
			writeBinary(ws, []byte(buf))
		}
		app.terminalWSReadPump(ws, connID, termName, true)
		return
	}

	term := app.Terms.Create(termName, terminal.TypePTY)
	term.AddWriter(connID, terminalBinaryWriter(ws))

	if _, err := exec.LookPath("bash"); err != nil {
		shell = "sh"
	}
	cmd := exec.Command(shell)
	cmd.Env = os.Environ()
	cmd.Dir = app.StacksDir

	if err := term.StartPTY(cmd); err != nil {
		slog.Error("terminal ws console start", "err", err)
		app.Terms.Remove(termName)
		writeErrorAndClose(ws, "failed to start terminal: "+err.Error())
		return
	}

	mainTerminalMu.Lock()
	app.MainTerminalName = termName
	mainTerminalMu.Unlock()

	app.terminalWSReadPump(ws, connID, termName, true)
}

// terminalWSReadPump reads from the WebSocket until closed.
// For interactive terminals, it parses the binary protocol:
//   - 0x00 + UTF-8 bytes = terminal input
//   - 0x01 + 4 bytes (uint16 rows BE + uint16 cols BE) = resize
//
// For display-only terminals, it just waits for close.
func (app *App) terminalWSReadPump(ws *websocket.Conn, connID, termName string, interactive bool) {
	defer func() {
		app.Terms.RemoveWriterAndCleanup(termName, connID)
		ws.CloseNow()
		slog.Debug("terminal ws disconnected", "id", connID, "term", termName)
	}()

	ctx := context.Background()

	for {
		typ, data, err := ws.Read(ctx)
		if err != nil {
			slog.Debug("terminal ws read exit", "id", connID, "term", termName, "err", err)
			return
		}

		if !interactive || typ != websocket.MessageBinary || len(data) < 1 {
			continue
		}

		term := app.Terms.Get(termName)
		if term == nil {
			continue
		}

		switch data[0] {
		case 0x00: // input
			if len(data) > 1 {
				term.Input(string(data[1:]))
			}
		case 0x01: // resize
			if len(data) >= 5 {
				rows := binary.BigEndian.Uint16(data[1:3])
				cols := binary.BigEndian.Uint16(data[3:5])
				if rows > 0 && cols > 0 {
					term.Resize(rows, cols)
				}
			}
		}
	}
}
