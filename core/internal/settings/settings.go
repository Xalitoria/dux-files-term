// Package settings reads the UI settings file the quickshell frontend owns.
// The daemon only consumes it, so values are re-read on demand; defaults
// mirror the JsonAdapter in quickshell/Common/SettingsData.qml.
package settings

import (
	"encoding/json"
	"os"
	"path/filepath"
)

const FileName = "ui-settings.json"

type UISettings struct {
	Terminal            string   `json:"terminal"`
	RegisterFileManager bool     `json:"registerFileManager"`
	RecentFiles         bool     `json:"recentFiles"`
	SearchExcludes      []string `json:"searchExcludes"`
}

func Defaults() UISettings {
	return UISettings{RegisterFileManager: true, RecentFiles: true}
}

// Load returns defaults when the file is missing or unreadable; a partial file
// only overrides the keys it contains.
func Load(configDir string) UISettings {
	return loadFrom(filepath.Join(configDir, FileName))
}

func loadFrom(path string) UISettings {
	out := Defaults()
	data, err := os.ReadFile(path)
	if err != nil {
		return out
	}
	if err := json.Unmarshal(data, &out); err != nil {
		return Defaults()
	}
	return out
}
