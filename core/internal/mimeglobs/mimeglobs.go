package mimeglobs

import (
	"bufio"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type DB struct {
	globs    map[string][]string
	aliases  map[string]string
	children map[string][]string
}

func Load(dataHome string, dataDirs []string) *DB {
	db := &DB{
		globs:    map[string][]string{},
		aliases:  map[string]string{},
		children: map[string][]string{},
	}
	for _, dir := range append([]string{dataHome}, dataDirs...) {
		if dir == "" {
			continue
		}
		base := filepath.Join(dir, "mime")
		eachLine(filepath.Join(base, "globs2"), db.addGlob)
		eachLine(filepath.Join(base, "aliases"), db.addAlias)
		eachLine(filepath.Join(base, "subclasses"), db.addSubclass)
	}
	return db
}

func (db *DB) Globs(mime string) []string {
	mime = strings.ToLower(strings.TrimSpace(mime))
	if canonical, ok := db.aliases[mime]; ok {
		mime = canonical
	}
	set := map[string]struct{}{}
	visited := map[string]bool{}
	switch {
	case strings.HasSuffix(mime, "/*"):
		for _, t := range db.typesWithPrefix(strings.TrimSuffix(mime, "*")) {
			db.collect(t, set, visited)
		}
	default:
		db.collect(mime, set, visited)
	}
	if len(set) == 0 {
		return nil
	}
	out := make([]string, 0, len(set))
	for pattern := range set {
		out = append(out, pattern)
	}
	sort.Strings(out)
	return out
}

func (db *DB) collect(mime string, set map[string]struct{}, visited map[string]bool) {
	if visited[mime] {
		return
	}
	visited[mime] = true
	for _, pattern := range db.globs[mime] {
		set[pattern] = struct{}{}
	}
	for _, child := range db.children[mime] {
		db.collect(child, set, visited)
	}
}

func (db *DB) typesWithPrefix(prefix string) []string {
	set := map[string]struct{}{}
	for mime := range db.globs {
		if strings.HasPrefix(mime, prefix) {
			set[mime] = struct{}{}
		}
	}
	for alias, canonical := range db.aliases {
		if strings.HasPrefix(alias, prefix) {
			set[canonical] = struct{}{}
		}
	}
	for parent, children := range db.children {
		if strings.HasPrefix(parent, prefix) {
			set[parent] = struct{}{}
		}
		for _, child := range children {
			if strings.HasPrefix(child, prefix) {
				set[child] = struct{}{}
			}
		}
	}
	out := make([]string, 0, len(set))
	for mime := range set {
		out = append(out, mime)
	}
	sort.Strings(out)
	return out
}

func (db *DB) addGlob(line string) {
	parts := strings.Split(line, ":")
	if len(parts) < 3 {
		return
	}
	mime := strings.ToLower(parts[1])
	pattern := strings.ToLower(parts[2])
	if !strings.Contains(mime, "/") || pattern == "" {
		return
	}
	db.globs[mime] = append(db.globs[mime], pattern)
}

func (db *DB) addAlias(line string) {
	fields := strings.Fields(line)
	if len(fields) != 2 {
		return
	}
	db.aliases[strings.ToLower(fields[0])] = strings.ToLower(fields[1])
}

func (db *DB) addSubclass(line string) {
	fields := strings.Fields(line)
	if len(fields) != 2 {
		return
	}
	child, parent := strings.ToLower(fields[0]), strings.ToLower(fields[1])
	db.children[parent] = append(db.children[parent], child)
}

func eachLine(path string, fn func(line string)) {
	f, err := os.Open(path)
	if err != nil {
		return
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		fn(line)
	}
}
