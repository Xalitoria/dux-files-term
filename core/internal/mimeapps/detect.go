package mimeapps

import (
	"os"
	"sync"
	"unicode/utf8"
)

const (
	TypeDirectory = "inode/directory"
	TypeSymlink   = "inode/symlink"
	TypeDesktop   = "application/x-desktop"
	TypeText      = "text/plain"
	TypeBinary    = "application/octet-stream"
	TypeExe       = "application/x-executable"
)

// magicWinsPriority is the shared-mime-info threshold above which a magic
// match overrides a glob match, as gio does it.
const magicWinsPriority = 80

const minSniffExtent = 256

type Detector struct {
	db   *DB
	bufs sync.Pool
}

func NewDetector(db *DB) *Detector { return &Detector{db: db} }

func (d *Detector) DB() *DB { return d.db }

// FromName matches the glob database and never touches the disk.
func (d *Detector) FromName(name string) string { return d.db.globs.match(name) }

// Sniff reads the head of a file and returns its magic type, falling back to
// text or binary.
func (d *Detector) Sniff(path string) string {
	buf := d.borrow()
	defer d.release(buf)

	head, ok := d.head(path, *buf)
	if !ok {
		return ""
	}
	if mimeType, _ := d.db.sniffData(head); mimeType != "" {
		return mimeType
	}
	if looksTextual(head) {
		return TypeText
	}
	return TypeBinary
}

// Detect applies the full spec order for one path: a glob match stands unless
// magic outranks it or there was no glob match at all.
func (d *Detector) Detect(path string, isDir bool) string {
	if isDir {
		return TypeDirectory
	}

	name := baseName(path)
	globbed := d.FromName(name)

	buf := d.borrow()
	defer d.release(buf)

	head, ok := d.head(path, *buf)
	if !ok {
		return globbed
	}

	sniffed, priority := d.db.sniffData(head)
	switch {
	case sniffed == "" && globbed != "":
		return globbed
	case sniffed == "" && looksTextual(head):
		return TypeText
	case sniffed == "":
		return TypeBinary
	case globbed == "" || priority >= magicWinsPriority || d.db.IsSubclassOf(sniffed, globbed):
		return sniffed
	}
	return globbed
}

func (d *Detector) MimeIcon(mimeType string) (specific, generic string) {
	return d.db.MimeIcon(mimeType)
}

func (d *Detector) head(path string, buf []byte) ([]byte, bool) {
	f, err := os.Open(path)
	if err != nil {
		return nil, false
	}
	defer f.Close()

	n, _ := f.Read(buf)
	if n == 0 {
		return nil, false
	}
	return buf[:n], true
}

func (d *Detector) borrow() *[]byte {
	if buf, ok := d.bufs.Get().(*[]byte); ok {
		return buf
	}
	buf := make([]byte, max(d.db.maxExtent, minSniffExtent))
	return &buf
}

func (d *Detector) release(buf *[]byte) { d.bufs.Put(buf) }

func baseName(path string) string {
	for i := len(path) - 1; i >= 0; i-- {
		if path[i] == '/' {
			return path[i+1:]
		}
	}
	return path
}

func looksTextual(head []byte) bool {
	if !utf8.Valid(head) && !utf8.Valid(trimPartialRune(head)) {
		return false
	}
	for _, b := range head {
		if b == 0 {
			return false
		}
		if b < 0x09 || (b > 0x0d && b < 0x20 && b != 0x1b) {
			return false
		}
	}
	return true
}

func trimPartialRune(head []byte) []byte {
	for i := len(head) - 1; i >= 0 && i > len(head)-5; i-- {
		if utf8.RuneStart(head[i]) {
			return head[:i]
		}
	}
	return head
}

// DesktopDisplay returns the name and icon a trusted desktop file should be
// listed under. An untrusted file keeps its raw name.
func (d *Detector) DesktopDisplay(path string, trusted bool) (name, icon string, ok bool) {
	if !trusted {
		return "", "", false
	}
	entry := parseDesktopFile(path, baseName(path))
	if entry == nil || entry.Name == "" {
		return "", "", false
	}
	return entry.Name, entry.Icon, true
}
