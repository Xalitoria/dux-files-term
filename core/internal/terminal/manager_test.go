package terminal

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/duxlabs-dev/dux-files/core/internal/ipctest"
	"github.com/duxlabs-dev/dux-go/ipc"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type managerHarness struct {
	manager *Manager
	nextID  int
	seqs    map[string]int

	mu     sync.Mutex
	topics map[string]int
}

func (h *managerHarness) publish(topic string, _ any) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.topics[topic]++
}

func newManagerHarness(t *testing.T) *managerHarness {
	t.Helper()
	shell, err := exec.LookPath("sh")
	if err != nil {
		t.Skip("no sh")
	}
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, ".config"))
	t.Setenv("XDG_DATA_HOME", filepath.Join(home, ".local", "share"))
	t.Setenv("XDG_CONFIG_DIRS", "")
	t.Setenv("XDG_DATA_DIRS", "")
	t.Setenv("XDG_CURRENT_DESKTOP", "")
	t.Setenv("TERMINAL", "")
	t.Setenv("SHELL", shell)
	t.Setenv("PS1", "$ ")
	t.Setenv("HISTFILE", "/dev/null")

	h := &managerHarness{seqs: map[string]int{}, topics: map[string]int{}}
	h.manager = NewManager(h.publish, false, false)
	t.Cleanup(func() {
		h.manager.mu.Lock()
		running := make([]*Session, 0, len(h.manager.sessions))
		for _, session := range h.manager.sessions {
			running = append(running, session)
		}
		h.manager.mu.Unlock()
		h.manager.Close()
		for _, session := range running {
			select {
			case <-session.done:
			case <-time.After(5 * time.Second):
			}
		}
	})
	return h
}

func (h *managerHarness) call(ctx context.Context, t *testing.T, method string, params map[string]any) map[string]any {
	t.Helper()
	h.nextID++
	req := ipc.Request{ID: h.nextID, Method: method, Params: params}
	return ipctest.Call(t, func(w *ipc.ConnWriter) { h.manager.Handle(ctx, w, req) })
}

func (h *managerHarness) open(ctx context.Context, t *testing.T, cwd string) (id, topic string) {
	t.Helper()
	reply := h.call(ctx, t, "terminal.open", map[string]any{"cwd": cwd, "cols": 80, "rows": 24})
	require.Nil(t, reply["error"])
	result, ok := reply["result"].(map[string]any)
	require.True(t, ok, "open replied %v", reply)
	return result["id"].(string), result["topic"].(string)
}

func (h *managerHarness) run(ctx context.Context, t *testing.T, id, command string) {
	t.Helper()
	for _, press := range []map[string]any{{"text": command}, {"key": "enter"}} {
		press["id"], press["seq"] = id, h.seqs[id]
		h.seqs[id]++
		h.call(ctx, t, "terminal.input", press)
	}
}

func (h *managerHarness) screen(id string) string {
	session := h.manager.session(id)
	if session == nil {
		return ""
	}
	return screenText(session)
}

func (h *managerHarness) published(topic string) int {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.topics[topic]
}

func TestEveryOpenGetsItsOwnShell(t *testing.T) {
	h := newManagerHarness(t)
	ctx := context.Background()
	first, second := t.TempDir(), t.TempDir()

	idA, topicA := h.open(ctx, t, first)
	idB, topicB := h.open(ctx, t, second)

	assert.NotEqual(t, idA, idB)
	assert.NotEqual(t, topicA, topicB)
	assert.NotSame(t, h.manager.session(idA), h.manager.session(idB))
	assert.NotEqual(t, h.manager.session(idA).cmd.Process.Pid, h.manager.session(idB).cmd.Process.Pid)

	h.run(ctx, t, idA, "pwd; echo alpha-$((20+1))")
	h.run(ctx, t, idB, "pwd; echo beta-$((20+2))")

	assert.Eventually(t, func() bool {
		return strings.Contains(h.screen(idA), "alpha-21") && strings.Contains(h.screen(idB), "beta-22")
	}, 5*time.Second, 20*time.Millisecond)
	assert.Contains(t, h.screen(idA), first)
	assert.Contains(t, h.screen(idB), second)
	assert.NotContains(t, h.screen(idA), "beta-22")
	assert.NotContains(t, h.screen(idB), "alpha-21")
	assert.Eventually(t, func() bool { return h.published(topicA) > 0 && h.published(topicB) > 0 }, 5*time.Second, 20*time.Millisecond)
}

func TestShellVariablesStayInTheirOwnTab(t *testing.T) {
	h := newManagerHarness(t)
	ctx := context.Background()
	idA, _ := h.open(ctx, t, t.TempDir())
	idB, _ := h.open(ctx, t, t.TempDir())

	h.run(ctx, t, idA, "TAB=one; echo set-$TAB")
	assert.Eventually(t, func() bool { return strings.Contains(h.screen(idA), "set-one") }, 5*time.Second, 20*time.Millisecond)
	h.run(ctx, t, idB, "echo other-[$TAB]")

	assert.Eventually(t, func() bool { return strings.Contains(h.screen(idB), "other-[]") }, 5*time.Second, 20*time.Millisecond)
}

func TestClosingOneTerminalLeavesTheOthersRunning(t *testing.T) {
	h := newManagerHarness(t)
	ctx := context.Background()
	idA, topicA := h.open(ctx, t, t.TempDir())
	idB, _ := h.open(ctx, t, t.TempDir())
	idC, _ := h.open(ctx, t, t.TempDir())
	closed := h.manager.session(idB)

	h.call(ctx, t, "terminal.close", map[string]any{"id": idB})

	select {
	case <-closed.done:
	case <-time.After(5 * time.Second):
		t.Fatal("the closed shell kept running")
	}
	assert.Eventually(t, func() bool { return h.manager.session(idB) == nil }, 5*time.Second, 20*time.Millisecond)

	reply := h.call(ctx, t, "terminal.resize", map[string]any{"id": idB, "cols": 80, "rows": 24})
	assert.Contains(t, reply["error"], "no such terminal")

	for _, id := range []string{idA, idC} {
		h.run(ctx, t, id, "echo still-$((6*7))-"+id)
		assert.Eventually(t, func() bool { return strings.Contains(h.screen(id), "still-42-"+id) }, 5*time.Second, 20*time.Millisecond)
	}
	assert.Eventually(t, func() bool { return h.published(topicA) > 0 }, 5*time.Second, 20*time.Millisecond)
}

func TestTerminalIDsAreNeverReused(t *testing.T) {
	h := newManagerHarness(t)
	ctx := context.Background()
	idA, _ := h.open(ctx, t, t.TempDir())
	h.call(ctx, t, "terminal.close", map[string]any{"id": idA})
	assert.Eventually(t, func() bool { return h.manager.session(idA) == nil }, 5*time.Second, 20*time.Millisecond)

	idB, _ := h.open(ctx, t, t.TempDir())

	assert.NotEqual(t, idA, idB)
}

func TestTerminalsEndWithTheConnectionThatOpenedThem(t *testing.T) {
	h := newManagerHarness(t)
	ui, drop := context.WithCancel(context.Background())
	other, keep := context.WithCancel(context.Background())
	t.Cleanup(keep)

	idA, _ := h.open(ui, t, t.TempDir())
	idB, _ := h.open(other, t, t.TempDir())
	idC, _ := h.open(ui, t, t.TempDir())
	sessionB := h.manager.session(idB)

	drop()

	assert.Eventually(t, func() bool {
		return h.manager.session(idA) == nil && h.manager.session(idC) == nil
	}, 5*time.Second, 20*time.Millisecond)
	select {
	case <-sessionB.done:
		t.Fatal("a shell of another connection ended")
	case <-time.After(200 * time.Millisecond):
	}
	assert.Same(t, sessionB, h.manager.session(idB))
}

func TestFollowingAFolderMovesOnlyThatTerminal(t *testing.T) {
	h := newManagerHarness(t)
	ctx := context.Background()
	start := t.TempDir()
	target := t.TempDir()
	idA, _ := h.open(ctx, t, start)
	idB, _ := h.open(ctx, t, start)
	sessionA, sessionB := h.manager.session(idA), h.manager.session(idB)
	assert.Eventually(t, func() bool { return sessionA.shellIdle() && sessionB.shellIdle() }, 5*time.Second, 20*time.Millisecond)

	reply := h.call(ctx, t, "terminal.cd", map[string]any{"id": idA, "path": target})

	assert.Equal(t, map[string]any{"sent": true}, reply["result"])
	assert.Eventually(t, func() bool { return sessionA.shellCwd() == target }, 5*time.Second, 20*time.Millisecond)
	assert.Equal(t, start, sessionB.shellCwd())
}

func TestOpeningFallsBackToHomeForAMissingFolder(t *testing.T) {
	h := newManagerHarness(t)
	ctx := context.Background()
	idA, _ := h.open(ctx, t, filepath.Join(t.TempDir(), "gone"))

	home, err := os.UserHomeDir()
	require.NoError(t, err)
	assert.Eventually(t, func() bool { return h.manager.session(idA).shellCwd() == home }, 5*time.Second, 20*time.Millisecond)
}
