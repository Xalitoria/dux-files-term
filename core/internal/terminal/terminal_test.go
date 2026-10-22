package terminal

import (
	"testing"

	uv "github.com/charmbracelet/ultraviolet"
	"github.com/charmbracelet/x/vt"
	"github.com/stretchr/testify/assert"
)

func screenRow(t *testing.T, cols int, output string) []run {
	t.Helper()
	emu := vt.NewEmulator(cols, 2)
	_, err := emu.WriteString(output)
	assert.NoError(t, err)
	return rowRuns(cols, func(x int) *uv.Cell { return emu.CellAt(x, 0) }, emu.IndexedColor)
}

func TestRowRuns(t *testing.T) {
	tests := []struct {
		name   string
		output string
		want   []run
	}{
		{"empty row", "", []run{}},
		{"plain text drops trailing blanks", "hi  there", []run{{X: 0, W: 9, Text: "hi  there"}}},
		{"style change splits runs", "a\x1b[1;31mb\x1b[0mc", []run{{X: 0, W: 1, Text: "a"}, {X: 1, W: 1, Text: "b", Fg: "#800000", Attr: attrBold}, {X: 2, W: 1, Text: "c"}}},
		{"truecolor background", "\x1b[48;2;1;2;3mx", []run{{X: 0, W: 1, Text: "x", Bg: "#010203"}}},
		{"coloured trailing space is kept", "a\x1b[44m  ", []run{{X: 0, W: 1, Text: "a"}, {X: 1, W: 2, Text: "  ", Bg: "#000080"}}},
		{"inverse, italic and underline", "\x1b[7;3;4mz", []run{{X: 0, W: 1, Text: "z", Attr: attrInverse | attrItalic | attrUnderline}}},
		{"wide characters sit on the grid", "a漢b", []run{{X: 0, W: 1, Text: "a"}, {X: 1, W: 2, Text: "漢"}, {X: 3, W: 1, Text: "b"}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, screenRow(t, 12, tt.output))
		})
	}
}

type recordedKeys struct {
	keys []uv.KeyEvent
	text string
}

func (r *recordedKeys) SendKey(k uv.KeyEvent) { r.keys = append(r.keys, k) }
func (r *recordedKeys) SendText(s string)     { r.text += s }

func TestSendKey(t *testing.T) {
	tests := []struct {
		name     string
		key      string
		text     string
		mods     int
		wantText string
		wantKey  uv.KeyEvent
	}{
		{name: "typed text", text: "é", wantText: "é"},
		{name: "control character", text: "\x03", mods: modCtrl, wantText: "\x03"},
		{name: "alt prefixes escape", text: "f", mods: modAlt, wantText: "\x1bf"},
		{name: "nothing to send", mods: modCtrl},
		{name: "plain arrow uses the emulator", key: "up", wantKey: uv.KeyPressEvent{Code: uv.KeyUp}},
		{name: "ctrl arrow", key: "left", mods: modCtrl, wantText: "\x1b[1;5D"},
		{name: "shift alt end", key: "end", mods: modShift | modAlt, wantText: "\x1b[1;4F"},
		{name: "ctrl delete", key: "delete", mods: modCtrl, wantText: "\x1b[3;5~"},
		{name: "shift f5", key: "f5", mods: modShift, wantText: "\x1b[15;2~"},
		{name: "backtab", key: "tab", mods: modShift, wantKey: uv.KeyPressEvent{Code: uv.KeyTab, Mod: uv.ModShift}},
		{name: "alt enter", key: "enter", mods: modAlt, wantKey: uv.KeyPressEvent{Code: uv.KeyEnter, Mod: uv.ModAlt}},
		{name: "ctrl space", key: "space", mods: modCtrl, wantKey: uv.KeyPressEvent{Code: uv.KeySpace, Mod: uv.ModCtrl}},
		{name: "shift space", key: "space", mods: modShift, wantText: " "},
		{name: "ctrl backspace", key: "backspace", mods: modCtrl, wantText: "\x08"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var sink recordedKeys
			sendKey(&sink, tt.key, tt.text, tt.mods)
			assert.Equal(t, tt.wantText, sink.text)
			if tt.wantKey == nil {
				assert.Empty(t, sink.keys)
				return
			}
			assert.Equal(t, []uv.KeyEvent{tt.wantKey}, sink.keys)
		})
	}
}

func TestCdCommand(t *testing.T) {
	assert.Equal(t, "\x05\x15 cd -- '/tmp/a b'\r", cdCommand("/tmp/a b"))
	assert.Equal(t, `'it'\''s'`, shellQuote("it's"))
}

func TestShellArgv(t *testing.T) {
	assert.Equal(t, []string{"/usr/bin/fish"}, shellArgv(false, "/home", "/usr/bin/fish"))
	assert.Equal(t, []string{"/bin/sh"}, shellArgv(false, "/home", ""))
	sandboxed := shellArgv(true, "/home/me", "/bin/sh")
	assert.Equal(t, []string{"flatpak-spawn", "--host", "--watch-bus", "--directory=/home/me"}, sandboxed[:4])
}

func TestClampSize(t *testing.T) {
	cols, rows := clampSize(0, -3)
	assert.Equal(t, []int{2, 1}, []int{cols, rows})
	cols, rows = clampSize(9000, 9000)
	assert.Equal(t, []int{maxCols, maxRows}, []int{cols, rows})
}
