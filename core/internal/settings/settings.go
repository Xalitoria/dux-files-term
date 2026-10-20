// Defaults mirror the JsonAdapter in quickshell/Common/SettingsData.qml.
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
