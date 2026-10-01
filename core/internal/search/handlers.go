package search

import (
	"context"
	"strconv"
	"strings"

	"github.com/AvengeMedia/dankgo/files"
	"github.com/AvengeMedia/dankgo/ipc"
	"github.com/AvengeMedia/dankgo/ipc/params"
)

type errorResponse struct {
	ID    int        `json:"id,omitempty"`
	Error string     `json:"error"`
	Code  files.Code `json:"code"`
}

func Handles(method string) bool {
	return strings.HasPrefix(method, "search.")
}

func (s *Service) Handle(ctx context.Context, w *ipc.ConnWriter, req ipc.Request) {
	switch req.Method {
	case "search.start":
		s.handleStart(ctx, w, req)
	case "search.cancel":
		id := params.StringOpt(req.Params, "searchId", "")
		ipc.Respond(w, req.ID, map[string]any{"searchId": id, "cancelled": s.Cancel(id)})
	default:
		respondCode(w, req.ID, files.CodeNotSupported, "unknown search method: "+req.Method)
	}
}

func (s *Service) handleStart(ctx context.Context, w *ipc.ConnWriter, req ipc.Request) {
	p := req.Params
	opts := Options{
		IncludeHidden:   params.BoolLoose(p, "includeHidden"),
		CrossDevice:     params.BoolLoose(p, "crossDevice"),
		Types:           stringsParam(p, "types"),
		ModifiedAfterMs: int64Param(p, "modifiedAfterMs"),
		MinSize:         int64Param(p, "minSize"),
		MaxSize:         int64Param(p, "maxSize"),
		Limit:           int(int64Param(p, "limit")),
	}
	id, topic, err := s.Start(ctx, params.StringOpt(p, "root", ""), params.StringOpt(p, "query", ""), opts)
	if err != nil {
		respondCode(w, req.ID, files.CodeOf(err), err.Error())
		return
	}
	ipc.Respond(w, req.ID, map[string]any{"searchId": id, "topic": topic, "limit": opts.limit()})
}

func stringsParam(p map[string]any, key string) []string {
	if values := params.StringSlice(p, key); len(values) > 0 {
		return values
	}
	raw := strings.TrimSpace(params.StringOpt(p, key, ""))
	if raw == "" {
		return nil
	}
	return strings.Split(raw, ",")
}

func int64Param(p map[string]any, key string) int64 {
	switch v := p[key].(type) {
	case float64:
		return int64(v)
	case int:
		return int64(v)
	case int64:
		return v
	case string:
		parsed, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			return 0
		}
		return parsed
	}
	return 0
}

func respondCode(w *ipc.ConnWriter, id int, code files.Code, msg string) {
	_ = w.WriteResponse(errorResponse{ID: id, Error: msg, Code: code})
}
