package files

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

type Entry struct {
	Name          string `json:"name"`
	Path          string `json:"path"`
	IsDir         bool   `json:"isDir"`
	IsSymlink     bool   `json:"isSymlink"`
	SymlinkTarget string `json:"symlinkTarget"`
	Size          int64  `json:"size"`
	MtimeMs       int64  `json:"mtimeMs"`
	Mode          string `json:"mode"`
	Hidden        bool   `json:"hidden"`
}

func entryFromDirEntry(dir string, de fs.DirEntry) (Entry, error) {
	info, err := de.Info()
	if err != nil {
		return Entry{}, err
	}
	full := filepath.Join(dir, de.Name())
	e := Entry{
		Name:    de.Name(),
		Path:    full,
		IsDir:   info.IsDir(),
		Size:    info.Size(),
		MtimeMs: info.ModTime().UnixMilli(),
		Mode:    info.Mode().String(),
		Hidden:  strings.HasPrefix(de.Name(), "."),
	}
	if info.Mode()&fs.ModeSymlink == 0 {
		return e, nil
	}
	e.IsSymlink = true
	e.SymlinkTarget, _ = os.Readlink(full)
	target, err := os.Stat(full)
	if err != nil {
		return e, nil
	}
	e.IsDir = target.IsDir()
	e.Size = target.Size()
	return e, nil
}
