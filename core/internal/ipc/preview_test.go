package ipc

import (
	"image"
	"image/gif"
	"image/jpeg"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"syscall"
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

func TestImageSizeReadsTheHeader(t *testing.T) {
	dir := t.TempDir()
	canvas := image.NewRGBA(image.Rect(0, 0, 640, 360))
	encoders := map[string]func(*os.File) error{
		"png":  func(f *os.File) error { return png.Encode(f, canvas) },
		"jpeg": func(f *os.File) error { return jpeg.Encode(f, canvas, nil) },
		"gif":  func(f *os.File) error { return gif.Encode(f, canvas, nil) },
	}
	id := 10
	for format, encode := range encoders {
		path := filepath.Join(dir, "shot."+format)
		file, err := os.Create(path)
		require.NoError(t, err)
		require.NoError(t, encode(file))
		require.NoError(t, file.Close())

		id++
		out := routeAndRead(t, Request{ID: id, Method: "files.imageSize", Params: map[string]any{"path": path}}, Deps{})
		result, ok := out["result"].(map[string]any)
		require.True(t, ok, format)
		assert.EqualValues(t, 640, result["width"], format)
		assert.EqualValues(t, 360, result["height"], format)
		assert.Equal(t, format, result["format"])
	}
}

func TestImageSizeRejectsWhatIsNotAnImage(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "notes.txt")
	require.NoError(t, os.WriteFile(path, []byte("plain text"), 0o644))

	out := routeAndRead(t, Request{ID: 20, Method: "files.imageSize", Params: map[string]any{"path": path}}, Deps{})
	assert.Contains(t, out["error"], "unknown format")

	out = routeAndRead(t, Request{ID: 21, Method: "files.imageSize", Params: map[string]any{"path": dir}}, Deps{})
	assert.NotEmpty(t, out["error"])

	out = routeAndRead(t, Request{ID: 22, Method: "files.imageSize", Params: map[string]any{"path": "shot.png"}}, Deps{})
	assert.Contains(t, out["error"], "absolute")
}

func TestPreviewRefusesFifosWithoutBlocking(t *testing.T) {
	path := filepath.Join(t.TempDir(), "stream.log")
	require.NoError(t, syscall.Mkfifo(path, 0o600))

	out := routeAndRead(t, Request{ID: 30, Method: "files.preview", Params: map[string]any{"path": path}}, Deps{})
	assert.Contains(t, out["error"], "regular file")

	out = routeAndRead(t, Request{ID: 31, Method: "files.imageSize", Params: map[string]any{"path": path}}, Deps{})
	assert.Contains(t, out["error"], "regular file")
}
