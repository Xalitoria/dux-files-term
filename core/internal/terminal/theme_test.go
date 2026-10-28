package terminal

import (
	"image/color"
	"testing"

	"github.com/duxlabs-dev/dux-files/core/internal/termtheme"
	"github.com/stretchr/testify/assert"
)

func TestSessionAppliesColors(t *testing.T) {
	session, _ := startShell(t, &published{})
	colors := termtheme.Profile{Bg: color.RGBA{0x0f, 0, 0x28, 0xff}, Opacity: 0.9}
	colors.ANSI[1] = color.RGBA{0x7d, 0x16, 0x25, 0xff}

	session.applyColors(colors)

	session.mu.Lock()
	defer session.mu.Unlock()
	assert.Equal(t, "#7d1625", hexColor(session.emu.IndexedColor(1), nil))
	assert.Equal(t, "#0f0028", hexColor(session.emu.BackgroundColor(), nil))
	assert.Equal(t, "#00ff00", hexColor(session.emu.IndexedColor(10), session.emu.IndexedColor), "colours the config leaves out keep their default")
}

func TestColorsPayload(t *testing.T) {
	payload := colorsPayload(termtheme.Profile{Fg: color.RGBA{0xdd, 0xdd, 0xdd, 0xff}, Opacity: 0.9})

	assert.Equal(t, map[string]any{"fg": "#dddddd", "bg": "", "cursor": "", "opacity": 0.9, "selectionFg": "", "selectionBg": ""}, payload)
}

func TestBehaviorPayload(t *testing.T) {
	profile := termtheme.Profile{WordChars: "-.", CopyOnSelect: true, FontFamily: "Iosevka", FontSize: 12, Keys: map[string][]string{"copy": {"ctrl+c"}}}

	assert.Equal(t, map[string]any{
		"wordChars": "-.", "wordDelims": "", "copyOnSelect": true, "fontFamily": "Iosevka", "fontSize": 12.0,
		"cursorShape": "", "cursorBlink": false,
		"keys": map[string][]string{"copy": {"ctrl+c"}},
	}, behaviorPayload(profile))
	assert.Equal(t, map[string][]string{}, behaviorPayload(termtheme.Profile{})["keys"], "no bindings is an empty object, not null")
}
