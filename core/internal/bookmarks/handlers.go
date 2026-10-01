package bookmarks

import (
	"strings"

	"github.com/AvengeMedia/dankgo/ipc"
	"github.com/AvengeMedia/dankgo/ipc/params"
)

func Handles(method string) bool {
	return strings.HasPrefix(method, "bookmarks.")
}

func (s *Store) Handle(w *ipc.ConnWriter, req ipc.Request) {
	switch req.Method {
	case "bookmarks.list":
		ipc.Respond(w, req.ID, map[string]any{"bookmarks": s.List(), "gtkImportAvailable": s.GTKImportAvailable()})
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
	case "bookmarks.reorder":
		items, err := s.Reorder(params.StringSlice(req.Params, "paths"))
		respond(w, req.ID, items, err)
	case "bookmarks.importGtk":
		items, added, err := s.ImportGTK()
		if err != nil {
			ipc.RespondError(w, req.ID, err.Error())
			return
		}
		ipc.Respond(w, req.ID, map[string]any{"bookmarks": items, "added": added})
	default:
		ipc.RespondError(w, req.ID, "unknown bookmarks method: "+req.Method)
	}
}

func respond(w *ipc.ConnWriter, id int, items []Bookmark, err error) {
	if err != nil {
		ipc.RespondError(w, id, err.Error())
		return
	}
	ipc.Respond(w, id, map[string]any{"bookmarks": items})
}
