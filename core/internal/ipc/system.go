package ipc

import (
	"context"

	"github.com/AvengeMedia/dankfiles/core/internal/sandbox"
)

func HandleSystem(ctx context.Context, w *ConnWriter, req Request, deps Deps) {
	switch req.Method {
	case "system.colorScheme.get":
		if deps.ColorScheme == nil {
			Respond(w, req.ID, map[string]any{"available": false, "colorScheme": uint32(0)})
			return
		}
		Respond(w, req.ID, deps.ColorScheme.State())
	case "system.capabilities":
		Respond(w, req.ID, capabilities(deps))
	case "system.elevation":
		if deps.FileOps == nil {
			Respond(w, req.ID, map[string]any{"available": false, "reason": "NOROUTE"})
			return
		}
		Respond(w, req.ID, deps.FileOps.Elevation())
	case "system.openUri":
		if deps.Apps == nil {
			RespondError(w, req.ID, "uri opener unavailable")
			return
		}
		uri := ParamString(req.Params, "uri")
		if uri == "" {
			RespondError(w, req.ID, "uri is required")
			return
		}
		if err := deps.Apps.OpenURI(ctx, uri); err != nil {
			RespondError(w, req.ID, err.Error())
			return
		}
		Respond(w, req.ID, map[string]any{"opened": true})
	default:
		RespondError(w, req.ID, "unknown system method: "+req.Method)
	}
}

type capabilitiesResponse struct {
	sandbox.Capabilities
	Thumbnailers []string `json:"thumbnailers"`
}

// capabilities overlays the live answers onto the boot-time table where the
// sandbox alone cannot decide: elevation also needs pkexec and the policy.
func capabilities(deps Deps) capabilitiesResponse {
	caps := deps.Sandbox
	if deps.FileOps != nil {
		e := deps.FileOps.Elevation()
		caps = caps.WithFeature("elevation", sandbox.Feature{Available: e.Available, Reason: e.Reason})
	}
	if caps.Portals == nil {
		caps.Portals = map[string]uint32{}
	}
	if caps.Features == nil {
		caps.Features = map[string]sandbox.Feature{}
	}
	if caps.Filesystems == nil {
		caps.Filesystems = []string{}
	}
	out := capabilitiesResponse{Capabilities: caps, Thumbnailers: []string{}}
	if deps.Files != nil {
		out.Thumbnailers = append(out.Thumbnailers, deps.Files.Capabilities()...)
	}
	return out
}
