package mimeapps

import (
	"context"

	"github.com/AvengeMedia/dankgo/ipc"
	"github.com/AvengeMedia/dankgo/ipc/params"
)

type errorResponse struct {
	ID    int    `json:"id,omitempty"`
	Error string `json:"error"`
	Code  string `json:"code"`
}

type appView struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	GenericName string `json:"genericName,omitempty"`
	Comment     string `json:"comment,omitempty"`
	Icon        string `json:"icon,omitempty"`
	Terminal    bool   `json:"terminal"`
	IsDefault   bool   `json:"isDefault"`
}

type appsResult struct {
	MimeType  string    `json:"mimeType,omitempty"`
	DefaultID string    `json:"defaultId,omitempty"`
	Apps      []appView `json:"apps"`
}

// Handles reports whether a method belongs to this service.
func Handles(method string) bool {
	switch method {
	case "files.open", "files.openWith", "files.openTerminal", "files.run",
		"files.apps.forMime", "files.apps.all", "files.apps.setDefault", "files.apps.terminals":
		return true
	}
	return false
}

func (s *Service) Handle(ctx context.Context, w *ipc.ConnWriter, req ipc.Request) {
	switch req.Method {
	case "files.open":
		s.handleOpen(ctx, w, req)
	case "files.openWith":
		s.handleOpenWith(ctx, w, req)
	case "files.run":
		s.handleRun(w, req)
	case "files.openTerminal":
		s.handleOpenTerminal(w, req)
	case "files.apps.forMime":
		s.handleForMime(w, req)
	case "files.apps.all":
		ipc.Respond(w, req.ID, appsResult{Apps: viewsOf(s.Apps(), "")})
	case "files.apps.setDefault":
		s.handleSetDefault(w, req)
	case "files.apps.terminals":
		ipc.Respond(w, req.ID, map[string]any{"terminals": Terminals()})
	default:
		respondCode(w, req.ID, "NOTSUPPORTED", "unknown method: "+req.Method)
	}
}

func (s *Service) handleOpen(ctx context.Context, w *ipc.ConnWriter, req ipc.Request) {
	paths := params.StringSlice(req.Params, "paths")
	if len(paths) == 0 {
		respondCode(w, req.ID, "EINVAL", "files.open requires paths")
		return
	}
	if err := s.Open(ctx, paths, launchOptions(req)); err != nil {
		respondErr(w, req.ID, err)
		return
	}
	ipc.Respond(w, req.ID, map[string]any{"opened": true})
}

func (s *Service) handleOpenWith(ctx context.Context, w *ipc.ConnWriter, req ipc.Request) {
	paths := params.StringSlice(req.Params, "paths")
	desktopID := params.StringOpt(req.Params, "desktopId", "")
	if len(paths) == 0 || desktopID == "" {
		respondCode(w, req.ID, "EINVAL", "files.openWith requires paths and a desktopId")
		return
	}
	if err := s.OpenWith(ctx, desktopID, paths, launchOptions(req)); err != nil {
		respondErr(w, req.ID, err)
		return
	}
	ipc.Respond(w, req.ID, map[string]any{"opened": true})
}

func (s *Service) handleRun(w *ipc.ConnWriter, req ipc.Request) {
	path := params.StringOpt(req.Params, "path", "")
	if path == "" {
		respondCode(w, req.ID, "EINVAL", "files.run requires a path")
		return
	}
	if err := s.Run(path, launchOptions(req)); err != nil {
		respondErr(w, req.ID, err)
		return
	}
	ipc.Respond(w, req.ID, map[string]any{"started": true})
}

func (s *Service) handleOpenTerminal(w *ipc.ConnWriter, req ipc.Request) {
	path := params.StringOpt(req.Params, "path", "")
	if path == "" {
		respondCode(w, req.ID, "EINVAL", "files.openTerminal requires a path")
		return
	}
	if err := s.OpenTerminal(path, params.StringOpt(req.Params, "terminal", "")); err != nil {
		respondErr(w, req.ID, err)
		return
	}
	ipc.Respond(w, req.ID, map[string]any{"opened": true})
}

func (s *Service) handleForMime(w *ipc.ConnWriter, req ipc.Request) {
	mimeType := params.StringOpt(req.Params, "mime", "")
	if path := params.StringOpt(req.Params, "path", ""); mimeType == "" && path != "" {
		resolved, err := s.MimeOf(path)
		if err != nil {
			respondErr(w, req.ID, err)
			return
		}
		mimeType = resolved
	}
	if mimeType == "" {
		respondCode(w, req.ID, "EINVAL", "files.apps.forMime requires a mime or a path")
		return
	}

	defaultID := s.Default(mimeType)
	ipc.Respond(w, req.ID, appsResult{
		MimeType:  mimeType,
		DefaultID: defaultID,
		Apps:      viewsOf(s.Associations(mimeType), defaultID),
	})
}

func (s *Service) handleSetDefault(w *ipc.ConnWriter, req ipc.Request) {
	mimeType := params.StringOpt(req.Params, "mime", "")
	desktopID := params.StringOpt(req.Params, "desktopId", "")
	if mimeType == "" || desktopID == "" {
		respondCode(w, req.ID, "EINVAL", "files.apps.setDefault requires a mime and a desktopId")
		return
	}
	if Sandboxed() && !writableConfig() {
		respondCode(w, req.ID, "SANDBOXED", "the sandbox cannot write mimeapps.list")
		return
	}
	if err := s.SetDefault(mimeType, desktopID); err != nil {
		respondCode(w, req.ID, "EACCES", err.Error())
		return
	}
	ipc.Respond(w, req.ID, map[string]any{"mimeType": mimeType, "desktopId": desktopID})
}

func launchOptions(req ipc.Request) LaunchOptions {
	return LaunchOptions{
		WorkingDir:      params.StringOpt(req.Params, "cwd", ""),
		ActivationToken: params.StringOpt(req.Params, "activationToken", ""),
		Terminal:        params.StringOpt(req.Params, "terminal", ""),
	}
}

func viewsOf(entries []*Entry, defaultID string) []appView {
	out := make([]appView, 0, len(entries))
	for _, entry := range entries {
		out = append(out, appView{
			ID:          entry.ID,
			Name:        entry.Name,
			GenericName: entry.GenericName,
			Comment:     entry.Comment,
			Icon:        entry.Icon,
			Terminal:    entry.Terminal,
			IsDefault:   entry.ID == defaultID,
		})
	}
	return out
}

func respondErr(w *ipc.ConnWriter, id int, err error) {
	respondCode(w, id, CodeOf(err), err.Error())
}

func respondCode(w *ipc.ConnWriter, id int, code, msg string) {
	_ = w.WriteResponse(errorResponse{ID: id, Error: msg, Code: code})
}
