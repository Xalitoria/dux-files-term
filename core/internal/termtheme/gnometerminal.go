package termtheme

import (
	"context"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const gnomeProfiles = "/org/gnome/terminal/legacy/profiles:/"

var (
	quoted = regexp.MustCompile(`'([^']*)'`)
	number = regexp.MustCompile(`\d+$`)
)

// GNOME Terminal keeps its profiles in dconf. One dump of them is read through the dconf tool.
func readGnomeTerminal(s sources) (Profile, bool) {
	if s.dconf == nil {
		return Profile{}, false
	}
	dump := parseINI([]byte(s.dconf(gnomeProfiles)))
	id := firstQuoted(dump["/"]["default"])
	if id == "" {
		id = firstQuoted(dump["/"]["list"])
	}
	profile := dump[":"+id]
	if id == "" || profile == nil {
		return Profile{}, false
	}

	colors := newProfile()
	if profile["use-theme-colors"] != "true" {
		colors.Fg = tint(profile["foreground-color"])
		colors.Bg = tint(profile["background-color"])
		for i, match := range quoted.FindAllStringSubmatch(profile["palette"], -1) {
			if i < len(colors.ANSI) {
				colors.ANSI[i] = tint(match[1])
			}
		}
	}
	if profile["cursor-colors-set"] == "true" {
		colors.Cursor = tint(profile["cursor-background-color"])
	}
	if profile["use-transparent-background"] == "true" {
		if percent, err := strconv.Atoi(number.FindString(profile["background-transparency-percent"])); err == nil {
			colors.Opacity = clampOpacity(1 - float64(percent)/100)
		}
	}
	return colors, colors.usable()
}

func firstQuoted(text string) string {
	match := quoted.FindStringSubmatch(text)
	if match == nil {
		return strings.Trim(text, "'")
	}
	return match[1]
}

func dconfDump(path string) string {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	out, err := exec.CommandContext(ctx, "dconf", "dump", path).Output()
	if err != nil {
		return ""
	}
	return string(out)
}
