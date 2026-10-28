package terminal

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"image"
	"image/png"
	"os"
	"strings"
	"testing"

	"github.com/charmbracelet/x/vt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type graphicsRig struct {
	g       *graphics
	emu     *vt.Emulator
	replies []string
}

func newRig(cols, rows int) *graphicsRig {
	emu := vt.NewEmulator(cols, rows)
	emu.SetScrollbackSize(100)
	rig := &graphicsRig{emu: emu}
	rig.g = newGraphics(emu, func(b []byte) { rig.replies = append(rig.replies, string(b)) }, func() (int, int) { return 10, 20 })
	return rig
}

func apc(keys string, payload []byte) string {
	return "\x1b_G" + keys + ";" + base64.StdEncoding.EncodeToString(payload) + "\x1b\\"
}

func pixels(w, h int) []byte {
	return bytes.Repeat([]byte{200, 100, 50}, w*h)
}

func (r *graphicsRig) placements(t *testing.T, scroll int) []placementView {
	t.Helper()
	frame, _ := r.g.frame(r.g.history()-scroll, r.emu.Height())
	if frame.Placements == nil {
		return nil
	}
	var views []placementView
	require.NoError(t, json.Unmarshal(frame.Placements, &views))
	return views
}

func TestGraphicsAnswersSupportQueries(t *testing.T) {
	rig := newRig(80, 24)

	rig.g.feed([]byte(apc("i=31,s=1,v=1,a=q,t=d,f=24", []byte("123"))))
	rig.g.feed([]byte(apc("i=32,s=1,v=1,a=q,t=d,f=24,q=1", []byte("123"))))
	rig.g.feed([]byte(apc("s=1,v=1,a=q,t=d,f=24", []byte("123"))))
	rig.g.feed([]byte(apc("i=33,s=2,v=2,a=q,t=d,f=24", []byte("123"))))

	require.Len(t, rig.replies, 2)
	assert.Equal(t, "\x1b_Gi=31;OK\x1b\\", rig.replies[0])
	assert.Equal(t, "\x1b_Gi=33;EINVAL:data does not match the size\x1b\\", rig.replies[1])
	assert.Empty(t, rig.g.images)
}

func TestGraphicsPlacesTransmittedImagesAtTheCursor(t *testing.T) {
	rig := newRig(80, 24)

	rig.g.feed([]byte("\x1b[3;5H" + apc("a=T,i=7,f=24,s=20,v=40", pixels(20, 40))))

	views := rig.placements(t, 0)
	require.Len(t, views, 1)
	assert.Equal(t, placementView{Image: 1, X: 4, Y: 2, Cols: 2, Rows: 2, W: 2, H: 2, SW: 20, SH: 40}, views[0])
	assert.Equal(t, 6, rig.emu.CursorPosition().X)
	assert.Equal(t, 3, rig.emu.CursorPosition().Y)
	assert.Equal(t, []string{"\x1b_Gi=7;OK\x1b\\"}, rig.replies)

	frame, _ := rig.g.frame(0, 24)
	assert.Empty(t, frame.Images, "an image is delivered once")
	rig.g.resync()
	frame, _ = rig.g.frame(0, 24)
	require.Len(t, frame.Images, 1)
	data, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(frame.Images[0].Src, "data:image/png;base64,"))
	require.NoError(t, err)
	decoded, err := png.Decode(bytes.NewReader(data))
	require.NoError(t, err)
	assert.Equal(t, image.Rect(0, 0, 20, 40), decoded.Bounds())
}

func TestGraphicsJoinsChunksSplitAcrossReads(t *testing.T) {
	rig := newRig(80, 24)
	data := pixels(10, 20)
	first := apc("a=T,i=9,f=24,s=10,v=20,m=1", data[:300])
	last := apc("m=0", data[300:])
	stream := "before" + first + last + "after"

	for i := 0; i < len(stream); i += 7 {
		rig.g.feed([]byte(stream[i:min(i+7, len(stream))]))
	}

	require.Len(t, rig.placements(t, 0), 1)
	assert.Contains(t, rig.emu.String(), "before")
	assert.Contains(t, rig.emu.String(), "after")
	assert.NotContains(t, rig.emu.String(), "_G")
}

func TestGraphicsKeepsImagesWithTheirText(t *testing.T) {
	rig := newRig(20, 5)
	rig.g.feed([]byte(apc("a=T,i=1,f=24,s=10,v=40,C=1", pixels(10, 40))))
	rig.g.feed([]byte(strings.Repeat("\n", 5)))

	views := rig.placements(t, 0)
	require.Len(t, views, 1)
	assert.Equal(t, -1, views[0].Y, "scrolled up with the text")

	scrolledBack := rig.placements(t, 1)
	require.Len(t, scrolledBack, 1)
	assert.Equal(t, 0, scrolledBack[0].Y)

}

func TestGraphicsDropsImagesWhenTheScreenIsCleared(t *testing.T) {
	rig := newRig(20, 5)
	rig.g.feed([]byte(apc("a=T,i=1,f=24,s=10,v=40,C=1", pixels(10, 40))))
	require.Len(t, rig.placements(t, 0), 1)

	rig.g.feed([]byte("\x1b[2J"))

	assert.Empty(t, rig.placements(t, 0))
}

func TestGraphicsDeletesPlacementsAndFreesImages(t *testing.T) {
	rig := newRig(80, 24)
	rig.g.feed([]byte(apc("a=T,i=1,f=24,s=10,v=20", pixels(10, 20))))
	rig.g.feed([]byte(apc("a=p,i=1,p=2,C=1", nil)))
	require.Len(t, rig.placements(t, 0), 2)

	rig.g.feed([]byte("\x1b_Ga=d,d=i,i=1,p=2\x1b\\"))
	assert.Len(t, rig.placements(t, 0), 1)
	assert.Len(t, rig.g.images, 1, "lowercase keeps the image data")

	rig.g.feed([]byte("\x1b_Ga=d,d=A\x1b\\"))
	assert.Empty(t, rig.placements(t, 0))
	assert.Empty(t, rig.g.images)
	assert.Zero(t, rig.g.stored)
}

func TestGraphicsReplacesAPlacementWithTheSameID(t *testing.T) {
	rig := newRig(80, 24)
	rig.g.feed([]byte(apc("a=T,i=1,p=3,f=24,s=10,v=20,C=1", pixels(10, 20))))
	rig.g.feed([]byte("\x1b[5;5H\x1b_Ga=p,i=1,p=3,C=1\x1b\\"))

	views := rig.placements(t, 0)
	require.Len(t, views, 1)
	assert.Equal(t, 4, views[0].Y)
}

func TestGraphicsEvictsTheOldestImagesOverTheBudget(t *testing.T) {
	rig := newRig(80, 24)
	big := &storedImage{uid: 100, png: make([]byte, maxStoredBytes-10)}
	rig.g.images[big.uid] = big
	rig.g.stored = len(big.png)

	rig.g.feed([]byte(apc("a=t,i=1,f=24,s=100,v=100", pixels(100, 100))))

	assert.NotContains(t, rig.g.images, 100)
	assert.NotNil(t, rig.g.imageByID(1))
	assert.Equal(t, []int{100}, rig.g.dropped)
}

func TestGraphicsAssignsIDsForImageNumbers(t *testing.T) {
	rig := newRig(80, 24)

	rig.g.feed([]byte(apc("a=t,I=13,f=24,s=1,v=1", []byte("123"))))

	require.Len(t, rig.replies, 1)
	assert.Equal(t, fmt.Sprintf("\x1b_Gi=%d,I=13;OK\x1b\\", autoIDBase+1), rig.replies[0])
}

func TestGraphicsIgnoresOtherEscapeStrings(t *testing.T) {
	rig := newRig(80, 24)

	rig.g.feed([]byte("a\x1b_Xnot graphics\x1b\\b\x1b"))
	rig.g.feed([]byte("[1;1Hc"))

	assert.True(t, strings.HasPrefix(strings.TrimSpace(rig.emu.String()), "c"), rig.emu.String())
	assert.Empty(t, rig.replies)
}

func TestGraphicsReadsImagesFromFilesWithUnpaddedPaths(t *testing.T) {
	rig := newRig(80, 24)
	var encoded bytes.Buffer
	require.NoError(t, png.Encode(&encoded, image.NewNRGBA(image.Rect(0, 0, 30, 20))))
	path := t.TempDir() + "/pic.png"
	require.NoError(t, os.WriteFile(path, encoded.Bytes(), 0o600))
	unpadded := strings.TrimRight(base64.StdEncoding.EncodeToString([]byte(path)), "=")

	rig.g.feed([]byte("\x1b_Ga=T,i=4,f=100,t=f;" + unpadded + "\x1b\\"))
	rig.g.feed([]byte("\x1b_Ga=T,i=5,f=100,t=f;" + base64.StdEncoding.EncodeToString([]byte("/proc/self/environ")) + "\x1b\\"))

	views := rig.placements(t, 0)
	require.Len(t, views, 1)
	assert.Equal(t, 3, views[0].Cols)
	assert.Equal(t, []string{"\x1b_Gi=4;OK\x1b\\", "\x1b_Gi=5;EINVAL:unsupported path\x1b\\"}, rig.replies)
	_, err := os.Stat(path)
	assert.NoError(t, err, "files sent as t=f are kept")
}

func TestGraphicsSizesPlacementsFromCellCounts(t *testing.T) {
	rig := newRig(80, 24)

	rig.g.feed([]byte(apc("a=T,i=1,f=24,s=100,v=50,c=4,C=1", pixels(100, 50))))
	rig.g.feed([]byte("\x1b[3;1H" + apc("a=T,i=2,f=24,s=100,v=50,r=6,C=1", pixels(100, 50))))

	views := rig.placements(t, 0)
	require.Len(t, views, 2)
	assert.Equal(t, placementView{Image: 1, Cols: 4, Rows: 1, W: 4, H: 1, SW: 100, SH: 50}, views[0])
	assert.Equal(t, 6, views[1].Rows)
	assert.InDelta(t, 24, views[1].W, 0.01)
}
