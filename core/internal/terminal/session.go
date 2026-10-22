package terminal

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	uv "github.com/charmbracelet/ultraviolet"
	"github.com/charmbracelet/x/vt"
	"github.com/creack/pty"
	"golang.org/x/sys/unix"
)

const (
	frameInterval   = 16 * time.Millisecond
	scrollbackLines = 1000
	maxCols         = 500
	maxRows         = 200
)

type Publisher func(topic string, data any)

type frameLine struct {
	Y    int             `json:"y"`
	Runs json.RawMessage `json:"runs"`
}

type cursor struct {
	X       int  `json:"x"`
	Y       int  `json:"y"`
	Visible bool `json:"visible"`
}

type frame struct {
	Cols       int         `json:"cols"`
	Rows       int         `json:"rows"`
	Lines      []frameLine `json:"lines"`
	Cursor     cursor      `json:"cursor"`
	Scroll     int         `json:"scroll"`
	Scrollback int         `json:"scrollback"`
	Title      string      `json:"title"`
}

type input struct {
	key   string
	text  string
	mods  int
	paste bool
}

type Session struct {
	topic   string
	publish Publisher
	cmd     *exec.Cmd
	pty     *os.File
	follow  bool
	done    chan struct{}
	writes  *writeQueue

	mu         sync.Mutex
	emu        *vt.Emulator
	rows       []json.RawMessage
	lastCursor cursor
	lastTitle  string
	title      string
	scroll     int
	scheduled  bool
	closed     bool

	inputMu sync.Mutex
	nextSeq int
	queued  map[int]input
}

func startSession(topic string, argv []string, cwd string, cols, rows int, follow bool, publish Publisher) (*Session, error) {
	if len(argv) == 0 {
		return nil, errors.New("no shell to run")
	}
	cols, rows = clampSize(cols, rows)
	cmd := exec.Command(argv[0], argv[1:]...)
	cmd.Dir = cwd
	cmd.Env = append(os.Environ(), "TERM=xterm-256color", "COLORTERM=truecolor")
	ptmx, err := pty.StartWithSize(cmd, &pty.Winsize{Cols: uint16(cols), Rows: uint16(rows)})
	if err != nil {
		return nil, err
	}

	emu := vt.NewEmulator(cols, rows)
	emu.SetScrollbackSize(scrollbackLines)
	s := &Session{
		topic:   topic,
		publish: publish,
		cmd:     cmd,
		pty:     ptmx,
		follow:  follow,
		done:    make(chan struct{}),
		writes:  newWriteQueue(ptmx),
		emu:     emu,
		queued:  map[int]input{},
	}
	emu.SetCallbacks(vt.Callbacks{Title: func(title string) { s.title = title }})

	go s.writes.run()
	go s.pumpInput()
	go s.readOutput()
	return s, nil
}

func clampSize(cols, rows int) (int, int) {
	return min(max(cols, 2), maxCols), min(max(rows, 1), maxRows)
}

func (s *Session) pumpInput() {
	buf := make([]byte, 4096)
	for {
		n, err := s.emu.Read(buf)
		if n > 0 {
			s.writes.push(bytes.Clone(buf[:n]))
		}
		if err != nil {
			return
		}
	}
}

func (s *Session) readOutput() {
	buf := make([]byte, 32*1024)
	for {
		n, err := s.pty.Read(buf)
		if n > 0 {
			s.mu.Lock()
			_, _ = s.emu.Write(buf[:n])
			s.scheduleLocked()
			s.mu.Unlock()
		}
		if err != nil {
			return
		}
	}
}

func (s *Session) wait(onExit func()) {
	_ = s.cmd.Wait()
	s.Close()
	s.publish(s.topic, map[string]any{"exited": true})
	onExit()
}

func (s *Session) Close() {
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return
	}
	s.closed = true
	s.mu.Unlock()

	close(s.done)
	if s.cmd.Process != nil {
		_ = syscall.Kill(-s.cmd.Process.Pid, syscall.SIGHUP)
	}
	s.writes.close()
	_ = s.pty.Close()
	_ = s.emu.Close()
}

func (s *Session) scheduleLocked() {
	if s.scheduled || s.closed {
		return
	}
	s.scheduled = true
	time.AfterFunc(frameInterval, s.flush)
}

func (s *Session) flush() {
	s.mu.Lock()
	s.scheduled = false
	if s.closed {
		s.mu.Unlock()
		return
	}
	next, changed := s.frameLocked()
	s.mu.Unlock()
	if changed {
		s.publish(s.topic, next)
	}
}

func (s *Session) frameLocked() (frame, bool) {
	width, height := s.emu.Width(), s.emu.Height()
	history := 0
	if !s.emu.IsAltScreen() {
		history = s.emu.ScrollbackLen()
	}
	s.scroll = min(max(s.scroll, 0), history)
	if len(s.rows) != height {
		s.rows = make([]json.RawMessage, height)
	}

	lines := []frameLine{}
	for y := range height {
		data := encodeRow(rowRuns(width, s.lineSource(history-s.scroll+y, history), s.emu.IndexedColor))
		if s.rows[y] != nil && bytes.Equal(data, s.rows[y]) {
			continue
		}
		s.rows[y] = data
		lines = append(lines, frameLine{Y: y, Runs: data})
	}

	position := s.emu.CursorPosition()
	current := cursor{X: position.X, Y: position.Y + s.scroll, Visible: !s.emu.CursorHidden() && s.scroll == 0}
	changed := len(lines) > 0 || current != s.lastCursor || s.title != s.lastTitle
	s.lastCursor = current
	s.lastTitle = s.title
	return frame{Cols: width, Rows: height, Lines: lines, Cursor: current, Scroll: s.scroll, Scrollback: history, Title: s.title}, changed
}

func (s *Session) lineSource(index, history int) func(x int) *uv.Cell {
	if index < history {
		return func(x int) *uv.Cell { return s.emu.ScrollbackCellAt(x, index) }
	}
	row := index - history
	return func(x int) *uv.Cell { return s.emu.CellAt(x, row) }
}

// Input arrives on concurrent request goroutines; seq restores the order the keys were typed in.
func (s *Session) Input(seq int, in input) {
	s.inputMu.Lock()
	defer s.inputMu.Unlock()
	if seq < s.nextSeq {
		return
	}
	s.queued[seq] = in
	for {
		next, ok := s.queued[s.nextSeq]
		if !ok {
			return
		}
		delete(s.queued, s.nextSeq)
		s.nextSeq++
		s.apply(next)
	}
}

func (s *Session) apply(in input) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return
	}
	s.jumpToBottomLocked()
	if in.paste {
		s.emu.Paste(in.text)
		return
	}
	sendKey(s.emu, in.key, in.text, in.mods)
}

func (s *Session) jumpToBottomLocked() {
	if s.scroll == 0 {
		return
	}
	s.scroll = 0
	s.scheduleLocked()
}

func (s *Session) Resize(cols, rows int) {
	cols, rows = clampSize(cols, rows)
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return
	}
	if cols != s.emu.Width() || rows != s.emu.Height() {
		s.emu.Resize(cols, rows)
		_ = pty.Setsize(s.pty, &pty.Winsize{Cols: uint16(cols), Rows: uint16(rows)})
	}
	s.rows = nil
	s.lastCursor = cursor{X: -1}
	s.scheduleLocked()
}

func (s *Session) Scroll(lines int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed || lines == 0 {
		return
	}
	if s.emu.IsAltScreen() {
		code := uv.KeyUp
		if lines < 0 {
			code = uv.KeyDown
		}
		for range min(abs(lines), maxRows) {
			s.emu.SendKey(uv.KeyPressEvent{Code: code})
		}
		return
	}
	s.scroll += lines
	s.scheduleLocked()
}

func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}

// Only an idle shell is told to cd, so nothing gets typed into a running program.
func (s *Session) FollowDirectory(dir string) bool {
	if !s.follow || !isDir(dir) || !s.shellIdle() || s.shellCwd() == filepath.Clean(dir) {
		return false
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return false
	}
	s.jumpToBottomLocked()
	s.emu.SendText(cdCommand(dir))
	return true
}

func (s *Session) shellIdle() bool {
	conn, err := s.pty.SyscallConn()
	if err != nil {
		return false
	}
	foreground := -1
	_ = conn.Control(func(fd uintptr) {
		if pgrp, err := unix.IoctlGetInt(int(fd), unix.TIOCGPGRP); err == nil {
			foreground = pgrp
		}
	})
	return s.cmd.Process != nil && foreground == s.cmd.Process.Pid
}

func (s *Session) shellCwd() string {
	if s.cmd.Process == nil {
		return ""
	}
	cwd, _ := os.Readlink("/proc/" + strconv.Itoa(s.cmd.Process.Pid) + "/cwd")
	return cwd
}

func isDir(path string) bool {
	info, err := os.Stat(path)
	return filepath.IsAbs(path) && err == nil && info.IsDir()
}

// Ctrl+E Ctrl+U clears whatever was typed at the prompt; the leading space keeps the cd out of shell history.
func cdCommand(dir string) string {
	return "\x05\x15 cd -- " + shellQuote(dir) + "\r"
}

func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}
