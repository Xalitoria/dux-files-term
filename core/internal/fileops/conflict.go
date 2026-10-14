package fileops

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/AvengeMedia/dankgo/files"
)

type resolution struct {
	dst      string
	skip     bool
	merge    bool
	replaced bool
	abort    bool
	// unknown: the target could not be stat'ed, so nothing may replace it
	// silently later.
	unknown bool
}

func (r *runner) resolve(src, dst string, info os.FileInfo, move bool) resolution {
	target, err := os.Lstat(dst)
	switch {
	case errors.Is(err, fs.ErrNotExist):
		return resolution{dst: dst}
	case err != nil:
		return resolution{dst: dst, unknown: true}
	case os.SameFile(info, target) && move:
		return resolution{skip: true}
	case os.SameFile(info, target):
		return resolution{dst: uniquePath(dst)}
	}

	action := r.conflictAll
	if action == "" {
		action = r.askConflict(src, dst, info.IsDir() && target.IsDir())
	}

	switch action {
	case ActionReplace:
		return resolution{dst: dst, replaced: true}
	case ActionMerge:
		return resolution{dst: dst, merge: true}
	case ActionKeepBoth:
		return resolution{dst: uniquePath(dst)}
	case ActionSkip:
		r.skip()
		return resolution{skip: true}
	default:
		return resolution{abort: true}
	}
}

func (r *runner) askConflict(src, dst string, bothDirs bool) Action {
	conflict := &Conflict{
		ID:      r.job.nextID("c"),
		Source:  r.describe(src),
		Target:  r.describe(dst),
		Actions: []Action{ActionSkip, ActionReplace, ActionKeepBoth, ActionCancel},
	}
	if bothDirs {
		conflict.Actions = []Action{ActionMerge, ActionSkip, ActionReplace, ActionKeepBoth, ActionCancel}
	}

	answer := r.job.ask(event{Kind: "conflict", JobID: r.job.id, Conflict: conflict}, StatePausedConflict)
	if answer.ApplyToAll {
		r.conflictAll = answer.Action
	}
	return answer.Action
}

func (r *runner) describe(path string) files.Entry {
	if r.stater == nil {
		return files.Entry{Path: path, Name: filepath.Base(path)}
	}
	entry, err := r.stater.Stat(path)
	if err != nil {
		return files.Entry{Path: path, Name: filepath.Base(path)}
	}
	return entry
}

// uniquePath appends "(copy)", then "(copy 2)", "(copy 3)"... before the
// extension until the name is free.
func uniquePath(path string) string {
	dir := filepath.Dir(path)
	stem, ext := splitName(filepath.Base(path))

	for i := 1; ; i++ {
		suffix := " (copy)"
		if i > 1 {
			suffix = " (copy " + strconv.Itoa(i) + ")"
		}
		candidate := filepath.Join(dir, stem+suffix+ext)
		if _, err := os.Lstat(candidate); err != nil {
			return candidate
		}
	}
}

func numberedPath(path string) string {
	if _, err := os.Lstat(path); err != nil {
		return path
	}
	dir := filepath.Dir(path)
	stem, ext := splitName(filepath.Base(path))

	for i := 2; ; i++ {
		candidate := filepath.Join(dir, stem+" "+strconv.Itoa(i)+ext)
		if _, err := os.Lstat(candidate); err != nil {
			return candidate
		}
	}
}

func splitName(name string) (stem, ext string) {
	if strings.HasPrefix(name, ".") {
		rest := name[1:]
		dot := strings.LastIndex(rest, ".")
		if dot <= 0 {
			return name, ""
		}
		return name[:dot+1], rest[dot:]
	}
	ext = filepath.Ext(name)
	return strings.TrimSuffix(name, ext), ext
}
