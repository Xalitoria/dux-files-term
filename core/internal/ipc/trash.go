package ipc

import (
	"context"

	"github.com/AvengeMedia/dankgo/trash"
)

func HandleTrash(_ context.Context, w *ConnWriter, req Request) {
	switch req.Method {
	case "trash.list":
		entries, err := trash.List()
		if err != nil {
			w.RespondError(req.ID, "trash.list: "+err.Error())
			return
		}
		if entries == nil {
			entries = []trash.Entry{}
		}
		w.Respond(req.ID, map[string]any{"entries": entries})
	case "trash.count":
		count, err := trash.Count()
		if err != nil {
			w.RespondError(req.ID, "trash.count: "+err.Error())
			return
		}
		w.Respond(req.ID, map[string]any{"count": count})
	default:
		w.RespondError(req.ID, "unknown trash method: "+req.Method)
	}
}
