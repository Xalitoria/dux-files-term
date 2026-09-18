package files

import (
	"cmp"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

type ListOptions struct {
	IncludeHidden bool
}

func List(dir string, opts ListOptions) ([]Entry, error) {
	if !filepath.IsAbs(dir) {
		return nil, &Error{Code: CodeInvalid, Path: dir, Err: errors.New("path must be absolute")}
	}
	dirEntries, err := os.ReadDir(dir)
	if err != nil {
		return nil, wrap(dir, err)
	}

	entries := make([]Entry, 0, len(dirEntries))
	for _, de := range dirEntries {
		if !opts.IncludeHidden && strings.HasPrefix(de.Name(), ".") {
			continue
		}
		e, err := entryFromDirEntry(dir, de)
		if errors.Is(err, fs.ErrNotExist) {
			continue
		}
		if err != nil {
			return nil, wrap(filepath.Join(dir, de.Name()), err)
		}
		entries = append(entries, e)
	}
	slices.SortFunc(entries, compareEntries)
	return entries, nil
}

func compareEntries(a, b Entry) int {
	if a.IsDir != b.IsDir {
		if a.IsDir {
			return -1
		}
		return 1
	}
	if c := cmp.Compare(strings.ToLower(a.Name), strings.ToLower(b.Name)); c != 0 {
		return c
	}
	return cmp.Compare(a.Name, b.Name)
}
