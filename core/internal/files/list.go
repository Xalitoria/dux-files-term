package files

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

const DefaultPageSize = 500

type ListOptions struct {
	IncludeHidden bool
	Sort          SortSpec
	Limit         int
	Cursor        string
}

type Page struct {
	Entries []Entry
	Total   int
	Cursor  string
}

type builder struct {
	mime     MimeResolver
	owners   *ownerCache
	icons    *iconIndex
	keys     *nameKeys
	supports func(mimeType string) bool
}

func newBuilder(resolver MimeResolver, supports func(string) bool) *builder {
	if resolver == nil {
		resolver = extensionResolver{}
	}
	if supports == nil {
		supports = func(string) bool { return false }
	}
	return &builder{mime: resolver, owners: newOwnerCache(), icons: &iconIndex{}, keys: newNameKeys(), supports: supports}
}

func (b *builder) readDir(dir string) ([]Entry, error) {
	if !filepath.IsAbs(dir) {
		return nil, &Error{Code: CodeInvalid, Path: dir, Err: errors.New("path must be absolute")}
	}
	names, err := os.ReadDir(dir)
	if err != nil {
		return nil, Wrap(dir, err)
	}

	hidden := hiddenNames(dir)
	entries := make([]Entry, 0, len(names))
	for _, de := range names {
		e, err := b.entry(dir, de.Name(), hidden[de.Name()])
		switch {
		case errors.Is(err, fs.ErrNotExist):
			continue
		case err != nil:
			entries = append(entries, b.unreadable(dir, de.Name(), hidden[de.Name()]))
			continue
		}
		entries = append(entries, e)
	}
	return entries, nil
}

func List(dir string, opts ListOptions) ([]Entry, error) {
	entries, err := newBuilder(nil, nil).readDir(dir)
	if err != nil {
		return nil, err
	}
	entries = filterHidden(entries, opts.IncludeHidden)
	sortEntries(entries, opts.Sort)
	return entries, nil
}

func pageOf(entries []Entry, cursor string, limit int) Page {
	if limit <= 0 {
		limit = DefaultPageSize
	}
	start := resumeAt(entries, cursor)
	end := min(start+limit, len(entries))
	if start >= len(entries) {
		return Page{Entries: []Entry{}, Total: len(entries)}
	}

	page := Page{Entries: entries[start:end], Total: len(entries)}
	if end < len(entries) {
		page.Cursor = strconv.Itoa(end-1) + ":" + entries[end-1].Name
	}
	return page
}

func resumeAt(entries []Entry, cursor string) int {
	if cursor == "" {
		return 0
	}
	rawIndex, name, ok := strings.Cut(cursor, ":")
	if !ok {
		return 0
	}
	index, err := strconv.Atoi(rawIndex)
	if err != nil {
		return 0
	}
	if index >= 0 && index < len(entries) && entries[index].Name == name {
		return index + 1
	}
	for i, e := range entries {
		if e.Name == name {
			return i + 1
		}
	}
	return min(index, len(entries))
}
