package termtheme

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSettings(t *testing.T) {
	tests := []struct {
		name       string
		terminal   string
		files      map[string]string
		dconf      string
		scrollback int
		wheel      float64
		bell       []string
	}{
		{name: "kitty default", terminal: "kitty", files: map[string]string{".config/kitty/kitty.conf": "background #101010\n"}, scrollback: 2000, wheel: 5, bell: []string{BellAudio, BellAttention}},
		{name: "kitty configured", terminal: "kitty", files: map[string]string{".config/kitty/kitty.conf": "scrollback_lines -1\nwheel_scroll_multiplier 2.5\nenable_audio_bell no\nvisual_bell_duration 0.2\nwindow_alert_on_bell no\nbell_on_tab no\n"}, scrollback: UnlimitedScrollback, wheel: 2.5, bell: []string{BellVisual}},
		{name: "kitty keeps the rest of its bell", terminal: "kitty", files: map[string]string{".config/kitty/kitty.conf": "window_alert_on_bell no\nbell_on_tab no\n"}, scrollback: 2000, wheel: 5, bell: []string{BellAudio}},
		{name: "ghostty default", terminal: "ghostty", files: map[string]string{".config/ghostty/config": "background = #101010\n"}, scrollback: UnlimitedScrollback, wheel: 3, bell: []string{BellAttention}},
		{name: "ghostty configured", terminal: "ghostty", files: map[string]string{".config/ghostty/config": "scrollback-limit-lines = 5000\nmouse-scroll-multiplier = precision:1,discrete:2\nbell-features = audio,no-attention,no-title\n"}, scrollback: 5000, wheel: 2, bell: []string{BellAudio}},
		{name: "alacritty default", terminal: "alacritty", files: map[string]string{".config/alacritty/alacritty.toml": "[colors.primary]\nbackground = \"#101010\"\n"}, scrollback: 10000, wheel: 3, bell: []string{}},
		{name: "alacritty configured", terminal: "alacritty", files: map[string]string{".config/alacritty/alacritty.toml": "[scrolling]\nhistory = 20000\nmultiplier = 5\n[bell]\nduration = 100\n"}, scrollback: 20000, wheel: 5, bell: []string{BellVisual}},
		{name: "foot default", terminal: "foot", files: map[string]string{".config/foot/foot.ini": "[colors]\nbackground=101010\n"}, scrollback: 1000, wheel: 3, bell: []string{}},
		{name: "foot configured", terminal: "foot", files: map[string]string{".config/foot/foot.ini": "[scrollback]\nlines=4000\nmultiplier=1.5\n[bell]\nurgent=yes\n"}, scrollback: 4000, wheel: 1.5, bell: []string{BellAttention}},
		{name: "wezterm default", terminal: "wezterm", files: map[string]string{".config/wezterm/wezterm.lua": "return { font_size = 12.0 }\n"}, scrollback: 3500, wheel: 3, bell: []string{BellAudio}},
		{name: "wezterm configured", terminal: "wezterm", files: map[string]string{".config/wezterm/wezterm.lua": "return {\n  scrollback_lines = 7000,\n  audible_bell = 'Disabled',\n  visual_bell = { fade_in_duration_ms = 75, fade_out_duration_ms = 75 },\n}\n"}, scrollback: 7000, wheel: 3, bell: []string{BellVisual}},
		{name: "terminator default", terminal: "terminator", files: map[string]string{".config/terminator/config": "[profiles]\n  [[default]]\n    background_color = \"#101010\"\n    palette = \"#000000:#111111\"\n"}, scrollback: 500, wheel: 3, bell: []string{BellAttention}},
		{name: "terminator configured", terminal: "terminator", files: map[string]string{".config/terminator/config": "[profiles]\n  [[default]]\n    palette = \"#000000:#111111\"\n    scrollback_infinite = True\n    audible_bell = True\n    icon_bell = False\n"}, scrollback: UnlimitedScrollback, wheel: 3, bell: []string{BellAudio}},
		{name: "xfce default", terminal: "xfce4-terminal", files: map[string]string{".config/xfce4/terminal/terminalrc": "[Configuration]\nColorBackground=#101010\n"}, scrollback: 1000, wheel: 3, bell: []string{}},
		{name: "xfce configured", terminal: "xfce4-terminal", files: map[string]string{".config/xfce4/terminal/terminalrc": "[Configuration]\nScrollingUnlimited=TRUE\nMiscBell=TRUE\nMiscBellUrgent=TRUE\n"}, scrollback: UnlimitedScrollback, wheel: 3, bell: []string{BellAudio, BellAttention}},
		{name: "konsole default", terminal: "konsole", files: map[string]string{
			".config/konsolerc":                     "[Desktop Entry]\nDefaultProfile=Test.profile\n",
			".local/share/konsole/Test.profile":     "[Appearance]\nColorScheme=Dark\n",
			".local/share/konsole/Dark.colorscheme": "[Background]\nColor=16,16,16\n",
		}, scrollback: 1000, wheel: 3, bell: []string{BellAudio}},
		{name: "konsole configured", terminal: "konsole", files: map[string]string{
			".config/konsolerc":                     "[Desktop Entry]\nDefaultProfile=Test.profile\n",
			".local/share/konsole/Test.profile":     "[Appearance]\nColorScheme=Dark\n[Scrolling]\nHistoryMode=2\nHistorySize=3000\n[Terminal Features]\nBellMode=2\n",
			".local/share/konsole/Dark.colorscheme": "[Background]\nColor=16,16,16\n",
		}, scrollback: UnlimitedScrollback, wheel: 3, bell: []string{BellVisual}},
		{name: "xterm", terminal: "xterm", files: map[string]string{".Xresources": "xterm*background: #101010\nxterm*saveLines: 4096\n"}, scrollback: 4096, wheel: 3, bell: []string{BellAudio}},
		{name: "rio", terminal: "rio", files: map[string]string{".config/rio/config.toml": "[colors]\nbackground = '#101010'\n[scroll]\nmultiplier = 4.0\n"}, scrollback: 0, wheel: 4, bell: nil},
		{name: "gnome-terminal", terminal: "gnome-terminal", dconf: "[/]\ndefault='abc'\nlist=['abc']\n\n[:abc]\nuse-theme-colors=false\nbackground-color='#101010'\nscrollback-lines=uint32 20000\naudible-bell=false\n", scrollback: 20000, wheel: 3, bell: []string{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := fixture(t, tt.terminal, tt.files)
			if tt.dconf != "" {
				s.dconf = func(string) string { return tt.dconf }
			}

			profile, ok := load(s)

			require.True(t, ok)
			assert.Equal(t, tt.scrollback, profile.ScrollbackLines, "scrollback")
			assert.InDelta(t, tt.wheel, profile.WheelLines, 0.0001, "wheel")
			assert.Equal(t, tt.bell, profile.Bell, "bell")
		})
	}
}
