package termtheme

import (
	"path/filepath"
	"strconv"
	"strings"
)

func readKonsole(s sources) (Profile, bool) {
	rc, _, ok := readFirst(s.configFiles("konsolerc"))
	if !ok {
		return Profile{}, false
	}
	profile := parseINI(rc).get("Desktop Entry", "DefaultProfile")
	scheme := ""
	settings := ini{}
	if data, _, found := readFirst(s.dataFiles(filepath.Join("konsole", profile))); found && profile != "" {
		settings = parseINI(data)
		scheme = settings.get("Appearance", "ColorScheme")
	}
	if scheme == "" {
		return Profile{}, false
	}
	data, _, ok := readFirst(s.dataFiles(filepath.Join("konsole", scheme+".colorscheme")))
	if !ok {
		return Profile{}, false
	}
	config := parseINI(data)

	colors := newProfile()
	colors.Fg, _ = parseRGBTriple(config.get("Foreground", "Color"))
	colors.Bg, _ = parseRGBTriple(config.get("Background", "Color"))
	for i := range colors.ANSI {
		section := "Color" + strconv.Itoa(i%8)
		if i >= 8 {
			section += "Intense"
		}
		colors.ANSI[i], _ = parseRGBTriple(config.get(section, "Color"))
	}
	if opacity, ok := parseOpacity(config.get("General", "Opacity")); ok {
		colors.Opacity = opacity
	}
	colors.ScrollbackLines = konsoleHistory(settings.get("Scrolling", "HistoryMode"), settings.get("Scrolling", "HistorySize"))
	colors.Bell = konsoleBell(settings.get("Terminal Features", "BellMode"))
	return colors, colors.usable()
}

// Mode 1 keeps HistorySize lines and 2 keeps everything. Mode 0 keeps nothing, which the dock does not do.
func konsoleHistory(mode, size string) int {
	switch strings.TrimSpace(mode) {
	case "2":
		return UnlimitedScrollback
	case "0":
		return 0
	}
	return parseLines(size)
}

// Modes are 0 for the system beep, 1 for a notification, 2 for a flash and 3 for nothing.
func konsoleBell(mode string) []string {
	switch strings.TrimSpace(mode) {
	case "":
		return nil
	case "2":
		return bellOf(false, true, false)
	case "3":
		return quietBell
	}
	return bellOf(true, false, false)
}
