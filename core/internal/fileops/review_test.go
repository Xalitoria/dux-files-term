package fileops

import (
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/AvengeMedia/dankgo/ipc"
	"github.com/stretchr/testify/require"
)

func TestDuplicateAnswerDoesNotLeakIntoTheNextPrompt(t *testing.T) {
	root := t.TempDir()
	src := filepath.Join(root, "src")
	write(t, filepath.Join(src, "a.txt"), "new")
	write(t, filepath.Join(src, "b.txt"), "new")
	dest := filepath.Join(root, "dest")
	write(t, filepath.Join(dest, "a.txt"), "old")
	write(t, filepath.Join(dest, "b.txt"), "old")

	manager, bus := newHarness(t)
	var asked []string
	bus.onConflict = func(c Conflict) Answer {
		asked = append(asked, c.ID)
		var wg sync.WaitGroup
		for range 4 {
			wg.Add(1)
			go func() {
				defer wg.Done()
				_ = manager.Respond(c.ID, Answer{ID: c.ID, Action: ActionReplace})
			}()
		}
		wg.Wait()
		return Answer{ID: c.ID, Action: ActionSkip}
	}

	view := run(t, manager, bus, spec{
		kind:    KindCopy,
		sources: []string{filepath.Join(src, "a.txt"), filepath.Join(src, "b.txt")},
		dest:    dest,
	})

	require.Equal(t, StateDone, view.State)
	require.Len(t, asked, 2, "both files must raise their own prompt")
	require.Equal(t, "old", read(t, filepath.Join(dest, "a.txt")))
	require.Equal(t, "old", read(t, filepath.Join(dest, "b.txt")))
}

func TestAnswerForAnotherJobIsRefused(t *testing.T) {
	root := t.TempDir()
	src := write(t, filepath.Join(root, "src", "a.txt"), "new")
	dest := filepath.Join(root, "dest")
	write(t, filepath.Join(dest, "a.txt"), "old")

	manager, bus := newHarness(t)
	bus.onConflict = func(c Conflict) Answer {
		require.Error(t, manager.Respond("j404", Answer{ID: c.ID, Action: ActionReplace}))
		require.Error(t, manager.Respond("j1", Answer{ID: "c99", Action: ActionReplace}))
		return Answer{ID: c.ID, Action: ActionSkip}
	}
	view := run(t, manager, bus, spec{kind: KindCopy, sources: []string{src}, dest: dest})

	require.Equal(t, StateDone, view.State)
	require.Equal(t, "old", read(t, filepath.Join(dest, "a.txt")))
}

func TestCancelAfterTheJobFinishedKeepsTheTerminalState(t *testing.T) {
	root := t.TempDir()
	src := write(t, filepath.Join(root, "src", "a.txt"), "a")
	dest := filepath.Join(root, "dest")
	require.NoError(t, os.Mkdir(dest, 0o755))

	manager, bus := newHarness(t)
	job := manager.enqueue(spec{kind: KindCopy, sources: []string{src}, dest: dest})

	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if err := manager.Cancel(job.id); err != nil {
			break
		}
	}
	view := bus.wait(t, job.id)

	listed := manager.Jobs()
	require.Len(t, listed, 1)
	require.Contains(t, []State{StateDone, StateFailed}, listed[0].State)
	require.Equal(t, view.State, listed[0].State)
}

func TestCopyIntoItsOwnSubdirectoryIsRefused(t *testing.T) {
	root := t.TempDir()
	src := filepath.Join(root, "src")
	write(t, filepath.Join(src, "a.txt"), "a")
	inner := filepath.Join(src, "inner")
	require.NoError(t, os.Mkdir(inner, 0o755))

	manager, _ := newHarness(t)
	for _, method := range []string{"files.op.copy", "files.op.move"} {
		out := call(t, manager, ipc.Request{Method: method, Params: map[string]any{
			"paths": []any{src},
			"dest":  inner,
		}})
		require.Contains(t, out, "error", method)
		require.Equal(t, "EINVAL", out["code"], method)
	}
	require.Empty(t, manager.Jobs())
}

func TestRelativePathInAListIsRefused(t *testing.T) {
	manager, _ := newHarness(t)

	out := call(t, manager, ipc.Request{Method: "files.op.trash", Params: map[string]any{
		"paths": "/home/someone/a,b.txt",
	}})
	require.Contains(t, out, "error")
	require.Empty(t, manager.Jobs(), "nothing is trashed when one entry is relative")
}

func TestRestoreNamesCannotEscapeTheTrash(t *testing.T) {
	trashDir := trashHome(t)
	manager, _ := newHarness(t)

	for _, name := range []string{"../../../etc/passwd", "sub/file", "..", ""} {
		out := call(t, manager, ipc.Request{Method: "files.op.restore", Params: map[string]any{
			"names":    []any{name},
			"trashDir": trashDir,
		}})
		require.Contains(t, out, "error", name)
	}
	require.Empty(t, manager.Jobs())
}

func TestUndoIsNotOfferedAfterReplacing(t *testing.T) {
	root := t.TempDir()
	src := write(t, filepath.Join(root, "src", "a.txt"), "new")
	dest := filepath.Join(root, "dest")
	write(t, filepath.Join(dest, "a.txt"), "irreplaceable")

	manager, bus := newHarness(t)
	bus.onConflict = func(c Conflict) Answer { return Answer{ID: c.ID, Action: ActionReplace} }
	view := run(t, manager, bus, spec{kind: KindCopy, scope: "w1", sources: []string{src}, dest: dest})

	require.Equal(t, StateDone, view.State)
	require.False(t, view.Undoable)
	require.Equal(t, "new", read(t, filepath.Join(dest, "a.txt")))

	_, err := manager.Undo("w1")
	require.Error(t, err, "undoing would delete the copy and the replaced file is already gone")
}

func TestRestoreReplacesAndMergesDirectories(t *testing.T) {
	trashDir := trashHome(t)
	root := t.TempDir()
	dir := filepath.Join(root, "docs")
	write(t, filepath.Join(dir, "trashed.txt"), "trashed")

	manager, bus := newHarness(t)
	run(t, manager, bus, spec{kind: KindTrash, sources: []string{dir}})
	write(t, filepath.Join(dir, "recreated.txt"), "recreated")

	var offered []Action
	bus.onConflict = func(c Conflict) Answer {
		offered = c.Actions
		return Answer{ID: c.ID, Action: ActionMerge}
	}
	view := run(t, manager, bus, spec{kind: KindRestore, sources: []string{"docs"}, trashDir: trashDir})

	require.Equal(t, StateDone, view.State)
	require.Contains(t, offered, ActionMerge, "two directories must be offered a merge")
	require.Equal(t, "trashed", read(t, filepath.Join(dir, "trashed.txt")))
	require.Equal(t, "recreated", read(t, filepath.Join(dir, "recreated.txt")))
	require.NoFileExists(t, filepath.Join(trashDir, "info", "docs.trashinfo"))
}

func TestDirectorySizeIsLeftOutWhenTheScanDidNotFinish(t *testing.T) {
	previous := scanBudget
	scanBudget = 0
	t.Cleanup(func() { scanBudget = previous })

	trashDir := trashHome(t)
	root := t.TempDir()
	dir := filepath.Join(root, "docs")
	write(t, filepath.Join(dir, "a.txt"), "12345")

	manager, bus := newHarness(t)
	view := run(t, manager, bus, spec{kind: KindTrash, sources: []string{dir}})

	require.Equal(t, StateDone, view.State)
	require.DirExists(t, filepath.Join(trashDir, "files", "docs"))
	require.NoFileExists(t, filepath.Join(trashDir, "directorysizes"), "a guessed size is worse than none")
}

func TestTransferOntoItselfKeepsTheSource(t *testing.T) {
	root := t.TempDir()
	write(t, filepath.Join(root, "dir", "inner.txt"), "keep me")
	write(t, filepath.Join(root, "a.txt"), "keep me too")
	sources := []string{filepath.Join(root, "dir"), filepath.Join(root, "a.txt")}
	manager, bus := newHarness(t)
	bus.onConflict = func(c Conflict) Answer {
		t.Errorf("asked about %s", c.Target.Path)
		return Answer{ID: c.ID, Action: ActionReplace}
	}

	moved := run(t, manager, bus, spec{kind: KindMove, sources: sources, dest: root})
	require.Equal(t, StateDone, moved.State)
	require.False(t, moved.Undoable)
	require.Equal(t, "keep me", read(t, filepath.Join(root, "dir", "inner.txt")))
	require.Equal(t, "keep me too", read(t, filepath.Join(root, "a.txt")))

	copied := run(t, manager, bus, spec{kind: KindCopy, sources: sources, dest: root})
	require.Equal(t, StateDone, copied.State)
	require.Equal(t, "keep me", read(t, filepath.Join(root, "dir", "inner.txt")))
	require.Equal(t, "keep me", read(t, filepath.Join(root, "dir (copy)", "inner.txt")))
	require.Equal(t, "keep me too", read(t, filepath.Join(root, "a (copy).txt")))
}
