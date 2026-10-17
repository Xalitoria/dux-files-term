package fileops

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFinishedJobsReportMovedFolders(t *testing.T) {
	root := t.TempDir()
	write(t, filepath.Join(root, "photos", "a.jpg"), "a")
	write(t, filepath.Join(root, "note.txt"), "n")
	require.NoError(t, os.MkdirAll(filepath.Join(root, "archive"), 0o755))
	manager, bus := newHarness(t)

	renamed := run(t, manager, bus, spec{kind: KindRename, scope: "w1", sources: []string{filepath.Join(root, "photos")}, name: "pictures"})
	require.Equal(t, StateDone, renamed.State)
	require.Equal(t, []Moved{{From: filepath.Join(root, "photos"), To: filepath.Join(root, "pictures")}}, renamed.Moved)

	moved := run(t, manager, bus, spec{kind: KindMove, scope: "w1", sources: []string{filepath.Join(root, "pictures"), filepath.Join(root, "note.txt")}, dest: filepath.Join(root, "archive")})
	require.Equal(t, StateDone, moved.State)
	require.Equal(t, []Moved{{From: filepath.Join(root, "pictures"), To: filepath.Join(root, "archive", "pictures")}}, moved.Moved)

	undone := undo(t, manager, bus, "w1")
	require.Equal(t, []Moved{{From: filepath.Join(root, "archive", "pictures"), To: filepath.Join(root, "pictures")}}, undone.Moved)

	copied := run(t, manager, bus, spec{kind: KindCopy, scope: "w1", sources: []string{filepath.Join(root, "pictures")}, dest: filepath.Join(root, "archive")})
	require.Equal(t, StateDone, copied.State)
	require.Empty(t, copied.Moved)

	grouped := run(t, manager, bus, spec{kind: KindNewFolderWith, scope: "w1", sources: []string{filepath.Join(root, "pictures"), filepath.Join(root, "note.txt")}, dest: root, name: "Group"})
	require.Equal(t, StateDone, grouped.State)
	require.Equal(t, []Moved{{From: filepath.Join(root, "pictures"), To: filepath.Join(root, "Group", "pictures")}}, grouped.Moved)
}
