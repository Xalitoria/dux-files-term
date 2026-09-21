package sandbox

import (
	"maps"
	"sync"
)

const (
	ReasonSandboxed     = "SANDBOXED"
	ReasonNoHostFS      = "NOHOSTFS"
	ReasonNoUDisks      = "NOUDISKS"
	ReasonNoGvfs        = "NOGVFS"
	ReasonNoHostSpawn   = "NOHOSTSPAWN"
	ReasonPortalChooser = "PORTALCHOOSER"
)

const (
	portalDesktopDest   = "org.freedesktop.portal.Desktop"
	portalDesktopPath   = "/org/freedesktop/portal/desktop"
	portalDocumentsDest = "org.freedesktop.portal.Documents"
	portalDocumentsPath = "/org/freedesktop/portal/documents"
	portalIfacePrefix   = "org.freedesktop.portal."
)

// Feature is one user-visible capability. Reason is a stable code the UI maps
// to a sentence; it is set when the feature is missing or Degraded.
type Feature struct {
	Available bool   `json:"available"`
	Degraded  bool   `json:"degraded,omitempty"`
	Reason    string `json:"reason,omitempty"`
}

// Capabilities is what the environment permits, computed once at boot.
// Whether a permitted service is actually running is reported live by that
// service (devices, network, elevation), not here.
type Capabilities struct {
	Sandboxed      bool               `json:"sandboxed"`
	AppID          string             `json:"appId,omitempty"`
	HostFilesystem bool               `json:"hostFilesystem"`
	UserFilesystem bool               `json:"userFilesystem"`
	HostOSRoot     string             `json:"hostOsRoot,omitempty"`
	Filesystems    []string           `json:"filesystems"`
	UDisks         bool               `json:"udisks"`
	Gvfs           bool               `json:"gvfs"`
	HostSpawn      bool               `json:"hostSpawn"`
	Portals        map[string]uint32  `json:"portals"`
	Dirs           Dirs               `json:"dirs"`
	Features       map[string]Feature `json:"features"`
}

var (
	currentOnce sync.Once
	current     Capabilities
)

// Current probes the real environment once and caches the answer.
func Current() Capabilities {
	currentOnce.Do(func() {
		probe := newOSProbe()
		defer probe.close()
		current = Detect(probe)
	})
	return current
}

func Detect(p Probe) Capabilities {
	if p.Getenv("FLATPAK_ID") == "" {
		return native(p)
	}
	return sandboxed(p)
}

func native(p Probe) Capabilities {
	caps := Capabilities{
		HostFilesystem: true,
		UserFilesystem: true,
		Filesystems:    []string{},
		UDisks:         true,
		Gvfs:           true,
		HostSpawn:      true,
		Portals:        map[string]uint32{},
		Dirs:           nativeDirs(p.Getenv),
	}
	caps.Features = features(caps)
	return caps
}

func sandboxed(p Probe) Capabilities {
	data, _ := p.ReadFile(flatpakInfoPath)
	info := parseFlatpakInfo(data)
	if info.appID == "" {
		info.appID = p.Getenv("FLATPAK_ID")
	}

	caps := Capabilities{
		Sandboxed:   true,
		AppID:       info.appID,
		Filesystems: sortedKeys(info.filesystems),
		UDisks:      canTalk(info.systemBus, "org.freedesktop.UDisks2"),
		Gvfs:        gvfsPermitted(info),
		HostSpawn:   hostSpawnPermitted(info, p),
		Portals:     portalVersions(p),
	}
	// The policy alone is not proof: an override can drop `host` after the
	// info file was written, so the host OS mount is checked as well.
	caps.HostFilesystem = info.hasFilesystem("host") && p.Exists(hostOSRoot+"/usr")
	caps.UserFilesystem = caps.HostFilesystem || info.hasFilesystem("home")
	caps.Dirs = nativeDirs(p.Getenv)
	if caps.HostFilesystem {
		caps.HostOSRoot = hostOSRoot
	}
	if caps.UserFilesystem {
		caps.Dirs = hostDirs(p.Getenv, caps.HostFilesystem)
	}
	caps.Features = features(caps)
	return caps
}

func gvfsPermitted(info flatpakInfo) bool {
	return canTalk(info.sessionBus, "org.gtk.vfs.Daemon") &&
		info.hasFilesystem("xdg-run/gvfs") &&
		info.hasFilesystem("xdg-run/gvfsd")
}

func hostSpawnPermitted(info flatpakInfo, p Probe) bool {
	if !canTalk(info.sessionBus, "org.freedesktop.Flatpak") {
		return false
	}
	_, err := p.LookPath("flatpak-spawn")
	return err == nil
}

func portalVersions(p Probe) map[string]uint32 {
	out := map[string]uint32{}
	for _, iface := range []string{"Trash", "OpenURI", "Settings"} {
		if v, ok := p.PortalVersion(portalDesktopDest, portalDesktopPath, portalIfacePrefix+iface); ok {
			out[iface] = v
		}
	}
	if v, ok := p.PortalVersion(portalDocumentsDest, portalDocumentsPath, portalIfacePrefix+"FileTransfer"); ok {
		out["FileTransfer"] = v
	}
	return out
}

func (c Capabilities) PortalVersion(iface string) uint32 { return c.Portals[iface] }

// WithFeature returns a copy carrying one overridden feature, for callers
// that overlay live service state onto the boot-time table.
func (c Capabilities) WithFeature(name string, f Feature) Capabilities {
	out := c
	out.Features = maps.Clone(c.Features)
	if out.Features == nil {
		out.Features = map[string]Feature{}
	}
	out.Features[name] = f
	return out
}
