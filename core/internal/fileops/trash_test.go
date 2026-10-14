package fileops

import (
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/AvengeMedia/dankgo/trash"
	"github.com/stretchr/testify/require"
)

func trashHome(t *testing.T) string {
	t.Helper()
	dataHome := t.TempDir()
	t.Setenv("XDG_DATA_HOME", dataHome)
	home := filepath.Join(dataHome, "Trash")
	listTrash = func() ([]trash.Entry, error) {
		all, err := trash.List()
		return slices.DeleteFunc(all, func(e trash.Entry) bool { return e.TrashDir != home }), err
	}
	t.Cleanup(func() { listTrash = trash.List })
	return home
}

func infoNames(t *testing.T, trashDir string) []string {
	t.Helper()
	entries, err := os.ReadDir(filepath.Join(trashDir, "info"))
	require.NoError(t, err)
	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		names = append(names, entry.Name())
	}
	return names
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

	require.Empty(t, infoNames(t, trashDir))
	require.NoFileExists(t, filepath.Join(trashDir, "directorysizes"))
}

func TestDeleteFromTheTrashDropsItsInfoAndDirectorySize(t *testing.T) {
	trashDir := trashHome(t)
	root := t.TempDir()
	write(t, filepath.Join(root, "docs", "a.txt"), "a")
	write(t, filepath.Join(root, "notes.txt"), "n")

	manager, bus := newHarness(t)
	run(t, manager, bus, spec{kind: KindTrash, sources: []string{filepath.Join(root, "docs"), filepath.Join(root, "notes.txt")}})
	require.Contains(t, read(t, filepath.Join(trashDir, "directorysizes")), " docs")

	view := run(t, manager, bus, spec{kind: KindDelete, sources: []string{filepath.Join(trashDir, "files", "docs")}})
	require.Equal(t, StateDone, view.State)
	require.NoDirExists(t, filepath.Join(trashDir, "files", "docs"))
	require.NoFileExists(t, filepath.Join(trashDir, "info", "docs.trashinfo"))
	require.NoFileExists(t, filepath.Join(trashDir, "directorysizes"))
	require.Equal(t, []string{"notes.txt.trashinfo"}, infoNames(t, trashDir))
}

func TestDeleteOutsideATrashKeepsLookalikeInfo(t *testing.T) {
	trashHome(t)
	root := t.TempDir()
	target := write(t, filepath.Join(root, "files", "a.txt"), "a")
	info := write(t, filepath.Join(root, "info", "a.txt.trashinfo"), "[Trash Info]\n")

	manager, bus := newHarness(t)
	view := run(t, manager, bus, spec{kind: KindDelete, sources: []string{target}})

	require.Equal(t, StateDone, view.State)
	require.NoFileExists(t, target)
	require.FileExists(t, info)
}

func TestRestoreRecreatesAMissingParent(t *testing.T) {
	trashDir := trashHome(t)
	root := t.TempDir()
	src := write(t, filepath.Join(root, "gone", "deeper", "a.txt"), "body")
	blocked := write(t, filepath.Join(root, "blocked", "b.txt"), "kept")

	manager, bus := newHarness(t)
	run(t, manager, bus, spec{kind: KindTrash, sources: []string{src, blocked}})
	require.NoError(t, os.RemoveAll(filepath.Join(root, "gone")))
	require.NoError(t, os.RemoveAll(filepath.Join(root, "blocked")))
	write(t, filepath.Join(root, "blocked"), "a file where the folder was")

	bus.onFailure = func(f Failure) Answer { return Answer{ID: f.ID, Action: ActionSkip} }
	view := run(t, manager, bus, spec{kind: KindRestore, sources: []string{"a.txt", "b.txt"}, trashDir: trashDir})
	require.Equal(t, StateDone, view.State)
	require.Equal(t, 1, view.Skipped)
	require.Equal(t, "body", read(t, src))
	require.Equal(t, []string{"b.txt.trashinfo"}, infoNames(t, trashDir))
}

func TestRestoreRefusesAPathLeavingItsVolume(t *testing.T) {
	trashHome(t)
	volume := t.TempDir()
	outside := filepath.Join(volume, "outside")
	require.NoError(t, os.Mkdir(outside, 0o755))
	disk := filepath.Join(volume, "disk")
	trashDir := filepath.Join(disk, ".Trash-"+strconv.Itoa(os.Getuid()))
	require.NoError(t, os.MkdirAll(disk, 0o755))
	require.NoError(t, os.Symlink(outside, filepath.Join(disk, "link")))

	stored := map[string]string{
		"relative.txt": "../outside/relative.txt",
		"absolute.txt": filepath.Join(outside, "absolute.txt"),
		"linked.txt":   "link/linked.txt",
	}
	for name, path := range stored {
		write(t, filepath.Join(trashDir, "files", name), "body")
		write(t, filepath.Join(trashDir, "info", name+trashInfoExt), "[Trash Info]\nPath="+path+"\nDeletionDate=2026-10-14T10:00:00\n")
	}

	manager, bus := newHarness(t)
	bus.onFailure = func(f Failure) Answer { return Answer{ID: f.ID, Action: ActionSkip} }
	view := run(t, manager, bus, spec{kind: KindRestore, sources: []string{"relative.txt", "absolute.txt", "linked.txt"}, trashDir: trashDir})

	require.Equal(t, StateDone, view.State)
	require.Equal(t, 3, view.Skipped)
	left, err := os.ReadDir(outside)
	require.NoError(t, err)
	require.Empty(t, left)
}

func TestRestorePath(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", "/home/me/.local/share")
	uid := strconv.Itoa(os.Getuid())
	tests := []struct {
		trashDir, stored, want string
		ok                     bool
	}{
		{"/home/me/.local/share/Trash", "/home/me/docs/a.txt", "/home/me/docs/a.txt", true},
		{"/run/media/me/disk/.Trash-" + uid, "photos/a b.jpg", "/run/media/me/disk/photos/a b.jpg", true},
		{"/run/media/me/disk/.Trash/" + uid, "photos/a.jpg", "/run/media/me/disk/photos/a.jpg", true},
		{"/run/media/me/disk/.Trash-" + uid, "/run/media/me/disk/photos/a.jpg", "/run/media/me/disk/photos/a.jpg", true},
		{"/run/media/me/disk/.Trash-" + uid, "../other/a.jpg", "/run/media/me/other/a.jpg", false},
		{"/run/media/me/disk/.Trash-" + uid, "/home/me/.config/autostart/x.desktop", "/home/me/.config/autostart/x.desktop", false},
		{"/run/media/me/disk/.Trash-" + uid, "", "", false},
	}
	for _, tt := range tests {
		t.Run(tt.stored, func(t *testing.T) {
			got, ok := RestorePath(tt.trashDir, tt.stored)
			require.Equal(t, tt.want, got)
			require.Equal(t, tt.ok, ok)
		})
	}
}
