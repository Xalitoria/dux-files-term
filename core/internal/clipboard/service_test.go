package clipboard

import (
	"errors"
	"testing"

	"github.com/AvengeMedia/dankgo/wlclipboard"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeSelection struct {
	offers []wlclipboard.Offer
	closed int
	err    error
}

func (f *fakeSelection) Set(offers []wlclipboard.Offer) error {
	if f.err != nil {
		return f.err
	}
	f.offers = offers
	return nil
}

func (f *fakeSelection) Close() {
	f.closed++
	f.offers = nil
}

func (f *fakeSelection) data(mime string) ([]byte, bool) {
	for _, offer := range f.offers {
		if offer.MimeType == mime {
			return offer.Data, true
		}
	}
	return nil, false
}

func newFake(t *testing.T) (*Service, *fakeSelection) {
	t.Helper()
	sel := &fakeSelection{}
	svc := &Service{newOwner: func() (selection, error) { return sel, nil }}
	svc.paste = func() ([]byte, string, error) {
		data, ok := sel.data(MimeTextUTF8)
		if !ok {
			return nil, "", wlclipboard.ErrNoData
		}
		return data, MimeTextUTF8, nil
	}
	return svc, sel
}

func TestCopyPublishesEveryFlavour(t *testing.T) {
	svc, sel := newFake(t)
	require.NoError(t, svc.Copy([]string{"/home/me/a.txt"}, false))

	gnome, ok := sel.data(MimeGnomeCopiedFiles)
	require.True(t, ok)
	assert.Equal(t, "copy\nfile:///home/me/a.txt\n", string(gnome))

	uris, ok := sel.data(MimeUriList)
	require.True(t, ok)
	assert.Equal(t, "file:///home/me/a.txt\r\n", string(uris))

	plain, ok := sel.data(MimeTextUTF8)
	require.True(t, ok)
	assert.Equal(t, "/home/me/a.txt", string(plain))

	_, ok = sel.data(MimeKdeCutSelection)
	assert.False(t, ok, "a copy must not advertise the KDE cut marker")
}

func TestCutAddsTheKdeMarker(t *testing.T) {
	svc, sel := newFake(t)
	require.NoError(t, svc.Copy([]string{"/home/me/a.txt"}, true))

	marker, ok := sel.data(MimeKdeCutSelection)
	require.True(t, ok)
	assert.Equal(t, "1", string(marker))

	gnome, _ := sel.data(MimeGnomeCopiedFiles)
	assert.Equal(t, "cut\nfile:///home/me/a.txt\n", string(gnome))
}

func TestCopyServesLegacyTextAliases(t *testing.T) {
	svc, sel := newFake(t)
	require.NoError(t, svc.Copy([]string{"/a"}, false))
	for _, alias := range []string{"text/plain", "UTF8_STRING", "STRING", "TEXT"} {
		_, ok := sel.data(alias)
		assert.True(t, ok, alias)
	}
}

func TestCopyRefusesRelativePaths(t *testing.T) {
	svc, _ := newFake(t)
	require.Error(t, svc.Copy([]string{"a.txt"}, false))
	require.Error(t, svc.Copy(nil, false))
}

func TestPathsReturnsOurOwnSelectionExactly(t *testing.T) {
	svc, _ := newFake(t)
	require.NoError(t, svc.Copy([]string{"/home/me/a.txt", "/home/me/b c.txt"}, true))

	paths, cut, source := svc.Paths()
	assert.Equal(t, []string{"/home/me/a.txt", "/home/me/b c.txt"}, paths)
	assert.True(t, cut)
	assert.Equal(t, "dfiles", source)
}

func TestPathsParsesAForeignSelection(t *testing.T) {
	svc, _ := newFake(t)
	require.NoError(t, svc.Copy([]string{"/home/me/mine.txt"}, true))
	svc.paste = func() ([]byte, string, error) {
		return GnomePayload([]string{"/home/me/theirs.txt"}, false), MimeGnomeCopiedFiles, nil
	}

	paths, cut, source := svc.Paths()
	assert.Equal(t, []string{"/home/me/theirs.txt"}, paths, "a selection we no longer own must not paste our stale paths")
	assert.False(t, cut)
	assert.Equal(t, MimeGnomeCopiedFiles, source)
}

func TestPathsOnAnEmptyClipboard(t *testing.T) {
	svc, _ := newFake(t)
	paths, cut, source := svc.Paths()
	assert.Nil(t, paths)
	assert.False(t, cut)
	assert.Empty(t, source)
}

func TestClearReleasesTheSelection(t *testing.T) {
	svc, sel := newFake(t)
	require.NoError(t, svc.Copy([]string{"/a"}, false))
	svc.Clear()
	assert.Equal(t, 1, sel.closed)

	paths, _, _ := svc.Paths()
	assert.Nil(t, paths)
}

func TestCopyReusesOneOwner(t *testing.T) {
	made := 0
	sel := &fakeSelection{}
	svc := &Service{
		newOwner: func() (selection, error) { made++; return sel, nil },
		paste:    func() ([]byte, string, error) { return nil, "", wlclipboard.ErrNoData },
	}
	require.NoError(t, svc.Copy([]string{"/a"}, false))
	require.NoError(t, svc.Copy([]string{"/b"}, false))
	assert.Equal(t, 1, made)
}

func TestAFailedOwnerMarksTheClipboardUnusable(t *testing.T) {
	svc := &Service{
		newOwner: func() (selection, error) { return nil, errors.New("no data-control") },
		paste:    func() ([]byte, string, error) { return nil, "", wlclipboard.ErrNoData },
	}
	assert.True(t, svc.Available())
	require.Error(t, svc.Copy([]string{"/a"}, false))
	assert.False(t, svc.Available())
}
