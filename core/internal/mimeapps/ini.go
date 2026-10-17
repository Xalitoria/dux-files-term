package mimeapps

import (
	"bufio"
	"bytes"
	"os"
	"strings"
	"sync"
)

type iniGroup struct {
	keys map[string]string
}

func (g *iniGroup) value(key string) string {
	if g == nil {
		return ""
	}
	return unescapeValue(g.keys[key])
}

func (g *iniGroup) localized(key string) string {
	if g == nil {
		return ""
	}
	for _, tag := range localeCandidates() {
		if v, ok := g.keys[key+"["+tag+"]"]; ok {
			return unescapeValue(v)
		}
	}
	return g.value(key)
}

func parseGroups(data []byte) map[string]*iniGroup {
	groups := map[string]*iniGroup{}
	var current *iniGroup

	scanner := bufio.NewScanner(bytes.NewReader(data))
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		switch {
		case line == "" || strings.HasPrefix(line, "#"):
		case strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]"):
			name := line[1 : len(line)-1]
			if groups[name] == nil {
				groups[name] = &iniGroup{keys: map[string]string{}}
			}
			current = groups[name]
		case current != nil:
			key, value, ok := strings.Cut(line, "=")
			if !ok {
				continue
			}
			key = strings.TrimSpace(key)
			if _, exists := current.keys[key]; !exists {
				current.keys[key] = strings.TrimSpace(value)
			}
		}
	}
	return groups
}

func readGroups(path string) map[string]*iniGroup {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	return parseGroups(data)
}

func splitList(value string) []string {
	var out []string
	for _, item := range strings.Split(unescapeValue(value), ";") {
		if item = strings.TrimSpace(item); item != "" {
			out = append(out, item)
		}
	}
	return out
}

func unescapeValue(value string) string {
	if !strings.ContainsRune(value, '\\') {
		return value
	}
	var b strings.Builder
	for i := 0; i < len(value); i++ {
		if value[i] != '\\' || i+1 >= len(value) {
			b.WriteByte(value[i])
			continue
		}
		i++
		switch value[i] {
		case 's':
			b.WriteByte(' ')
		case 'n':
			b.WriteByte('\n')
		case 't':
			b.WriteByte('\t')
		case 'r':
			b.WriteByte('\r')
		case ';':
			b.WriteByte(';')
		case '\\':
			b.WriteByte('\\')
		default:
			b.WriteByte('\\')
			b.WriteByte(value[i])
		}
	}
	return b.String()
}

var localeCandidates = sync.OnceValue(sessionLocaleCandidates)

func sessionLocaleCandidates() []string {
	return expandLocale(sessionLocale())
}

func sessionLocale() string {
	for _, key := range []string{"LC_ALL", "LC_MESSAGES", "LANG"} {
		if value := os.Getenv(key); value != "" {
			return value
		}
	}
	return ""
}

func expandLocale(locale string) []string {
	locale = StripParams(locale)
	if locale == "" || locale == "C" || locale == "POSIX" {
		return nil
	}

	modifier := ""
	if base, mod, ok := strings.Cut(locale, "@"); ok {
		locale, modifier = base, mod
	}
	if base, _, ok := strings.Cut(locale, "."); ok {
		locale = base
	}
	lang, country, hasCountry := strings.Cut(locale, "_")

	var tags []string
	switch {
	case hasCountry && modifier != "":
		tags = append(tags, lang+"_"+country+"@"+modifier, lang+"_"+country, lang+"@"+modifier, lang)
	case hasCountry:
		tags = append(tags, lang+"_"+country, lang)
	case modifier != "":
		tags = append(tags, lang+"@"+modifier, lang)
	default:
		tags = append(tags, lang)
	}
	return tags
}
