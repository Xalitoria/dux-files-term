// Package termtheme reads the colours of the user's own terminal from its config files.
package termtheme

import (
	"image/color"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

const maxIncludeDepth = 4

// Profile is the look and behaviour of a terminal. Anything its config leaves out stays unset.
type Profile struct {
	Fg, Bg, Cursor           color.Color
	SelectionFg, SelectionBg color.Color
	ANSI                     [16]color.Color
	Opacity                  float64

	// Word characters say what a word is, delimiters say what ends one. Terminals use one or the other.
	WordChars, WordDelims string
	CopyOnSelect          bool
	Keys                  map[string][]string
	FontFamily            string
	FontSize              float64

	// Shape is "block", "bar" or "underline". Programs can still change it with escape sequences.
	CursorShape string
	CursorBlink bool
}

func newProfile() Profile {
	return Profile{Opacity: 1}
}

// A config that sets nothing this package follows has nothing to follow.
func (c Profile) usable() bool {
	switch {
	case c.Fg != nil, c.Bg != nil, c.Cursor != nil, c.SelectionFg != nil, c.SelectionBg != nil, c.Opacity < 1:
		return true
	case c.WordChars != "", c.WordDelims != "", c.CopyOnSelect, len(c.Keys) > 0, c.FontFamily != "", c.FontSize > 0, c.CursorShape != "", c.CursorBlink:
		return true
	}
	for _, ansi := range c.ANSI {
		if ansi != nil {
			return true
		}
	}
	return false
}

type sources struct {
	home       string
	configHome string
	dataHome   string
	configDirs []string
	dataDirs   []string
	desktop    string
	terminal   string
	dconf      func(path string) string
}

type reader func(s sources) (Profile, bool)

type terminal struct {
	ids  []string
	read reader
	// ctrlZoom is set for terminals that zoom with Ctrl+Plus, Ctrl+Minus and Ctrl+0 out of the box.
	ctrlZoom bool
}

// Names that say nothing about which terminal runs, so they don't count as a default.
var genericTerminals = []string{"x-terminal-emulator", "xdg-terminal-exec", "terminal"}

var terminals = []terminal{
	{[]string{"kitty"}, readKitty, false},
	{[]string{"ghostty", "com.mitchellh.ghostty"}, readGhostty, true},
	{[]string{"alacritty"}, readAlacritty, true},
	{[]string{"foot", "footclient"}, readFoot, true},
	{[]string{"wezterm", "org.wezfurlong.wezterm"}, readWezterm, true},
	{[]string{"konsole", "org.kde.konsole"}, readKonsole, true},
	{[]string{"rio", "com.raphamorim.rio"}, readRio, true},
	{[]string{"xfce4-terminal"}, readXfce, true},
	{[]string{"gnome-terminal", "org.gnome.terminal"}, readGnomeTerminal, true},
	{[]string{"terminator"}, readTerminator, true},
	{[]string{"urxvt", "urxvtc", "rxvt-unicode"}, readXresources("urxvt", "rxvt"), false},
	{[]string{"xterm", "uxterm"}, readXresources("xterm", "uxterm"), false},
}

// Load follows the colours of the default terminal, or of the first supported one that has a config.
func Load() (Profile, bool) {
	return load(hostSources())
}

func load(s sources) (Profile, bool) {
	if name := s.defaultTerminal(); name != "" && !slices.Contains(genericTerminals, name) {
		for _, t := range terminals {
			if slices.Contains(t.ids, name) {
				return t.load(s)
			}
		}
		return Profile{}, false
	}
	for _, t := range terminals {
		if profile, ok := t.load(s); ok {
			return profile, true
		}
	}
	return Profile{}, false
}

func (t terminal) load(s sources) (Profile, bool) {
	profile, ok := t.read(s)
	if ok && t.ctrlZoom {
		profile.addKeys(ctrlZoomKeys)
	}
	return profile, ok
}

func hostSources() sources {
	home, _ := os.UserHomeDir()
	configHome := envOr("XDG_CONFIG_HOME", filepath.Join(home, ".config"))
	dataHome := envOr("XDG_DATA_HOME", filepath.Join(home, ".local", "share"))
	configDirs := filepath.SplitList(envOr("XDG_CONFIG_DIRS", "/etc/xdg"))
	dataDirs := filepath.SplitList(envOr("XDG_DATA_DIRS", "/usr/local/share:/usr/share"))
	// A Flatpak moves the XDG folders into ~/.var, where the host's terminal configs are not.
	configDirs = append([]string{filepath.Join(home, ".config")}, configDirs...)
	dataDirs = append([]string{filepath.Join(home, ".local", "share")}, dataDirs...)
	return sources{
		home:       home,
		configHome: configHome,
		dataHome:   dataHome,
		configDirs: configDirs,
		dataDirs:   dataDirs,
		desktop:    strings.ToLower(strings.Split(os.Getenv("XDG_CURRENT_DESKTOP"), ":")[0]),
		terminal:   os.Getenv("TERMINAL"),
		dconf:      dconfDump,
	}
}

func envOr(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func (s sources) configFiles(rel ...string) []string {
	var paths []string
	for _, dir := range append([]string{s.configHome}, s.configDirs...) {
		for _, name := range rel {
			paths = append(paths, filepath.Join(dir, name))
		}
	}
	return paths
}

func (s sources) dataFiles(rel string) []string {
	paths := []string{filepath.Join(s.dataHome, rel)}
	for _, dir := range s.dataDirs {
		paths = append(paths, filepath.Join(dir, rel))
	}
	return paths
}

// The xdg-terminal-exec lists come first, then $TERMINAL.
func (s sources) defaultTerminal() string {
	lists := []string{"xdg-terminals.list"}
	if s.desktop != "" {
		lists = []string{s.desktop + "-xdg-terminals.list", "xdg-terminals.list"}
	}
	for _, path := range s.configFiles(lists...) {
		if name := firstTerminalEntry(path); name != "" {
			return name
		}
	}
	return normalizeTerminal(s.terminal)
}

func firstTerminalEntry(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		line, _, _ = strings.Cut(line, ":")
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, "-") {
			continue
		}
		return normalizeTerminal(strings.TrimPrefix(line, "+"))
	}
	return ""
}

func normalizeTerminal(name string) string {
	fields := strings.Fields(name)
	if len(fields) == 0 {
		return ""
	}
	return strings.TrimSuffix(strings.ToLower(filepath.Base(fields[0])), ".desktop")
}

func readFirst(paths []string) ([]byte, string, bool) {
	for _, path := range paths {
		if data, err := os.ReadFile(path); err == nil {
			return data, path, true
		}
	}
	return nil, "", false
}

func resolveFrom(path, included string) string {
	if filepath.IsAbs(included) {
		return included
	}
	return filepath.Join(filepath.Dir(path), included)
}

func clampOpacity(value float64) float64 {
	return min(max(value, 0), 1)
}
