package terminal

import (
	"strconv"

	uv "github.com/charmbracelet/ultraviolet"
)

const (
	modShift = 1 << iota
	modAlt
	modCtrl
)

type namedKey struct {
	code rune
	csi  string // final byte of the xterm modified form, CSI 1;<mod><csi>
	num  int    // CSI <num>;<mod>~ when csi is empty
}

var namedKeys = map[string]namedKey{
	"up":        {code: uv.KeyUp, csi: "A"},
	"down":      {code: uv.KeyDown, csi: "B"},
	"right":     {code: uv.KeyRight, csi: "C"},
	"left":      {code: uv.KeyLeft, csi: "D"},
	"home":      {code: uv.KeyHome, csi: "H"},
	"end":       {code: uv.KeyEnd, csi: "F"},
	"insert":    {code: uv.KeyInsert, num: 2},
	"delete":    {code: uv.KeyDelete, num: 3},
	"pageUp":    {code: uv.KeyPgUp, num: 5},
	"pageDown":  {code: uv.KeyPgDown, num: 6},
	"f1":        {code: uv.KeyF1, csi: "P"},
	"f2":        {code: uv.KeyF2, csi: "Q"},
	"f3":        {code: uv.KeyF3, csi: "R"},
	"f4":        {code: uv.KeyF4, csi: "S"},
	"f5":        {code: uv.KeyF5, num: 15},
	"f6":        {code: uv.KeyF6, num: 17},
	"f7":        {code: uv.KeyF7, num: 18},
	"f8":        {code: uv.KeyF8, num: 19},
	"f9":        {code: uv.KeyF9, num: 20},
	"f10":       {code: uv.KeyF10, num: 21},
	"f11":       {code: uv.KeyF11, num: 23},
	"f12":       {code: uv.KeyF12, num: 24},
	"enter":     {code: uv.KeyEnter},
	"tab":       {code: uv.KeyTab},
	"backspace": {code: uv.KeyBackspace},
	"escape":    {code: uv.KeyEscape},
	"space":     {code: uv.KeySpace},
}

type keySink interface {
	SendKey(uv.KeyEvent)
	SendText(string)
}

func sendKey(sink keySink, name, text string, mods int) {
	key, named := namedKeys[name]
	switch {
	case named:
		sendNamed(sink, key, mods)
	case text == "":
		return
	case mods&modAlt != 0:
		sink.SendText("\x1b" + text)
	default:
		sink.SendText(text)
	}
}

func sendNamed(sink keySink, key namedKey, mods int) {
	if seq := modifiedSequence(key, mods); seq != "" {
		sink.SendText(seq)
		return
	}
	event := uv.KeyPressEvent{Code: key.code}
	switch {
	case key.code == uv.KeyTab && mods&modShift != 0:
		event.Mod = uv.ModShift
	case key.code == uv.KeySpace && mods&modCtrl != 0:
		event.Mod = uv.ModCtrl
	case key.code == uv.KeySpace:
		sink.SendText(altPrefix(mods) + " ")
		return
	case key.code == uv.KeyBackspace && mods&modCtrl != 0:
		sink.SendText(altPrefix(mods) + "\x08")
		return
	}
	if mods&modAlt != 0 {
		event.Mod |= uv.ModAlt
	}
	sink.SendKey(event)
}

func altPrefix(mods int) string {
	if mods&modAlt != 0 {
		return "\x1b"
	}
	return ""
}

// The emulator only encodes unmodified cursor and function keys, so modified ones use the xterm form.
func modifiedSequence(key namedKey, mods int) string {
	if mods == 0 || (key.csi == "" && key.num == 0) {
		return ""
	}
	param := strconv.Itoa(mods + 1)
	if key.csi != "" {
		return "\x1b[1;" + param + key.csi
	}
	return "\x1b[" + strconv.Itoa(key.num) + ";" + param + "~"
}
