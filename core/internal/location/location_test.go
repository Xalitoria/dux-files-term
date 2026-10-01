package location

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func fixture(t *testing.T) (string, Env) {
	t.Helper()
	root := t.TempDir()
	for _, dir := range []string{"docs", "docs/sub", "data/Trash/files/old", "with space"} {
		require.NoError(t, os.MkdirAll(filepath.Join(root, dir), 0o755))
	}
	for _, file := range []string{"docs/a.txt", "docs/b.txt", "with space/c.txt"} {
		require.NoError(t, os.WriteFile(filepath.Join(root, file), nil, 0o644))
	}
	return root, HostEnv(filepath.Join(root, "data"))
}

func TestParse(t *testing.T) {
	root, env := fixture(t)
	trash := filepath.Join(root, "data", "Trash", "files")

	tests := []struct {
		raw  string
		want Location
	}{
		{"/tmp/../etc/", Location{Kind: KindFolder, Path: "/etc", Select: []string{}}},
		{"file:///home/me/My%20Files", Location{Kind: KindFolder, Path: "/home/me/My Files", Select: []string{}}},
		{"file://localhost/srv", Location{Kind: KindFolder, Path: "/srv", Select: []string{}}},
		{"FILE:///srv", Location{Kind: KindFolder, Path: "/srv", Select: []string{}}},
		{"trash:///", Location{Kind: KindFolder, Path: trash, Select: []string{}}},
		{"trash:///old", Location{Kind: KindFolder, Path: filepath.Join(trash, "old"), Select: []string{}}},
		{"trash:///../../etc", Location{Kind: KindFolder, Path: filepath.Join(trash, "etc"), Select: []string{}}},
		{"computer:///", Location{Kind: KindFolder, Path: "/", Select: []string{}}},
		{"admin:///etc/fstab", Location{Kind: KindFolder, Path: "/etc/fstab", Select: []string{}}},
		{"network:///", Location{Kind: KindServers, Select: []string{}}},
		{"sftp://me@host:2222/srv/www", Location{Kind: KindRemote, URI: "sftp://me@host:2222/srv/www", Select: []string{}}},
		{"smb://nas/share/dir", Location{Kind: KindRemote, URI: "smb://nas/share/dir", Select: []string{}}},
	}
	for _, tt := range tests {
		t.Run(tt.raw, func(t *testing.T) {
			got, err := Parse(tt.raw, env)
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestParseRefuses(t *testing.T) {
	_, env := fixture(t)
	for _, raw := range []string{"", "docs", "file://otherhost/srv", "file:relative", "http://example.com/", "recent:///", "sftp:///nohost"} {
		t.Run(raw, func(t *testing.T) {
			_, err := Parse(raw, env)
			assert.Error(t, err)
		})
	}
}

func TestAbsolute(t *testing.T) {
	assert.Equal(t, "/work/docs", Absolute("docs", "/work"))
	assert.Equal(t, "/work/a b", Absolute("./a b", "/work"))
	assert.Equal(t, "/etc", Absolute("/etc/", "/work"))
	assert.Equal(t, "trash:///", Absolute("trash:///", "/work"))
	assert.Equal(t, "sftp://host/x", Absolute("sftp://host/x", "/work"))
	assert.Equal(t, "/work/notes:draft", Absolute("notes:draft", "/work"))
	assert.Equal(t, "/work/computer:notes", Absolute("computer:notes", "/work"))
	assert.Equal(t, "mtp://phone/", Absolute("mtp://phone/", "/work"), "an unknown scheme stays a URI so Parse can refuse it")
}

func TestResolve(t *testing.T) {
	root, env := fixture(t)
	docs := filepath.Join(root, "docs")

	t.Run("folders open as themselves, once", func(t *testing.T) {
		got, err := Resolve([]string{docs, "file://" + docs, filepath.Join(root, "with space")}, nil, env)
		require.NoError(t, err)
		assert.Equal(t, []Location{
			{Kind: KindFolder, Path: docs, Select: []string{}},
			{Kind: KindFolder, Path: filepath.Join(root, "with space"), Select: []string{}},
		}, got)
	})

	t.Run("a file opens its folder selected", func(t *testing.T) {
		got, err := Resolve([]string{filepath.Join(docs, "a.txt")}, nil, env)
		require.NoError(t, err)
		assert.Equal(t, []Location{{Kind: KindFolder, Path: docs, Select: []string{filepath.Join(docs, "a.txt")}}}, got)
	})

	t.Run("selections group by folder in first-seen order", func(t *testing.T) {
		got, err := Resolve(nil, []string{
			filepath.Join(docs, "b.txt"),
			"file://" + filepath.Join(root, "with%20space", "c.txt"),
			filepath.Join(docs, "sub"),
			filepath.Join(docs, "b.txt"),
		}, env)
		require.NoError(t, err)
		assert.Equal(t, []Location{
			{Kind: KindFolder, Path: docs, Select: []string{filepath.Join(docs, "b.txt"), filepath.Join(docs, "sub")}},
			{Kind: KindFolder, Path: filepath.Join(root, "with space"), Select: []string{filepath.Join(root, "with space", "c.txt")}},
		}, got)
	})

	t.Run("selecting root opens root", func(t *testing.T) {
		got, err := Resolve(nil, []string{"/"}, env)
		require.NoError(t, err)
		assert.Equal(t, []Location{{Kind: KindFolder, Path: "/", Select: []string{}}}, got)
	})

	t.Run("remote and server locations pass through", func(t *testing.T) {
		got, err := Resolve([]string{"sftp://host/srv", "network:///"}, nil, env)
		require.NoError(t, err)
		assert.Equal(t, []Location{
			{Kind: KindRemote, URI: "sftp://host/srv", Select: []string{}},
			{Kind: KindServers, Select: []string{}},
		}, got)
	})

	t.Run("missing paths fail", func(t *testing.T) {
		_, err := Resolve([]string{filepath.Join(root, "nope")}, nil, env)
		assert.ErrorContains(t, err, "does not exist")
		_, err = Resolve(nil, []string{filepath.Join(root, "nope")}, env)
		assert.ErrorContains(t, err, "does not exist")
	})

	t.Run("names keep their trailing spaces", func(t *testing.T) {
		spaced := filepath.Join(docs, "spaced ")
		require.NoError(t, os.WriteFile(spaced, nil, 0o644))
		got, err := Resolve([]string{spaced}, nil, env)
		require.NoError(t, err)
		assert.Equal(t, []string{spaced}, got[0].Select)
	})

	t.Run("a dangling symlink is selected in its folder", func(t *testing.T) {
		link := filepath.Join(docs, "dangling")
		require.NoError(t, os.Symlink(filepath.Join(root, "gone"), link))
		got, err := Resolve([]string{link}, []string{link}, env)
		require.NoError(t, err)
		assert.Equal(t, []Location{{Kind: KindFolder, Path: docs, Select: []string{link}}}, got)
		paths, err := LocalPaths([]string{link}, env)
		require.NoError(t, err)
		assert.Equal(t, []string{link}, paths)
	})

	t.Run("remote locations cannot be selected", func(t *testing.T) {
		_, err := Resolve(nil, []string{"sftp://host/srv/file"}, env)
		assert.Error(t, err)
	})
}

func TestLocalPaths(t *testing.T) {
	root, env := fixture(t)
	docs := filepath.Join(root, "docs")

	got, err := LocalPaths([]string{docs + "/a.txt", "file://" + filepath.Join(root, "with%20space")}, env)
	require.NoError(t, err)
	assert.Equal(t, []string{filepath.Join(docs, "a.txt"), filepath.Join(root, "with space")}, got)

	_, err = LocalPaths([]string{"sftp://host/srv"}, env)
	assert.ErrorContains(t, err, "not a local path")
	_, err = LocalPaths([]string{filepath.Join(root, "nope")}, env)
	assert.ErrorContains(t, err, "does not exist")
}
