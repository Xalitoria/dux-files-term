package ipc

import (
	"context"
	"strings"

	"github.com/AvengeMedia/dankfiles/core/internal/devices"
	"github.com/AvengeMedia/dankfiles/core/internal/mimeapps"
	"github.com/AvengeMedia/dankfiles/core/internal/netmount"
)

func Route(ctx context.Context, w *ConnWriter, req Request, deps Deps) {
	switch req.Method {
	case "version":
		Respond(w, req.ID, map[string]any{"version": deps.Version, "apiVersion": APIVersion})
		return
	case "describe":
		Respond(w, req.ID, map[string]any{"apiVersion": APIVersion, "methods": Methods})
		return
	}

	switch {
	case devices.Handles(req.Method):
		if deps.Devices == nil {
			RespondError(w, req.ID, "device service unavailable")
			return
		}
		deps.Devices.Handle(w, req)
	case netmount.Handles(req.Method):
		if deps.Network == nil {
			RespondError(w, req.ID, "network service unavailable")
			return
		}
		deps.Network.Handle(w, req)
	case mimeapps.Handles(req.Method):
		if deps.Apps == nil {
			RespondError(w, req.ID, "application service unavailable")
			return
		}
		deps.Apps.Handle(ctx, w, req)
	case strings.HasPrefix(req.Method, "files.op."):
		if deps.FileOps == nil {
			RespondError(w, req.ID, "file operations unavailable")
			return
		}
		deps.FileOps.Handle(ctx, w, req)
	case strings.HasPrefix(req.Method, "ui."):
		HandleUI(ctx, w, req, deps)
	case strings.HasPrefix(req.Method, "system."):
		HandleSystem(ctx, w, req, deps)
	case strings.HasPrefix(req.Method, "files."):
		if deps.Files == nil {
			RespondError(w, req.ID, "file service unavailable")
			return
		}
		deps.Files.Handle(ctx, w, req)
	default:
		RespondError(w, req.ID, "unknown method: "+req.Method)
	}
}
