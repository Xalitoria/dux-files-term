package fileops

import (
	"context"
	"errors"
	"os/user"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	"github.com/AvengeMedia/dankgo/files"
	"github.com/AvengeMedia/dankgo/ipc"
	"github.com/AvengeMedia/dankgo/ipc/params"
)

type errorResponse struct {
	ID    int    `json:"id,omitempty"`
	Error string `json:"error"`
	Code  string `json:"code"`
}

func (m *Manager) Handle(_ context.Context, w *ipc.ConnWriter, req ipc.Request) {
	switch req.Method {
	case "files.op.jobs":
		w.Respond(req.ID, map[string]any{"jobs": m.Jobs()})
		return
	case "files.op.cancel":
		m.handleCancel(w, req)
		return
	case "files.op.respond":
		m.handleRespond(w, req)
		return
	case "files.op.undo":
		m.handleUndo(w, req)
		return
	case "files.op.dropAction":
		handleDropAction(w, req)
		return
	}

	s, err := specOf(req)
	if err != nil {
		respondErr(w, req.ID, err)
		return
	}
	w.Respond(req.ID, map[string]any{"job": m.enqueue(s).view()})
}

func handleDropAction(w *ipc.ConnWriter, req ipc.Request) {
	sources, err := absPaths(req, "paths")
	if err != nil {
		respondErr(w, req.ID, err)
		return
	}
	dest := params.StringOpt(req.Params, "dest", "")
	if len(sources) == 0 || !filepath.IsAbs(dest) {
		respondErr(w, req.ID, invalid("files.op.dropAction requires paths and an absolute dest"))
		return
	}

	destDev, same := deviceOf(dest)
	for _, src := range sources {
		if !same {
			break
		}
		srcDev, found := deviceOf(src)
		same = found && srcDev == destDev
	}
	action := "copy"
	if same {
		action = "move"
	}
	w.Respond(req.ID, map[string]any{"action": action, "sameDevice": same})
}

func specOf(req ipc.Request) (spec, error) {
	s := spec{scope: params.StringOpt(req.Params, "scope", "")}

	switch req.Method {
	case "files.op.copy", "files.op.move", "files.op.link":
		s.kind = Kind(strings.TrimPrefix(req.Method, "files.op."))
		sources, err := absPaths(req, "paths")
		if err != nil {
			return spec{}, err
		}
		s.sources = sources
		s.dest = params.StringOpt(req.Params, "dest", "")
		if len(s.sources) == 0 || !filepath.IsAbs(s.dest) {
			return spec{}, invalid(req.Method + " requires paths and an absolute dest")
		}
		if nested := nestedSource(s.sources, s.dest); nested != "" {
			return spec{}, invalid("cannot put " + nested + " inside itself")
		}
	case "files.op.rename":
		s.kind = KindRename
		sources, err := absPaths(req, "path")
		if err != nil {
			return spec{}, err
		}
		s.sources = sources
		s.name = params.StringOpt(req.Params, "name", "")
		if len(s.sources) != 1 || !validName(s.name) {
			return spec{}, invalid("files.op.rename requires an absolute path and a name")
		}
	case "files.op.newFolder", "files.op.newFile":
		s.kind = Kind(strings.TrimPrefix(req.Method, "files.op."))
		s.dest = params.StringOpt(req.Params, "path", "")
		s.name = params.StringOpt(req.Params, "name", "")
		s.template = params.StringOpt(req.Params, "template", "")
		s.unique = params.BoolLoose(req.Params, "unique")
		if !filepath.IsAbs(s.dest) || !validName(s.name) {
			return spec{}, invalid(req.Method + " requires an absolute path and a name")
		}
	case "files.op.trash", "files.op.delete":
		s.kind = Kind(strings.TrimPrefix(req.Method, "files.op."))
		sources, err := absPaths(req, "paths")
		if err != nil {
			return spec{}, err
		}
		s.sources = sources
		if len(s.sources) == 0 {
			return spec{}, invalid(req.Method + " requires absolute paths")
		}
	case "files.op.restore":
		s.kind = KindRestore
		s.sources = stringsParam(req.Params, "names")
		s.trashDir = params.StringOpt(req.Params, "trashDir", "")
		if len(s.sources) == 0 || !filepath.IsAbs(s.trashDir) {
			return spec{}, invalid("files.op.restore requires names and an absolute trashDir")
		}
		for _, name := range s.sources {
			if !validName(name) {
				return spec{}, invalid("not a trash entry name: " + name)
			}
		}
	case "files.op.emptyTrash":
		s.kind = KindEmptyTrash
	case "files.op.compress":
		s.kind = KindCompress
		sources, err := absPaths(req, "paths")
		if err != nil {
			return spec{}, err
		}
		s.sources = sources
		s.dest = params.StringOpt(req.Params, "dest", "")
		s.name = params.StringOpt(req.Params, "name", "")
		s.format = params.StringOpt(req.Params, "format", FormatZip)
		if len(s.sources) == 0 || !filepath.IsAbs(s.dest) || !validName(s.name) {
			return spec{}, invalid("files.op.compress requires paths, an absolute dest and a name")
		}
		if !slices.Contains(compressFormats, s.format) {
			return spec{}, invalid("files.op.compress format must be one of " + strings.Join(compressFormats, ", "))
		}
		s.password = params.StringOpt(req.Params, "password", "")
		if s.password != "" && s.format != FormatZip {
			return spec{}, invalid("files.op.compress only protects zip archives with a password")
		}
	case "files.op.extract":
		s.kind = KindExtract
		sources, err := absPaths(req, "paths")
		if err != nil {
			return spec{}, err
		}
		s.sources = sources
		s.dest = params.StringOpt(req.Params, "dest", "")
		if len(s.sources) == 0 || !filepath.IsAbs(s.dest) {
			return spec{}, invalid("files.op.extract requires paths and an absolute dest")
		}
	case "files.op.setOwner":
		s.kind = KindSetOwner
		sources, err := absPaths(req, "paths")
		if err != nil {
			return spec{}, err
		}
		s.sources = sources
		s.recursive = params.BoolLoose(req.Params, "recursive")
		s.uid, err = lookupID(textParam(req.Params, "owner"), user.Lookup, func(u *user.User) string { return u.Uid })
		if err != nil {
			return spec{}, err
		}
		s.gid, err = lookupID(textParam(req.Params, "group"), user.LookupGroup, func(g *user.Group) string { return g.Gid })
		if err != nil {
			return spec{}, err
		}
		if len(s.sources) == 0 || (s.uid < 0 && s.gid < 0) {
			return spec{}, invalid("files.op.setOwner requires paths and an owner or a group")
		}
	case "files.op.setPermissions":
		s.kind = KindSetPermissions
		sources, err := absPaths(req, "paths")
		if err != nil {
			return spec{}, err
		}
		s.sources = sources
		s.recursive = params.BoolLoose(req.Params, "recursive")
		mode, parseErr := strconv.ParseUint(params.StringOpt(req.Params, "mode", ""), 8, 32)
		if parseErr != nil || len(s.sources) == 0 {
			return spec{}, invalid("files.op.setPermissions requires paths and an octal mode")
		}
		s.mode = uint32(mode)
	default:
		return spec{}, &files.Error{Code: files.CodeNotSupported, Err: errors.New("unknown files.op method: " + req.Method)}
	}
	return s, nil
}

func (m *Manager) handleCancel(w *ipc.ConnWriter, req ipc.Request) {
	id := params.StringOpt(req.Params, "jobId", "")
	if err := m.Cancel(id); err != nil {
		respondErr(w, req.ID, invalid(err.Error()))
		return
	}
	w.Respond(req.ID, map[string]any{"jobId": id, "cancelling": true})
}

func (m *Manager) handleRespond(w *ipc.ConnWriter, req ipc.Request) {
	jobID := params.StringOpt(req.Params, "jobId", "")
	answer := Answer{
		ID:         params.StringOpt(req.Params, "id", ""),
		Action:     Action(params.StringOpt(req.Params, "action", "")),
		ApplyToAll: params.BoolLoose(req.Params, "applyToAll"),
		Password:   params.StringOpt(req.Params, "password", ""),
	}
	if !knownAction(answer.Action) {
		respondErr(w, req.ID, invalid("unknown action: "+string(answer.Action)))
		return
	}
	if err := m.Respond(jobID, answer); err != nil {
		respondErr(w, req.ID, invalid(err.Error()))
		return
	}
	w.Respond(req.ID, map[string]any{"jobId": jobID, "accepted": true})
}

func (m *Manager) handleUndo(w *ipc.ConnWriter, req ipc.Request) {
	view, err := m.Undo(params.StringOpt(req.Params, "scope", ""))
	if err != nil {
		respondErr(w, req.ID, invalid(err.Error()))
		return
	}
	w.Respond(req.ID, map[string]any{"job": view})
}

// maxID excludes (uid_t)-1, which the kernel reads as "leave unchanged".
const maxID = 1<<32 - 1

// lookupID accepts a name or a numeric id; empty keeps the current value.
func lookupID[T any](raw string, lookup func(string) (*T, error), id func(*T) string) (int, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return -1, nil
	}
	if number, err := strconv.Atoi(raw); err == nil && number >= 0 {
		if number >= maxID {
			return -1, invalid("owner or group id out of range: " + raw)
		}
		return number, nil
	}
	found, err := lookup(raw)
	if err != nil {
		return -1, invalid("unknown owner or group: " + raw)
	}
	number, err := strconv.Atoi(id(found))
	if err != nil || number < 0 || number >= maxID {
		return -1, invalid("unknown owner or group: " + raw)
	}
	return number, nil
}

// textParam reads a string or a JSON number, since a uid arrives either way.
func textParam(p map[string]any, key string) string {
	switch value := p[key].(type) {
	case string:
		return value
	case float64:
		return strconv.FormatInt(int64(value), 10)
	case int:
		return strconv.Itoa(value)
	}
	return ""
}

func knownAction(action Action) bool {
	switch action {
	case ActionSkip, ActionReplace, ActionKeepBoth, ActionMerge, ActionRetry, ActionRetryElevated, ActionCancel, ActionUnlock:
		return true
	}
	return false
}

func validName(name string) bool {
	if name == "" || name == "." || name == ".." {
		return false
	}
	return !strings.ContainsAny(name, "/\x00")
}

// absPaths refuses the whole request when one entry is relative: a list given
// as a comma-joined string must not silently act on the parts that happen to
// parse.
func absPaths(req ipc.Request, key string) ([]string, error) {
	values := stringsParam(req.Params, key)
	if len(values) == 0 {
		values = stringsParam(req.Params, "path")
	}

	out := make([]string, 0, len(values))
	for _, value := range values {
		if !filepath.IsAbs(value) {
			return nil, invalid("not an absolute path: " + value)
		}
		out = append(out, filepath.Clean(value))
	}
	return out, nil
}

func nestedSource(sources []string, dest string) string {
	for _, src := range sources {
		if dest == src || isUnder(dest, src) {
			return src
		}
	}
	return ""
}

func isUnder(path, dir string) bool {
	rel, err := filepath.Rel(dir, path)
	if err != nil {
		return false
	}
	return rel != "." && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) && !filepath.IsAbs(rel)
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

func invalid(msg string) error {
	return &files.Error{Code: files.CodeInvalid, Err: errors.New(msg)}
}

func respondErr(w *ipc.ConnWriter, id int, err error) {
	_ = w.WriteResponse(errorResponse{ID: id, Error: err.Error(), Code: string(files.CodeOf(err))})
}
