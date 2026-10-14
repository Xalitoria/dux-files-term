package fileops

import (
	"bufio"
	"context"
	"encoding/json"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/AvengeMedia/dankgo/ipc"
	"github.com/stretchr/testify/require"
)

func call(t *testing.T, m *Manager, req ipc.Request) map[string]any {
	t.Helper()
	client, srv := net.Pipe()
	t.Cleanup(func() {
		_ = client.Close()
		_ = srv.Close()
	})

	go m.Handle(context.Background(), ipc.NewConnWriter(srv), req)

	line, err := bufio.NewReader(client).ReadBytes('\n')
	require.NoError(t, err)
	var out map[string]any
	require.NoError(t, json.Unmarshal(line, &out))
	return out
}

func TestHandleRejectsBadParams(t *testing.T) {
	manager, _ := newHarness(t)

	cases := []struct {
		name   string
		req    ipc.Request
		reason string
	}{
		{"relative source", ipc.Request{Method: "files.op.copy", Params: map[string]any{"paths": "relative/a.txt", "dest": "/tmp"}}, "paths"},
		{"relative dest", ipc.Request{Method: "files.op.move", Params: map[string]any{"paths": "/tmp/a", "dest": "dest"}}, "dest"},
		{"rename with a slash", ipc.Request{Method: "files.op.rename", Params: map[string]any{"path": "/tmp/a", "name": "b/c"}}, "name"},
		{"rename to dotdot", ipc.Request{Method: "files.op.rename", Params: map[string]any{"path": "/tmp/a", "name": ".."}}, "name"},
		{"mode that is not octal", ipc.Request{Method: "files.op.setPermissions", Params: map[string]any{"paths": "/tmp/a", "mode": "9x9"}}, "mode"},
		{"unknown method", ipc.Request{Method: "files.op.nope"}, "unknown"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			out := call(t, manager, tc.req)
			require.Contains(t, out, "error")
			require.Contains(t, out, "code")
			require.Empty(t, manager.Jobs())
		})
	}
}

func TestHandleStartsAJobAndLists(t *testing.T) {
	root := t.TempDir()
	src := write(t, filepath.Join(root, "a.txt"), "a")
	dest := filepath.Join(root, "dest")
	write(t, filepath.Join(dest, "keep"), "")

	manager, bus := newHarness(t)
	out := call(t, manager, ipc.Request{ID: 3, Method: "files.op.copy", Params: map[string]any{
		"paths": []any{src},
		"dest":  dest,
		"scope": "w1",
	}})

	result := out["result"].(map[string]any)
	job := result["job"].(map[string]any)
	require.Equal(t, "copy", job["kind"])
	bus.wait(t, job["id"].(string))

	listed := call(t, manager, ipc.Request{Method: "files.op.jobs"})["result"].(map[string]any)["jobs"].([]any)
	require.Len(t, listed, 1)
	require.Equal(t, "done", listed[0].(map[string]any)["state"])
	require.Equal(t, "a", read(t, filepath.Join(dest, "a.txt")))
}

func TestHandleRespondNeedsTheCurrentAnswerID(t *testing.T) {
	root := t.TempDir()
	src := write(t, filepath.Join(root, "src", "a.txt"), "new")
	dest := filepath.Join(root, "dest")
	write(t, filepath.Join(dest, "a.txt"), "old")

	manager, bus := newHarness(t)
	bus.onConflict = func(c Conflict) Answer {
		require.Error(t, manager.Respond(c.ID, Answer{ID: c.ID, Action: ActionSkip}), "unknown job id")
		return Answer{ID: "stale", Action: ActionSkip}
	}

	job := manager.enqueue(spec{kind: KindCopy, sources: []string{src}, dest: dest})
	require.Eventually(t, func() bool { return job.view().State == StatePausedConflict }, 5*time.Second, time.Millisecond)

	require.NoError(t, manager.Cancel(job.id))
	view := bus.wait(t, job.id)
	require.True(t, view.Cancelled)
	require.Equal(t, "old", read(t, filepath.Join(dest, "a.txt")))
}

func TestHandleCancelAndUndoErrors(t *testing.T) {
	manager, _ := newHarness(t)

	out := call(t, manager, ipc.Request{Method: "files.op.cancel", Params: map[string]any{"jobId": "j404"}})
	require.Contains(t, out, "error")

	out = call(t, manager, ipc.Request{Method: "files.op.undo", Params: map[string]any{"scope": "w1"}})
	require.Contains(t, out, "error")

	out = call(t, manager, ipc.Request{Method: "files.op.respond", Params: map[string]any{"jobId": "j404", "id": "c1", "action": "banana"}})
	require.Contains(t, out, "error")
}

func TestJobsListingIsSafeWhileAJobRecords(t *testing.T) {
	root := t.TempDir()
	src := filepath.Join(root, "src")
	for i := range 40 {
		write(t, filepath.Join(src, "f"+strconv.Itoa(i)+".txt"), "body")
	}
	dest := filepath.Join(root, "dest")
	require.NoError(t, os.Mkdir(dest, 0o755))

	manager, bus := newHarness(t)
	sources := make([]string, 0, 40)
	for i := range 40 {
		sources = append(sources, filepath.Join(src, "f"+strconv.Itoa(i)+".txt"))
	}
	job := manager.enqueue(spec{kind: KindCopy, scope: "w1", sources: sources, dest: dest})

	done := make(chan struct{})
	go func() {
		defer close(done)
		for range 200 {
			manager.Jobs()
		}
	}()

	bus.wait(t, job.id)
	<-done
	require.FileExists(t, filepath.Join(dest, "f39.txt"))
}

func TestDropActionOnTheSameDeviceIsAMove(t *testing.T) {
	manager, _ := newHarness(t)
	root := t.TempDir()
	src := filepath.Join(root, "a.txt")
	require.NoError(t, os.WriteFile(src, []byte("a"), 0o644))
	dest := filepath.Join(root, "sub")
	require.NoError(t, os.Mkdir(dest, 0o755))

	result := call(t, manager, ipc.Request{Method: "files.op.dropAction", Params: map[string]any{"paths": src, "dest": dest}})["result"].(map[string]any)
	require.Equal(t, "move", result["action"])
	require.Equal(t, true, result["sameDevice"])
}

func TestDropActionRejectsBadParams(t *testing.T) {
	manager, _ := newHarness(t)
	for _, params := range []map[string]any{
		{"paths": "relative/a.txt", "dest": t.TempDir()},
		{"paths": filepath.Join(t.TempDir(), "a"), "dest": "relative"},
	} {
		out := call(t, manager, ipc.Request{Method: "files.op.dropAction", Params: params})
		require.NotEmpty(t, out["error"])
	}
}

func TestHandleNewFolderPassesUnique(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, os.Mkdir(filepath.Join(root, "Untitled folder"), 0o755))

	manager, bus := newHarness(t)
	out := call(t, manager, ipc.Request{Method: "files.op.newFolder", Params: map[string]any{
		"path":   root,
		"name":   "Untitled folder",
		"unique": true,
	}})

	job := out["result"].(map[string]any)["job"].(map[string]any)
	view := bus.wait(t, job["id"].(string))
	require.Equal(t, StateDone, view.State)
	require.DirExists(t, filepath.Join(root, "Untitled folder 2"))
}
