package sandbox

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
)

// Dirs are the XDG base directories user-visible state belongs to. Inside a
// sandbox with host access they point at the host's tree, because trash,
// thumbnails, mimeapps.list and user-dirs.dirs are only useful there.
type Dirs struct {
	ConfigHome string   `json:"configHome"`
	DataHome   string   `json:"dataHome"`
	CacheHome  string   `json:"cacheHome"`
	StateHome  string   `json:"stateHome"`
	ConfigDirs []string `json:"configDirs"`
	DataDirs   []string `json:"dataDirs"`
}

const hostOSRoot = "/run/host"

const (
	sandboxConfigDirs = "/app/etc/xdg:/etc/xdg"
	sandboxDataDirs   = "/app/share:/usr/share"
)

var hostConfigDirs = []string{hostOSRoot + "/etc/xdg"}

// Other Flatpak apps export their desktop entries under these two, which
// `host` plus `/var` make visible; the rest of the host OS sits under
// /run/host because /usr inside the sandbox is the runtime.
func hostDataDirs(dataHome string) []string {
	return []string{
		filepath.Join(dataHome, "flatpak", "exports", "share"),
		"/var/lib/flatpak/exports/share",
		hostOSRoot + "/usr/local/share",
		hostOSRoot + "/usr/share",
	}
}

func nativeDirs(env func(string) string) Dirs {
	home := env("HOME")
	return Dirs{
		ConfigHome: envOr(env, "XDG_CONFIG_HOME", filepath.Join(home, ".config")),
		DataHome:   envOr(env, "XDG_DATA_HOME", filepath.Join(home, ".local", "share")),
		CacheHome:  envOr(env, "XDG_CACHE_HOME", filepath.Join(home, ".cache")),
		StateHome:  envOr(env, "XDG_STATE_HOME", filepath.Join(home, ".local", "state")),
		ConfigDirs: splitPathList(envOr(env, "XDG_CONFIG_DIRS", "/etc/xdg")),
		DataDirs:   splitPathList(envOr(env, "XDG_DATA_DIRS", "/usr/local/share:/usr/share")),
	}
}

// Flatpak exports HOST_XDG_*_HOME only when the host overrides that variable,
// so the spec defaults under HOME stand in otherwise. The runtime's own
// XDG_*_DIRS stay first so the sandbox keeps finding its own resources.
func hostDirs(env func(string) string, hostOS bool) Dirs {
	home := env("HOME")
	dirs := Dirs{
		ConfigHome: envOr(env, "HOST_XDG_CONFIG_HOME", filepath.Join(home, ".config")),
		DataHome:   envOr(env, "HOST_XDG_DATA_HOME", filepath.Join(home, ".local", "share")),
		CacheHome:  envOr(env, "HOST_XDG_CACHE_HOME", filepath.Join(home, ".cache")),
		StateHome:  envOr(env, "HOST_XDG_STATE_HOME", filepath.Join(home, ".local", "state")),
		ConfigDirs: splitPathList(envOr(env, "XDG_CONFIG_DIRS", sandboxConfigDirs)),
		DataDirs:   splitPathList(envOr(env, "XDG_DATA_DIRS", sandboxDataDirs)),
	}
	if !hostOS {
		return dirs
	}
	dirs.ConfigDirs = appendMissing(dirs.ConfigDirs, hostConfigDirs...)
	dirs.DataDirs = appendMissing(dirs.DataDirs, hostDataDirs(dirs.DataHome)...)
	return dirs
}

// ExportHostDirs points this process's XDG variables at the host tree, so
// every reader (trash, mimeapps, thumbnails, user-dirs, Qt in the UI child)
// behaves as it would in a native install. No-op unless the sandbox can
// reach the user's home.
func ExportHostDirs(caps Capabilities) {
	if !caps.Sandboxed || !caps.UserFilesystem {
		return
	}
	exports := [][2]string{
		{"XDG_CONFIG_HOME", caps.Dirs.ConfigHome},
		{"XDG_DATA_HOME", caps.Dirs.DataHome},
		{"XDG_CACHE_HOME", caps.Dirs.CacheHome},
		{"XDG_STATE_HOME", caps.Dirs.StateHome},
		{"XDG_CONFIG_DIRS", strings.Join(caps.Dirs.ConfigDirs, ":")},
		{"XDG_DATA_DIRS", strings.Join(caps.Dirs.DataDirs, ":")},
	}
	for _, kv := range exports {
		_ = os.Setenv(kv[0], kv[1])
	}
}

func envOr(env func(string) string, key, fallback string) string {
	if v := env(key); v != "" {
		return v
	}
	return fallback
}

func splitPathList(raw string) []string {
	var out []string
	for _, dir := range strings.Split(raw, ":") {
		if dir = strings.TrimSpace(dir); dir != "" {
			out = append(out, dir)
		}
	}
	return out
}

func appendMissing(list []string, extra ...string) []string {
	for _, dir := range extra {
		if !slices.Contains(list, dir) {
			list = append(list, dir)
		}
	}
	return list
}
