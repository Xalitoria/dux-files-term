package sandbox

import (
	"maps"
	"slices"
)

// features derives the degraded-mode table every plan documents from the
// permission facts. Names are the ones the UI switches on.
func features(c Capabilities) map[string]Feature {
	userFS := gate(c.UserFilesystem, ReasonNoHostFS)
	hostSpawn := gate(c.HostSpawn, ReasonNoHostSpawn)
	return map[string]Feature{
		"hostFilesystem": gate(c.HostFilesystem, ReasonNoHostFS),
		"trash":          userFS,
		"thumbnails":     degradedUnless(c.UserFilesystem, ReasonNoHostFS),
		"setDefault":     userFS,
		"devices":        gate(c.UDisks, ReasonNoUDisks),
		"network":        gate(c.Gvfs, ReasonNoGvfs),
		"terminal":       hostSpawn,
		"run":            hostSpawn,
		"openWith":       degradedUnless(!c.Sandboxed, ReasonPortalChooser),
		"elevation":      gate(!c.Sandboxed, ReasonSandboxed),
	}
}

func gate(ok bool, reason string) Feature {
	if ok {
		return Feature{Available: true}
	}
	return Feature{Reason: reason}
}

func degradedUnless(ok bool, reason string) Feature {
	if ok {
		return Feature{Available: true}
	}
	return Feature{Available: true, Degraded: true, Reason: reason}
}

func sortedKeys(m map[string]string) []string {
	if len(m) == 0 {
		return []string{}
	}
	return slices.Sorted(maps.Keys(m))
}
