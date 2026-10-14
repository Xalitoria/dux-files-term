package ipc

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLocationResolve(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "notes.txt")
	require.NoError(t, os.WriteFile(file, nil, 0o644))
	t.Setenv("HOME", dir)

	out := routeAndRead(t, Request{ID: 1, Method: "location.resolve", Params: map[string]any{"text": "notes.txt", "cwd": dir}}, Deps{})
	assert.Equal(t, map[string]any{"location": map[string]any{"kind": "folder", "path": dir, "select": []any{file}}}, out["result"])

	out = routeAndRead(t, Request{ID: 2, Method: "location.resolve", Params: map[string]any{"text": "~/notes.txt"}}, Deps{})
	assert.Equal(t, map[string]any{"location": map[string]any{"kind": "folder", "path": dir, "select": []any{file}}}, out["result"])

	out = routeAndRead(t, Request{ID: 3, Method: "location.resolve", Params: map[string]any{"text": "sftp://me:pw@host/srv"}}, Deps{})
	assert.Equal(t, map[string]any{"location": map[string]any{"kind": "remote", "uri": "sftp://me@host/srv", "password": "pw", "select": []any{}}}, out["result"])
}

func TestLocationResolveErrorsCarryACode(t *testing.T) {
	dir := t.TempDir()

	out := routeAndRead(t, Request{ID: 1, Method: "location.resolve", Params: map[string]any{"text": "missing", "cwd": dir}}, Deps{})
	assert.Equal(t, "ENOENT", out["code"])
	assert.Contains(t, out["error"], filepath.Join(dir, "missing"))

	out = routeAndRead(t, Request{ID: 2, Method: "location.resolve", Params: map[string]any{"text": "~root/x", "cwd": dir}}, Deps{})
	assert.Equal(t, "EOTHERHOME", out["code"])
}

func TestLocationMissingReportsWhatIsNoLongerAFolder(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "file")
	link := filepath.Join(dir, "dangling")
	unplugged := filepath.Join(dir, "media", "stick", "photos")
	require.NoError(t, os.WriteFile(file, nil, 0o644))
	require.NoError(t, os.Symlink(filepath.Join(dir, "gone"), link))

	out := routeAndRead(t, Request{ID: 1, Method: "location.missing", Params: map[string]any{"paths": []any{dir, file, link, filepath.Join(dir, "gone"), unplugged, "relative", "sftp://host/"}}}, Deps{})
	assert.Equal(t, map[string]any{"missing": []any{file, link, filepath.Join(dir, "gone")}}, out["result"])
}

func TestMissingFoldersSkipsNetworkMounts(t *testing.T) {
	mountinfo := `22 1 259:2 / / rw,relatime shared:1 - ext4 /dev/nvme0n1p2 rw
98 22 0:55 / /mnt/nas\040share rw,relatime shared:60 - nfs4 nas:/export rw
99 22 0:56 / /run/user/1000/gvfs rw,nosuid shared:61 - fuse.gvfsd-fuse gvfsd-fuse rw
100 22 0:57 / /home/me/remote rw - fuse.sshfs me@host: rw
`
	mounts := networkMounts(mountinfo)
	assert.Equal(t, []string{"/mnt/nas share", "/run/user/1000/gvfs", "/home/me/remote"}, mounts)

	dir := t.TempDir()
	gone := filepath.Join(dir, "gone")
	assert.Equal(t, []string{gone}, missingFolders([]string{gone, "/mnt/nas share/stale", "/home/me/remote/x", "/run/user/1000/gvfs/sftp:host=h"}, mounts))
	assert.Empty(t, missingFolders([]string{gone}, []string{dir}))
}
