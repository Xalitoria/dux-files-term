package fileops

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestCopyPreservesModeAndTime(t *testing.T) {
	root := t.TempDir()
	src := write(t, filepath.Join(root, "src", "note.txt"), "hello")
	require.NoError(t, os.Chmod(src, 0o640))
	stamp := time.Now().Add(-48 * time.Hour).Truncate(time.Second)
	require.NoError(t, os.Chtimes(src, stamp, stamp))
	dest := filepath.Join(root, "dest")
	require.NoError(t, os.Mkdir(dest, 0o755))

	manager, bus := newHarness(t)
	view := run(t, manager, bus, spec{kind: KindCopy, sources: []string{src}, dest: dest})

	require.Equal(t, StateDone, view.State)
	require.Equal(t, "hello", read(t, filepath.Join(dest, "note.txt")))
	require.Equal(t, "hello", read(t, src))

	info, err := os.Lstat(filepath.Join(dest, "note.txt"))
	require.NoError(t, err)
	require.Equal(t, os.FileMode(0o640), info.Mode().Perm())
	require.Equal(t, stamp.Unix(), info.ModTime().Unix())
	require.Empty(t, tempsIn(t, dest))
}

func TestCopyTreeKeepsSymlinksAsLinks(t *testing.T) {
	root := t.TempDir()
	src := filepath.Join(root, "src")
	write(t, filepath.Join(src, "inner", "a.txt"), "a")
	require.NoError(t, os.Symlink("inner/a.txt", filepath.Join(src, "link")))
	require.NoError(t, os.Symlink("/nowhere", filepath.Join(src, "broken")))
	dest := filepath.Join(root, "dest")
	require.NoError(t, os.Mkdir(dest, 0o755))

	manager, bus := newHarness(t)
	view := run(t, manager, bus, spec{kind: KindCopy, sources: []string{src}, dest: dest})
	require.Equal(t, StateDone, view.State)

	target, err := os.Readlink(filepath.Join(dest, "src", "link"))
	require.NoError(t, err)
	require.Equal(t, "inner/a.txt", target)

	broken, err := os.Lstat(filepath.Join(dest, "src", "broken"))
	require.NoError(t, err)
	require.NotZero(t, broken.Mode()&os.ModeSymlink)
	require.Equal(t, "a", read(t, filepath.Join(dest, "src", "inner", "a.txt")))
}

func TestMoveSameDeviceRenames(t *testing.T) {
	root := t.TempDir()
	src := filepath.Join(root, "src")
	write(t, filepath.Join(src, "deep", "a.txt"), "a")
	dest := filepath.Join(root, "dest")
	require.NoError(t, os.Mkdir(dest, 0o755))

	manager, bus := newHarness(t)
	view := run(t, manager, bus, spec{kind: KindMove, sources: []string{src}, dest: dest})

	require.Equal(t, StateDone, view.State)
	require.Equal(t, "a", read(t, filepath.Join(dest, "src", "deep", "a.txt")))
	_, err := os.Lstat(src)
	require.True(t, os.IsNotExist(err))
	require.Zero(t, view.Progress.BytesTotal)
}

func TestMoveCrossDeviceCopiesAndRemoves(t *testing.T) {
	root := t.TempDir()
	dest := otherDevice(t, root)
	src := filepath.Join(root, "src")
	write(t, filepath.Join(src, "a.txt"), "a")
	write(t, filepath.Join(src, "sub", "b.txt"), "bb")

	manager, bus := newHarness(t)
	view := run(t, manager, bus, spec{kind: KindMove, sources: []string{src}, dest: dest})

	require.Equal(t, StateDone, view.State)
	require.Equal(t, "a", read(t, filepath.Join(dest, "src", "a.txt")))
	require.Equal(t, "bb", read(t, filepath.Join(dest, "src", "sub", "b.txt")))
	_, err := os.Lstat(src)
	require.True(t, os.IsNotExist(err))
	require.Equal(t, int64(3), view.Progress.BytesTotal)
	require.Empty(t, tempsIn(t, filepath.Join(dest, "src")))
}

func TestConflictReplaceKeepBothSkip(t *testing.T) {
	cases := []struct {
		action Action
		body   string
		extra  string
	}{
		{ActionReplace, "new", ""},
		{ActionSkip, "old", ""},
		{ActionKeepBoth, "old", "a (copy).txt"},
	}

	for _, tc := range cases {
		t.Run(string(tc.action), func(t *testing.T) {
			root := t.TempDir()
			src := write(t, filepath.Join(root, "src", "a.txt"), "new")
			dest := filepath.Join(root, "dest")
			write(t, filepath.Join(dest, "a.txt"), "old")

			manager, bus := newHarness(t)
			bus.onConflict = func(c Conflict) Answer { return Answer{ID: c.ID, Action: tc.action} }
			view := run(t, manager, bus, spec{kind: KindCopy, sources: []string{src}, dest: dest})

			require.Equal(t, StateDone, view.State)
			require.Equal(t, tc.body, read(t, filepath.Join(dest, "a.txt")))
			if tc.extra == "" {
				require.Empty(t, tempsIn(t, dest))
				return
			}
			require.Equal(t, "new", read(t, filepath.Join(dest, tc.extra)))
		})
	}
}

func TestConflictMergeKeepsExistingChildren(t *testing.T) {
	root := t.TempDir()
	src := filepath.Join(root, "src", "shared")
	write(t, filepath.Join(src, "new.txt"), "new")
	dest := filepath.Join(root, "dest")
	write(t, filepath.Join(dest, "shared", "old.txt"), "old")

	manager, bus := newHarness(t)
	var asked int
	bus.onConflict = func(c Conflict) Answer {
		asked++
		require.Contains(t, c.Actions, ActionMerge)
		return Answer{ID: c.ID, Action: ActionMerge, ApplyToAll: true}
	}
	view := run(t, manager, bus, spec{kind: KindCopy, sources: []string{src}, dest: dest})

	require.Equal(t, StateDone, view.State)
	require.Equal(t, 1, asked)
	require.Equal(t, "old", read(t, filepath.Join(dest, "shared", "old.txt")))
	require.Equal(t, "new", read(t, filepath.Join(dest, "shared", "new.txt")))
}

func TestConflictApplyToAllAsksOnce(t *testing.T) {
	root := t.TempDir()
	src := filepath.Join(root, "src")
	write(t, filepath.Join(src, "a.txt"), "new")
	write(t, filepath.Join(src, "b.txt"), "new")
	dest := filepath.Join(root, "dest")
	write(t, filepath.Join(dest, "a.txt"), "old")
	write(t, filepath.Join(dest, "b.txt"), "old")

	manager, bus := newHarness(t)
	var asked int
	bus.onConflict = func(c Conflict) Answer {
		asked++
		return Answer{ID: c.ID, Action: ActionSkip, ApplyToAll: true}
	}
	view := run(t, manager, bus, spec{
		kind:    KindCopy,
		sources: []string{filepath.Join(src, "a.txt"), filepath.Join(src, "b.txt")},
		dest:    dest,
	})

	require.Equal(t, 1, asked)
	require.Equal(t, 2, view.Skipped)
	require.Equal(t, "old", read(t, filepath.Join(dest, "b.txt")))
}

func TestErrorPausesAndSkips(t *testing.T) {
	root := t.TempDir()
	src := filepath.Join(root, "src")
	write(t, filepath.Join(src, "a.txt"), "a")
	dest := filepath.Join(root, "dest")
	require.NoError(t, os.Mkdir(dest, 0o755))

	manager, bus := newHarness(t)
	var seen Failure
	bus.onFailure = func(f Failure) Answer {
		seen = f
		return Answer{ID: f.ID, Action: ActionSkip}
	}
	view := run(t, manager, bus, spec{
		kind:    KindCopy,
		sources: []string{filepath.Join(src, "missing.txt"), filepath.Join(src, "a.txt")},
		dest:    dest,
	})

	require.Equal(t, StateDone, view.State)
	require.Equal(t, "ENOENT", seen.Code)
	require.False(t, seen.Escalatable)
	require.Equal(t, 1, view.Skipped)
	require.Equal(t, "a", read(t, filepath.Join(dest, "a.txt")))
}

func TestDeniedErrorIsEscalatable(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("root ignores mode bits")
	}
	root := t.TempDir()
	src := write(t, filepath.Join(root, "a.txt"), "a")
	dest := filepath.Join(root, "locked")
	require.NoError(t, os.Mkdir(dest, 0o500))

	manager, bus := newHarness(t)
	var seen Failure
	bus.onFailure = func(f Failure) Answer {
		seen = f
		return Answer{ID: f.ID, Action: ActionCancel}
	}
	run(t, manager, bus, spec{kind: KindCopy, sources: []string{src}, dest: dest})

	require.Equal(t, "EACCES", seen.Code)
	require.True(t, seen.Escalatable)
}

func TestCancelMidCopyLeavesNoTempFile(t *testing.T) {
	previous := copyChunk
	copyChunk = 1
	t.Cleanup(func() { copyChunk = previous })

	root := t.TempDir()
	src := write(t, filepath.Join(root, "big.bin"), string(make([]byte, 512<<10)))
	dest := otherDevice(t, root)

	manager, bus := newHarness(t)
	job := manager.enqueue(spec{kind: KindCopy, sources: []string{src}, dest: dest})

	require.Eventually(t, func() bool { return job.view().Progress.BytesDone > 0 }, 10*time.Second, time.Millisecond)
	require.NoError(t, manager.Cancel(job.id))

	view := bus.wait(t, job.id)
	require.True(t, view.Cancelled)
	require.Empty(t, tempsIn(t, dest))
	_, err := os.Lstat(filepath.Join(dest, "big.bin"))
	require.True(t, os.IsNotExist(err))
}

func TestCancelledCrossDeviceMoveReportsTheSplit(t *testing.T) {
	previous := copyChunk
	copyChunk = 1
	t.Cleanup(func() { copyChunk = previous })

	root := t.TempDir()
	src := filepath.Join(root, "src")
	write(t, filepath.Join(src, "a.txt"), "a")
	write(t, filepath.Join(src, "b.bin"), string(make([]byte, 512<<10)))
	dest := otherDevice(t, root)

	manager, bus := newHarness(t)
	job := manager.enqueue(spec{kind: KindMove, sources: []string{src}, dest: dest})

	require.Eventually(t, func() bool { return job.view().Progress.BytesDone > 1 }, 10*time.Second, time.Millisecond)
	require.NoError(t, manager.Cancel(job.id))
	view := bus.wait(t, job.id)

	require.True(t, view.Cancelled)
	require.Equal(t, "a", read(t, filepath.Join(dest, "src", "a.txt")))
	_, err := os.Lstat(filepath.Join(src, "a.txt"))
	require.True(t, os.IsNotExist(err), "moved file stays moved")
	require.FileExists(t, filepath.Join(src, "b.bin"), "unfinished file stays at the source")
	require.Empty(t, tempsIn(t, filepath.Join(dest, "src")))
}

func TestUniquePathNaming(t *testing.T) {
	root := t.TempDir()
	write(t, filepath.Join(root, "a.txt"), "")
	require.Equal(t, filepath.Join(root, "a (copy).txt"), uniquePath(filepath.Join(root, "a.txt")))

	write(t, filepath.Join(root, "a (copy).txt"), "")
	require.Equal(t, filepath.Join(root, "a (copy 2).txt"), uniquePath(filepath.Join(root, "a.txt")))

	write(t, filepath.Join(root, ".bashrc"), "")
	require.Equal(t, filepath.Join(root, ".bashrc (copy)"), uniquePath(filepath.Join(root, ".bashrc")))

	write(t, filepath.Join(root, "notes"), "")
	require.Equal(t, filepath.Join(root, "notes (copy)"), uniquePath(filepath.Join(root, "notes")))
}

func TestJobsLeaveNoGoroutinesBehind(t *testing.T) {
	root := t.TempDir()
	dest := filepath.Join(root, "dest")
	require.NoError(t, os.Mkdir(dest, 0o755))

	baseline := runtime.NumGoroutine()
	manager, bus := newHarness(t)

	for i := range 5 {
		name := filepath.Join(root, "src", "file"+string(rune('a'+i))+".txt")
		write(t, name, "body")
		run(t, manager, bus, spec{kind: KindCopy, sources: []string{name}, dest: dest})
	}

	settled := baseline
	for range 100 {
		settled = runtime.NumGoroutine()
		if settled <= baseline {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	require.LessOrEqual(t, settled, baseline)
}
