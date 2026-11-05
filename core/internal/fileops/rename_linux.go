package fileops

import (
	"errors"
	"io/fs"
	"os"

	"golang.org/x/sys/unix"
)

func renameNoReplace(from, to string) error {
	err := unix.Renameat2(unix.AT_FDCWD, from, unix.AT_FDCWD, to, unix.RENAME_NOREPLACE)
	switch {
	case err == nil:
		return nil
	case errors.Is(err, unix.EINVAL), errors.Is(err, unix.ENOSYS), errors.Is(err, unix.ENOTSUP):
		return renameUnlessExists(from, to)
	}
	return &os.LinkError{Op: "rename", Old: from, New: to, Err: err}
}

// Filesystems like sshfs and FUSE mounts without flag support take this path.
func renameUnlessExists(from, to string) error {
	_, err := os.Lstat(to)
	switch {
	case err == nil:
		return &os.LinkError{Op: "rename", Old: from, New: to, Err: fs.ErrExist}
	case !errors.Is(err, fs.ErrNotExist):
		return err
	}
	return os.Rename(from, to)
}
