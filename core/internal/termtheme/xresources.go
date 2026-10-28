package termtheme

import (
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

var (
	resourceKey = regexp.MustCompile(`^(color(\d+)|foreground|background|cursorcolor)$`)
	includeLine = regexp.MustCompile(`^#\s*include\s+"([^"]+)"`)
)

// Specific resources (URxvt.color1) beat loose ones (*color1), whatever their order.
type resources struct {
	loose, specific map[string]string
}

// xterm and urxvt take their colours from the X resources, so the apps argument is their class names.
func readXresources(apps ...string) reader {
	return func(s sources) (Profile, bool) {
		files := []string{filepath.Join(s.home, ".Xresources"), filepath.Join(s.home, ".Xdefaults")}
		found := resources{loose: map[string]string{}, specific: map[string]string{}}
		read := false
		for _, path := range files {
			if readResources(path, 0, apps, found) {
				read = true
				break
			}
		}
		if !read {
			return Profile{}, false
		}
		lookup := func(key string) string {
			if value := found.specific[key]; value != "" {
				return value
			}
			return found.loose[key]
		}

		colors := newProfile()
		colors.Fg = tint(lookup("foreground"))
		colors.Bg = tint(lookup("background"))
		colors.Cursor = tint(lookup("cursorcolor"))
		for i := range colors.ANSI {
			colors.ANSI[i] = tint(lookup("color" + strconv.Itoa(i)))
		}
		return colors, colors.usable()
	}
}

func readResources(path string, depth int, apps []string, into resources) bool {
	data, err := os.ReadFile(path)
	if err != nil || depth > maxIncludeDepth {
		return false
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if match := includeLine.FindStringSubmatch(line); match != nil {
			readResources(resolveFrom(path, match[1]), depth+1, apps, into)
			continue
		}
		name, value, found := strings.Cut(line, ":")
		if !found || strings.HasPrefix(line, "!") || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.FieldsFunc(strings.ToLower(strings.TrimSpace(name)), func(r rune) bool { return r == '.' || r == '*' })
		if len(parts) == 0 || !resourceKey.MatchString(parts[len(parts)-1]) {
			continue
		}
		key := parts[len(parts)-1]
		app := ""
		if !strings.HasPrefix(name, "*") && !strings.HasPrefix(name, ".") && len(parts) > 1 {
			app = parts[0]
		}
		switch {
		case app == "":
			into.loose[key] = strings.TrimSpace(value)
		case containsFold(apps, app):
			into.specific[key] = strings.TrimSpace(value)
		}
	}
	return true
}

func containsFold(list []string, item string) bool {
	for _, entry := range list {
		if strings.EqualFold(entry, item) {
			return true
		}
	}
	return false
}
