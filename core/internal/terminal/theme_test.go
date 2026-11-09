package terminal

import (
	"image/color"
	"strings"
	"testing"
	"time"

	"github.com/duxlabs-dev/dux-files/core/internal/termtheme"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

func themeBackground(t *testing.T, events *published) string {
	t.Helper()
	theme := events.lastTheme()
	require.NotNil(t, theme, "no theme was published")
	return theme["colors"].(map[string]any)["bg"].(string)
}

func TestRestyleAppliesAtOnceWhenTheShellIsIdle(t *testing.T) {
	events := &published{}
	session, _ := startShell(t, events)
	require.Eventually(t, session.shellIdle, 5*time.Second, 20*time.Millisecond)

	session.Restyle(termtheme.Profile{Bg: color.RGBA{0x0f, 0, 0x28, 0xff}, Opacity: 1})

	assert.Equal(t, "#0f0028", themeBackground(t, events))
}

func TestRestyleWaitsForAnUnfocusedBusyTerminal(t *testing.T) {
	events := &published{}
	session, _ := startShell(t, events)
	require.Eventually(t, session.shellIdle, 5*time.Second, 20*time.Millisecond)
	session.Input(0, input{text: "sleep 1; echo done-$((40+2))"})
	session.Input(1, input{key: "enter"})
	require.Eventually(t, func() bool { return !session.shellIdle() }, 5*time.Second, 20*time.Millisecond)

	session.Restyle(termtheme.Profile{Bg: color.RGBA{0x0f, 0, 0x28, 0xff}, Opacity: 1})
	assert.Nil(t, events.lastTheme(), "a running program is left alone")

	require.Eventually(t, func() bool { return strings.Contains(screenText(session), "done-42") }, 5*time.Second, 20*time.Millisecond)
	assert.Eventually(t, func() bool { return events.lastTheme() != nil }, 5*time.Second, 20*time.Millisecond, "applies once the shell is back at its prompt")
	assert.Equal(t, "#0f0028", themeBackground(t, events))
}

func TestRestyleAppliesToABusyTerminalOnceItHasFocus(t *testing.T) {
	events := &published{}
	session, _ := startShell(t, events)
	require.Eventually(t, session.shellIdle, 5*time.Second, 20*time.Millisecond)
	session.Input(0, input{text: "sleep 30"})
	session.Input(1, input{key: "enter"})
	require.Eventually(t, func() bool { return !session.shellIdle() }, 5*time.Second, 20*time.Millisecond)

	session.Restyle(termtheme.Profile{Bg: color.RGBA{0x0f, 0, 0x28, 0xff}, Opacity: 1})
	assert.Nil(t, events.lastTheme())

	session.Focus(true)
	assert.Equal(t, "#0f0028", themeBackground(t, events))
}

func TestRestyleKeepsOnlyTheLatestLook(t *testing.T) {
	events := &published{}
	session, _ := startShell(t, events)
	require.Eventually(t, session.shellIdle, 5*time.Second, 20*time.Millisecond)
	session.Input(0, input{text: "sleep 30"})
	session.Input(1, input{key: "enter"})
	require.Eventually(t, func() bool { return !session.shellIdle() }, 5*time.Second, 20*time.Millisecond)

	session.Restyle(termtheme.Profile{Bg: color.RGBA{0x11, 0x11, 0x11, 0xff}, Opacity: 1})
	session.Restyle(termtheme.Profile{Bg: color.RGBA{0x22, 0x22, 0x22, 0xff}, Opacity: 1})
	session.Focus(true)

	assert.Equal(t, "#222222", themeBackground(t, events))
}
