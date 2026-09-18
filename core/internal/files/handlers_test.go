package files

import (
	"bufio"
	"context"
	"encoding/json"
	"net"
	"os"
	"path/filepath"
	"testing"

	"github.com/AvengeMedia/dankgo/ipc"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func handleAndRead(t *testing.T, req ipc.Request) map[string]any {
	t.Helper()
	client, srv := net.Pipe()
	t.Cleanup(func() {
		_ = client.Close()
		_ = srv.Close()
	})

	go Handle(context.Background(), ipc.NewConnWriter(srv), req)

	line, err := bufio.NewReader(client).ReadBytes('\n')
	require.NoError(t, err)
	var out map[string]any
	require.NoError(t, json.Unmarshal(line, &out))
	return out
}

func TestHandleListReturnsEntries(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(root, "a.txt"), []byte("hi"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(root, ".secret"), nil, 0o644))

	out := handleAndRead(t, ipc.Request{ID: 7, Method: "files.list", Params: map[string]any{"path": root}})

	assert.Equal(t, float64(7), out["id"])
	result, ok := out["result"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, root, result["path"])
	entries, ok := result["entries"].([]any)
	require.True(t, ok)
	require.Len(t, entries, 1)
	entry := entries[0].(map[string]any)
	assert.Equal(t, "a.txt", entry["name"])
	assert.Equal(t, float64(2), entry["size"])
	assert.Equal(t, false, entry["isDir"])
	assert.Contains(t, entry, "symlinkTarget")
	assert.Contains(t, entry, "mtimeMs")
}

func TestHandleListIncludeHidden(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(root, ".secret"), nil, 0o644))

	out := handleAndRead(t, ipc.Request{ID: 1, Method: "files.list", Params: map[string]any{"path": root, "includeHidden": "true"}})

	result := out["result"].(map[string]any)
	assert.Len(t, result["entries"], 1)
}

func TestHandleListErrorsCarryCode(t *testing.T) {
	out := handleAndRead(t, ipc.Request{ID: 2, Method: "files.list", Params: map[string]any{"path": filepath.Join(t.TempDir(), "gone")}})
	assert.Equal(t, "ENOENT", out["code"])
	assert.Nil(t, out["result"])
	assert.NotEmpty(t, out["error"])

	out = handleAndRead(t, ipc.Request{ID: 3, Method: "files.list"})
	assert.Equal(t, "EINVAL", out["code"])

	out = handleAndRead(t, ipc.Request{ID: 4, Method: "files.nothing"})
	assert.Equal(t, "NOTSUPPORTED", out["code"])
}
