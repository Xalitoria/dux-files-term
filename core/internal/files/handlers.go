package files

import (
	"context"
	"strings"

	"github.com/AvengeMedia/dankgo/ipc"
	"github.com/AvengeMedia/dankgo/ipc/params"
)

type errorResponse struct {
	ID    int    `json:"id,omitempty"`
	Error string `json:"error"`
	Code  Code   `json:"code"`
}

type listResult struct {
	Path    string  `json:"path"`
	Entries []Entry `json:"entries"`
}

func Handle(_ context.Context, w *ipc.ConnWriter, req ipc.Request) {
	switch req.Method {
	case "files.list":
		handleList(w, req)
	default:
		respondCode(w, req.ID, CodeNotSupported, "unknown files method: "+req.Method)
	}
}

func handleList(w *ipc.ConnWriter, req ipc.Request) {
	dir := strings.TrimSpace(params.StringOpt(req.Params, "path", ""))
	if dir == "" {
		respondCode(w, req.ID, CodeInvalid, "files.list requires a path")
		return
	}
	entries, err := List(dir, ListOptions{IncludeHidden: params.BoolLoose(req.Params, "includeHidden")})
	if err != nil {
		respondErr(w, req.ID, err)
		return
	}
	ipc.Respond(w, req.ID, listResult{Path: dir, Entries: entries})
}

func respondErr(w *ipc.ConnWriter, id int, err error) {
	respondCode(w, id, CodeOf(err), err.Error())
}

func respondCode(w *ipc.ConnWriter, id int, code Code, msg string) {
	_ = w.WriteResponse(errorResponse{ID: id, Error: msg, Code: code})
}
