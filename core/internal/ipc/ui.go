package ipc

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

// publishUI delivers a "ui" payload live when the GUI is listening, else stashes
// it for the next "ui" subscriber so a cold-started window still honors it.
func publishUI(deps Deps, payload map[string]any) {
	if deps.Bus.HasSubscriber("ui") {
		deps.Bus.Publish("ui", payload)
		return
	}
	if deps.Pending != nil {
		deps.Pending.Set(payload)
	}
}

func HandleUI(_ context.Context, w *ConnWriter, req Request, deps Deps) {
	switch req.Method {
	case "ui.show", "ui.hide", "ui.toggle":
		action := strings.TrimPrefix(req.Method, "ui.")
		deps.Bus.Publish("ui", map[string]any{"action": action})
		Respond(w, req.ID, map[string]any{"ok": true})
	case "ui.open":
		path := strings.TrimSpace(ParamString(req.Params, "path"))
		if path == "" {
			RespondError(w, req.ID, "ui.open requires a path")
			return
		}
		if !filepath.IsAbs(path) {
			RespondError(w, req.ID, "ui.open requires an absolute path")
			return
		}
		publishUI(deps, map[string]any{"action": "open", "path": path})
		Respond(w, req.ID, map[string]any{"ok": true})
	case "ui.quit":
		Respond(w, req.ID, map[string]any{"ok": true})
		go func() {
			time.Sleep(100 * time.Millisecond)
			_ = syscall.Kill(os.Getpid(), syscall.SIGTERM)
		}()
	default:
		RespondError(w, req.ID, "unknown ui method: "+req.Method)
	}
}
