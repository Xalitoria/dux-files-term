package bookmarks

import (
	"strings"

	"github.com/duxlabs-dev/dux-go/ipc"
	"github.com/duxlabs-dev/dux-go/ipc/params"
)

func Handles(method string) bool {
	return strings.HasPrefix(method, "bookmarks.")
}

func (s *Store) Handle(w *ipc.ConnWriter, req ipc.Request) {
	switch req.Method {
	case "bookmarks.list":
		w.Respond(req.ID, map[string]any{"bookmarks": s.List(), "seeded": s.Seeded(), "gtkImportAvailable": s.GTKImportAvailable()})
	case "bookmarks.add":
		items, err := s.Add(Bookmark{
			Path:  params.StringOpt(req.Params, "path", ""),
			Label: params.StringOpt(req.Params, "label", ""),
			Icon:  params.StringOpt(req.Params, "icon", ""),
		})
		respond(w, req.ID, items, err)
	case "bookmarks.remove":
		items, err := s.Remove(params.StringOpt(req.Params, "path", ""))
		respond(w, req.ID, items, err)
	case "bookmarks.rename":
		items, err := s.Rename(params.StringOpt(req.Params, "path", ""), params.StringOpt(req.Params, "label", ""))
		respond(w, req.ID, items, err)
	case "bookmarks.seed":
		items, err := s.Seed(params.StringSlice(req.Params, "paths"))
		respond(w, req.ID, items, err)
	case "bookmarks.reorder":
		items, err := s.Reorder(params.StringSlice(req.Params, "paths"))
		respond(w, req.ID, items, err)
	case "bookmarks.importGtk":
		items, added, err := s.ImportGTK()
		if err != nil {
			w.RespondError(req.ID, err.Error())
			return
		}
		w.Respond(req.ID, map[string]any{"bookmarks": items, "added": added})
	default:
		w.RespondError(req.ID, "unknown bookmarks method: "+req.Method)
	}
}

func respond(w *ipc.ConnWriter, id int, items []Bookmark, err error) {
	if err != nil {
		w.RespondError(id, err.Error())
		return
	}
	w.Respond(id, map[string]any{"bookmarks": items})
}
