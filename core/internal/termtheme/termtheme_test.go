package termtheme

import (
	"fmt"
	"image/color"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func hexOf(c color.Color) string {
	if c == nil {
		return ""
	}
	r, g, b, _ := c.RGBA()
	return fmt.Sprintf("#%02x%02x%02x", r>>8, g>>8, b>>8)
}

func fixture(t *testing.T, terminal string, files map[string]string) sources {
	t.Helper()
	root := t.TempDir()
	t.Setenv("KITTY_CONFIG_DIRECTORY", "")
	t.Setenv("GHOSTTY_RESOURCES_DIR", "")
	for name, content := range files {
		path := filepath.Join(root, name)
		require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
		require.NoError(t, os.WriteFile(path, []byte(content), 0o644))
	}
	return sources{
		home:       root,
		configHome: filepath.Join(root, ".config"),
		dataHome:   filepath.Join(root, ".local/share"),
		configDirs: []string{filepath.Join(root, "etc/xdg")},
		dataDirs:   []string{filepath.Join(root, "usr/share")},
		terminal:   terminal,
	}
}

type want struct {
	fg, bg, cursor string
	ansi           map[int]string
	opacity        float64
}

func assertColors(t *testing.T, w want, colors Profile) {
	t.Helper()
	assert.Equal(t, w.fg, hexOf(colors.Fg), "foreground")
	assert.Equal(t, w.bg, hexOf(colors.Bg), "background")
	assert.Equal(t, w.cursor, hexOf(colors.Cursor), "cursor")
	assert.InDelta(t, w.opacity, colors.Opacity, 0.0001, "opacity")
	for index, hex := range w.ansi {
		assert.Equal(t, hex, hexOf(colors.ANSI[index]), "ansi %d", index)
	}
}

func TestReaders(t *testing.T) {
	tests := []struct {
		name     string
		terminal string
		files    map[string]string
		dconf    string
		want     want
	}{
		{
			name:     "kitty: includes are read in place and later lines win",
			terminal: "kitty",
			files: map[string]string{
				".config/kitty/kitty.conf": "foreground #ddd\nbackground #0f0028\ninclude theme.conf\ncolor1 #7d1625\nbackground_opacity 0.90\n",
				".config/kitty/theme.conf": "background #111111\ncolor1 #000000\ncolor15 #8c91fa\ncursor nope\n",
			},
			want: want{fg: "#dddddd", bg: "#111111", opacity: 0.9, ansi: map[int]string{1: "#7d1625", 15: "#8c91fa", 2: ""}},
		},
		{
			name:     "ghostty: the user's settings sit on top of the theme",
			terminal: "com.mitchellh.ghostty.desktop",
			files: map[string]string{
				".config/ghostty/config":       "theme = \"light:Day,dark:Night\"\nbackground = #0f0028\nconfig-file = extra\nbackground-opacity = 0.8\npalette = 1=#7d1625\n",
				".config/ghostty/extra":        "cursor-color = #8c91fa\n",
				".config/ghostty/themes/Night": "background = #000000\nforeground = #dddddd\npalette = 1=#ff0000\npalette = 2=#00ff00\n",
				".config/ghostty/themes/Day":   "foreground = #ffffff\n",
			},
			want: want{fg: "#dddddd", bg: "#0f0028", cursor: "#8c91fa", opacity: 0.8, ansi: map[int]string{1: "#7d1625", 2: "#00ff00"}},
		},
		{
			name:     "alacritty: imports first, 0x colours and window opacity",
			terminal: "Alacritty",
			files: map[string]string{
				".config/alacritty/alacritty.toml": "[general]\nimport = [\"~/theme.toml\"]\n[colors.primary]\nbackground = \"#070b14\"\n[window]\nopacity = 0.85\n",
				"theme.toml":                       "[colors.primary]\nbackground = \"#ffffff\"\nforeground = \"0xe6edf7\"\n[colors.normal]\nred = \"#e6484c\"\n[colors.bright]\nwhite = \"#d8d3dd\"\n[colors.cursor]\ncursor = \"#441199\"\n",
			},
			want: want{fg: "#e6edf7", bg: "#070b14", cursor: "#441199", opacity: 0.85, ansi: map[int]string{1: "#e6484c", 15: "#d8d3dd"}},
		},
		{
			name:     "foot: colours are written without a hash and alpha is the opacity",
			terminal: "foot",
			files: map[string]string{
				".config/foot/foot.ini": "include=" + "base.ini\n[colors]\nbackground=0f0028\nalpha=0.9\nregular1=7d1625\n",
				".config/foot/base.ini": "[colors]\nforeground=dddddd\nregular1=000000\nbright7=8c91fa\n[cursor]\ncolor=050014 8c91fa\n",
			},
			want: want{fg: "#dddddd", bg: "#0f0028", cursor: "#8c91fa", opacity: 0.9, ansi: map[int]string{1: "#7d1625", 15: "#8c91fa"}},
		},
		{
			name:     "wezterm: literal lua settings and a scheme file",
			terminal: "org.wezfurlong.wezterm",
			files: map[string]string{
				".config/wezterm/wezterm.lua":       "local config = {}\nconfig.color_scheme = 'Night'\nconfig.window_background_opacity = 0.9\nconfig.colors = {\n  background = '#0f0028',\n  ansi = { '#000000', '#7d1625' },\n}\nreturn config\n",
				".config/wezterm/colors/Night.toml": "[colors]\nforeground = \"#dddddd\"\nbackground = \"#222222\"\ncursor_bg = \"#8c91fa\"\nbrights = [\"#111111\", \"#e05167\"]\n",
			},
			want: want{fg: "#dddddd", bg: "#0f0028", cursor: "#8c91fa", opacity: 0.9, ansi: map[int]string{0: "#000000", 1: "#7d1625", 8: "#111111", 9: "#e05167"}},
		},
		{
			name:     "konsole: the default profile names the colour scheme",
			terminal: "org.kde.konsole",
			files: map[string]string{
				".config/konsolerc":                         "[Desktop Entry]\nDefaultProfile=Main.profile\n",
				".local/share/konsole/Main.profile":         "[Appearance]\nColorScheme=Lavender\n",
				".local/share/konsole/Lavender.colorscheme": "[Background]\nColor=15,0,40\n[Foreground]\nColor=221,221,221\n[Color1]\nColor=125,22,37\n[Color1Intense]\nColor=224,81,103\n[General]\nOpacity=0.9\n",
			},
			want: want{fg: "#dddddd", bg: "#0f0028", opacity: 0.9, ansi: map[int]string{1: "#7d1625", 9: "#e05167"}},
		},
		{
			name:     "rio: a named theme sits under the config",
			terminal: "rio",
			files: map[string]string{
				".config/rio/config.toml":       "theme = \"night\"\n[window]\nopacity = 0.95\n[colors]\nbackground = \"#0f0028\"\n",
				".config/rio/themes/night.toml": "[colors]\nforeground = \"#dddddd\"\nbackground = \"#000000\"\nred = \"#7d1625\"\nlight_red = \"#e05167\"\n",
			},
			want: want{fg: "#dddddd", bg: "#0f0028", opacity: 0.95, ansi: map[int]string{1: "#7d1625", 9: "#e05167"}},
		},
		{
			name:     "xfce4-terminal: a palette of sixteen and a darkness for transparency",
			terminal: "xfce4-terminal",
			files: map[string]string{
				".config/xfce4/terminal/terminalrc": "[Configuration]\nColorForeground=#dddddd\nColorBackground=#0f0028\nColorPalette=#000000;#7d1625;#337e6f\nBackgroundMode=TERMINAL_BACKGROUND_TRANSPARENT\nBackgroundDarkness=0.9\n",
			},
			want: want{fg: "#dddddd", bg: "#0f0028", opacity: 0.9, ansi: map[int]string{1: "#7d1625", 2: "#337e6f"}},
		},
		{
			name:     "terminator: colon separated palette, darkness only when transparent",
			terminal: "terminator",
			files: map[string]string{
				".config/terminator/config": "[profiles]\n  [[default]]\n    palette = \"#000000:#7d1625:#337e6f\"\n    foreground_color = \"#dddddd\"\n    background_color = \"#0f0028\"\n    background_type = transparent\n    background_darkness = 0.7\n",
			},
			want: want{fg: "#dddddd", bg: "#0f0028", opacity: 0.7, ansi: map[int]string{1: "#7d1625"}},
		},
		{
			name:     "xterm: its own resources beat loose ones, whatever the order",
			terminal: "xterm",
			files: map[string]string{
				".Xresources": "! comment\nXTerm*color1: #7d1625\n*color1: #ffffff\n*.background: #0f0028\nURxvt.foreground: #123456\n#include \"extra\"\n",
				"extra":       "*foreground: #dddddd\n",
			},
			want: want{fg: "#dddddd", bg: "#0f0028", opacity: 1, ansi: map[int]string{1: "#7d1625"}},
		},
		{
			name:     "urxvt: only its own class is read",
			terminal: "urxvt",
			files: map[string]string{
				".Xdefaults": "URxvt.foreground: rgb:dd/dd/dd\nXTerm*foreground: #123456\n",
			},
			want: want{fg: "#dddddd", opacity: 1},
		},
		{
			name:     "wezterm: keys of nested tables don't replace the colors table's own",
			terminal: "wezterm",
			files: map[string]string{
				".config/wezterm/wezterm.lua": "return {\n  background = { { source = { Color = '#000001' } } },\n  colors = {\n    foreground = '#dddddd',\n    background = '#0f0028',\n    tab_bar = {\n      background = '#11111b',\n    },\n    cursor_bg = '#8c91fa',\n  },\n}\n",
			},
			want: want{fg: "#dddddd", bg: "#0f0028", cursor: "#8c91fa", opacity: 1},
		},
		{
			name:     "terminator: profiles of layouts are ignored and the fallback profile is stable",
			terminal: "terminator",
			files: map[string]string{
				".config/terminator/config": "[layouts]\n  [[default]]\n    palette = \"#ffffff:#ffffff\"\n    background_color = \"#ffffff\"\n[profiles]\n  [[work]]\n    palette = \"#000000:#7d1625\"\n    background_color = \"#0f0028\"\n  [[zeta]]\n    palette = \"#000000:#111111\"\n  [[default]]\n    foreground_color = \"#dddddd\"\n",
			},
			want: want{bg: "#0f0028", opacity: 1, ansi: map[int]string{1: "#7d1625"}},
		},
		{
			name:     "ghostty: a theme given only for light is still used",
			terminal: "ghostty",
			files: map[string]string{
				".config/ghostty/config":      "theme = light: Dawn\n",
				".config/ghostty/themes/Dawn": "background = #faf4ed\n",
			},
			want: want{bg: "#faf4ed", opacity: 1},
		},
		{
			name:     "gnome-terminal: profile keys come from dconf",
			terminal: "org.gnome.Terminal.desktop",
			dconf:    "[/]\ndefault='abc'\nlist=['abc']\n\n[:abc]\npalette=['#000000', '#7d1625']\nforeground-color='#dddddd'\nbackground-color='#0f0028'\nuse-transparent-background=true\nbackground-transparency-percent=uint32 10\n",
			want:     want{fg: "#dddddd", bg: "#0f0028", opacity: 0.9, ansi: map[int]string{1: "#7d1625"}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := fixture(t, tt.terminal, tt.files)
			s.dconf = func(string) string { return tt.dconf }

			colors, ok := load(s)

			require.True(t, ok)
			assertColors(t, tt.want, colors)
		})
	}
}

func TestDefaultTerminalDecidesWhoIsFollowed(t *testing.T) {
	configs := map[string]string{
		".config/kitty/kitty.conf":   "background #111111\n",
		".config/ghostty/config":     "background = #222222\n",
		".config/foot/foot.ini":      "[colors]\nbackground=333333\n",
		".config/xdg-terminals.list": "# default\nghostty.desktop\nkitty.desktop\n",
	}
	with := func(extra map[string]string) map[string]string {
		merged := map[string]string{}
		for name, content := range configs {
			merged[name] = content
		}
		for name, content := range extra {
			merged[name] = content
		}
		return merged
	}
	background := func(t *testing.T, s sources) string {
		t.Helper()
		colors, ok := load(s)
		require.True(t, ok)
		return hexOf(colors.Bg)
	}

	t.Run("the first entry of the xdg list wins over $TERMINAL", func(t *testing.T) {
		assert.Equal(t, "#222222", background(t, fixture(t, "foot", configs)))
	})

	t.Run("a desktop specific list wins over the general one", func(t *testing.T) {
		s := fixture(t, "", with(map[string]string{".config/niri-xdg-terminals.list": "kitty.desktop\n"}))
		s.desktop = "niri"
		assert.Equal(t, "#111111", background(t, s))
	})

	t.Run("$TERMINAL is used without a list, paths and arguments included", func(t *testing.T) {
		s := fixture(t, "/usr/bin/foot --app-id=x", with(map[string]string{".config/xdg-terminals.list": ""}))
		assert.Equal(t, "#333333", background(t, s))
	})

	t.Run("an unsupported default is not replaced by another terminal's colours", func(t *testing.T) {
		_, ok := load(fixture(t, "st", map[string]string{".config/kitty/kitty.conf": "background #111111\n"}))
		assert.False(t, ok)
	})

	t.Run("without a default the first terminal that has a config is used", func(t *testing.T) {
		assert.Equal(t, "#333333", background(t, fixture(t, "", map[string]string{".config/foot/foot.ini": "[colors]\nbackground=333333\n"})))
	})

	t.Run("a default with no config follows nothing", func(t *testing.T) {
		_, ok := load(fixture(t, "kitty", nil))
		assert.False(t, ok)
	})
}

func TestParseColor(t *testing.T) {
	tests := map[string]string{
		"#abc":          "#aabbcc",
		"#0f0028":       "#0f0028",
		"0xE6EDF7":      "#e6edf7",
		"0f0028":        "#0f0028",
		"rgb:dd/dd/dd":  "#dddddd",
		"#7d7d16162525": "#7d1625",
		`"#112233"`:     "#112233",
		"not-a-colour":  "",
		"#12345":        "",
		"":              "",
	}
	for text, want := range tests {
		t.Run(text, func(t *testing.T) {
			assert.Equal(t, want, hexOf(tint(text)))
		})
	}
}

func TestGenericDefaultsAreScannedPast(t *testing.T) {
	s := fixture(t, "x-terminal-emulator", map[string]string{".config/foot/foot.ini": "[colors]\nbackground=333333\n"})

	colors, ok := load(s)

	require.True(t, ok)
	assert.Equal(t, "#333333", hexOf(colors.Bg))
}

func TestSandboxedHomeStillFindsHostConfigs(t *testing.T) {
	home := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(home, ".config", "kitty"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(home, ".config", "kitty", "kitty.conf"), []byte("background #111111\n"), 0o644))
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, ".var", "app", "x", "config"))
	t.Setenv("KITTY_CONFIG_DIRECTORY", "")
	t.Setenv("TERMINAL", "kitty")

	colors, ok := Load()

	require.True(t, ok)
	assert.Equal(t, "#111111", hexOf(colors.Bg))
}
