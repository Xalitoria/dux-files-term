package ipc

import (
	"context"
	"errors"
	"os"
	"strings"
	"syscall"
	"time"

	"github.com/AvengeMedia/dankfiles/core/internal/location"
	"github.com/AvengeMedia/dankgo/paths"
)

func publishUI(deps Deps, payload map[string]any) {
	if deps.Pending == nil {
		deps.Bus.Publish("ui", payload)
		return
	}
	deps.Pending.Deliver(deps.Bus, payload)
}

func OpenLocations(deps Deps, open, sel []string) error {
	if len(open)+len(sel) == 0 {
		return errors.New("nothing to open")
	}
	locations, err := location.Resolve(open, sel, location.HostEnv(paths.XDGDataHome()))
	if err != nil {
		return err
	}
	publishUI(deps, map[string]any{"action": "open", "locations": locations})
	return nil
}

func ShowProperties(deps Deps, targets []string) error {
	if len(targets) == 0 {
		return errors.New("nothing to show")
	}
	resolved, err := location.LocalPaths(targets, location.HostEnv(paths.XDGDataHome()))
	if err != nil {
		return err
	}
	publishUI(deps, map[string]any{"action": "properties", "paths": resolved})
	return nil
}

func HandleUI(_ context.Context, w *ConnWriter, req Request, deps Deps) {
	switch req.Method {
	case "ui.show", "ui.hide", "ui.toggle":
		action := strings.TrimPrefix(req.Method, "ui.")
		deps.Bus.Publish("ui", map[string]any{"action": action})
		Respond(w, req.ID, map[string]any{"ok": true})
	case "ui.open":
		open := stringList(req.Params, "paths")
		if path := ParamString(req.Params, "path"); path != "" {
			open = append([]string{path}, open...)
		}
		if err := OpenLocations(deps, open, stringList(req.Params, "select")); err != nil {
			RespondError(w, req.ID, "ui.open: "+err.Error())
			return
		}
		Respond(w, req.ID, map[string]any{"ok": true})
	case "ui.properties":
		if err := ShowProperties(deps, stringList(req.Params, "paths")); err != nil {
			RespondError(w, req.ID, "ui.properties: "+err.Error())
			return
		}
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

func stringList(p map[string]any, key string) []string {
	if value := ParamString(p, key); value != "" {
		return []string{value}
	}
	return ParamStringSlice(p, key)
}
