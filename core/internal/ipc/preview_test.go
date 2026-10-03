package ipc

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPreviewReturnsTextHead(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "notes.txt")
	require.NoError(t, os.WriteFile(path, []byte("one\ntwo\nthree"), 0o644))

	out := routeAndRead(t, Request{ID: 1, Method: "files.preview", Params: map[string]any{"path": path}}, Deps{})
	result := out["result"].(map[string]any)
	assert.Equal(t, "one\ntwo\nthree", result["text"])
	assert.Equal(t, false, result["binary"])
	assert.Equal(t, false, result["truncated"])
	assert.EqualValues(t, 3, result["lines"])
	assert.EqualValues(t, 13, result["size"])
}

func TestPreviewFlagsBinary(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "blob.bin")
	require.NoError(t, os.WriteFile(path, []byte{0x89, 'P', 'N', 'G', 0, 1, 2}, 0o644))

	out := routeAndRead(t, Request{ID: 2, Method: "files.preview", Params: map[string]any{"path": path}}, Deps{})
	result := out["result"].(map[string]any)
	assert.Equal(t, true, result["binary"])
	assert.Equal(t, "", result["text"])
}

func TestPreviewTruncatesOnRuneBoundary(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "long.txt")
	content := strings.Repeat("é", 100)
	require.NoError(t, os.WriteFile(path, []byte(content), 0o644))

	out := routeAndRead(t, Request{ID: 3, Method: "files.preview", Params: map[string]any{"path": path, "maxBytes": 11}}, Deps{})
	result := out["result"].(map[string]any)
	assert.Equal(t, true, result["truncated"])
	assert.Equal(t, strings.Repeat("é", 5), result["text"])
	assert.EqualValues(t, 200, result["size"])
}

func TestPreviewRejectsDirectoriesAndRelativePaths(t *testing.T) {
	dir := t.TempDir()

	out := routeAndRead(t, Request{ID: 4, Method: "files.preview", Params: map[string]any{"path": dir}}, Deps{})
	assert.Contains(t, out["error"], "directory")

	out = routeAndRead(t, Request{ID: 5, Method: "files.preview", Params: map[string]any{"path": "notes.txt"}}, Deps{})
	assert.Contains(t, out["error"], "absolute")
}
