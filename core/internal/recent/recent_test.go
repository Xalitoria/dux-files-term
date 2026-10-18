package recent

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const sample = `<?xml version="1.0" encoding="UTF-8"?>
<xbel version="1.0"
      xmlns:bookmark="http://www.freedesktop.org/standards/desktop-bookmarks"
      xmlns:mime="http://www.freedesktop.org/standards/shared-mime-info"
>
  <bookmark href="file:///home/me/repos/dank%20files" added="2025-08-10T00:03:18.164545Z" modified="2026-02-07T22:07:13.775350Z" visited="2025-08-10T00:03:18.164547Z">
    <info>
      <metadata owner="http://freedesktop.org">
        <mime:mime-type type="inode/directory"/>
        <bookmark:applications>
          <bookmark:application name="xdg-desktop-portal-gtk" exec="&apos;xdg-desktop-portal-gtk %u&apos;" modified="2026-02-07T22:07:13.775348Z" count="3"/>
        </bookmark:applications>
      </metadata>
    </info>
  </bookmark>
  <bookmark href="file:///home/me/secret.txt" added="2007-07-15T12:34:56Z" modified="2007-07-15T12:34:56Z" visited="2007-07-15T12:34:56Z">
    <title>Secret</title>
    <info>
      <metadata owner="http://freedesktop.org">
        <mime:mime-type type="text/plain"/>
        <bookmark:groups>
          <bookmark:group>notes</bookmark:group>
        </bookmark:groups>
        <bookmark:applications>
          <bookmark:application name="gedit" exec="&apos;gedit %u&apos;" modified="2007-07-15T12:34:56Z" count="1"/>
        </bookmark:applications>
        <bookmark:private/>
        <bookmark:icon href="file:///icon.png" type="image/png"/>
      </metadata>
    </info>
  </bookmark>
</xbel>`

func TestParseReadsTheGtkFormat(t *testing.T) {
	items, err := Parse([]byte(sample))
	require.NoError(t, err)
	require.Len(t, items, 2)

	folder := items[0]
	path, ok := folder.Path()
	require.True(t, ok)
	assert.Equal(t, "/home/me/repos/dank files", path)
	assert.Equal(t, "inode/directory", folder.MimeType)
	assert.Equal(t, time.Date(2026, 2, 7, 22, 7, 13, 775350000, time.UTC), folder.Used())
	assert.False(t, folder.Private)
	require.Len(t, folder.Apps, 1)
	assert.Equal(t, Application{Name: "xdg-desktop-portal-gtk", Exec: "'xdg-desktop-portal-gtk %u'", Modified: time.Date(2026, 2, 7, 22, 7, 13, 775348000, time.UTC), Count: 3}, folder.Apps[0])

	secret := items[1]
	assert.True(t, secret.Private)
	assert.Equal(t, "Secret", secret.Title)
	assert.Equal(t, []string{"notes"}, secret.Groups)
	assert.Equal(t, Icon{Href: "file:///icon.png", Type: "image/png"}, secret.Icon)
	assert.Equal(t, time.Date(2007, 7, 15, 12, 34, 56, 0, time.UTC), secret.Used())
}

func TestMarshalRoundTrips(t *testing.T) {
	items, err := Parse([]byte(sample))
	require.NoError(t, err)
	again, err := Parse(Marshal(items))
	require.NoError(t, err)
	assert.Equal(t, items, again)
	assert.Contains(t, string(Marshal(items)), `xmlns:bookmark="http://www.freedesktop.org/standards/desktop-bookmarks"`)
}

func TestParseTolerates(t *testing.T) {
	items, err := Parse(nil)
	require.NoError(t, err)
	assert.Empty(t, items)

	_, err = Parse([]byte("<xbel><bookmark"))
	assert.Error(t, err)

	_, ok := PathOf("https://example.com/a")
	assert.False(t, ok)
	_, ok = PathOf("file://otherhost/a")
	assert.False(t, ok)
	path, ok := PathOf("file:///a/b%20c/../d")
	assert.True(t, ok)
	assert.Equal(t, "/a/d", path)
}

func TestFileURIEscapesLikeGLib(t *testing.T) {
	assert.Equal(t, "file:///tmp/it's%20here.txt", FileURI("/tmp/it's here.txt"))
	assert.Equal(t, "file:///tmp/a&b%20(1).png", FileURI("/tmp/a&b (1).png"))
	assert.Equal(t, "file:///tmp/semi%3Bcolon.txt", FileURI("/tmp/semi;colon.txt"))
	assert.Equal(t, "file:///tmp/%C3%BCn%C3%AF.txt", FileURI("/tmp/ünï.txt"))
	for _, path := range []string{"/tmp/it's here.txt", "/tmp/a&b (1).png", "/tmp/semi;colon.txt", "/tmp/ünï.txt", "/tmp/hash#q?.txt"} {
		back, ok := PathOf(FileURI(path))
		assert.True(t, ok)
		assert.Equal(t, path, back)
	}
}

func TestParseDefaultsMatchGLib(t *testing.T) {
	items, err := Parse([]byte(`<xbel version="1.0"><bookmark href="file:///x"><info><metadata owner="http://freedesktop.org"><bookmark:applications xmlns:bookmark="http://www.freedesktop.org/standards/desktop-bookmarks"><bookmark:application name="a" exec="a %u"/></bookmark:applications></metadata></info></bookmark></xbel>`))
	require.NoError(t, err)
	require.Len(t, items, 1)
	assert.Equal(t, 1, items[0].Apps[0].Count)
	again, err := Parse(Marshal(items))
	require.NoError(t, err)
	assert.WithinDuration(t, time.Now(), again[0].Added, time.Minute)
}

func TestAddRegistersAndBumpsExistingItems(t *testing.T) {
	store := NewStore(t.TempDir())
	first := time.Date(2026, 10, 1, 9, 0, 0, 0, time.UTC)
	later := first.Add(time.Hour)

	require.NoError(t, store.Add("/home/me/a.txt", "text/plain", "Editor", "'gio open %u'", first))
	require.NoError(t, store.Add("/home/me/b.png", "image/png", "Viewer", "'gio open %u'", first))
	require.NoError(t, store.Add("/home/me/a.txt", "", "Editor", "'gio open %u'", later))
	require.NoError(t, store.Add("/home/me/a.txt", "", "Other", "'gio open %u'", later))

	items, stamp, err := store.Load()
	require.NoError(t, err)
	assert.NotEmpty(t, stamp)
	require.Len(t, items, 2)
	a := items[0]
	assert.Equal(t, "file:///home/me/a.txt", a.URI)
	assert.Equal(t, "text/plain", a.MimeType)
	assert.Equal(t, first, a.Added)
	assert.Equal(t, later, a.Modified)
	assert.Equal(t, first, a.Visited)
	require.Len(t, a.Apps, 2)
	assert.Equal(t, 2, a.Apps[0].Count)
	assert.Equal(t, later, a.Apps[0].Modified)
	assert.Equal(t, "Other", a.Apps[1].Name)

	info, err := os.Stat(store.Path)
	require.NoError(t, err)
	assert.Equal(t, defaultMode, info.Mode().Perm())
}

func TestRemoveAndClear(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, FileName), []byte(sample), 0o644))
	store := NewStore(dir)

	require.NoError(t, store.Remove([]string{"/home/me/repos/dank files/", "/nowhere"}))
	items, _, err := store.Load()
	require.NoError(t, err)
	require.Len(t, items, 1)
	assert.Equal(t, "file:///home/me/secret.txt", items[0].URI)
	info, err := os.Stat(store.Path)
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0o644), info.Mode().Perm())

	require.NoError(t, store.Clear())
	items, _, err = store.Load()
	require.NoError(t, err)
	assert.Empty(t, items)
	data, err := os.ReadFile(store.Path)
	require.NoError(t, err)
	assert.Contains(t, string(data), "<xbel")
}

func TestStampFollowsTheFile(t *testing.T) {
	store := NewStore(t.TempDir())
	assert.Equal(t, "", store.Stamp())
	items, stamp, err := store.Load()
	require.NoError(t, err)
	assert.Empty(t, items)
	assert.Equal(t, "", stamp)

	require.NoError(t, store.Add("/x", "text/plain", "App", "'gio open %u'", time.Now()))
	first := store.Stamp()
	assert.NotEqual(t, "", first)
	require.NoError(t, store.Add("/y", "text/plain", "App", "'gio open %u'", time.Now()))
	assert.NotEqual(t, first, store.Stamp())
}
