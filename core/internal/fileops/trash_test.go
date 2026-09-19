package fileops

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/AvengeMedia/dankgo/trash"
	"github.com/stretchr/testify/require"
)

func trashHome(t *testing.T) string {
	t.Helper()
	dataHome := t.TempDir()
	t.Setenv("XDG_DATA_HOME", dataHome)
	return filepath.Join(dataHome, "Trash")
}

func TestTrashMovesIntoHomeTrash(t *testing.T) {
	trashDir := trashHome(t)
	root := t.TempDir()
	src := write(t, filepath.Join(root, "a file.txt"), "body")

	manager, bus := newHarness(t)
	view := run(t, manager, bus, spec{kind: KindTrash, sources: []string{src}})

	require.Equal(t, StateDone, view.State)
	require.NoFileExists(t, src)
	require.Equal(t, "body", read(t, filepath.Join(trashDir, "files", "a file.txt")))

	info := read(t, filepath.Join(trashDir, "info", "a file.txt.trashinfo"))
	require.Contains(t, info, "[Trash Info]")
	require.Contains(t, info, "Path="+strings.ReplaceAll(src, " ", "%20"))
}

func TestTrashWritesDirectorySizes(t *testing.T) {
	trashDir := trashHome(t)
	root := t.TempDir()
	dir := filepath.Join(root, "álbum")
	write(t, filepath.Join(dir, "a.txt"), "12345")
	write(t, filepath.Join(dir, "b.txt"), "67890")

	manager, bus := newHarness(t)
	run(t, manager, bus, spec{kind: KindTrash, sources: []string{dir}})

	line := strings.TrimSpace(read(t, filepath.Join(trashDir, "directorysizes")))
	fields := strings.SplitN(line, " ", 3)
	require.Len(t, fields, 3)
	require.Equal(t, "10", fields[0])
	require.Equal(t, "%C3%A1lbum", fields[2])
	require.Equal(t, "álbum", directorySizeName(line))
}

func TestRestoreReturnsFileAndDropsDirectorySize(t *testing.T) {
	trashDir := trashHome(t)
	root := t.TempDir()
	dir := filepath.Join(root, "docs")
	write(t, filepath.Join(dir, "a.txt"), "a")

	manager, bus := newHarness(t)
	run(t, manager, bus, spec{kind: KindTrash, sources: []string{dir}})
	require.FileExists(t, filepath.Join(trashDir, "directorysizes"))

	view := run(t, manager, bus, spec{kind: KindRestore, sources: []string{"docs"}, trashDir: trashDir})
	require.Equal(t, StateDone, view.State)
	require.Equal(t, "a", read(t, filepath.Join(dir, "a.txt")))
	require.NoFileExists(t, filepath.Join(trashDir, "info", "docs.trashinfo"))
	require.NoFileExists(t, filepath.Join(trashDir, "directorysizes"))
}

func TestRestoreConflictKeepsBoth(t *testing.T) {
	trashDir := trashHome(t)
	root := t.TempDir()
	src := write(t, filepath.Join(root, "a.txt"), "trashed")

	manager, bus := newHarness(t)
	run(t, manager, bus, spec{kind: KindTrash, sources: []string{src}})
	write(t, src, "recreated")

	bus.onConflict = func(c Conflict) Answer { return Answer{ID: c.ID, Action: ActionKeepBoth} }
	view := run(t, manager, bus, spec{kind: KindRestore, sources: []string{"a.txt"}, trashDir: trashDir})

	require.Equal(t, StateDone, view.State)
	require.Equal(t, "recreated", read(t, src))
	require.Equal(t, "trashed", read(t, filepath.Join(root, "a (copy).txt")))
}

func TestEmptyTrashRemovesEverything(t *testing.T) {
	trashDir := trashHome(t)
	root := t.TempDir()
	write(t, filepath.Join(root, "a.txt"), "a")
	write(t, filepath.Join(root, "dir", "b.txt"), "b")

	manager, bus := newHarness(t)
	run(t, manager, bus, spec{kind: KindTrash, sources: []string{
		filepath.Join(root, "a.txt"),
		filepath.Join(root, "dir"),
	}})

	view := run(t, manager, bus, spec{kind: KindEmptyTrash})
	require.Equal(t, StateDone, view.State)
	require.Equal(t, 2, view.Progress.FilesDone)

	entries, err := trash.List()
	require.NoError(t, err)
	require.Empty(t, entries)
	require.NoFileExists(t, filepath.Join(trashDir, "directorysizes"))
}
