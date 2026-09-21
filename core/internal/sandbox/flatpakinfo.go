package sandbox

import (
	"strings"
)

const flatpakInfoPath = "/.flatpak-info"

// flatpakInfo is the subset of /.flatpak-info the daemon consults: the
// filesystem grants and the bus policies flatpak resolved for this instance,
// overrides included.
type flatpakInfo struct {
	appID       string
	filesystems map[string]string
	sessionBus  map[string]string
	systemBus   map[string]string
}

func parseFlatpakInfo(data []byte) flatpakInfo {
	info := flatpakInfo{
		filesystems: map[string]string{},
		sessionBus:  map[string]string{},
		systemBus:   map[string]string{},
	}
	section := ""
	for _, raw := range strings.Split(string(data), "\n") {
		line := strings.TrimSpace(raw)
		switch {
		case line == "" || strings.HasPrefix(line, "#"):
			continue
		case strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]"):
			section = line[1 : len(line)-1]
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		switch section {
		case "Application":
			if key == "name" {
				info.appID = value
			}
		case "Context":
			if key == "filesystems" {
				info.filesystems = parseFilesystems(value)
			}
		case "Session Bus Policy":
			info.sessionBus[key] = value
		case "System Bus Policy":
			info.systemBus[key] = value
		}
	}
	return info
}

func parseFilesystems(value string) map[string]string {
	out := map[string]string{}
	for _, entry := range strings.Split(value, ";") {
		path, mode := splitFilesystemMode(strings.TrimSpace(entry))
		switch {
		case path == "":
			continue
		case mode == "reset":
			clear(out)
		case strings.HasPrefix(path, "!"):
			delete(out, path[1:])
		default:
			out[path] = mode
		}
	}
	return out
}

func splitFilesystemMode(entry string) (string, string) {
	i := strings.LastIndex(entry, ":")
	if i < 0 {
		return entry, "rw"
	}
	switch mode := entry[i+1:]; mode {
	case "ro", "rw", "create", "reset":
		return entry[:i], mode
	}
	return entry, "rw"
}

func (fi flatpakInfo) hasFilesystem(path string) bool {
	_, ok := fi.filesystems[path]
	return ok
}

// canTalk reports whether the policy lets the app call name: an exact entry
// or a `prefix.*` wildcard at talk or own level, matched like xdg-dbus-proxy.
func canTalk(policy map[string]string, name string) bool {
	for pattern, level := range policy {
		if level != "talk" && level != "own" {
			continue
		}
		if pattern == name {
			return true
		}
		prefix, ok := strings.CutSuffix(pattern, ".*")
		if !ok {
			continue
		}
		if strings.HasPrefix(name, prefix+".") {
			return true
		}
	}
	return false
}
