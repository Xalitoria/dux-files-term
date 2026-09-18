package ipc

import (
	"context"
	"strings"

	"github.com/AvengeMedia/dankfiles/core/internal/files"
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
	case strings.HasPrefix(req.Method, "ui."):
		HandleUI(ctx, w, req, deps)
	case strings.HasPrefix(req.Method, "system."):
		HandleSystem(ctx, w, req, deps)
	case strings.HasPrefix(req.Method, "files."):
		files.Handle(ctx, w, req)
	default:
		RespondError(w, req.ID, "unknown method: "+req.Method)
	}
}
