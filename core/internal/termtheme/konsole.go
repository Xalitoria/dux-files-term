package termtheme

import (
	"path/filepath"
	"strconv"
)

func readKonsole(s sources) (Profile, bool) {
	rc, _, ok := readFirst(s.configFiles("konsolerc"))
	if !ok {
		return Profile{}, false
	}
	profile := parseINI(rc).get("Desktop Entry", "DefaultProfile")
	scheme := ""
	if data, _, found := readFirst(s.dataFiles(filepath.Join("konsole", profile))); found && profile != "" {
		scheme = parseINI(data).get("Appearance", "ColorScheme")
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
	return colors, colors.usable()
}
