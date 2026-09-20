package devices

import (
	"github.com/AvengeMedia/dankgo/ipc"
	"github.com/AvengeMedia/dankgo/ipc/params"
	"github.com/godbus/dbus/v5"
)

type errorResponse struct {
	ID    int    `json:"id,omitempty"`
	Error string `json:"error"`
	Code  string `json:"code"`
}

func Handles(method string) bool {
	switch method {
	case "devices.list", "devices.usage", "devices.mount", "devices.unmount",
		"devices.eject", "devices.powerOff", "devices.unlock", "devices.lock":
		return true
	}
	return false
}

func (s *Service) Handle(w *ipc.ConnWriter, req ipc.Request) {
	switch req.Method {
	case "devices.list":
		ipc.Respond(w, req.ID, s.Snapshot())
	case "devices.usage":
		ipc.Respond(w, req.ID, map[string]any{"usage": s.Usage(params.StringSlice(req.Params, "paths"), params.BoolLoose(req.Params, "force"))})
	case "devices.mount":
		s.handleMount(w, req)
	case "devices.unmount":
		s.handleUnmount(w, req)
	case "devices.eject":
		s.handleEject(w, req)
	case "devices.powerOff":
		s.handlePowerOff(w, req)
	case "devices.unlock":
		s.handleUnlock(w, req)
	case "devices.lock":
		s.handleLock(w, req)
	default:
		respondCode(w, req.ID, "NOTSUPPORTED", "unknown devices method: "+req.Method)
	}
}

func (s *Service) handleMount(w *ipc.ConnWriter, req ipc.Request) {
	id, ok := deviceID(w, req)
	if !ok {
		return
	}
	mountPoint, err := s.Mount(id)
	if err != nil {
		respondErr(w, req.ID, err)
		return
	}
	ipc.Respond(w, req.ID, map[string]any{"id": id, "mountPoint": mountPoint})
}

func (s *Service) handleUnmount(w *ipc.ConnWriter, req ipc.Request) {
	id, ok := deviceID(w, req)
	if !ok {
		return
	}
	if err := s.Unmount(id, params.BoolLoose(req.Params, "force")); err != nil {
		respondErr(w, req.ID, err)
		return
	}
	ipc.Respond(w, req.ID, map[string]any{"id": id, "mounted": false})
}

func (s *Service) handleEject(w *ipc.ConnWriter, req ipc.Request) {
	id, ok := deviceID(w, req)
	if !ok {
		return
	}
	if err := s.Eject(id); err != nil {
		respondErr(w, req.ID, err)
		return
	}
	ipc.Respond(w, req.ID, map[string]any{"id": id, "ejected": true})
}

func (s *Service) handlePowerOff(w *ipc.ConnWriter, req ipc.Request) {
	id, ok := deviceID(w, req)
	if !ok {
		return
	}
	if err := s.PowerOff(id); err != nil {
		respondErr(w, req.ID, err)
		return
	}
	ipc.Respond(w, req.ID, map[string]any{"id": id, "poweredOff": true})
}

func (s *Service) handleUnlock(w *ipc.ConnWriter, req ipc.Request) {
	id, ok := deviceID(w, req)
	if !ok {
		return
	}
	passphrase := params.StringOpt(req.Params, "passphrase", "")
	if passphrase == "" {
		respondCode(w, req.ID, "EINVAL", "devices.unlock requires a passphrase")
		return
	}

	cleartext, err := s.Unlock(id, passphrase)
	if err != nil {
		respondErr(w, req.ID, err)
		return
	}

	result := map[string]any{"id": id, "cleartext": cleartext}
	if params.BoolLoose(req.Params, "mount") {
		mountPoint, err := s.mountPath(dbus.ObjectPath(cleartext))
		switch err {
		case nil:
			result["mountPoint"] = mountPoint
		default:
			result["mountError"] = err.Error()
			result["mountCode"] = CodeOf(err)
		}
	}
	ipc.Respond(w, req.ID, result)
}

func (s *Service) handleLock(w *ipc.ConnWriter, req ipc.Request) {
	id, ok := deviceID(w, req)
	if !ok {
		return
	}
	if err := s.Lock(id); err != nil {
		respondErr(w, req.ID, err)
		return
	}
	ipc.Respond(w, req.ID, map[string]any{"id": id, "unlocked": false})
}

// The object table has not seen the cleartext device yet right after an unlock.
func (s *Service) mountPath(path dbus.ObjectPath) (string, error) {
	var mountPoint string
	if err := s.call(path, FsIface+".Mount", noOptions).Store(&mountPoint); err != nil {
		return "", wrapDBus(err)
	}
	s.usage.forget(mountPoint)
	return mountPoint, nil
}

func deviceID(w *ipc.ConnWriter, req ipc.Request) (string, bool) {
	id := params.StringOpt(req.Params, "id", "")
	if id == "" {
		respondCode(w, req.ID, "EINVAL", req.Method+" requires a device id")
		return "", false
	}
	return id, true
}

func respondErr(w *ipc.ConnWriter, id int, err error) {
	respondCode(w, id, CodeOf(err), err.Error())
}

func respondCode(w *ipc.ConnWriter, id int, code, msg string) {
	_ = w.WriteResponse(errorResponse{ID: id, Error: msg, Code: code})
}
