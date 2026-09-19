package files

import (
	"os"
	"path/filepath"
	"strings"
)

var userDirIcons = map[string]string{
	"XDG_DESKTOP_DIR":     "user-desktop",
	"XDG_DOCUMENTS_DIR":   "folder-documents",
	"XDG_DOWNLOAD_DIR":    "folder-download",
	"XDG_MUSIC_DIR":       "folder-music",
	"XDG_PICTURES_DIR":    "folder-pictures",
	"XDG_PUBLICSHARE_DIR": "folder-publicshare",
	"XDG_TEMPLATES_DIR":   "folder-templates",
	"XDG_VIDEOS_DIR":      "folder-videos",
}

func loadUserDirs() map[string]string {
	home, err := os.UserHomeDir()
	if err != nil {
		return map[string]string{}
	}

	dirs := map[string]string{home: "user-home"}
	for key, icon := range userDirIcons {
		if path := os.Getenv(key); path != "" {
			dirs[filepath.Clean(path)] = icon
		}
	}

	config := os.Getenv("XDG_CONFIG_HOME")
	if config == "" {
		config = filepath.Join(home, ".config")
	}
	data, err := os.ReadFile(filepath.Join(config, "user-dirs.dirs"))
	if err != nil {
		return dirs
	}

	for _, line := range strings.Split(string(data), "\n") {
		key, value, ok := parseUserDir(line)
		if !ok {
			continue
		}
		icon, known := userDirIcons[key]
		if !known {
			continue
		}
		if _, fromEnv := dirs[value]; fromEnv {
			continue
		}
		dirs[value] = icon
	}
	return dirs
}

func parseUserDir(line string) (string, string, bool) {
	line = strings.TrimSpace(line)
	if strings.HasPrefix(line, "#") {
		return "", "", false
	}
	key, value, ok := strings.Cut(line, "=")
	if !ok {
		return "", "", false
	}
	value = strings.Trim(strings.TrimSpace(value), `"`)
	if !strings.HasPrefix(value, "$HOME") {
		if !filepath.IsAbs(value) {
			return "", "", false
		}
		return strings.TrimSpace(key), filepath.Clean(value), true
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", "", false
	}
	return strings.TrimSpace(key), filepath.Clean(filepath.Join(home, strings.TrimPrefix(value, "$HOME"))), true
}
