// Package mimeapps resolves mime types, default applications and launching.
package mimeapps

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
)

type DB struct {
	globs        *globSet
	magic        []magicRule
	maxExtent    int
	aliases      map[string]string
	subclasses   map[string][]string
	genericIcons map[string]string
	icons        map[string]string
}

// MimeDirs is the shared-mime-info search path, most specific first.
func MimeDirs() []string {
	return appendXDG(nil, "mime")
}

func appendXDG(dirs []string, leaf string) []string {
	seen := map[string]bool{}
	add := func(dir string) {
		if dir == "" || seen[dir] {
			return
		}
		seen[dir] = true
		dirs = append(dirs, dir)
	}

	add(filepath.Join(xdgDataHome(), leaf))
	for _, dir := range xdgDataDirs() {
		add(filepath.Join(dir, leaf))
	}
	return dirs
}

func xdgDataHome() string {
	if dir := os.Getenv("XDG_DATA_HOME"); dir != "" {
		return dir
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".local", "share")
}

func xdgDataDirs() []string {
	raw := os.Getenv("XDG_DATA_DIRS")
	if raw == "" {
		raw = "/usr/local/share:/usr/share"
	}
	var out []string
	for _, dir := range strings.Split(raw, ":") {
		if dir = strings.TrimSpace(dir); dir != "" {
			out = append(out, dir)
		}
	}
	return out
}

func xdgConfigHome() string {
	if dir := os.Getenv("XDG_CONFIG_HOME"); dir != "" {
		return dir
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".config")
}

func xdgConfigDirs() []string {
	raw := os.Getenv("XDG_CONFIG_DIRS")
	if raw == "" {
		raw = "/etc/xdg"
	}
	var out []string
	for _, dir := range strings.Split(raw, ":") {
		if dir = strings.TrimSpace(dir); dir != "" {
			out = append(out, dir)
		}
	}
	return out
}

func LoadDB(dirs []string) *DB {
	db := &DB{
		globs:        newGlobSet(),
		aliases:      map[string]string{},
		subclasses:   map[string][]string{},
		genericIcons: map[string]string{},
		icons:        map[string]string{},
	}

	for _, dir := range dirs {
		db.globs.load(filepath.Join(dir, "globs2"))
		db.loadMagic(filepath.Join(dir, "magic"))
		readPairs(filepath.Join(dir, "aliases"), " ", func(alias, target string) {
			if _, ok := db.aliases[alias]; !ok {
				db.aliases[alias] = target
			}
		})
		readPairs(filepath.Join(dir, "subclasses"), " ", func(child, parent string) {
			db.subclasses[child] = append(db.subclasses[child], parent)
		})
		readPairs(filepath.Join(dir, "generic-icons"), ":", func(mimeType, icon string) {
			if _, ok := db.genericIcons[mimeType]; !ok {
				db.genericIcons[mimeType] = icon
			}
		})
		readPairs(filepath.Join(dir, "icons"), ":", func(mimeType, icon string) {
			if _, ok := db.icons[mimeType]; !ok {
				db.icons[mimeType] = icon
			}
		})
	}

	db.globs.sort()
	return db
}

func readPairs(path, sep string, fn func(key, value string)) {
	f, err := os.Open(path)
	if err != nil {
		return
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, value, ok := strings.Cut(line, sep)
		if !ok {
			continue
		}
		fn(strings.TrimSpace(key), strings.TrimSpace(value))
	}
}

// Canonical folds an alias onto the type it stands for.
func (db *DB) Canonical(mimeType string) string {
	mimeType = StripParams(mimeType)
	if target, ok := db.aliases[mimeType]; ok {
		return target
	}
	return mimeType
}

// Chain is the type itself followed by its ancestors, breadth first.
func (db *DB) Chain(mimeType string) []string {
	root := db.Canonical(mimeType)
	if root == "" {
		return nil
	}

	seen := map[string]bool{root: true}
	chain := []string{root}
	for i := 0; i < len(chain); i++ {
		for _, parent := range db.subclasses[chain[i]] {
			parent = db.Canonical(parent)
			if seen[parent] {
				continue
			}
			seen[parent] = true
			chain = append(chain, parent)
		}
	}
	return chain
}

func (db *DB) IsSubclassOf(mimeType, ancestor string) bool {
	ancestor = db.Canonical(ancestor)
	for _, m := range db.Chain(mimeType) {
		if m == ancestor {
			return true
		}
	}
	return false
}

// MimeIcon returns the database's specific icon name and generic icon name.
func (db *DB) MimeIcon(mimeType string) (specific, generic string) {
	canonical := db.Canonical(mimeType)
	return db.icons[canonical], db.genericIcons[canonical]
}

func StripParams(mimeType string) string {
	if cut := strings.IndexByte(mimeType, ';'); cut >= 0 {
		mimeType = mimeType[:cut]
	}
	return strings.TrimSpace(mimeType)
}
