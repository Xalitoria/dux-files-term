package netmount

import (
	"strings"

	"github.com/AvengeMedia/dankfiles/core/internal/location"
	"github.com/AvengeMedia/dankgo/ipc"
	"github.com/AvengeMedia/dankgo/ipc/params"
)

type errorResponse struct {
	ID    int    `json:"id,omitempty"`
	Error string `json:"error"`
	Code  string `json:"code"`
}

func Handles(method string) bool {
	switch method {
	case "network.mounts", "network.connect", "network.disconnect", "network.answer",
		"network.recent", "network.forget", "network.mountPortable":
		return true
	}
	return false
}

func (s *Service) Handle(w *ipc.ConnWriter, req ipc.Request) {
	switch req.Method {
	case "network.mounts":
		snapshot := s.Snapshot()
		w.Respond(req.ID, map[string]any{
			"available": snapshot.Available,
			"reason":    snapshot.Reason,
			"schemes":   snapshot.Schemes,
			"mounts":    snapshot.Mounts,
			"portables": s.Portables(),
		})
	case "network.connect":
		s.handleConnect(w, req)
	case "network.disconnect":
		s.handleDisconnect(w, req)
	case "network.answer":
		s.handleAnswer(w, req)
	case "network.recent":
		w.Respond(req.ID, map[string]any{"servers": s.Servers()})
	case "network.forget":
		s.Forget(params.StringOpt(req.Params, "uri", ""))
		w.Respond(req.ID, map[string]any{"servers": s.Servers()})
	case "network.mountPortable":
		s.handleMountPortable(w, req)
	default:
		respondCode(w, req.ID, "NOTSUPPORTED", "unknown network method: "+req.Method)
	}
}

func (s *Service) handleConnect(w *ipc.ConnWriter, req ipc.Request) {
	uri := params.StringOpt(req.Params, "uri", "")
	if uri == "" {
		respondCode(w, req.ID, "EINVAL", "network.connect requires a uri")
		return
	}

	remote := location.SplitPassword(strings.TrimSpace(uri))
	uri = remote.URI
	password := params.StringOpt(req.Params, "password", remote.Password)
	mount, err := s.Connect(uri, password)
	if err != nil {
		respondErr(w, req.ID, err)
		return
	}
	location := mount.Path
	if spec, err := ParseURI(uri, s.mountableList()); err == nil {
		location = LocationIn(mount, URIOf(spec, s.mountableList()))
	}
	w.Respond(req.ID, map[string]any{"mount": mount, "location": location})
}

func (s *Service) handleDisconnect(w *ipc.ConnWriter, req ipc.Request) {
	id := params.StringOpt(req.Params, "id", "")
	if id == "" {
		respondCode(w, req.ID, "EINVAL", "network.disconnect requires an id")
		return
	}
	if err := s.Disconnect(id); err != nil {
		respondErr(w, req.ID, err)
		return
	}
	w.Respond(req.ID, map[string]any{"id": id, "mounted": false})
}

func (s *Service) handleAnswer(w *ipc.ConnWriter, req ipc.Request) {
	id := params.StringOpt(req.Params, "id", "")
	if id == "" {
		respondCode(w, req.ID, "EINVAL", "network.answer requires an id")
		return
	}

	answer := Answer{
		Abort:     params.BoolLoose(req.Params, "abort"),
		Password:  params.StringOpt(req.Params, "password", ""),
		User:      params.StringOpt(req.Params, "user", ""),
		Domain:    params.StringOpt(req.Params, "domain", ""),
		Anonymous: params.BoolLoose(req.Params, "anonymous"),
		Save:      uint32(params.IntOpt(req.Params, "save", 0)),
		Choice:    uint32(params.IntOpt(req.Params, "choice", 0)),
	}
	if err := s.Answer(id, answer); err != nil {
		respondErr(w, req.ID, err)
		return
	}
	w.Respond(req.ID, map[string]any{"id": id, "answered": true})
}

func (s *Service) handleMountPortable(w *ipc.ConnWriter, req ipc.Request) {
	monitor := params.StringOpt(req.Params, "monitor", "")
	id := params.StringOpt(req.Params, "id", "")
	if monitor == "" || id == "" {
		respondCode(w, req.ID, "EINVAL", "network.mountPortable requires a monitor and an id")
		return
	}
	if err := s.MountPortable(monitor, id); err != nil {
		respondErr(w, req.ID, err)
		return
	}
	w.Respond(req.ID, map[string]any{"id": id, "mounted": true})
}

func respondErr(w *ipc.ConnWriter, id int, err error) {
	respondCode(w, id, CodeOf(err), err.Error())
}

func respondCode(w *ipc.ConnWriter, id int, code, msg string) {
	_ = w.WriteResponse(errorResponse{ID: id, Error: msg, Code: code})
}
