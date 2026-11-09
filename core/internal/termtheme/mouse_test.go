package termtheme

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClicks(t *testing.T) {
	tests := []struct {
		name     string
		terminal string
		files    map[string]string
		right    string
		middle   string
	}{
		{"kitty default", "kitty", map[string]string{".config/kitty/kitty.conf": "background #101010\n"}, ClickExtend, ClickPrimaryPaste},
		{"kitty remapped", "kitty", map[string]string{".config/kitty/kitty.conf": "background #101010\nmouse_map right press ungrabbed paste_from_clipboard\nmouse_map middle release ungrabbed no_op\n"}, ClickPaste, ClickNone},
		{"kitty move-end still extends", "kitty", map[string]string{".config/kitty/kitty.conf": "background #101010\nmouse_map right press ungrabbed mouse_selection move-end\n"}, ClickExtend, ClickPrimaryPaste},
		{"kitty ignores grabbed and modified maps", "kitty", map[string]string{".config/kitty/kitty.conf": "background #101010\nmouse_map right press grabbed no_op\nmouse_map ctrl+middle release ungrabbed no_op\n"}, ClickExtend, ClickPrimaryPaste},
		{"kitty clear all", "kitty", map[string]string{".config/kitty/kitty.conf": "background #101010\nclear_all_mouse_actions yes\nmouse_map middle release ungrabbed paste_from_selection\n"}, ClickNone, ClickPrimaryPaste},
		{"ghostty default", "ghostty", map[string]string{".config/ghostty/config": "background = #101010\n"}, ClickMenu, ClickPrimaryPaste},
		{"ghostty configured", "ghostty", map[string]string{".config/ghostty/config": "background = #101010\nright-click-action = copy-or-paste\nmiddle-click-action = ignore\n"}, ClickCopyOrPaste, ClickNone},
		{"alacritty default", "alacritty", map[string]string{".config/alacritty/alacritty.toml": "[colors.primary]\nbackground = \"#101010\"\n"}, ClickExtend, ClickPrimaryPaste},
		{"alacritty bindings", "alacritty", map[string]string{".config/alacritty/alacritty.toml": `[colors.primary]
background = "#101010"
[[mouse.bindings]]
mouse = "Right"
action = "Paste"
[[mouse.bindings]]
mouse = "Middle"
mods = "Control"
action = "Paste"
`}, ClickPaste, ClickPrimaryPaste},
		{"foot default", "foot", map[string]string{".config/foot/foot.ini": "[colors]\nbackground=101010\n"}, ClickExtend, ClickPrimaryPaste},
		{"foot paste on right, middle unbound", "foot", map[string]string{".config/foot/foot.ini": "[colors]\nbackground=101010\n[mouse-bindings]\nclipboard-paste=BTN_RIGHT\nprimary-paste=none\n"}, ClickPaste, ClickNone},
		{"foot extend unbound", "foot", map[string]string{".config/foot/foot.ini": "[colors]\nbackground=101010\n[mouse-bindings]\nselect-extend=none\n"}, ClickNone, ClickPrimaryPaste},
		{"terminator", "terminator", map[string]string{".config/terminator/config": "[profiles]\n  [[default]]\n    background_color = \"#101010\"\n    palette = \"#000000:#111111\"\n"}, ClickMenu, ClickPrimaryPaste},
		{"wezterm", "wezterm", map[string]string{".config/wezterm/wezterm.lua": "return { font_size = 12.0 }\n"}, ClickNone, ClickPrimaryPaste},
		{"rio leaves it to the view", "rio", map[string]string{".config/rio/config.toml": "[colors]\nbackground = \"#101010\"\n"}, "", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			profile, ok := load(fixture(t, tt.terminal, tt.files))

			require.True(t, ok)
			assert.Equal(t, tt.right, profile.RightClick, "right click")
			assert.Equal(t, tt.middle, profile.MiddleClick, "middle click")
		})
	}
}
