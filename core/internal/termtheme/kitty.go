package termtheme

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"unicode"
)

type kittyConfig struct {
	settings  map[string]string
	maps      [][]string
	mouseMaps [][]string
}

func readKitty(s sources) (Profile, bool) {
	candidates := s.configFiles(filepath.Join("kitty", "kitty.conf"))
	if dir := os.Getenv("KITTY_CONFIG_DIRECTORY"); dir != "" {
		candidates = []string{filepath.Join(dir, "kitty.conf")}
	}
	_, path, ok := readFirst(candidates)
	if !ok {
		return Profile{}, false
	}
	config := kittyConfig{settings: map[string]string{}}
	readKittyFile(path, 0, &config)
	value := func(key string) string {
		return firstField(config.settings[key])
	}

	profile := newProfile()
	profile.Fg = tint(value("foreground"))
	profile.Bg = tint(value("background"))
	profile.Cursor = tint(value("cursor"))
	profile.SelectionFg = tint(value("selection_foreground"))
	profile.SelectionBg = tint(value("selection_background"))
	for i := range profile.ANSI {
		profile.ANSI[i] = tint(value("color" + strconv.Itoa(i)))
	}
	if opacity, ok := parseOpacity(value("background_opacity")); ok {
		profile.Opacity = opacity
	}
	profile.WordChars = value("select_by_word_characters")
	profile.CopyOnSelect = value("copy_on_select") == "clipboard"
	profile.FontFamily = kittyFamily(config.settings["font_family"])
	profile.FontSize, _ = strconv.ParseFloat(value("font_size"), 64)
	blink := value("cursor_blink_interval")
	profile.setCursor(value("cursor_shape"), blink != "" && blink != "0")
	kittyKeys(&profile, config)
	kittyClicks(&profile, config)
	profile.ScrollbackLines = parseLines(value("scrollback_lines"))
	profile.WheelLines = parsePositive(value("wheel_scroll_multiplier"))
	kittyBell(&profile, config)
	return profile, profile.usable()
}

// Later lines win, and an include is read where it appears, as kitty does.
func readKittyFile(path string, depth int, into *kittyConfig) {
	if depth > maxIncludeDepth {
		return
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return
	}
	for _, line := range strings.Split(string(data), "\n") {
		fields := strings.Fields(line)
		if len(fields) < 2 || strings.HasPrefix(fields[0], "#") {
			continue
		}
		switch fields[0] {
		case "include":
			readKittyFile(resolveFrom(path, fields[1]), depth+1, into)
		case "map":
			into.maps = append(into.maps, fields[1:])
		case "mouse_map":
			into.mouseMaps = append(into.mouseMaps, fields[1:])
		default:
			into.settings[fields[0]] = strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(line), fields[0]))
		}
	}
}

func kittyKeys(profile *Profile, config kittyConfig) {
	mod := "ctrl+shift"
	if custom := firstField(config.settings["kitty_mod"]); custom != "" {
		mod = custom
	}
	for _, fields := range config.maps {
		// A mapping with options, such as --when-focus-on, only applies under conditions.
		if len(fields) < 2 || strings.HasPrefix(fields[0], "--") {
			continue
		}
		chord, ok := parseChord(strings.ReplaceAll(fields[0], "kitty_mod", mod))
		if !ok {
			continue
		}
		switch fields[1] {
		case "copy_to_clipboard", "copy_or_interrupt", "copy_and_clear_or_interrupt":
			profile.bind(ActionCopy, chord)
		case "paste_from_clipboard":
			profile.bind(ActionPaste, chord)
		case "show_scrollback":
			profile.bind(ActionSearch, chord)
		case "change_font_size":
			if action := kittyZoom(fields[2:]); action != "" {
				profile.bind(action, chord)
			}
		}
	}
}

// Kitty's bell plays a sound, marks the tab and alerts the window unless the config says otherwise.
func kittyBell(profile *Profile, config kittyConfig) {
	audio, visual, alert := config.settings["enable_audio_bell"], config.settings["visual_bell_duration"], config.settings["window_alert_on_bell"]
	onTab, tabSet := config.settings["bell_on_tab"]
	if !anySet(audio, visual, alert) && !tabSet {
		return
	}
	onTab = strings.Trim(strings.TrimSpace(onTab), `"'`)
	marksTab := !tabSet || (onTab != "" && onTab != "no" && onTab != "none")
	profile.Bell = bellOf(boolOr(audio, true), parsePositive(firstField(visual)) > 0, boolOr(alert, true) || marksTab)
}

// Only plain clicks of the left-out buttons count. Mappings with modifiers, or for a grabbed mouse, are the program's.
func kittyClicks(profile *Profile, config kittyConfig) {
	if truthy(firstField(config.settings["clear_all_mouse_actions"])) {
		profile.RightClick, profile.MiddleClick = ClickNone, ClickNone
	}
	for _, fields := range config.mouseMaps {
		if len(fields) < 3 || !strings.Contains(fields[2], "ungrabbed") {
			continue
		}
		switch fields[0] {
		case "right":
			profile.RightClick = kittyClick(fields[3:])
		case "middle":
			profile.MiddleClick = kittyClick(fields[3:])
		}
	}
}

// `change_font_size all +2.0` grows, `-2.0` shrinks and `0` resets. Anything else is a fixed size.
func kittyZoom(args []string) string {
	if len(args) == 0 {
		return ""
	}
	switch change := args[len(args)-1]; {
	case strings.HasPrefix(change, "+"):
		return ActionZoomIn
	case strings.HasPrefix(change, "-"):
		return ActionZoomOut
	case change == "0":
		return ActionZoomReset
	}
	return ""
}

// kitty takes a bare name or key=value pairs like `family='Name' postscript_name=Name`, with either quote.
func kittyFamily(text string) string {
	text = strings.TrimSpace(text)
	if !strings.Contains(text, "=") {
		return namedFamily(strings.Trim(text, `"'`))
	}
	for _, token := range kittyTokens(text) {
		if key, value, ok := strings.Cut(token, "="); ok && key == "family" {
			return namedFamily(value)
		}
	}
	return ""
}

// kittyTokens splits on whitespace outside quotes and drops the quotes.
func kittyTokens(text string) []string {
	var tokens []string
	var token strings.Builder
	var quote rune
	started := false
	flush := func() {
		if started {
			tokens = append(tokens, token.String())
		}
		token.Reset()
		started = false
	}
	for _, r := range text {
		switch {
		case quote != 0 && r == quote:
			quote = 0
		case quote != 0:
			token.WriteRune(r)
		case r == '"' || r == '\'':
			quote, started = r, true
		case unicode.IsSpace(r):
			flush()
		default:
			token.WriteRune(r)
			started = true
		}
	}
	flush()
	return tokens
}
