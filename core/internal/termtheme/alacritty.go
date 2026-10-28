package termtheme

import (
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
)

type alacrittyANSI struct {
	Black, Red, Green, Yellow, Blue, Magenta, Cyan, White string
}

func (a alacrittyANSI) list() [8]string {
	return [8]string{a.Black, a.Red, a.Green, a.Yellow, a.Blue, a.Magenta, a.Cyan, a.White}
}

type alacrittyConfig struct {
	Import  []string
	General struct{ Import []string }
	Colors  struct {
		Primary   struct{ Background, Foreground string }
		Cursor    struct{ Cursor string }
		Selection struct{ Text, Background string }
		Normal    alacrittyANSI
		Bright    alacrittyANSI
	}
	Window    struct{ Opacity *float64 }
	Selection struct {
		SemanticEscapeChars string `toml:"semantic_escape_chars"`
		SaveToClipboard     bool   `toml:"save_to_clipboard"`
	}
	Cursor struct {
		Style struct{ Shape, Blinking string }
	}
	Font struct {
		Size   float64
		Normal struct{ Family string }
	}
	Keyboard struct {
		Bindings []struct{ Key, Mods, Action, Mode string }
	}
}

func readAlacritty(s sources) (Profile, bool) {
	home := filepath.Join(s.home, ".alacritty.toml")
	_, path, ok := readFirst(append(s.configFiles("alacritty/alacritty.toml"), home))
	if !ok {
		return Profile{}, false
	}
	var config alacrittyConfig
	loadAlacritty(s, path, 0, &config)

	profile := newProfile()
	profile.Fg = tint(config.Colors.Primary.Foreground)
	profile.Bg = tint(config.Colors.Primary.Background)
	profile.Cursor = tint(config.Colors.Cursor.Cursor)
	profile.SelectionFg = tint(config.Colors.Selection.Text)
	profile.SelectionBg = tint(config.Colors.Selection.Background)
	normal, bright := config.Colors.Normal.list(), config.Colors.Bright.list()
	for i := range 8 {
		profile.ANSI[i] = tint(normal[i])
		profile.ANSI[i+8] = tint(bright[i])
	}
	if config.Window.Opacity != nil {
		profile.Opacity = clampOpacity(*config.Window.Opacity)
	}
	profile.WordDelims = config.Selection.SemanticEscapeChars
	profile.CopyOnSelect = config.Selection.SaveToClipboard
	profile.setCursor(config.Cursor.Style.Shape, truthy(config.Cursor.Style.Blinking))
	profile.FontFamily = namedFamily(config.Font.Normal.Family)
	profile.FontSize = config.Font.Size
	for _, binding := range config.Keyboard.Bindings {
		if !alwaysActive(binding.Mode) {
			continue
		}
		mods := strings.Split(strings.ReplaceAll(binding.Mods, "None", ""), "|")
		chord, ok := chordOf(mods, binding.Key)
		if !ok {
			continue
		}
		switch binding.Action {
		case "Copy":
			profile.bind(ActionCopy, chord)
		case "Paste":
			profile.bind(ActionPaste, chord)
		case "IncreaseFontSize":
			profile.bind(ActionZoomIn, chord)
		case "DecreaseFontSize":
			profile.bind(ActionZoomOut, chord)
		case "ResetFontSize":
			profile.bind(ActionZoomReset, chord)
		}
	}
	return profile, profile.usable()
}

// A binding that needs a mode such as Vi or Search does not apply while typing in the shell.
func alwaysActive(mode string) bool {
	for _, term := range strings.Split(mode, "|") {
		if term = strings.TrimSpace(term); term != "" && !strings.HasPrefix(term, "~") {
			return false
		}
	}
	return true
}

// Imports load first so the file that imports them overrides them.
func loadAlacritty(s sources, path string, depth int, into *alacrittyConfig) {
	if depth > maxIncludeDepth {
		return
	}
	var head alacrittyConfig
	if _, err := toml.DecodeFile(path, &head); err != nil {
		return
	}
	for _, included := range append(head.Import, head.General.Import...) {
		if strings.HasPrefix(included, "~/") {
			included = filepath.Join(s.home, included[2:])
		}
		loadAlacritty(s, resolveFrom(path, included), depth+1, into)
	}
	_, _ = toml.DecodeFile(path, into)
}
