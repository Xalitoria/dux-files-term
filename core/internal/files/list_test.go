package files

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func buildTree(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	require.NoError(t, os.Mkdir(filepath.Join(root, "zeta"), 0o755))
	require.NoError(t, os.Mkdir(filepath.Join(root, "Alpha"), 0o755))
	require.NoError(t, os.Mkdir(filepath.Join(root, ".hiddendir"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(root, "b.txt"), []byte("12345"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(root, "A.txt"), []byte("1"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(root, ".hidden"), nil, 0o644))
	require.NoError(t, os.Symlink(filepath.Join(root, "b.txt"), filepath.Join(root, "link")))
	require.NoError(t, os.Symlink(filepath.Join(root, "zeta"), filepath.Join(root, "dirlink")))
	require.NoError(t, os.Symlink(filepath.Join(root, "missing"), filepath.Join(root, "dangling")))
	return root
}

func names(entries []Entry) []string {
	out := make([]string, len(entries))
	for i, e := range entries {
		out[i] = e.Name
	}
	return out
}

func byName(t *testing.T, entries []Entry, name string) Entry {
	t.Helper()
	for _, e := range entries {
		if e.Name == name {
			return e
		}
	}
	t.Fatalf("entry %q missing", name)
	return Entry{}
}

func TestListSortsDirectoriesFirstCaseInsensitive(t *testing.T) {
	root := buildTree(t)
	entries, err := List(root, ListOptions{})
	require.NoError(t, err)
	assert.Equal(t, []string{"Alpha", "dirlink", "zeta", "A.txt", "b.txt", "dangling", "link"}, names(entries))
}

func TestListIncludesHiddenOnRequest(t *testing.T) {
	root := buildTree(t)
	entries, err := List(root, ListOptions{IncludeHidden: true})
	require.NoError(t, err)
	assert.Equal(t, []string{".hiddendir", "Alpha", "dirlink", "zeta", ".hidden", "A.txt", "b.txt", "dangling", "link"}, names(entries))
	assert.True(t, byName(t, entries, ".hidden").Hidden)
	assert.False(t, byName(t, entries, "A.txt").Hidden)
}

func TestListEntryFields(t *testing.T) {
	root := buildTree(t)
	stamp := time.Date(2024, 3, 4, 5, 6, 7, 0, time.UTC)
	require.NoError(t, os.Chtimes(filepath.Join(root, "b.txt"), stamp, stamp))
	entries, err := List(root, ListOptions{})
	require.NoError(t, err)

	file := byName(t, entries, "b.txt")
	assert.Equal(t, filepath.Join(root, "b.txt"), file.Path)
	assert.False(t, file.IsDir)
	assert.False(t, file.IsSymlink)
	assert.Equal(t, int64(5), file.Size)
	assert.Equal(t, stamp.UnixMilli(), file.MtimeMs)
	assert.Equal(t, "-rw-r--r--", file.Mode)

	dir := byName(t, entries, "zeta")
	assert.True(t, dir.IsDir)
	assert.Equal(t, "drwxr-xr-x", dir.Mode)
}

func TestListSymlinksFollowTargetType(t *testing.T) {
	root := buildTree(t)
	entries, err := List(root, ListOptions{})
	require.NoError(t, err)

	link := byName(t, entries, "link")
	assert.True(t, link.IsSymlink)
	assert.False(t, link.IsDir)
	assert.Equal(t, filepath.Join(root, "b.txt"), link.SymlinkTarget)
	assert.Equal(t, int64(5), link.Size)

	dirlink := byName(t, entries, "dirlink")
	assert.True(t, dirlink.IsSymlink)
	assert.True(t, dirlink.IsDir)

	dangling := byName(t, entries, "dangling")
	assert.True(t, dangling.IsSymlink)
	assert.False(t, dangling.IsDir)
	assert.Equal(t, filepath.Join(root, "missing"), dangling.SymlinkTarget)
}

func TestListEmptyDirectory(t *testing.T) {
	entries, err := List(t.TempDir(), ListOptions{})
	require.NoError(t, err)
	assert.Empty(t, entries)
	assert.NotNil(t, entries)
}

func TestListErrorCodes(t *testing.T) {
	root := buildTree(t)

	_, err := List("relative/path", ListOptions{})
	assert.Equal(t, CodeInvalid, CodeOf(err))

	_, err = List(filepath.Join(root, "nope"), ListOptions{})
	assert.Equal(t, CodeNotFound, CodeOf(err))

	_, err = List(filepath.Join(root, "b.txt"), ListOptions{})
	assert.Equal(t, CodeNotDir, CodeOf(err))

	if os.Geteuid() == 0 {
		t.Skip("root bypasses permission bits")
	}
	locked := filepath.Join(root, "locked")
	require.NoError(t, os.Mkdir(locked, 0o000))
	t.Cleanup(func() { _ = os.Chmod(locked, 0o755) })
	_, err = List(locked, ListOptions{})
	assert.Equal(t, CodeAccess, CodeOf(err))
}
