package terminal

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	uv "github.com/charmbracelet/ultraviolet"
	"github.com/charmbracelet/x/ansi"
	"github.com/charmbracelet/x/vt"
	"github.com/creack/pty"
	"github.com/duxlabs-dev/dux-files/core/internal/sshfs"
	"github.com/duxlabs-dev/dux-files/core/internal/termtheme"
	"golang.org/x/sys/unix"
)

const (
	frameInterval   = 16 * time.Millisecond
	scrollbackLines = 1000
	maxCols         = 500
	maxCellPixels   = 64
	maxRows         = 200

	maxClipboardBytes = 1 << 20
	syncTimeout       = 150 * time.Millisecond
	leavePoll         = 50 * time.Millisecond
	leaveAttempts     = 60
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

	// Set once a program picks a cursor of its own; until then the terminal's own setting applies.
	Shape string `json:"shape,omitempty"`
	Blink bool   `json:"blink,omitempty"`
}

type frame struct {
	Cols       int         `json:"cols"`
	Rows       int         `json:"rows"`
	Lines      []frameLine `json:"lines"`
	Cursor     cursor      `json:"cursor"`
	Scroll     int         `json:"scroll"`
	Scrollback int         `json:"scrollback"`
	Title      string      `json:"title"`
	Mouse      int         `json:"mouse"`
	Clipboard  *string     `json:"clipboard,omitempty"`

	Images     []frameImage    `json:"images,omitempty"`
	Dropped    []int           `json:"dropped,omitempty"`
	Placements json.RawMessage `json:"placements,omitempty"`
}

type input struct {
	key    string
	text   string
	mods   int
	paste  bool
	mouse  string
	x, y   int
	button string
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
	gfx        *graphics
	rows       []json.RawMessage
	lastCursor cursor
	lastTitle  string
	mouse      mouseTracker
	cellW      int
	cellH      int
	sized      pty.Winsize
	lastMouse  int
	styled     bool
	bracketed  bool
	elevation  func(user, host string, port int) sshfs.Elevation
	remoteKey  string
	remoteDir  string
	pending    string
	leaving    bool
	clipboard  *string
	syncSince  time.Time
	title      string
	scroll     int
	scheduled  bool
	closed     bool

	inputMu sync.Mutex
	nextSeq int
	queued  map[int]input
}

func startSession(topic string, argv []string, cwd string, cols, rows int, follow bool, colors *termtheme.Profile, publish Publisher) (*Session, error) {
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
	s.gfx = newGraphics(emu, func(b []byte) { s.writes.push(b) }, func() (int, int) { return s.cellW, s.cellH })
	emu.SetCallbacks(vt.Callbacks{
		Title:       func(title string) { s.title = title },
		EnableMode:  func(mode ansi.Mode) { s.setMode(mode, true) },
		DisableMode: func(mode ansi.Mode) { s.setMode(mode, false) },
		CursorStyle: func(vt.CursorStyle, bool) { s.styled = true },
	})
	emu.RegisterOscHandler(52, s.clipboardWrite)

	emu.RegisterCsiHandler('t', s.windowReport)
	if colors != nil {
		s.applyColors(*colors)
	}

	go s.writes.run()
	go s.pumpInput()
	go s.readOutput()
	return s, nil
}

func (s *Session) setMode(mode ansi.Mode, enabled bool) {
	s.mouse.set(mode, enabled)
	if mode == ansi.ModeBracketedPaste {
		s.bracketed = enabled
	}
	if mode != ansi.ModeSynchronizedOutput {
		return
	}
	s.syncSince = time.Time{}
	if enabled {
		s.syncSince = time.Now()
	}
}

// Only the write half of OSC 52: a program may fill the clipboard but never read it.
func (s *Session) clipboardWrite(data []byte) bool {
	parts := bytes.SplitN(data, []byte(";"), 3)
	if len(parts) != 3 || len(parts[2]) > maxClipboardBytes*4/3 {
		return true
	}
	text, err := base64.StdEncoding.DecodeString(string(parts[2]))
	if err != nil {
		return true
	}
	value := string(text)
	s.clipboard = &value
	return true
}

// XTWINOPS size reports, which image tools use to learn the cell size.
func (s *Session) windowReport(params ansi.Params) bool {
	op, _, _ := params.Param(0, 0)
	width, height := s.emu.Width(), s.emu.Height()
	switch {
	case s.cellW <= 0 || s.cellH <= 0:
		return false
	case op == 14:
		s.writes.push([]byte(fmt.Sprintf("\x1b[4;%d;%dt", height*s.cellH, width*s.cellW)))
	case op == 16:
		s.writes.push([]byte(fmt.Sprintf("\x1b[6;%d;%dt", s.cellH, s.cellW)))
	case op == 18:
		s.writes.push([]byte(fmt.Sprintf("\x1b[8;%d;%dt", height, width)))
	default:
		return false
	}
	return true
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
			s.gfx.feed(buf[:n])
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
	if !s.syncSince.IsZero() && time.Since(s.syncSince) < syncTimeout {
		s.scheduleLocked()
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
	if s.styled {
		style, blink := s.emu.CursorStyle()
		current.Shape, current.Blink = cursorShapes[style], blink
	}
	mouse := s.mouse.level()
	images, imagesChanged := s.gfx.frame(history-s.scroll, height)
	clipboard := s.clipboard
	s.clipboard = nil
	changed := len(lines) > 0 || imagesChanged || clipboard != nil || current != s.lastCursor || s.title != s.lastTitle || mouse != s.lastMouse
	s.lastCursor = current
	s.lastTitle = s.title
	s.lastMouse = mouse
	return frame{Cols: width, Rows: height, Lines: lines, Cursor: current, Scroll: s.scroll, Scrollback: history, Title: s.title, Mouse: mouse, Images: images.Images, Dropped: images.Dropped, Placements: images.Placements, Clipboard: clipboard}, changed
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
	if in.mouse != "" {
		s.sendMouse(mouseEventOf(in))
		return
	}
	s.jumpToBottomLocked()
	if in.paste {
		s.emu.Paste(in.text)
		return
	}
	sendKey(s.emu, in.key, in.text, in.mods)
}

func (s *Session) sendMouse(event uv.MouseEvent) {
	switch {
	case event == nil:
	case s.mouse.pixels && s.cellW > 0 && s.cellH > 0:
		if s.mouse.level() > 0 {
			s.writes.push([]byte(pixelMouse(event, s.cellW, s.cellH)))
		}
	default:
		s.emu.SendMouse(event)
	}
}

var cursorShapes = map[vt.CursorStyle]string{vt.CursorBlock: "block", vt.CursorUnderline: "underline", vt.CursorBar: "bar"}

// Programs that ask for focus events (mode 1004) learn when the view gains or loses focus.
func (s *Session) Focus(focused bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	switch {
	case s.closed:
	case focused:
		s.emu.Focus()
	default:
		s.emu.Blur()
	}
}

func (s *Session) jumpToBottomLocked() {
	if s.scroll == 0 {
		return
	}
	s.scroll = 0
	s.scheduleLocked()
}

// The cell size is in pixels. Programs that report the mouse in pixels need it to find the cell.
func (s *Session) Resize(cols, rows, cellW, cellH int) {
	cols, rows = clampSize(cols, rows)
	cellW, cellH = min(max(cellW, 0), maxCellPixels), min(max(cellH, 0), maxCellPixels)
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return
	}
	if cols != s.emu.Width() || rows != s.emu.Height() {
		s.emu.Resize(cols, rows)
	}
	size := pty.Winsize{Cols: uint16(cols), Rows: uint16(rows), X: uint16(cols * cellW), Y: uint16(rows * cellH)}
	if size != s.sized {
		s.cellW, s.cellH, s.sized = cellW, cellH, size
		_ = pty.Setsize(s.pty, &size)
	}
	s.rows = nil
	s.gfx.resync()
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
	if !s.follow || !filepath.IsAbs(dir) {
		return false
	}
	target, isRemote := s.remoteAt(dir)
	switch {
	case s.inRemote() && (!isRemote || s.remoteKey != target.key()):
		return (isRemote || isDir(dir)) && s.leaveRemote(dir)
	case isRemote:
		return isDir(dir) && s.followRemote(target)
	case !isDir(dir) || !s.shellIdle() || s.shellCwd() == filepath.Clean(dir):
		return false
	}
	return s.send(cdCommand(dir), "", "")
}

func (s *Session) remoteAt(dir string) (remote, bool) {
	target, ok := parseRemote(dir)
	if ok && !target.as.Active() && s.elevation != nil {
		target.as = s.elevation(target.user, target.host, target.port)
	}
	return target, ok
}

func (s *Session) inRemote() bool {
	s.mu.Lock()
	remote := s.remoteKey != ""
	s.mu.Unlock()
	return remote && !s.shellIdle() && commandName(s.foreground()) == "ssh"
}

// Ends the ssh session when its shell sits at a prompt, then follows the folder from the local shell.
func (s *Session) leaveRemote(dir string) bool {
	s.mu.Lock()
	ready, waiting := s.bracketed && !s.emu.IsAltScreen(), s.leaving
	if ready || waiting {
		s.pending = dir
	}
	s.leaving = s.leaving || ready
	key, path := s.remoteKey, s.remoteDir
	s.mu.Unlock()
	switch {
	case waiting:
		return true
	case !ready || !s.send(leaveCommand, key, path):
		return false
	}
	go s.followAfterLeaving()
	return true
}

func (s *Session) followAfterLeaving() {
	defer func() {
		s.mu.Lock()
		s.leaving, s.pending = false, ""
		s.mu.Unlock()
	}()
	for range leaveAttempts {
		select {
		case <-s.done:
			return
		case <-time.After(leavePoll):
		}
		if !s.shellIdle() {
			continue
		}
		s.mu.Lock()
		dir := s.pending
		s.remoteKey, s.remoteDir, s.leaving = "", "", false
		s.mu.Unlock()
		s.FollowDirectory(dir)
		return
	}
}

// An idle local shell logs in over ssh. Once logged in, moving around the same server cd's the remote shell,
// which is only done at a prompt: a shell reading a line has bracketed paste on.
func (s *Session) followRemote(target remote) bool {
	if s.shellIdle() {
		return s.send(sshCommand(target), target.key(), target.dir)
	}
	s.mu.Lock()
	moves := s.remoteKey == target.key() && s.remoteDir != target.dir && s.bracketed && !s.emu.IsAltScreen()
	s.mu.Unlock()
	return moves && commandName(s.foreground()) == "ssh" && s.send(remoteCdCommand(target.dir), target.key(), target.dir)
}

func (s *Session) send(text, remoteKey, remoteDir string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return false
	}
	s.remoteKey, s.remoteDir = remoteKey, remoteDir
	s.jumpToBottomLocked()
	s.emu.SendText(text)
	return true
}

func (s *Session) foreground() int {
	conn, err := s.pty.SyscallConn()
	if err != nil {
		return -1
	}
	foreground := -1
	_ = conn.Control(func(fd uintptr) {
		if pgrp, err := unix.IoctlGetInt(int(fd), unix.TIOCGPGRP); err == nil {
			foreground = pgrp
		}
	})
	return foreground
}

func (s *Session) shellIdle() bool {
	return s.cmd.Process != nil && s.foreground() == s.cmd.Process.Pid
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
