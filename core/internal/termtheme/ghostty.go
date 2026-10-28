package termtheme

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type ghosttyConfig struct {
	values  map[string]string
	palette map[int]string
	binds   []string
}

func newGhosttyConfig() *ghosttyConfig {
	return &ghosttyConfig{values: map[string]string{}, palette: map[int]string{}}
}

func (c *ghosttyConfig) overlay(top *ghosttyConfig) {
	for key, value := range top.values {
		c.values[key] = value
	}
	for index, value := range top.palette {
		c.palette[index] = value
	}
	c.binds = append(c.binds, top.binds...)
}

func readGhostty(s sources) (Profile, bool) {
	_, path, ok := readFirst(s.configFiles("ghostty/config.ghostty", "ghostty/config"))
	if !ok {
		return Profile{}, false
	}
	user := newGhosttyConfig()
	readGhosttyFile(path, 0, user)

	config := newGhosttyConfig()
	if theme := ghosttyThemeFile(s, user.values["theme"]); theme != "" {
		readGhosttyFile(theme, maxIncludeDepth, config)
	}
	config.overlay(user)

	profile := newProfile()
	profile.Fg = tint(config.values["foreground"])
	profile.Bg = tint(config.values["background"])
	profile.Cursor = tint(config.values["cursor-color"])
	profile.SelectionFg = tint(config.values["selection-foreground"])
	profile.SelectionBg = tint(config.values["selection-background"])
	for index, value := range config.palette {
		profile.ANSI[index] = tint(value)
	}
	if opacity, ok := parseOpacity(config.values["background-opacity"]); ok {
		profile.Opacity = opacity
	}
	profile.WordDelims = config.values["selection-word-chars"]
	profile.CopyOnSelect = config.values["copy-on-select"] == "clipboard"
	profile.FontFamily = namedFamily(config.values["font-family"])
	profile.FontSize, _ = strconv.ParseFloat(config.values["font-size"], 64)
	profile.setCursor(config.values["cursor-style"], truthy(config.values["cursor-style-blink"]))
	for _, bind := range config.binds {
		ghosttyBind(&profile, bind)
	}
	return profile, profile.usable()
}

// A bind reads "ctrl+shift+c=copy_to_clipboard". The chord can end in "=", so the split is at the last one.
func ghosttyBind(profile *Profile, bind string) {
	cut := strings.LastIndex(bind, "=")
	if cut < 0 {
		return
	}
	chord, action := bind[:cut], bind[cut+1:]
	for _, prefix := range []string{"global:", "all:", "unconsumed:", "performable:"} {
		chord = strings.TrimPrefix(chord, prefix)
	}
	chord, ok := parseChord(chord)
	if !ok {
		return
	}
	action, _, _ = strings.Cut(action, ":")
	switch action {
	case "copy_to_clipboard":
		profile.bind(ActionCopy, chord)
	case "paste_from_clipboard":
		profile.bind(ActionPaste, chord)
	case "increase_font_size":
		profile.bind(ActionZoomIn, chord)
	case "decrease_font_size":
		profile.bind(ActionZoomOut, chord)
	case "reset_font_size":
		profile.bind(ActionZoomReset, chord)
	}
}

func readGhosttyFile(path string, depth int, into *ghosttyConfig) {
	if depth > maxIncludeDepth {
		return
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return
	}
	for _, line := range strings.Split(string(data), "\n") {
		key, value, found := strings.Cut(strings.TrimSpace(line), "=")
		key, value = strings.TrimSpace(key), strings.Trim(strings.TrimSpace(value), `"`)
		if !found || strings.HasPrefix(key, "#") {
			continue
		}
		switch key {
		case "config-file":
			readGhosttyFile(resolveFrom(path, strings.TrimPrefix(value, "?")), depth+1, into)
		case "keybind":
			into.binds = append(into.binds, value)
		case "font-family":
			if _, set := into.values[key]; !set {
				into.values[key] = value
			}
		case "palette":
			index, color, _ := strings.Cut(value, "=")
			if n, err := strconv.Atoi(strings.TrimSpace(index)); err == nil && n >= 0 && n < 16 {
				into.palette[n] = color
			}
		default:
			into.values[key] = value
		}
	}
}

// A "light:A,dark:B" theme follows the system scheme in ghostty. The dock takes the dark one, or the light one when that is all there is.
func ghosttyThemeFile(s sources, theme string) string {
	var plain, dark, light string
	for _, part := range strings.Split(theme, ",") {
		mode, name, found := strings.Cut(part, ":")
		switch {
		case found && strings.TrimSpace(mode) == "dark":
			dark = strings.TrimSpace(name)
		case found && strings.TrimSpace(mode) == "light":
			light = strings.TrimSpace(name)
		default:
			plain = strings.TrimSpace(part)
		}
	}
	theme = plain
	for _, preferred := range []string{dark, light} {
		if preferred != "" {
			theme = preferred
			break
		}
	}
	if theme == "" {
		return ""
	}
	if filepath.IsAbs(theme) {
		return theme
	}
	candidates := s.configFiles(filepath.Join("ghostty", "themes", theme))
	if dir := os.Getenv("GHOSTTY_RESOURCES_DIR"); dir != "" {
		candidates = append(candidates, filepath.Join(dir, "themes", theme))
	}
	candidates = append(candidates, s.dataFiles(filepath.Join("ghostty", "themes", theme))...)
	for _, path := range candidates {
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}
	return ""
}
