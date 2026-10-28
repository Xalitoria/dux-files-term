package termtheme

import (
	"os"
	"strconv"
	"strings"
)

func readFoot(s sources) (Profile, bool) {
	_, path, ok := readFirst(s.configFiles("foot/foot.ini"))
	if !ok {
		return Profile{}, false
	}
	config := readFootINI(path, 0)

	section := "colors"
	if _, dark := config["colors-dark"]; dark {
		section = "colors-dark"
	}
	colors := newProfile()
	colors.Fg = tint(config.get(section, "foreground"))
	colors.Bg = tint(config.get(section, "background"))
	colors.SelectionFg = tint(config.get(section, "selection-foreground"))
	colors.SelectionBg = tint(config.get(section, "selection-background"))
	for i := range colors.ANSI {
		prefix := "regular"
		if i >= 8 {
			prefix = "bright"
		}
		colors.ANSI[i] = tint(config.get(section, prefix+strconv.Itoa(i%8)))
	}
	if _, cursor, found := strings.Cut(config.get("cursor", "color"), " "); found {
		colors.Cursor = tint(cursor)
	}
	if opacity, ok := parseOpacity(config.get(section, "alpha")); ok {
		colors.Opacity = opacity
	}
	colors.WordDelims = config.get("main", "word-delimiters")
	target := config.get("main", "selection-target")
	colors.CopyOnSelect = target == "clipboard" || target == "both"
	colors.setCursor(config.get("cursor", "style"), truthy(config.get("cursor", "blink")))
	colors.FontFamily, colors.FontSize = footFont(config.get("main", "font"))
	for action, name := range footActions {
		for _, text := range strings.Fields(config.get("key-bindings", name)) {
			if chord, ok := parseChord(text); ok {
				colors.bind(action, chord)
			}
		}
	}
	return colors, colors.usable()
}

var footActions = map[string]string{
	ActionCopy:      "clipboard-copy",
	ActionPaste:     "clipboard-paste",
	ActionZoomIn:    "font-increase",
	ActionZoomOut:   "font-decrease",
	ActionZoomReset: "font-reset",
}

// The first of foot's fonts, written "Family:size=12:weight=bold".
func footFont(text string) (string, float64) {
	first, _, _ := strings.Cut(text, ",")
	family, attrs, _ := strings.Cut(first, ":")
	var size float64
	for _, attr := range strings.Split(attrs, ":") {
		if value, ok := strings.CutPrefix(attr, "size="); ok {
			size, _ = strconv.ParseFloat(value, 64)
		}
	}
	return namedFamily(family), size
}

// An include is read first, so the including file keeps the last word.
func readFootINI(path string, depth int) ini {
	data, err := os.ReadFile(path)
	if err != nil || depth > maxIncludeDepth {
		return ini{}
	}
	config := parseINI(data)
	merged := ini{}
	for _, section := range config {
		if included := section["include"]; included != "" {
			mergeINI(merged, readFootINI(resolveFrom(path, included), depth+1))
		}
	}
	mergeINI(merged, config)
	return merged
}

func mergeINI(into, from ini) {
	for name, section := range from {
		if into[name] == nil {
			into[name] = map[string]string{}
		}
		for key, value := range section {
			into[name][key] = value
		}
	}
}
