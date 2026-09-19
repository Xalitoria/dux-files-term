package files

import (
	"os"
	"path/filepath"
	"strings"
)

func hiddenNames(dir string) map[string]bool {
	data, err := os.ReadFile(filepath.Join(dir, ".hidden"))
	if err != nil {
		return nil
	}
	names := map[string]bool{}
	for _, line := range strings.Split(string(data), "\n") {
		name := strings.TrimSpace(line)
		if name == "" {
			continue
		}
		names[name] = true
	}
	return names
}

func filterHidden(entries []Entry, includeHidden bool) []Entry {
	if includeHidden {
		return entries
	}
	out := make([]Entry, 0, len(entries))
	for _, e := range entries {
		if e.Hidden {
			continue
		}
		out = append(out, e)
	}
	return out
}
