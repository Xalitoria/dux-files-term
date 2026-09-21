package fileops

import (
	"context"
	"errors"
	"os"
	"path/filepath"
)

func (r *runner) rename() error {
	src := r.job.spec.sources[0]
	dst := filepath.Join(filepath.Dir(src), r.job.spec.name)
	r.setTotals(1, 0, false)

	if dst == src {
		return nil
	}
	if _, err := os.Lstat(dst); err == nil {
		return errors.New("a file named " + r.job.spec.name + " already exists")
	}
	if err := r.retry(src, func(fs FS) error { return fs.Rename(src, dst, true) }); err != nil {
		return skipToNil(err)
	}

	r.remember(KindMove, dst, src)
	r.advance(dst, 1, 0)
	return nil
}

func (r *runner) newFolder() error {
	path := r.target()
	r.setTotals(1, 0, false)

	if err := r.retry(path, func(fs FS) error { return fs.Mkdir(path, 0o755) }); err != nil {
		return skipToNil(err)
	}
	r.remember(KindDelete, path, "")
	r.advance(path, 1, 0)
	return nil
}

func (r *runner) newFile() error {
	path := r.target()
	r.setTotals(1, 0, false)

	if _, err := os.Lstat(path); err == nil {
		return errors.New("a file named " + r.job.spec.name + " already exists")
	}

	template := r.job.spec.template
	if template == "" {
		if err := r.retry(path, func(fs FS) error { return create(fs, path) }); err != nil {
			return skipToNil(err)
		}
		r.remember(KindDelete, path, "")
		r.advance(path, 1, 0)
		return nil
	}

	var info os.FileInfo
	if err := r.retry(template, func(fs FS) error {
		var statErr error
		info, statErr = fs.Lstat(template)
		return statErr
	}); err != nil {
		return skipToNil(err)
	}
	if err := r.transferFile(template, path, info, false, true); err != nil {
		return skipToNil(err)
	}
	r.remember(KindDelete, path, "")
	return nil
}

func create(fs FS, path string) error {
	file, err := fs.Create(path, 0o644)
	if err != nil {
		return err
	}
	return file.Close()
}

func (r *runner) link() error {
	sources := r.job.spec.sources
	r.setTotals(len(sources), 0, false)

	for _, src := range sources {
		if r.job.closed() {
			return nil
		}
		dst := filepath.Join(r.job.spec.dest, filepath.Base(src))
		if _, err := os.Lstat(dst); err == nil {
			dst = uniquePath(dst)
		}
		if err := r.retry(dst, func(fs FS) error { return fs.Symlink(src, dst) }); err != nil {
			if errors.Is(err, errSkipped) {
				continue
			}
			return skipToNil(err)
		}
		r.remember(KindDelete, dst, "")
		r.advance(dst, 1, 0)
	}
	return nil
}

func (r *runner) setPermissions() error {
	mode := os.FileMode(r.job.spec.mode).Perm()
	sources := r.job.spec.sources
	r.setTotals(len(sources), 0, false)

	for _, path := range sources {
		if r.job.closed() {
			return nil
		}
		err := r.retry(path, func(fs FS) error { return fs.Chmod(path, mode, r.job.spec.recursive) })
		switch {
		case errors.Is(err, errSkipped):
			continue
		case errors.Is(err, context.Canceled):
			return nil
		case err != nil:
			return err
		}
		r.advance(path, 1, 0)
	}
	return nil
}

func (r *runner) setOwner() error {
	sources := r.job.spec.sources
	r.setTotals(len(sources), 0, false)

	for _, path := range sources {
		if r.job.closed() {
			return nil
		}
		err := r.retry(path, func(fs FS) error { return fs.Lchown(path, r.job.spec.uid, r.job.spec.gid, r.job.spec.recursive) })
		switch {
		case errors.Is(err, errSkipped):
			continue
		case errors.Is(err, context.Canceled):
			return nil
		case err != nil:
			return err
		}
		r.advance(path, 1, 0)
	}
	return nil
}

func (r *runner) deleteAll() error {
	sources := r.job.spec.sources
	count, bytes, complete := r.scan(sources)
	r.setTotals(count, bytes, !complete)

	for _, path := range sources {
		if r.job.closed() {
			return nil
		}
		if !r.guarded(path) {
			return errors.New("changed since the operation it would undo: " + path)
		}
		err := r.retry(path, func(fs FS) error { return fs.RemoveAll(path) })
		switch {
		case errors.Is(err, errSkipped):
			continue
		case errors.Is(err, context.Canceled):
			return nil
		case err != nil:
			return err
		}
		counted := r.scanned[path]
		r.advance(path, max(counted.files, 1), counted.bytes)
	}
	return nil
}

// guarded holds an undo back when the file it would remove or move is no
// longer the one the original operation produced.
func (r *runner) guarded(path string) bool {
	if !r.job.spec.undo {
		return true
	}
	for i, p := range r.job.spec.pairs {
		if p.src != path || i >= len(r.job.spec.guards) {
			continue
		}
		return r.job.spec.guards[i].matches(path)
	}
	return true
}

func (r *runner) target() string {
	return filepath.Join(r.job.spec.dest, r.job.spec.name)
}

func skipToNil(err error) error {
	if errors.Is(err, errSkipped) || errors.Is(err, context.Canceled) {
		return nil
	}
	return err
}
