package termtheme

import (
	"path/filepath"

	"github.com/BurntSushi/toml"
)

type rioColors struct {
	Background, Foreground, Cursor string
	Black, Red, Green, Yellow      string
	Blue, Magenta, Cyan, White     string
	LightBlack                     string `toml:"light_black"`
	LightRed                       string `toml:"light_red"`
	LightGreen                     string `toml:"light_green"`
	LightYellow                    string `toml:"light_yellow"`
	LightBlue                      string `toml:"light_blue"`
	LightMagenta                   string `toml:"light_magenta"`
	LightCyan                      string `toml:"light_cyan"`
	LightWhite                     string `toml:"light_white"`
}

type rioConfig struct {
	Theme  string
	Colors rioColors
	Window struct{ Opacity *float64 }
}

func readRio(s sources) (Profile, bool) {
	_, path, ok := readFirst(s.configFiles("rio/config.toml"))
	if !ok {
		return Profile{}, false
	}
	var config rioConfig
	if _, err := toml.DecodeFile(path, &config); err != nil {
		return Profile{}, false
	}
	if config.Theme != "" {
		var theme rioConfig
		if _, err := toml.DecodeFile(filepath.Join(filepath.Dir(path), "themes", config.Theme+".toml"), &theme); err == nil {
			_, _ = toml.DecodeFile(path, &theme)
			config = theme
		}
	}

	c := config.Colors
	colors := newProfile()
	colors.Fg, colors.Bg, colors.Cursor = tint(c.Foreground), tint(c.Background), tint(c.Cursor)
	names := [16]string{
		c.Black, c.Red, c.Green, c.Yellow, c.Blue, c.Magenta, c.Cyan, c.White,
		c.LightBlack, c.LightRed, c.LightGreen, c.LightYellow, c.LightBlue, c.LightMagenta, c.LightCyan, c.LightWhite,
	}
	for i, name := range names {
		colors.ANSI[i] = tint(name)
	}
	if config.Window.Opacity != nil {
		colors.Opacity = clampOpacity(*config.Window.Opacity)
	}
	return colors, colors.usable()
}
