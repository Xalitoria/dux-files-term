package terminal

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type published struct {
	mu     sync.Mutex
	exited bool
}

func (p *published) publish(_ string, data any) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if event, ok := data.(map[string]any); ok && event["exited"] == true {
		p.exited = true
	}
}

func (p *published) hasExited() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.exited
}

func screenText(s *Session) string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.emu.String()
}

func startShell(t *testing.T, events *published) (*Session, string) {
	t.Helper()
	shell, err := exec.LookPath("sh")
	if err != nil {
		t.Skip("no sh")
	}
	dir := t.TempDir()
	t.Setenv("PS1", "$ ")
	session, err := startSession("terminal:test", []string{shell}, dir, 80, 24, true, nil, events.publish)
	require.NoError(t, err)
	go session.wait(func() {})
	t.Cleanup(session.Close)
	return session, dir
}

func TestSessionRunsInputInOrder(t *testing.T) {
	session, _ := startShell(t, &published{})

	session.Input(1, input{key: "enter"})
	session.Input(0, input{text: "echo in-$((40+2))-order"})

	assert.Eventually(t, func() bool { return strings.Contains(screenText(session), "in-42-order") }, 5*time.Second, 20*time.Millisecond)
}

func TestSessionFollowsOnlyWhenIdle(t *testing.T) {
	session, dir := startShell(t, &published{})
	target := filepath.Join(dir, "it's here")
	require.NoError(t, os.Mkdir(target, 0o755))

	assert.Eventually(t, session.shellIdle, 5*time.Second, 20*time.Millisecond)
	assert.False(t, session.FollowDirectory(filepath.Join(dir, "missing")))
	assert.True(t, session.FollowDirectory(target))
	assert.Eventually(t, func() bool { return session.shellCwd() == target }, 5*time.Second, 20*time.Millisecond)
	assert.False(t, session.FollowDirectory(target), "already there")

	session.Input(0, input{text: "sleep 5"})
	session.Input(1, input{key: "enter"})
	assert.Eventually(t, func() bool { return !session.shellIdle() }, 5*time.Second, 20*time.Millisecond)
	assert.False(t, session.FollowDirectory(dir), "a running program is never typed into")
}

func TestSessionCloseEndsShell(t *testing.T) {
	events := &published{}
	session, _ := startShell(t, events)

	session.Close()

	assert.Eventually(t, events.hasExited, 5*time.Second, 20*time.Millisecond)
}

func TestSessionReportsMouseInPixelsWhenAsked(t *testing.T) {
	script := `printf '\033[?1003h\033[?1016h'; stty raw -echo; head -c 12 | od -An -c; sleep 5`
	session, err := startSession("terminal:pixels", []string{"sh", "-c", script}, t.TempDir(), 80, 24, false, nil, (&published{}).publish)
	require.NoError(t, err)
	go session.wait(func() {})
	t.Cleanup(session.Close)
	session.Resize(80, 24, 10, 20)
	require.Eventually(t, func() bool {
		session.mu.Lock()
		defer session.mu.Unlock()
		return session.mouse.pixels && session.mouse.level() == 3
	}, 5*time.Second, 20*time.Millisecond)

	session.Input(0, input{mouse: "move", x: 2, y: 1, button: "none"})

	assert.Eventually(t, func() bool { return strings.Contains(screenText(session), "3   5   ;   2   6   ;   3   1   M") }, 5*time.Second, 20*time.Millisecond, screenText(session))
}

func frameOf(s *Session) frame {
	s.mu.Lock()
	defer s.mu.Unlock()
	f, _ := s.frameLocked()
	return f
}

type clipboards struct {
	mu   sync.Mutex
	seen []string
}

func (c *clipboards) publish(_ string, data any) {
	f, ok := data.(frame)
	if !ok || f.Clipboard == nil {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.seen = append(c.seen, *f.Clipboard)
}

func (c *clipboards) all() []string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return append([]string(nil), c.seen...)
}

func TestSessionPassesClipboardWritesOnly(t *testing.T) {
	seen := &clipboards{}
	session, err := startSession("terminal:clip", []string{"sh", "-c", `printf '\033]52;c;aGVsbG8=\033\\'; sleep 0.3; printf '\033]52;c;?\033\\'; sleep 5`}, t.TempDir(), 80, 24, false, nil, seen.publish)
	require.NoError(t, err)
	go session.wait(func() {})
	t.Cleanup(session.Close)

	require.Eventually(t, func() bool { return len(seen.all()) > 0 }, 5*time.Second, 20*time.Millisecond)

	time.Sleep(500 * time.Millisecond)
	assert.Equal(t, []string{"hello"}, seen.all(), "a read request is ignored")
}

func TestSessionReportsTheCursorAProgramPicked(t *testing.T) {
	session, err := startSession("terminal:cursor", []string{"sh", "-c", `sleep 0.3; printf '\033[6 q'; sleep 5`}, t.TempDir(), 80, 24, false, nil, (&published{}).publish)
	require.NoError(t, err)
	go session.wait(func() {})
	t.Cleanup(session.Close)
	assert.Empty(t, frameOf(session).Cursor.Shape, "the terminal's own setting applies until a program picks one")

	require.Eventually(t, func() bool { return frameOf(session).Cursor.Shape == "bar" }, 5*time.Second, 20*time.Millisecond)

	assert.False(t, frameOf(session).Cursor.Blink)
}

func TestSessionHoldsFramesDuringSynchronizedOutput(t *testing.T) {
	session, err := startSession("terminal:sync", []string{"sh", "-c", `printf '\033[?2026hhalf'; sleep 5`}, t.TempDir(), 80, 24, false, nil, (&published{}).publish)
	require.NoError(t, err)
	go session.wait(func() {})
	t.Cleanup(session.Close)
	require.Eventually(t, func() bool { return strings.Contains(screenText(session), "half") }, 5*time.Second, 20*time.Millisecond)

	session.mu.Lock()
	held := !session.syncSince.IsZero()
	session.mu.Unlock()

	assert.True(t, held)
}
