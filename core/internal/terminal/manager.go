package terminal

import (
	"context"
	"errors"
	"os"
	"strconv"
	"strings"
	"sync"

	"github.com/duxlabs-dev/dux-files/core/internal/wire"
	"github.com/duxlabs-dev/dux-go/files"
	"github.com/duxlabs-dev/dux-go/ipc"
	"github.com/duxlabs-dev/dux-go/ipc/params"
)

type Manager struct {
	publish   Publisher
	sandboxed bool
	hostSpawn bool

	mu       sync.Mutex
	sessions map[string]*Session
	lastID   int
}

func NewManager(publish Publisher, sandboxed, hostSpawn bool) *Manager {
	return &Manager{publish: publish, sandboxed: sandboxed, hostSpawn: hostSpawn, sessions: map[string]*Session{}}
}

func Handles(method string) bool {
	return strings.HasPrefix(method, "terminal.")
}

func (m *Manager) Handle(ctx context.Context, w *ipc.ConnWriter, req ipc.Request) {
	if req.Method == "terminal.open" {
		m.handleOpen(ctx, w, req)
		return
	}
	session := m.session(params.StringOpt(req.Params, "id", ""))
	if session == nil {
		wire.RespondError(w, req.ID, wire.Invalid("no such terminal"))
		return
	}
	switch req.Method {
	case "terminal.input":
		session.Input(params.IntOpt(req.Params, "seq", 0), input{
			key:    params.StringOpt(req.Params, "key", ""),
			text:   params.StringOpt(req.Params, "text", ""),
			mods:   params.IntOpt(req.Params, "mods", 0),
			paste:  params.BoolOpt(req.Params, "paste", false),
			mouse:  params.StringOpt(req.Params, "mouse", ""),
			x:      params.IntOpt(req.Params, "x", 0),
			y:      params.IntOpt(req.Params, "y", 0),
			button: params.StringOpt(req.Params, "button", "none"),
		})
		w.Respond(req.ID, map[string]any{})
	case "terminal.resize":
		session.Resize(params.IntOpt(req.Params, "cols", 80), params.IntOpt(req.Params, "rows", 24))
		w.Respond(req.ID, map[string]any{})
	case "terminal.scroll":
		session.Scroll(params.IntOpt(req.Params, "lines", 0))
		w.Respond(req.ID, map[string]any{})
	case "terminal.cd":
		w.Respond(req.ID, map[string]any{"sent": session.FollowDirectory(params.StringOpt(req.Params, "path", ""))})
	case "terminal.close":
		session.Close()
		w.Respond(req.ID, map[string]any{})
	default:
		w.RespondError(req.ID, "unknown terminal method: "+req.Method)
	}
}

// The session lives as long as the connection that opened it, so a crashed UI leaves no shells behind.
func (m *Manager) handleOpen(ctx context.Context, w *ipc.ConnWriter, req ipc.Request) {
	if m.sandboxed && !m.hostSpawn {
		wire.RespondError(w, req.ID, &files.Error{Code: files.CodeSandboxed, Err: errors.New("the terminal needs to run commands on the host")})
		return
	}
	cwd := params.StringOpt(req.Params, "cwd", "")
	if !isDir(cwd) {
		cwd, _ = os.UserHomeDir()
	}

	m.mu.Lock()
	m.lastID++
	id := strconv.Itoa(m.lastID)
	m.mu.Unlock()

	topic := "terminal:" + id
	session, err := startSession(topic, shellArgv(m.sandboxed, cwd, os.Getenv("SHELL")), cwd,
		params.IntOpt(req.Params, "cols", 80), params.IntOpt(req.Params, "rows", 24), !m.sandboxed, m.publish)
	if err != nil {
		wire.RespondError(w, req.ID, err)
		return
	}

	m.mu.Lock()
	m.sessions[id] = session
	m.mu.Unlock()

	go session.wait(func() { m.forget(id) })
	go func() {
		select {
		case <-ctx.Done():
			session.Close()
		case <-session.done:
		}
	}()
	w.Respond(req.ID, map[string]any{"id": id, "topic": topic, "follow": session.follow})
}

// A sandboxed shell runs on the host, where the foreground process group cannot be read, so it never follows folders.
func shellArgv(sandboxed bool, cwd, shell string) []string {
	if sandboxed {
		return []string{"flatpak-spawn", "--host", "--watch-bus", "--directory=" + cwd,
			"--env=TERM=xterm-256color", "--env=COLORTERM=truecolor",
			"sh", "-c", `exec "${SHELL:-/bin/sh}"`}
	}
	if shell == "" {
		shell = "/bin/sh"
	}
	return []string{shell}
}

func (m *Manager) session(id string) *Session {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.sessions[id]
}

func (m *Manager) forget(id string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.sessions, id)
}

func (m *Manager) Close() {
	m.mu.Lock()
	sessions := make([]*Session, 0, len(m.sessions))
	for _, session := range m.sessions {
		sessions = append(sessions, session)
	}
	m.mu.Unlock()
	for _, session := range sessions {
		session.Close()
	}
}
