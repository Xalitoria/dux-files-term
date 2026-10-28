package termtheme

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type behavior struct {
	selectionFg, selectionBg string
	wordChars, wordDelims    string
	copyOnSelect             bool
	keys                     map[string][]string
	family                   string
	size                     float64
	cursorShape              string
	cursorBlink              bool
}

func TestReadersBehavior(t *testing.T) {
	tests := []struct {
		name     string
		terminal string
		files    map[string]string
		want     behavior
	}{
		{
			name:     "kitty",
			terminal: "kitty",
			files: map[string]string{
				".config/kitty/kitty.conf": `selection_foreground #111111
selection_background #222222
select_by_word_characters @-./_~?&=%+#:
copy_on_select clipboard
font_family family="JetBrainsMono Nerd Font" style=Regular
font_size 12.5
cursor_shape beam
cursor_blink_interval 0.5
kitty_mod ctrl+alt
map kitty_mod+c copy_to_clipboard
map ctrl+v paste_from_clipboard
map --when-focus-on var:IS_VIM ctrl+h paste_from_clipboard
map ctrl+shift+equal change_font_size all +2.0
map ctrl+shift+minus change_font_size all -2.0
map ctrl+shift+backspace change_font_size all 0
map ctrl+a>c new_tab
map f5 load_config_file
`,
			},
			want: behavior{
				selectionFg: "#111111", selectionBg: "#222222", wordChars: "@-./_~?&=%+#:", copyOnSelect: true,
				family: "JetBrainsMono Nerd Font", size: 12.5, cursorShape: "bar", cursorBlink: true,
				keys: map[string][]string{
					ActionCopy:      {"ctrl+alt+c"},
					ActionPaste:     {"ctrl+v"},
					ActionZoomIn:    {"ctrl+shift+plus"},
					ActionZoomOut:   {"ctrl+shift+minus"},
					ActionZoomReset: {"ctrl+shift+backspace"},
				},
			},
		},
		{
			name:     "kitty: generic fonts and the default clipboard behaviour follow nothing",
			terminal: "kitty",
			files:    map[string]string{".config/kitty/kitty.conf": "background #101010\nfont_family monospace\ncopy_on_select yes\n"},
			want:     behavior{},
		},
		{
			name:     "ghostty",
			terminal: "ghostty",
			files: map[string]string{
				".config/ghostty/config": `selection-foreground = #333333
selection-background = #444444
selection-word-chars = " ,;()"
copy-on-select = clipboard
font-family = "Iosevka Nerd Font"
font-family = "Noto Color Emoji"
font-size = 13
cursor-style = underline
cursor-style-blink = true
keybind = ctrl+c=copy_to_clipboard
keybind = performable:super+v=paste_from_clipboard
keybind = ctrl+==increase_font_size:2
keybind = ctrl+minus=decrease_font_size:1
keybind = ctrl+0=reset_font_size
keybind = ctrl+a>n=new_window
keybind = f5=reload_config
`,
			},
			want: behavior{
				selectionFg: "#333333", selectionBg: "#444444", wordDelims: " ,;()", copyOnSelect: true,
				family: "Iosevka Nerd Font", size: 13, cursorShape: "underline", cursorBlink: true,
				keys: map[string][]string{
					ActionCopy:      {"ctrl+c"},
					ActionPaste:     {"super+v"},
					ActionZoomIn:    {"ctrl+plus"},
					ActionZoomOut:   {"ctrl+minus"},
					ActionZoomReset: {"ctrl+0"},
				},
			},
		},
		{
			name:     "alacritty",
			terminal: "alacritty",
			files: map[string]string{
				".config/alacritty/alacritty.toml": `
[colors.selection]
text = "#555555"
background = "CellForeground"

[selection]
semantic_escape_chars = ",│|: ()[]{}<>"
save_to_clipboard = true

[cursor.style]
shape = "Beam"
blinking = "On"

[font]
size = 11.25
[font.normal]
family = "FiraCode Nerd Font"

[keyboard]
bindings = [
  { key = "C", mods = "Control", action = "Copy" },
  { key = "V", mods = "Control|Shift", action = "Paste" },
  { key = "Equals", mods = "Control", action = "IncreaseFontSize" },
  { key = "Key0", mods = "Control", action = "ResetFontSize" },
  { key = "C", mods = "Control", mode = "Vi", action = "ClearSelection" },
  { key = "Y", mods = "Control", mode = "Vi", action = "Copy" },
  { key = "N", mods = "Control", mode = "~Vi|~Search", action = "Paste" },
]
`,
			},
			want: behavior{
				selectionFg: "#555555", wordDelims: ",│|: ()[]{}<>", copyOnSelect: true,
				family: "FiraCode Nerd Font", size: 11.25, cursorShape: "bar", cursorBlink: true,
				keys: map[string][]string{
					ActionCopy:      {"ctrl+c"},
					ActionPaste:     {"ctrl+shift+v", "ctrl+n"},
					ActionZoomIn:    {"ctrl+plus"},
					ActionZoomReset: {"ctrl+0"},
				},
			},
		},
		{
			name:     "foot",
			terminal: "foot",
			files: map[string]string{
				".config/foot/foot.ini": `[main]
font=Hack Nerd Font:size=10:weight=bold, Noto Color Emoji:size=10
word-delimiters=,│|:"'()[]{}<>
selection-target=both

[cursor]
style=underline
blink=no

[colors]
selection-foreground=666666
selection-background=777777

[key-bindings]
clipboard-copy=Control+c XF86Copy
clipboard-paste=Control+Shift+v
font-increase=Control+plus Control+KP_Add
font-decrease=Control+minus
font-reset=Control+0
`,
			},
			want: behavior{
				selectionFg: "#666666", selectionBg: "#777777", wordDelims: `,│|:"'()[]{}<>`, copyOnSelect: true,
				family: "Hack Nerd Font", size: 10, cursorShape: "underline",
				keys: map[string][]string{
					ActionCopy:      {"ctrl+c"},
					ActionPaste:     {"ctrl+shift+v"},
					ActionZoomIn:    {"ctrl+plus"},
					ActionZoomOut:   {"ctrl+minus"},
					ActionZoomReset: {"ctrl+0"},
				},
			},
		},
		{
			name:     "wezterm",
			terminal: "wezterm",
			files: map[string]string{
				".config/wezterm/wezterm.lua": `local wezterm = require 'wezterm'
local act = wezterm.action
local config = {
  font = wezterm.font_with_fallback { 'Berkeley Mono', 'Noto Color Emoji' },
  font_size = 14.0,
  default_cursor_style = 'BlinkingBar',
  selection_word_boundary = " \t\n{}[]()",
  colors = { selection_fg = '#888888', selection_bg = '#999999' },
  keys = {
    { key = 'c', mods = 'CTRL', action = act.CopyTo 'Clipboard' },
    { key = 'y', mods = 'CTRL', action = act.CopyTo 'PrimarySelection' },
    { key = 'v', mods = 'CTRL|SHIFT', action = act.PasteFrom("Clipboard") },
    { key = 'phys:Equal', mods = 'CTRL', action = act.IncreaseFontSize },
    { key = '-', mods = 'CTRL', action = wezterm.action.DecreaseFontSize },
    { key = '0', mods = 'CTRL', action = act.ResetFontSize },
    { key = 'a', mods = 'LEADER', action = act.CopyTo 'Clipboard' },
  },
}
return config
`,
			},
			want: behavior{
				selectionFg: "#888888", selectionBg: "#999999", wordDelims: " \t\n{}[]()",
				family: "Berkeley Mono", size: 14, cursorShape: "bar", cursorBlink: true,
				keys: map[string][]string{
					ActionCopy:      {"ctrl+c"},
					ActionPaste:     {"ctrl+shift+v"},
					ActionZoomIn:    {"ctrl+plus"},
					ActionZoomOut:   {"ctrl+minus"},
					ActionZoomReset: {"ctrl+0"},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := fixture(t, tt.terminal, tt.files)

			profile, ok := load(s)

			require.True(t, ok)
			assert.Equal(t, tt.want.selectionFg, hexOf(profile.SelectionFg), "selection foreground")
			assert.Equal(t, tt.want.selectionBg, hexOf(profile.SelectionBg), "selection background")
			assert.Equal(t, tt.want.wordChars, profile.WordChars)
			assert.Equal(t, tt.want.wordDelims, profile.WordDelims)
			assert.Equal(t, tt.want.copyOnSelect, profile.CopyOnSelect)
			assert.Equal(t, tt.want.family, profile.FontFamily)
			assert.InDelta(t, tt.want.size, profile.FontSize, 0.0001)
			assert.Equal(t, tt.want.cursorShape, profile.CursorShape)
			assert.Equal(t, tt.want.cursorBlink, profile.CursorBlink)
			for action, chords := range tt.want.keys {
				switch tt.terminal {
				case "kitty":
					assert.ElementsMatch(t, chords, profile.Keys[action], action)
				default:
					assert.Subset(t, profile.Keys[action], chords, action)
				}
			}
		})
	}
}

func TestParseChord(t *testing.T) {
	tests := map[string]string{
		"ctrl+shift+c":    "ctrl+shift+c",
		"Control+Shift+c": "ctrl+shift+c",
		"shift+ctrl+c":    "ctrl+shift+c",
		"ctrl+equal":      "ctrl+plus",
		"ctrl+=":          "ctrl+plus",
		"ctrl++":          "ctrl+plus",
		"ctrl+-":          "ctrl+minus",
		"cmd+KP_Add":      "super+plus",
		"Mod1+Prior":      "alt+pageup",
		"ctrl+page_down":  "ctrl+pagedown",
		"ctrl+arrow_up":   "ctrl+up",
		"ctrl+Return":     "ctrl+enter",
		"ctrl+Key0":       "ctrl+0",
		"shift+insert":    "shift+insert",
		"ctrl+a>c":        "",
		"hyper+c":         "",
		"":                "",
		"ctrl+shift+":     "",
	}
	for text, want := range tests {
		chord, ok := parseChord(text)
		assert.Equal(t, want != "", ok, text)
		assert.Equal(t, want, chord, text)
	}
}
