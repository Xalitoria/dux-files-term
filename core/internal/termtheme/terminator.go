package termtheme

import (
	"path/filepath"
	"slices"
	"strings"
)

func readTerminator(s sources) (Profile, bool) {
	data, _, ok := readFirst(append(s.configFiles("terminator/config"), filepath.Join(s.home, ".terminator", "config")))
	if !ok {
		return Profile{}, false
	}
	profiles := terminatorProfiles(string(data))
	profile := profiles["default"]
	if profile["palette"] == "" {
		names := make([]string, 0, len(profiles))
		for name := range profiles {
			names = append(names, name)
		}
		slices.Sort(names)
		for _, name := range names {
			if profiles[name]["palette"] != "" {
				profile = profiles[name]
				break
			}
		}
	}

	colors := newProfile()
	colors.Fg = tint(profile["foreground_color"])
	colors.Bg = tint(profile["background_color"])
	colors.Cursor = tint(profile["cursor_color"])
	for i, entry := range strings.Split(profile["palette"], ":") {
		if i < len(colors.ANSI) {
			colors.ANSI[i] = tint(entry)
		}
	}
	if profile["background_type"] == "transparent" {
		if opacity, ok := parseOpacity(profile["background_darkness"]); ok {
			colors.Opacity = opacity
		}
	}
	return colors, colors.usable()
}

// Only the [[profile]] sections inside [profiles] count, a [[default]] of a layout is something else.
func terminatorProfiles(config string) map[string]map[string]string {
	profiles := map[string]map[string]string{}
	inProfiles := false
	var current map[string]string
	for _, line := range strings.Split(config, "\n") {
		line = strings.TrimSpace(line)
		switch {
		case strings.HasPrefix(line, "[[["):
			current = nil
		case strings.HasPrefix(line, "[["):
			current = nil
			if inProfiles {
				name := strings.Trim(line, "[] ")
				current = map[string]string{}
				profiles[name] = current
			}
		case strings.HasPrefix(line, "["):
			inProfiles = strings.Trim(line, "[] ") == "profiles"
			current = nil
		default:
			key, value, found := strings.Cut(line, "=")
			if found && current != nil {
				current[strings.TrimSpace(key)] = strings.Trim(strings.TrimSpace(value), `"'`)
			}
		}
	}
	return profiles
}
