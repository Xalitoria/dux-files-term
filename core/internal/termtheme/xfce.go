package termtheme

import "strings"

func readXfce(s sources) (Profile, bool) {
	data, _, ok := readFirst(s.configFiles("xfce4/terminal/terminalrc"))
	if !ok {
		return Profile{}, false
	}
	config := parseINI(data)["Configuration"]

	colors := newProfile()
	colors.Fg = tint(config["ColorForeground"])
	colors.Bg = tint(config["ColorBackground"])
	colors.Cursor = tint(config["ColorCursor"])
	for i, entry := range strings.Split(config["ColorPalette"], ";") {
		if i < len(colors.ANSI) {
			colors.ANSI[i] = tint(entry)
		}
	}
	if config["BackgroundMode"] == "TERMINAL_BACKGROUND_TRANSPARENT" {
		if opacity, ok := parseOpacity(config["BackgroundDarkness"]); ok {
			colors.Opacity = opacity
		}
	}
	return colors, colors.usable()
}
