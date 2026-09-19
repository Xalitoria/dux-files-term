package fileops

import (
	"context"
	"errors"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"github.com/AvengeMedia/dankgo/trash"
)

const trashInfoExt = ".trashinfo"

func (r *runner) trash() error {
	sources := r.job.spec.sources
	count, bytes, complete := r.scan(sources)
	r.setTotals(count, bytes, !complete)

	for _, path := range sources {
		if r.job.closed() {
			return nil
		}

		var entry trash.Entry
		err := r.retry(path, func() error {
			var putErr error
			entry, putErr = trash.Put(path)
			return putErr
		})
		switch {
		case errors.Is(err, errSkipped):
			continue
		case errors.Is(err, context.Canceled):
			return nil
		case err != nil:
			return err
		}

		counted, measured := r.scanned[path]
		if entry.IsDir && measured {
			writeDirectorySize(entry, counted.bytes)
		}
		r.rememberPair(KindRestore, pair{src: entry.Name, dst: entry.TrashDir}, guard{})
		r.advance(path, max(counted.files, 1), counted.bytes)
	}
	return nil
}

func (r *runner) restore() error {
	items := r.restoreItems()
	r.setTotals(len(items), 0, false)

	for _, item := range items {
		if r.job.closed() {
			return nil
		}

		source := filepath.Join(item.dst, "files", item.src)
		var target string
		err := r.retry(source, func() error {
			var resolveErr error
			target, resolveErr = restoreTarget(item.dst, item.src)
			return resolveErr
		})
		switch {
		case errors.Is(err, errSkipped):
			continue
		case errors.Is(err, context.Canceled):
			return nil
		case err != nil:
			return err
		}

		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}

		done, err := r.transfer(source, target, true)
		switch {
		case errors.Is(err, errSkipped):
			continue
		case errors.Is(err, context.Canceled):
			return nil
		case err != nil:
			return err
		}

		os.Remove(filepath.Join(item.dst, "info", item.src+trashInfoExt))
		dropDirectorySize(item.dst, item.src)
		if done.undoable {
			r.rememberPair(KindTrash, pair{src: done.dst, dst: ""}, guard{})
		}
	}
	return nil
}

func (r *runner) restoreItems() []pair {
	if len(r.job.spec.pairs) > 0 {
		return r.job.spec.pairs
	}
	dir := r.job.spec.trashDir
	items := make([]pair, 0, len(r.job.spec.sources))
	for _, name := range r.job.spec.sources {
		items = append(items, pair{src: name, dst: dir})
	}
	return items
}

// restoreTarget mirrors the spec's Path rule: absolute for the home trash,
// relative to the trash dir's topdir for a per-mountpoint trash.
func restoreTarget(trashDir, name string) (string, error) {
	body, err := os.ReadFile(filepath.Join(trashDir, "info", name+trashInfoExt))
	if err != nil {
		return "", err
	}

	stored := ""
	for _, line := range strings.Split(string(body), "\n") {
		value, ok := strings.CutPrefix(line, "Path=")
		if !ok {
			continue
		}
		stored = value
		break
	}
	if stored == "" {
		return "", errors.New("invalid .trashinfo: missing Path")
	}
	if decoded, err := url.PathUnescape(stored); err == nil {
		stored = decoded
	}
	if filepath.IsAbs(stored) {
		return stored, nil
	}

	topDir := filepath.Dir(trashDir)
	if filepath.Base(topDir) == ".Trash" {
		topDir = filepath.Dir(topDir)
	}
	return filepath.Join(topDir, stored), nil
}

func (r *runner) emptyTrash() error {
	entries, err := trash.List()
	if err != nil {
		return err
	}
	r.setTotals(len(entries), 0, false)

	dirs := map[string]bool{}
	for _, entry := range entries {
		if r.job.closed() {
			return nil
		}
		if err := r.retry(entry.FilesPath, func() error { return os.RemoveAll(entry.FilesPath) }); err != nil {
			if errors.Is(err, errSkipped) {
				continue
			}
			return skipToNil(err)
		}
		os.Remove(entry.InfoPath)
		dirs[entry.TrashDir] = true
		r.advance(entry.FilesPath, 1, entry.Size)
	}

	for dir := range dirs {
		os.Remove(filepath.Join(dir, "directorysizes"))
	}
	return nil
}

func directorySizesPath(trashDir string) string {
	return filepath.Join(trashDir, "directorysizes")
}

// writeDirectorySize maintains the optional cache other trash implementations
// read instead of walking the tree: "<size> <info mtime> <encoded name>".
func writeDirectorySize(entry trash.Entry, size int64) {
	info, err := os.Lstat(entry.InfoPath)
	if err != nil {
		return
	}
	line := strconv.FormatInt(size, 10) + " " + strconv.FormatInt(info.ModTime().Unix(), 10) + " " + url.PathEscape(entry.Name)
	rewriteDirectorySizes(entry.TrashDir, entry.Name, line)
}

func dropDirectorySize(trashDir, name string) {
	rewriteDirectorySizes(trashDir, name, "")
}

var directorySizesLock sync.Mutex

func rewriteDirectorySizes(trashDir, name, line string) {
	directorySizesLock.Lock()
	defer directorySizesLock.Unlock()

	path := directorySizesPath(trashDir)
	body, _ := os.ReadFile(path)

	kept := make([]string, 0, 8)
	for _, existing := range strings.Split(string(body), "\n") {
		if existing == "" || directorySizeName(existing) == name {
			continue
		}
		kept = append(kept, existing)
	}
	if line != "" {
		kept = append(kept, line)
	}
	if len(kept) == 0 {
		os.Remove(path)
		return
	}

	tmp, err := os.CreateTemp(trashDir, ".directorysizes-*")
	if err != nil {
		return
	}
	defer os.Remove(tmp.Name())

	if _, err := tmp.WriteString(strings.Join(kept, "\n") + "\n"); err != nil {
		tmp.Close()
		return
	}
	if err := tmp.Close(); err != nil {
		return
	}
	if err := os.Chmod(tmp.Name(), 0o600); err != nil {
		return
	}
	_ = os.Rename(tmp.Name(), path)
}

func directorySizeName(line string) string {
	fields := strings.SplitN(line, " ", 3)
	if len(fields) < 3 {
		return ""
	}
	name, err := url.PathUnescape(fields[2])
	if err != nil {
		return fields[2]
	}
	return name
}
