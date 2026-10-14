package ipc

import (
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/AvengeMedia/dankgo/files"
	"github.com/AvengeMedia/dankgo/trash"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func writeTrashed(t *testing.T, trashDir, name, stored, deleted string, dir bool) trash.Entry {
	t.Helper()
	filesPath := filepath.Join(trashDir, "files", name)
	infoPath := filepath.Join(trashDir, "info", name+".trashinfo")
	require.NoError(t, os.MkdirAll(filepath.Dir(infoPath), 0o700))
	switch {
	case dir:
		require.NoError(t, os.MkdirAll(filepath.Join(filesPath, "inside"), 0o700))
	default:
		require.NoError(t, os.MkdirAll(filepath.Dir(filesPath), 0o700))
		require.NoError(t, os.WriteFile(filesPath, []byte("body"), 0o600))
	}
	require.NoError(t, os.WriteFile(infoPath, []byte("[Trash Info]\nPath="+stored+"\nDeletionDate="+deleted+"\n"), 0o600))
	return trash.Entry{Name: name, OriginalPath: stored, DeletionDate: deleted, TrashDir: trashDir, FilesPath: filesPath, InfoPath: infoPath}
}

func TestTrashEntriesAcrossHomeAndVolumeTrash(t *testing.T) {
	root := t.TempDir()
	home := filepath.Join(root, "data", "Trash")
	volume := filepath.Join(root, "media", "disk")
	volumeTrash := filepath.Join(volume, ".Trash-"+strconv.Itoa(os.Getuid()))

	listed := []trash.Entry{
		writeTrashed(t, home, "notes.2.txt", filepath.Join(root, "docs", "notes.txt"), "2001-02-03T09:00:00", false),
		writeTrashed(t, volumeTrash, "photos", "pictures/photos", "2001-02-03T11:30:00", true),
		writeTrashed(t, home, "odd", filepath.Join(root, "odd"), "not a date", false),
		{Name: "orphan", OriginalPath: filepath.Join(root, "orphan"), DeletionDate: "2001-02-03T12:00:00", TrashDir: home, FilesPath: filepath.Join(home, "files", "orphan")},
	}
	listTrash = func() ([]trash.Entry, error) { return listed, nil }
	t.Cleanup(func() { listTrash = trash.List })

	out := routeAndRead(t, Request{ID: 7, Method: "trash.entries"}, Deps{Files: files.NewService(nil, t.TempDir(), nil)})
	result, ok := out["result"].(map[string]any)
	require.True(t, ok, out)
	items, ok := result["entries"].([]any)
	require.True(t, ok)
	require.Len(t, items, 3)

	odd := items[0].(map[string]any)
	assert.Equal(t, "odd", odd["name"])
	info, err := os.Lstat(listed[2].InfoPath)
	require.NoError(t, err)
	assert.Equal(t, float64(info.ModTime().UnixMilli()), odd["deletedMs"])

	photos := items[1].(map[string]any)
	assert.Equal(t, "photos", photos["name"])
	assert.Equal(t, filepath.Join(volume, "pictures", "photos"), photos["originalPath"])
	assert.Equal(t, volumeTrash, photos["trashDir"])
	assert.Equal(t, "photos", photos["trashName"])
	assert.Equal(t, filepath.Join(volumeTrash, "files", "photos"), photos["path"])
	assert.Equal(t, true, photos["isDir"])
	deleted := time.Date(2001, 2, 3, 11, 30, 0, 0, time.Local).UnixMilli()
	assert.Equal(t, float64(deleted), photos["deletedMs"])

	notes := items[2].(map[string]any)
	assert.Equal(t, "notes.txt", notes["name"])
	assert.Equal(t, "notes.2.txt", notes["trashName"])
	assert.Equal(t, filepath.Join(root, "docs", "notes.txt"), notes["originalPath"])
	assert.Equal(t, "text/plain", notes["mime"])
}

func TestTrashEntriesNeedsTheFileService(t *testing.T) {
	out := routeAndRead(t, Request{ID: 8, Method: "trash.entries"}, Deps{})
	assert.Equal(t, "file service unavailable", out["error"])
}

func TestTrashCountStampFollowsTheHomeTrash(t *testing.T) {
	dataHome := t.TempDir()
	t.Setenv("XDG_DATA_HOME", dataHome)
	info := filepath.Join(dataHome, "Trash", "info")
	require.NoError(t, os.MkdirAll(info, 0o700))

	require.NoError(t, os.Chtimes(info, time.Unix(1000, 0), time.Unix(1000, 0)))
	before := trashStamp(1)
	require.NoError(t, os.Chtimes(info, time.Unix(2000, 0), time.Unix(2000, 0)))

	assert.NotEqual(t, before, trashStamp(1), "a swap that keeps the count still changes the stamp")
	assert.NotEqual(t, trashStamp(1), trashStamp(2))
}
