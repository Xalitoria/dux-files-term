package clipboard

import (
	"net/url"
	"strings"
)

// The flavours a file manager selection is published under. The GNOME payload
// is the only one that carries the cut/copy distinction for GTK file managers;
// KDE carries it in a separate one-byte flavour.
//
// Sources:
// nautilus_clipboard_to_string, gitlab.gnome.org/GNOME/nautilus src/nautilus-clipboard.c
// KIO::setClipboardDataCut, invent.kde.org/frameworks/kio src/widgets/paste.cpp
const (
	MimeGnomeCopiedFiles = "x-special/gnome-copied-files"
	MimeUriList          = "text/uri-list"
	MimeKdeCutSelection  = "application/x-kde-cutselection"
	MimeTextUTF8         = "text/plain;charset=utf-8"
)

// GnomePayload is "copy\n" or "cut\n" followed by one file URI per line, each
// terminated by a newline.
func GnomePayload(paths []string, cut bool) []byte {
	action := "copy"
	if cut {
		action = "cut"
	}
	out := strings.Builder{}
	out.WriteString(action)
	out.WriteByte('\n')
	for _, path := range paths {
		out.WriteString(FileURI(path))
		out.WriteByte('\n')
	}
	return []byte(out.String())
}

// UriList is RFC 2483 text/uri-list: CRLF-separated URIs.
func UriList(paths []string) []byte {
	out := strings.Builder{}
	for _, path := range paths {
		out.WriteString(FileURI(path))
		out.WriteString("\r\n")
	}
	return []byte(out.String())
}

// PlainText is what a text editor pastes: one absolute path per line.
func PlainText(paths []string) []byte {
	return []byte(strings.Join(paths, "\n"))
}

func FileURI(path string) string {
	return (&url.URL{Scheme: "file", Path: path}).String()
}

// Parse reads a selection published by any file manager back into absolute
// paths plus whether it was cut. Only the GNOME flavour carries the cut flag,
// so every other flavour reads as a copy.
func Parse(data []byte, mime string) ([]string, bool) {
	switch {
	case strings.HasPrefix(mime, MimeGnomeCopiedFiles):
		return parseGnome(data)
	case strings.HasPrefix(mime, MimeUriList):
		return parseLines(data), false
	default:
		return parseLines(data), false
	}
}

func parseGnome(data []byte) ([]string, bool) {
	lines := strings.Split(strings.ReplaceAll(string(data), "\r\n", "\n"), "\n")
	if len(lines) == 0 {
		return nil, false
	}
	action := strings.TrimSpace(lines[0])
	if action != "copy" && action != "cut" {
		return parseLines(data), false
	}
	return collect(lines[1:]), action == "cut"
}

func parseLines(data []byte) []string {
	return collect(strings.Split(strings.ReplaceAll(string(data), "\r\n", "\n"), "\n"))
}

func collect(lines []string) []string {
	paths := make([]string, 0, len(lines))
	for _, line := range lines {
		path := toPath(strings.TrimSpace(line))
		if path == "" {
			continue
		}
		paths = append(paths, path)
	}
	if len(paths) == 0 {
		return nil
	}
	return paths
}

// toPath accepts a local file URI or a bare absolute path and rejects
// everything else, so a remote URI or a comment never becomes an operand.
func toPath(line string) string {
	switch {
	case line == "" || strings.HasPrefix(line, "#"):
		return ""
	case strings.HasPrefix(line, "/"):
		return line
	case !strings.HasPrefix(line, "file:"):
		return ""
	}
	parsed, err := url.Parse(line)
	if err != nil || (parsed.Host != "" && parsed.Host != "localhost") || !strings.HasPrefix(parsed.Path, "/") {
		return ""
	}
	return parsed.Path
}
