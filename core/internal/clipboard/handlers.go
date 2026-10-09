package clipboard

import (
	"strings"

	"github.com/AvengeMedia/dankgo/ipc"
	"github.com/AvengeMedia/dankgo/ipc/params"
)

func (s *Service) Handle(w *ipc.ConnWriter, req ipc.Request) {
	switch req.Method {
	case "clipboard.copy":
		s.handleCopy(w, req)
	case "clipboard.paths":
		s.handlePaths(w, req)
	case "clipboard.clear":
		s.Clear()
		w.Respond(req.ID, map[string]any{"cleared": true})
	default:
		w.RespondError(req.ID, "unknown clipboard method: "+req.Method)
	}
}

func (s *Service) handleCopy(w *ipc.ConnWriter, req ipc.Request) {
	paths := pathsParam(req.Params)
	if len(paths) == 0 {
		w.RespondError(req.ID, "clipboard.copy requires paths")
		return
	}
	cut := params.BoolLoose(req.Params, "cut")
	if err := s.Copy(paths, cut); err != nil {
		w.RespondError(req.ID, err.Error())
		return
	}
	w.Respond(req.ID, map[string]any{"paths": paths, "cut": cut})
}

func (s *Service) handlePaths(w *ipc.ConnWriter, req ipc.Request) {
	paths, cut, source := s.Paths()
	w.Respond(req.ID, map[string]any{"paths": paths, "cut": cut, "source": source})
}

func pathsParam(p map[string]any) []string {
	if values := params.StringSlice(p, "paths"); len(values) > 0 {
		return values
	}
	raw := strings.TrimSpace(params.StringOpt(p, "paths", ""))
	if raw == "" {
		return nil
	}
	return strings.Split(raw, ",")
}
