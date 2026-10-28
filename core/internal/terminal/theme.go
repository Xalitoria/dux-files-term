package terminal

import "github.com/duxlabs-dev/dux-files/core/internal/termtheme"

func colorsPayload(colors termtheme.Profile) map[string]any {
	return map[string]any{
		"fg":      hexColor(colors.Fg, nil),
		"bg":      hexColor(colors.Bg, nil),
		"cursor":  hexColor(colors.Cursor, nil),
		"opacity": colors.Opacity,

		"selectionFg": hexColor(colors.SelectionFg, nil),
		"selectionBg": hexColor(colors.SelectionBg, nil),
	}
}

func behaviorPayload(profile termtheme.Profile) map[string]any {
	keys := profile.Keys
	if keys == nil {
		keys = map[string][]string{}
	}
	return map[string]any{
		"wordChars":    profile.WordChars,
		"wordDelims":   profile.WordDelims,
		"copyOnSelect": profile.CopyOnSelect,
		"keys":         keys,
		"fontFamily":   profile.FontFamily,
		"fontSize":     profile.FontSize,
		"cursorShape":  profile.CursorShape,
		"cursorBlink":  profile.CursorBlink,
	}
}

func (s *Session) applyColors(colors termtheme.Profile) {
	for i, c := range colors.ANSI {
		if c != nil {
			s.emu.SetIndexedColor(i, c)
		}
	}
	if colors.Fg != nil {
		s.emu.SetDefaultForegroundColor(colors.Fg)
	}
	if colors.Bg != nil {
		s.emu.SetDefaultBackgroundColor(colors.Bg)
	}
	if colors.Cursor != nil {
		s.emu.SetDefaultCursorColor(colors.Cursor)
	}
}
