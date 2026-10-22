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
	session, err := startSession("terminal:test", []string{shell}, dir, 80, 24, true, events.publish)
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
