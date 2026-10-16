package portalsetup

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type FileStatus struct {
	Found bool   `json:"found"`
	Path  string `json:"path"`
	User  bool   `json:"user"`
}

type ConfigStatus struct {
	Path      string   `json:"path"`
	Exists    bool     `json:"exists"`
	Backends  []string `json:"backends"`
	Effective string   `json:"effective"`
}

type Status struct {
	Managed       bool         `json:"managed"`
	SystemManaged bool         `json:"systemManaged"`
	PortalFile    FileStatus   `json:"portalFile"`
	Service       FileStatus   `json:"service"`
	Config        ConfigStatus `json:"config"`
	Active        bool         `json:"active"`
	Kept          bool         `json:"kept,omitempty"`
}

type portalInfo struct {
	path       string
	busName    string
	interfaces []string
	user       bool
}

type configFile struct {
	path    string
	content string
}

func Inspect(env Env) Status {
	portals := installedPortals(env)
	_, managed := readState(env)
	status := Status{
		Managed:       managed,
		SystemManaged: env.PortalDir != "",
		PortalFile:    portalStatus(portals),
		Service:       serviceStatus(env),
		Config:        configStatus(env, portals),
	}
	status.Active = status.PortalFile.Found && status.Config.Effective == PortalName
	return status
}

func portalStatus(portals map[string]portalInfo) FileStatus {
	info, ok := portals[PortalName]
	if !ok {
		return FileStatus{}
	}
	return FileStatus{Found: true, Path: info.path, User: info.user}
}

func serviceStatus(env Env) FileStatus {
	for i, dir := range serviceDirs(env) {
		path := filepath.Join(dir, ServiceFile)
		if !isFile(path) {
			continue
		}
		return FileStatus{Found: true, Path: path, User: i == 0}
	}
	return FileStatus{Path: filepath.Join(userServiceDir(env), ServiceFile)}
}

func configStatus(env Env, portals map[string]portalInfo) ConfigStatus {
	path, exists := targetConfig(env)
	status := ConfigStatus{Path: path, Exists: exists, Effective: effectiveBackend(env, portals)}
	if !exists {
		return status
	}
	content, err := os.ReadFile(path)
	if err != nil {
		return status
	}
	line, found := preferredLine(string(content))
	if !found {
		return status
	}
	_, value, _ := strings.Cut(line, "=")
	status.Backends = backendList(value)
	return status
}

func effectiveBackend(env Env, portals map[string]portalInfo) string {
	for _, file := range configChain(env) {
		value, ok := lookupKey(file.content, preferredGroup, Interface)
		if !ok {
			value, ok = lookupKey(file.content, preferredGroup, "default")
		}
		if !ok {
			continue
		}
		name, stop := resolveBackends(value, portals)
		if stop || name != "" {
			return name
		}
	}
	return ""
}

func resolveBackends(value string, portals map[string]portalInfo) (string, bool) {
	for _, candidate := range backendList(value) {
		switch candidate {
		case "none":
			return "", true
		case "*":
			if name := firstSupporting(portals); name != "" {
				return name, false
			}
		default:
			if supports(portals, candidate) {
				return candidate, false
			}
		}
	}
	return "", false
}

func firstSupporting(portals map[string]portalInfo) string {
	names := make([]string, 0, len(portals))
	for name := range portals {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		if supports(portals, name) {
			return name
		}
	}
	return ""
}

func supports(portals map[string]portalInfo, name string) bool {
	info, ok := portals[name]
	if !ok {
		return false
	}
	for _, iface := range info.interfaces {
		if iface == Interface {
			return true
		}
	}
	return false
}

func configChain(env Env) []configFile {
	names := configFileNames(env)
	var chain []configFile
	for _, dir := range configDirs(env) {
		file, ok := configFileIn(dir, names)
		if !ok {
			continue
		}
		chain = append(chain, file)
	}
	return chain
}

func configFileIn(dir string, names []string) (configFile, bool) {
	for _, name := range names {
		path := filepath.Join(dir, name)
		content, err := os.ReadFile(path)
		if err != nil || !hasGroup(string(content), preferredGroup) {
			continue
		}
		return configFile{path: path, content: string(content)}, true
	}
	return configFile{}, false
}

func configDirs(env Env) []string {
	if env.PortalDir != "" {
		return []string{env.PortalDir}
	}
	dirs := []string{filepath.Join(env.ConfigHome, "xdg-desktop-portal")}
	for _, dir := range env.ConfigDirs {
		dirs = append(dirs, filepath.Join(dir, "xdg-desktop-portal"))
	}
	dirs = append(dirs, filepath.Join(env.SysConfDir, "xdg-desktop-portal"))
	dirs = append(dirs, filepath.Join(env.DataHome, "xdg-desktop-portal"))
	for _, dir := range env.DataDirs {
		dirs = append(dirs, filepath.Join(dir, "xdg-desktop-portal"))
	}
	return append(dirs, filepath.Join(env.DataDir, "xdg-desktop-portal"))
}

func installedPortals(env Env) map[string]portalInfo {
	portals := map[string]portalInfo{}
	for _, dir := range portalDirs(env) {
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		for _, entry := range entries {
			name, isPortal := strings.CutSuffix(entry.Name(), ".portal")
			if !isPortal || entry.IsDir() {
				continue
			}
			if _, seen := portals[name]; seen {
				continue
			}
			path := filepath.Join(dir, entry.Name())
			info, ok := readPortal(path)
			if !ok {
				continue
			}
			info.user = dir == userPortalDir(env)
			portals[name] = info
		}
	}
	return portals
}

func readPortal(path string) (portalInfo, bool) {
	data, err := os.ReadFile(path)
	if err != nil {
		return portalInfo{}, false
	}
	content := string(data)
	busName, ok := lookupKey(content, "portal", "DBusName")
	if !ok {
		return portalInfo{}, false
	}
	interfaces, _ := lookupKey(content, "portal", "Interfaces")
	return portalInfo{path: path, busName: busName, interfaces: backendList(interfaces)}, true
}

func portalDirs(env Env) []string {
	if env.PortalDir != "" {
		return []string{env.PortalDir}
	}
	dirs := []string{userPortalDir(env)}
	for _, dir := range env.DataDirs {
		dirs = append(dirs, filepath.Join(dir, "xdg-desktop-portal", "portals"))
	}
	return append(dirs, filepath.Join(env.DataDir, "xdg-desktop-portal", "portals"))
}

func serviceDirs(env Env) []string {
	dirs := []string{userServiceDir(env)}
	for _, dir := range env.DataDirs {
		dirs = append(dirs, filepath.Join(dir, "dbus-1", "services"))
	}
	return dirs
}

func userPortalDir(env Env) string {
	return filepath.Join(env.DataHome, "xdg-desktop-portal", "portals")
}

func userServiceDir(env Env) string {
	return filepath.Join(env.DataHome, "dbus-1", "services")
}
