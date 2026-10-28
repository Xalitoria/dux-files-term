package termtheme

import (
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/BurntSushi/toml"
)

var (
	luaHex        = regexp.MustCompile(`['"](#[0-9a-fA-F]{6})['"]`)
	luaKeyValue   = regexp.MustCompile(`\b([a-z_]+)\s*=\s*['"]?([^'",\s}]+)`)
	luaColorsOpen = regexp.MustCompile(`\bcolors\s*=\s*\{`)
	luaTable      = regexp.MustCompile(`\{[^{}]*\}`)
	luaKey        = regexp.MustCompile(`\bkey\s*=\s*['"]([^'"]+)['"]`)
	luaMods       = regexp.MustCompile(`\bmods\s*=\s*['"]([^'"]*)['"]`)
	luaAction     = regexp.MustCompile(`\baction\s*=\s*(?:wezterm\.)?act(?:ion)?\.(\w+)(.*)`)
	luaFont       = regexp.MustCompile(`wezterm\.font(?:_with_fallback)?\s*[({]*\s*['"]([^'"]+)['"]`)
	luaBoundary   = regexp.MustCompile(`\bselection_word_boundary\s*=\s*"((?:[^"\\]|\\.)*)"`)
	luaPalettes   = map[string]*regexp.Regexp{
		"ansi":    regexp.MustCompile(`ansi\s*=\s*\{([^}]*)\}`),
		"brights": regexp.MustCompile(`brights\s*=\s*\{([^}]*)\}`),
	}
)

type weztermScheme struct {
	Foreground string
	Background string
	CursorBg   string `toml:"cursor_bg"`
	Ansi       []string
	Brights    []string
}

// WezTerm is configured in Lua, so this reads the literal settings of the colors table
// and the color scheme files in its colors folder. Built-in scheme names are not known here.
func readWezterm(s sources) (Profile, bool) {
	data, path, ok := readFirst(append(s.configFiles("wezterm/wezterm.lua"), filepath.Join(s.home, ".wezterm.lua")))
	if !ok {
		return Profile{}, false
	}
	source := string(data)
	table := luaColorsTable(source)
	fields := luaTopLevel(table)

	var scheme weztermScheme
	if name := luaSetting(source, "color_scheme"); name != "" {
		_, _ = toml.DecodeFile(filepath.Join(filepath.Dir(path), "colors", name+".toml"), &struct {
			Colors *weztermScheme
		}{&scheme})
	}
	pick := func(key, fallback string) string {
		if strings.HasPrefix(fields[key], "#") {
			return fields[key]
		}
		return fallback
	}

	colors := newProfile()
	colors.Fg = tint(pick("foreground", scheme.Foreground))
	colors.Bg = tint(pick("background", scheme.Background))
	colors.Cursor = tint(pick("cursor_bg", scheme.CursorBg))
	colors.SelectionFg = tint(pick("selection_fg", ""))
	colors.SelectionBg = tint(pick("selection_bg", ""))
	for i, list := range [2][]string{luaList(table, "ansi", scheme.Ansi), luaList(table, "brights", scheme.Brights)} {
		for j := 0; j < min(len(list), 8); j++ {
			colors.ANSI[i*8+j] = tint(list[j])
		}
	}
	if opacity, err := strconv.ParseFloat(luaSetting(source, "window_background_opacity"), 64); err == nil {
		colors.Opacity = clampOpacity(opacity)
	}
	colors.FontSize, _ = strconv.ParseFloat(luaSetting(source, "font_size"), 64)
	if match := luaFont.FindStringSubmatch(source); match != nil {
		colors.FontFamily = namedFamily(match[1])
	}
	if match := luaBoundary.FindStringSubmatch(source); match != nil {
		colors.WordDelims = match[1]
		if text, err := strconv.Unquote(`"` + match[1] + `"`); err == nil {
			colors.WordDelims = text
		}
	}
	style := luaSetting(source, "default_cursor_style")
	colors.setCursor(style, strings.HasPrefix(style, "Blinking"))
	weztermKeys(&colors, source)
	return colors, colors.usable()
}

// Reads the `{ key = 'c', mods = 'CTRL|SHIFT', action = act.CopyTo 'Clipboard' }` entries that fit on a table of their own.
func weztermKeys(profile *Profile, source string) {
	for _, entry := range luaTable.FindAllString(source, -1) {
		key, mods, action := luaKey.FindStringSubmatch(entry), luaMods.FindStringSubmatch(entry), luaAction.FindStringSubmatch(entry)
		if key == nil || action == nil {
			continue
		}
		var held []string
		if mods != nil {
			held = strings.Split(strings.ReplaceAll(mods[1], "NONE", ""), "|")
		}
		name := key[1]
		if _, bare, found := strings.Cut(name, ":"); found {
			name = bare
		}
		chord, ok := chordOf(held, name)
		if !ok {
			continue
		}
		switch {
		case action[1] == "CopyTo" && strings.Contains(action[2], "Clipboard"):
			profile.bind(ActionCopy, chord)
		case action[1] == "PasteFrom" && strings.Contains(action[2], "Clipboard"):
			profile.bind(ActionPaste, chord)
		case action[1] == "IncreaseFontSize":
			profile.bind(ActionZoomIn, chord)
		case action[1] == "DecreaseFontSize":
			profile.bind(ActionZoomOut, chord)
		case action[1] == "ResetFontSize":
			profile.bind(ActionZoomReset, chord)
		}
	}
}

func luaSetting(source, key string) string {
	match := regexp.MustCompile(`\b` + key + `\s*=\s*['"]?([^'",\s}]+)`).FindStringSubmatch(source)
	if match == nil {
		return ""
	}
	return match[1]
}

// The body of the first `colors = { ... }` table, found by matching braces.
func luaColorsTable(source string) string {
	loc := luaColorsOpen.FindStringIndex(source)
	if loc == nil {
		return ""
	}
	depth := 1
	for i := loc[1]; i < len(source); i++ {
		switch source[i] {
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return source[loc[1]:i]
			}
		}
	}
	return ""
}

// Keys that sit directly in the table, not in a table inside it such as tab_bar.
func luaTopLevel(body string) map[string]string {
	fields := map[string]string{}
	depth := 0
	for _, line := range strings.Split(body, "\n") {
		for _, match := range luaKeyValue.FindAllStringSubmatchIndex(line, -1) {
			before := line[:match[0]]
			if depth+strings.Count(before, "{")-strings.Count(before, "}") == 0 {
				fields[line[match[2]:match[3]]] = line[match[4]:match[5]]
			}
		}
		depth += strings.Count(line, "{") - strings.Count(line, "}")
	}
	return fields
}

func luaList(table, name string, fallback []string) []string {
	match := luaPalettes[name].FindStringSubmatch(table)
	if match == nil {
		return fallback
	}
	var list []string
	for _, hex := range luaHex.FindAllStringSubmatch(match[1], -1) {
		list = append(list, hex[1])
	}
	return list
}
