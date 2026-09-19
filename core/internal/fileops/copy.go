package fileops

import (
	"context"
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"syscall"
)

var copyChunk = 256 << 10

type outcome struct {
	dst      string
	undoable bool
}

func (r *runner) transferAll(move bool) error {
	pairs := r.targets()

	var needScan []string
	for _, p := range pairs {
		if move && sameDevice(p.src, filepath.Dir(p.dst)) {
			continue
		}
		needScan = append(needScan, p.src)
	}

	switch len(needScan) {
	case 0:
		r.setTotals(len(pairs), 0, false)
	default:
		count, bytes, complete := r.scan(needScan)
		r.setTotals(count+len(pairs)-len(needScan), bytes, !complete)
	}

	for _, p := range pairs {
		if r.job.closed() {
			return nil
		}
		if !r.guarded(p.src) {
			return errors.New("changed since the operation it would undo: " + p.src)
		}
		done, err := r.transfer(p.src, p.dst, move)
		switch {
		case errors.Is(err, errSkipped):
			continue
		case errors.Is(err, context.Canceled):
			return nil
		case err != nil:
			return err
		case done.undoable:
			r.remember(inverseOf(move), done.dst, p.src)
		}
	}
	return nil
}

func (r *runner) transfer(src, dst string, move bool) (outcome, error) {
	var info os.FileInfo
	if err := r.retry(src, func() error {
		var statErr error
		info, statErr = os.Lstat(src)
		return statErr
	}); err != nil {
		return outcome{}, err
	}

	res := r.resolve(src, dst, info.IsDir())
	switch {
	case res.abort:
		return outcome{}, context.Canceled
	case res.skip:
		return outcome{}, errSkipped
	}
	dst = res.dst

	undoable := !res.merge && !res.replaced
	if move && !res.merge && r.renameInto(src, dst, info) {
		return outcome{dst: dst, undoable: undoable}, nil
	}
	if err := r.clearTarget(dst, res, info.IsDir()); err != nil {
		return outcome{}, err
	}

	switch {
	case info.IsDir():
		return outcome{dst: dst, undoable: undoable}, r.transferDir(src, dst, move, res.merge)
	case info.Mode()&fs.ModeSymlink != 0:
		return outcome{dst: dst, undoable: undoable}, r.transferLink(src, dst, move)
	case info.Mode().IsRegular():
		return outcome{dst: dst, undoable: undoable}, r.transferFile(src, dst, info, move)
	default:
		r.skip()
		return outcome{}, errSkipped
	}
}

func (r *runner) renameInto(src, dst string, info os.FileInfo) bool {
	if !sameDevice(src, filepath.Dir(dst)) {
		return false
	}
	if _, err := os.Lstat(dst); err == nil && info.IsDir() {
		return false
	}
	if err := os.Rename(src, dst); err != nil {
		r.scanLate(src)
		return false
	}

	counted := r.scanned[src]
	if counted.files == 0 {
		counted.files = 1
	}
	r.advance(dst, counted.files, counted.bytes)
	return true
}

func (r *runner) scanLate(src string) {
	if _, known := r.scanned[src]; known {
		return
	}
	count, bytes, complete := r.scan([]string{src})
	r.job.mu.Lock()
	r.job.progress.FilesTotal += count - 1
	r.job.progress.BytesTotal += bytes
	r.job.progress.Estimating = r.job.progress.Estimating || !complete
	r.job.mu.Unlock()
	r.estimating = r.estimating || !complete
}

func (r *runner) clearTarget(dst string, res resolution, srcIsDir bool) error {
	if res.merge {
		return nil
	}
	target, err := os.Lstat(dst)
	if err != nil {
		return nil
	}
	if !target.IsDir() && !srcIsDir {
		return nil
	}
	return r.retry(dst, func() error { return os.RemoveAll(dst) })
}

func (r *runner) transferDir(src, dst string, move, merge bool) error {
	var info os.FileInfo
	if err := r.retry(src, func() error {
		var statErr error
		info, statErr = os.Stat(src)
		return statErr
	}); err != nil {
		return err
	}

	if err := r.retry(dst, func() error { return os.MkdirAll(dst, info.Mode().Perm()|0o700) }); err != nil {
		return err
	}

	var children []os.DirEntry
	if err := r.retry(src, func() error {
		var readErr error
		children, readErr = os.ReadDir(src)
		return readErr
	}); err != nil {
		return err
	}
	if r.estimating {
		r.growTotals(len(children))
	}

	for _, child := range children {
		if r.job.closed() {
			return context.Canceled
		}
		_, err := r.transfer(filepath.Join(src, child.Name()), filepath.Join(dst, child.Name()), move)
		switch {
		case err == nil, errors.Is(err, errSkipped):
			continue
		default:
			return err
		}
	}

	r.preserve(src, dst, info)
	r.advance(dst, 1, 0)
	if move {
		os.Remove(src)
	}
	return nil
}

func (r *runner) transferLink(src, dst string, move bool) error {
	var target string
	if err := r.retry(src, func() error {
		var readErr error
		target, readErr = os.Readlink(src)
		return readErr
	}); err != nil {
		return err
	}

	tmp := r.tempPath(dst)
	if err := r.retry(dst, func() error {
		os.Remove(tmp)
		return os.Symlink(target, tmp)
	}); err != nil {
		return err
	}
	if err := r.commit(tmp, dst); err != nil {
		return err
	}

	r.advance(dst, 1, 0)
	return r.dropSource(src, move)
}

func (r *runner) transferFile(src, dst string, info os.FileInfo, move bool) error {
	tmp := r.tempPath(dst)
	if err := r.retry(src, func() error {
		os.Remove(tmp)
		return r.copyBytes(src, tmp, info)
	}); err != nil {
		os.Remove(tmp)
		return err
	}

	r.preserve(src, tmp, info)
	if err := r.commit(tmp, dst); err != nil {
		return err
	}

	r.advance(dst, 1, 0)
	return r.dropSource(src, move)
}

func (r *runner) commit(tmp, dst string) error {
	if err := r.retry(dst, func() error { return os.Rename(tmp, dst) }); err != nil {
		os.Remove(tmp)
		return err
	}
	return nil
}

// dropSource removes a source file once its bytes landed, so cancelling a
// cross-device move leaves a clean split instead of a half-moved tree.
func (r *runner) dropSource(src string, move bool) error {
	if !move {
		return nil
	}
	return r.retry(src, func() error { return os.Remove(src) })
}

func (r *runner) copyBytes(src, tmp string, info os.FileInfo) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.OpenFile(tmp, os.O_WRONLY|os.O_CREATE|os.O_EXCL, info.Mode().Perm())
	if err != nil {
		return err
	}
	defer out.Close()

	buf := make([]byte, copyChunk)
	for {
		if r.job.closed() {
			return context.Canceled
		}
		n, readErr := in.Read(buf)
		if n > 0 {
			if _, writeErr := out.Write(buf[:n]); writeErr != nil {
				return writeErr
			}
			r.advance(src, 0, int64(n))
		}
		if readErr == io.EOF {
			return out.Close()
		}
		if readErr != nil {
			return readErr
		}
	}
}

func (r *runner) preserve(src, dst string, info os.FileInfo) {
	if err := os.Chmod(dst, info.Mode().Perm()); err != nil {
		r.warn(dst, err)
	}
	if err := os.Chtimes(dst, info.ModTime(), info.ModTime()); err != nil {
		r.warn(dst, err)
	}
	if err := copyXattrs(src, dst); err != nil {
		r.warn(dst, err)
	}
}

func (r *runner) growTotals(children int) {
	r.job.mu.Lock()
	r.job.progress.FilesTotal += children
	r.job.mu.Unlock()
}

func (r *runner) tempPath(dst string) string {
	return filepath.Join(filepath.Dir(dst), ".dfiles-tmp-"+r.job.id+r.job.nextID("-t")+"-"+filepath.Base(dst))
}

func inverseOf(move bool) Kind {
	if move {
		return KindMove
	}
	return KindDelete
}

func sameDevice(src, destDir string) bool {
	srcDev, ok := deviceOf(src)
	if !ok {
		return false
	}
	dstDev, ok := deviceOf(destDir)
	if !ok {
		return false
	}
	return srcDev == dstDev
}

func deviceOf(path string) (uint64, bool) {
	for {
		var st syscall.Stat_t
		if err := syscall.Lstat(path, &st); err == nil {
			return uint64(st.Dev), true
		}
		parent := filepath.Dir(path)
		if parent == path {
			return 0, false
		}
		path = parent
	}
}
