package ipc

import (
	"context"
	"strings"

	"github.com/AvengeMedia/dankfiles/core/internal/mimeapps"
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
