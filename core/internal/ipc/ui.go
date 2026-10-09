package ipc

import (
	"context"
	"errors"
	"os"
	"strings"
	"syscall"
	"time"

	"github.com/AvengeMedia/dankfiles/core/internal/fileops"
	"github.com/AvengeMedia/dankfiles/core/internal/location"
	"github.com/AvengeMedia/dankgo/paths"
)

func hostEnv() location.Env {
	return location.HostEnv(paths.XDGDataHome(), fileops.IsArchive)
}

func publishUI(deps Deps, payload map[string]any) {
	if deps.Pending == nil {
		deps.Bus.Publish("ui", payload)
		return
	}
	deps.Pending.Deliver(deps.Bus, payload)
}

func OpenLocations(deps Deps, open, sel []string, newWindow bool) error {
	if len(open)+len(sel) == 0 && !newWindow {
		return errors.New("nothing to open")
	}
	locations, err := location.Resolve(open, sel, hostEnv())
	if err != nil {
		return err
	}
	if locations == nil {
		locations = []location.Location{}
	}
	payload := map[string]any{"action": "open", "locations": locations}
	if newWindow {
		payload["newWindow"] = true
	}
	publishUI(deps, payload)
	return nil
}

func ShowProperties(deps Deps, targets []string) error {
	if len(targets) == 0 {
		return errors.New("nothing to show")
	}
	resolved, err := location.LocalPaths(targets, hostEnv())
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
		w.Respond(req.ID, map[string]any{"ok": true})
	case "ui.open":
		open := stringList(req.Params, "paths")
		if path := ParamString(req.Params, "path"); path != "" {
			open = append([]string{path}, open...)
		}
		if err := OpenLocations(deps, open, stringList(req.Params, "select"), ParamBool(req.Params, "newWindow")); err != nil {
			w.RespondError(req.ID, "ui.open: "+err.Error())
			return
		}
		w.Respond(req.ID, map[string]any{"ok": true})
	case "ui.properties":
		if err := ShowProperties(deps, stringList(req.Params, "paths")); err != nil {
			w.RespondError(req.ID, "ui.properties: "+err.Error())
			return
		}
		w.Respond(req.ID, map[string]any{"ok": true})
	case "ui.quit":
		w.Respond(req.ID, map[string]any{"ok": true})
		go func() {
			time.Sleep(100 * time.Millisecond)
			_ = syscall.Kill(os.Getpid(), syscall.SIGTERM)
		}()
	default:
		w.RespondError(req.ID, "unknown ui method: "+req.Method)
	}
}

func stringList(p map[string]any, key string) []string {
	if value := ParamString(p, key); value != "" {
		return []string{value}
	}
	return ParamStringSlice(p, key)
}
