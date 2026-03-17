package handlers

import (
	"context"
	"encoding/binary"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/cfilipov/dockge/internal/compose"
	"github.com/cfilipov/dockge/internal/stack"
	"github.com/cfilipov/dockge/internal/terminal"
	"github.com/cfilipov/dockge/internal/ws"
)

// RegisterTerminalHandlers registers terminalJoin/terminalLeave WS events
// and the binary frame handler for terminal input/resize.
func RegisterTerminalHandlers(app *App) {
	app.WS.Handle("terminalJoin", func(c *ws.Conn, msg *ws.ClientMessage) {
		if checkLogin(c, msg) == 0 {
			return
		}
		args := parseArgs(msg)
		var joinArgs ws.TerminalJoinArgs
		if !argObject(args, 0, &joinArgs) {
			if msg.ID != nil {
				ws.SendAck(c, *msg.ID, ws.TerminalJoinResponse{OK: false, Msg: "invalid args"})
			}
			return
		}
		if joinArgs.Shell == "" {
			joinArgs.Shell = "bash"
		}

		app.handleTerminalJoin(c, msg, &joinArgs)
	})

	app.WS.Handle("terminalLeave", func(c *ws.Conn, msg *ws.ClientMessage) {
		if checkLogin(c, msg) == 0 {
			return
		}
		args := parseArgs(msg)
		var leaveArgs ws.TerminalLeaveArgs
		if !argObject(args, 0, &leaveArgs) {
			if msg.ID != nil {
				ws.SendAck(c, *msg.ID, ws.ErrorResponse{OK: false, Msg: "invalid args"})
			}
			return
		}

		session := c.RemoveSession(leaveArgs.SessionID)
		if session == nil {
			if msg.ID != nil {
				ws.SendAck(c, *msg.ID, ws.OkResponse{OK: true})
			}
			return
		}

		app.Terms.RemoveWriterAndCleanup(session.TermName, session.WriterKey)
		slog.Debug("terminalLeave", "session", leaveArgs.SessionID, "term", session.TermName)

		if msg.ID != nil {
			ws.SendAck(c, *msg.ID, ws.OkResponse{OK: true})
		}
	})

	// Binary frame handler: dispatches terminal input/resize
	app.WS.OnBinary(func(c *ws.Conn, session *ws.TermSession, data []byte) {
		if !session.Interactive || len(data) < 1 {
			return
		}

		term := app.Terms.Get(session.TermName)
		if term == nil {
			return
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
	})
}

// sessionBinaryWriter creates a terminal.WriteFunc that prepends the 2-byte
// session ID header before sending binary frames on the main WS connection.
func sessionBinaryWriter(c *ws.Conn, sessionID uint16) terminal.WriteFunc {
	header := make([]byte, 2)
	binary.BigEndian.PutUint16(header, sessionID)
	return func(data string) {
		buf := make([]byte, 2+len(data))
		copy(buf, header)
		copy(buf[2:], data)
		c.WriteBinary(buf)
	}
}

// handleTerminalJoin dispatches to type-specific terminal setup.
func (app *App) handleTerminalJoin(c *ws.Conn, msg *ws.ClientMessage, args *ws.TerminalJoinArgs) {
	// Validate stack name for terminal types that use it
	if args.Stack != "" {
		if err := stack.ValidateStackName(args.Stack); err != nil {
			sendJoinError(c, msg, err.Error())
			return
		}
	}

	switch args.Type {
	case "combined":
		if args.Stack == "" {
			sendJoinError(c, msg, "stack parameter required")
			return
		}
		app.joinCombined(c, msg, args)

	case "container-log":
		if args.Stack == "" || args.Service == "" {
			sendJoinError(c, msg, "stack and service parameters required")
			return
		}
		app.joinContainerLog(c, msg, args)

	case "container-log-by-name":
		if args.Container == "" {
			sendJoinError(c, msg, "container parameter required")
			return
		}
		app.joinContainerLogByName(c, msg, args)

	case "exec":
		if args.Stack == "" || args.Service == "" {
			sendJoinError(c, msg, "stack and service parameters required")
			return
		}
		app.joinExec(c, msg, args)

	case "exec-by-name":
		if args.Container == "" {
			sendJoinError(c, msg, "container parameter required")
			return
		}
		app.joinExecByName(c, msg, args)

	case "console":
		app.joinConsole(c, msg, args)

	case "compose":
		if args.Stack == "" {
			sendJoinError(c, msg, "stack parameter required")
			return
		}
		app.joinCompose(c, msg, args)

	case "container-action":
		if args.Container == "" {
			sendJoinError(c, msg, "container parameter required")
			return
		}
		app.joinContainerAction(c, msg, args)

	default:
		sendJoinError(c, msg, "unknown terminal type: "+args.Type)
	}
}

func sendJoinError(c *ws.Conn, msg *ws.ClientMessage, errMsg string) {
	if msg.ID != nil {
		ws.SendAck(c, *msg.ID, ws.TerminalJoinResponse{OK: false, Msg: errMsg})
	}
}

// allocJoinAndReplay atomically registers a writer via JoinAndGetBuffer and sends buffer replay.
func (app *App) allocJoinAndReplay(c *ws.Conn, msg *ws.ClientMessage, termName string, interactive bool, term *terminal.Terminal) (uint16, string) {
	session := &ws.TermSession{
		TermName:    termName,
		Interactive: interactive,
	}
	sessionID := c.AllocSession(session)

	writer := sessionBinaryWriter(c, sessionID)
	buf := term.JoinAndGetBuffer(session.WriterKey, writer)

	if msg.ID != nil {
		ws.SendAck(c, *msg.ID, ws.TerminalJoinResponse{OK: true, SessionID: sessionID})
	}

	// Send buffer replay as binary frames
	if buf != "" {
		header := make([]byte, 2)
		binary.BigEndian.PutUint16(header, sessionID)
		payload := make([]byte, 2+len(buf))
		copy(payload, header)
		copy(payload[2:], buf)
		c.WriteBinary(payload)
	}

	return sessionID, session.WriterKey
}

func (app *App) joinCombined(c *ws.Conn, msg *ws.ClientMessage, args *ws.TerminalJoinArgs) {
	termName := "combined-" + args.Stack

	term := app.Terms.Get(termName)
	if term == nil {
		term = app.startCombinedLogs(termName, args.Stack)
	}
	if term == nil {
		sendJoinError(c, msg, "failed to start combined logs")
		return
	}

	app.allocJoinAndReplay(c, msg, termName, false, term)
}

func (app *App) joinContainerLog(c *ws.Conn, msg *ws.ClientMessage, args *ws.TerminalJoinArgs) {
	termName := "container-log-" + args.Service

	term := app.Terms.Recreate(termName, terminal.TypePipe)

	ctx, cancel := context.WithCancel(context.Background())
	term.SetCancel(cancel)

	go app.runContainerLogLoop(ctx, term, termName, args.Stack, args.Service)

	session := &ws.TermSession{
		TermName:    termName,
		Interactive: false,
	}
	sessionID := c.AllocSession(session)

	writer := sessionBinaryWriter(c, sessionID)
	term.AddWriter(session.WriterKey, writer)

	if msg.ID != nil {
		ws.SendAck(c, *msg.ID, ws.TerminalJoinResponse{OK: true, SessionID: sessionID})
	}

	// Send buffer replay
	buf := term.Buffer()
	if buf != "" {
		header := make([]byte, 2)
		binary.BigEndian.PutUint16(header, sessionID)
		payload := make([]byte, 2+len(buf))
		copy(payload, header)
		copy(payload[2:], buf)
		c.WriteBinary(payload)
	}
}

func (app *App) joinContainerLogByName(c *ws.Conn, msg *ws.ClientMessage, args *ws.TerminalJoinArgs) {
	termName := "container-log-by-name-" + args.Container
	term := app.Terms.Recreate(termName, terminal.TypePipe)

	ctx, cancel := context.WithCancel(context.Background())
	term.SetCancel(cancel)

	go app.runContainerLogByNameLoop(ctx, term, termName, args.Container)

	session := &ws.TermSession{
		TermName:    termName,
		Interactive: false,
	}
	sessionID := c.AllocSession(session)

	writer := sessionBinaryWriter(c, sessionID)
	term.AddWriter(session.WriterKey, writer)

	if msg.ID != nil {
		ws.SendAck(c, *msg.ID, ws.TerminalJoinResponse{OK: true, SessionID: sessionID})
	}

	buf := term.Buffer()
	if buf != "" {
		header := make([]byte, 2)
		binary.BigEndian.PutUint16(header, sessionID)
		payload := make([]byte, 2+len(buf))
		copy(payload, header)
		copy(payload[2:], buf)
		c.WriteBinary(payload)
	}
}

func (app *App) joinExec(c *ws.Conn, msg *ws.ClientMessage, args *ws.TerminalJoinArgs) {
	termName := "container-exec-" + args.Stack + "-" + args.Service + "-0"

	term := app.Terms.Recreate(termName, terminal.TypePTY)

	session := &ws.TermSession{
		TermName:    termName,
		Interactive: true,
	}
	sessionID := c.AllocSession(session)

	writer := sessionBinaryWriter(c, sessionID)
	term.AddWriter(session.WriterKey, writer)

	dir := filepath.Join(app.StacksDir, args.Stack)
	execArgs := []string{"compose"}
	execArgs = append(execArgs, compose.GlobalEnvArgs(app.StacksDir, args.Stack)...)
	execArgs = append(execArgs, "exec", args.Service, args.Shell)
	cmd := exec.Command("docker", execArgs...)
	cmd.Dir = dir
	cmd.Env = os.Environ()

	if err := term.StartPTY(cmd); err != nil {
		slog.Error("terminalJoin exec start", "err", err, "stack", args.Stack, "service", args.Service)
		app.Terms.Remove(termName)
		c.RemoveSession(sessionID)
		sendJoinError(c, msg, "failed to start terminal: "+err.Error())
		return
	}

	term.OnExit(func() {
		app.Terms.RemoveAfter(termName, 30*time.Second)
		// Notify client that terminal exited
		ws.SendEvent(c, "terminalExited", ws.TerminalExitedData{SessionID: sessionID})
	})

	if msg.ID != nil {
		ws.SendAck(c, *msg.ID, ws.TerminalJoinResponse{OK: true, SessionID: sessionID})
	}

	buf := term.Buffer()
	if buf != "" {
		header := make([]byte, 2)
		binary.BigEndian.PutUint16(header, sessionID)
		payload := make([]byte, 2+len(buf))
		copy(payload, header)
		copy(payload[2:], buf)
		c.WriteBinary(payload)
	}
}

func (app *App) joinExecByName(c *ws.Conn, msg *ws.ClientMessage, args *ws.TerminalJoinArgs) {
	termName := "container-exec-by-name-" + args.Container

	// Check if already running
	existing := app.Terms.Get(termName)
	if existing != nil && existing.IsRunning() {
		session := &ws.TermSession{
			TermName:    termName,
			Interactive: true,
		}
		sessionID := c.AllocSession(session)
		writer := sessionBinaryWriter(c, sessionID)
		existing.AddWriter(session.WriterKey, writer)

		if msg.ID != nil {
			ws.SendAck(c, *msg.ID, ws.TerminalJoinResponse{OK: true, SessionID: sessionID})
		}
		buf := existing.Buffer()
		if buf != "" {
			header := make([]byte, 2)
			binary.BigEndian.PutUint16(header, sessionID)
			payload := make([]byte, 2+len(buf))
			copy(payload, header)
			copy(payload[2:], buf)
			c.WriteBinary(payload)
		}
		return
	}

	term := app.Terms.Recreate(termName, terminal.TypePTY)

	session := &ws.TermSession{
		TermName:    termName,
		Interactive: true,
	}
	sessionID := c.AllocSession(session)
	writer := sessionBinaryWriter(c, sessionID)
	term.AddWriter(session.WriterKey, writer)

	cmd := exec.Command("docker", "exec", "-it", args.Container, args.Shell)
	cmd.Env = os.Environ()

	if err := term.StartPTY(cmd); err != nil {
		slog.Error("terminalJoin exec-by-name start", "err", err, "container", args.Container)
		app.Terms.Remove(termName)
		c.RemoveSession(sessionID)
		sendJoinError(c, msg, "failed to start terminal: "+err.Error())
		return
	}

	term.OnExit(func() {
		app.Terms.RemoveAfter(termName, 30*time.Second)
		ws.SendEvent(c, "terminalExited", ws.TerminalExitedData{SessionID: sessionID})
	})

	if msg.ID != nil {
		ws.SendAck(c, *msg.ID, ws.TerminalJoinResponse{OK: true, SessionID: sessionID})
	}

	buf := term.Buffer()
	if buf != "" {
		header := make([]byte, 2)
		binary.BigEndian.PutUint16(header, sessionID)
		payload := make([]byte, 2+len(buf))
		copy(payload, header)
		copy(payload[2:], buf)
		c.WriteBinary(payload)
	}
}

func (app *App) joinConsole(c *ws.Conn, msg *ws.ClientMessage, args *ws.TerminalJoinArgs) {
	termName := "console"

	// Check if already running
	existing := app.Terms.Get(termName)
	if existing != nil && existing.IsRunning() {
		session := &ws.TermSession{
			TermName:    termName,
			Interactive: true,
		}
		sessionID := c.AllocSession(session)
		writer := sessionBinaryWriter(c, sessionID)
		existing.AddWriter(session.WriterKey, writer)

		if msg.ID != nil {
			ws.SendAck(c, *msg.ID, ws.TerminalJoinResponse{OK: true, SessionID: sessionID})
		}
		buf := existing.Buffer()
		if buf != "" {
			header := make([]byte, 2)
			binary.BigEndian.PutUint16(header, sessionID)
			payload := make([]byte, 2+len(buf))
			copy(payload, header)
			copy(payload[2:], buf)
			c.WriteBinary(payload)
		}
		return
	}

	term := app.Terms.Create(termName, terminal.TypePTY)

	session := &ws.TermSession{
		TermName:    termName,
		Interactive: true,
	}
	sessionID := c.AllocSession(session)
	writer := sessionBinaryWriter(c, sessionID)
	term.AddWriter(session.WriterKey, writer)

	shell := args.Shell
	if _, err := exec.LookPath("bash"); err != nil {
		shell = "sh"
	}
	cmd := exec.Command(shell)
	cmd.Env = os.Environ()
	cmd.Dir = app.StacksDir

	if err := term.StartPTY(cmd); err != nil {
		slog.Error("terminalJoin console start", "err", err)
		app.Terms.Remove(termName)
		c.RemoveSession(sessionID)
		sendJoinError(c, msg, "failed to start terminal: "+err.Error())
		return
	}

	mainTerminalMu.Lock()
	app.MainTerminalName = termName
	mainTerminalMu.Unlock()

	if msg.ID != nil {
		ws.SendAck(c, *msg.ID, ws.TerminalJoinResponse{OK: true, SessionID: sessionID})
	}
}

func (app *App) joinCompose(c *ws.Conn, msg *ws.ClientMessage, args *ws.TerminalJoinArgs) {
	termName := "compose-" + args.Stack
	term := app.Terms.GetOrCreate(termName)
	app.allocJoinAndReplay(c, msg, termName, false, term)
}

func (app *App) joinContainerAction(c *ws.Conn, msg *ws.ClientMessage, args *ws.TerminalJoinArgs) {
	termName := "container-" + args.Container
	term := app.Terms.GetOrCreate(termName)
	app.allocJoinAndReplay(c, msg, termName, false, term)
}
