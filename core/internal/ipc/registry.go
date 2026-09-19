package ipc

import "sort"

type ParamSpec struct {
	Name     string `json:"name"`
	Required bool   `json:"required"`
	Desc     string `json:"desc,omitempty"`
}

type MethodSpec struct {
	Name   string      `json:"name"`
	Group  string      `json:"group"`
	Desc   string      `json:"desc,omitempty"`
	Params []ParamSpec `json:"params,omitempty"`
}

func req(name, desc string) ParamSpec { return ParamSpec{Name: name, Required: true, Desc: desc} }
func opt(name, desc string) ParamSpec { return ParamSpec{Name: name, Desc: desc} }

// Methods drives shell completion, `ipc list`, and the `describe` method.
var Methods = []MethodSpec{
	{Name: "ping", Group: "core", Desc: "Liveness check"},
	{Name: "version", Group: "core", Desc: "Daemon and API version"},
	{Name: "describe", Group: "core", Desc: "List all IPC methods and their params"},
	{Name: "subscribe", Group: "core", Desc: "Subscribe to event topics", Params: []ParamSpec{opt("topics", "ui,colorScheme")}},
	{Name: "unsubscribe", Group: "core", Desc: "Unsubscribe from event topics", Params: []ParamSpec{opt("topics", "topics to drop")}},

	{Name: "ui.show", Group: "ui", Desc: "Show the file manager window"},
	{Name: "ui.hide", Group: "ui", Desc: "Hide the file manager window"},
	{Name: "ui.toggle", Group: "ui", Desc: "Toggle the file manager window"},
	{Name: "ui.open", Group: "ui", Desc: "Open a directory in the file manager", Params: []ParamSpec{req("path", "absolute directory path")}},
	{Name: "ui.quit", Group: "ui", Desc: "Quit the running daemon"},

	{Name: "files.list", Group: "files", Desc: "List a directory, or page a live watch when watchId is given", Params: []ParamSpec{req("path", "absolute directory path"), opt("watchId", "page from this watch instead of re-reading"), opt("includeHidden", "include dot-prefixed and .hidden entries"), opt("sort", "name|size|mtime|type"), opt("desc", "reverse the sort"), opt("dirsFirst", "directories before files (default true)"), opt("limit", "page size (default 500)"), opt("cursor", "cursor from the previous page")}},
	{Name: "files.watch", Group: "files", Desc: "List a directory and stream changes on files:<watchId>", Params: []ParamSpec{req("path", "absolute directory path"), opt("includeHidden", "include dot-prefixed and .hidden entries"), opt("sort", "name|size|mtime|type"), opt("desc", "reverse the sort"), opt("dirsFirst", "directories before files (default true)"), opt("limit", "page size (default 500)")}},
	{Name: "files.unwatch", Group: "files", Desc: "Close a watch and its event topic", Params: []ParamSpec{req("watchId", "id from files.watch")}},
	{Name: "files.stat", Group: "files", Desc: "Metadata for a single path", Params: []ParamSpec{req("path", "absolute path")}},
	{Name: "files.count", Group: "files", Desc: "Child counts for directories, capped at 10000", Params: []ParamSpec{req("paths", "absolute directory paths"), opt("includeHidden", "count dot-prefixed entries")}},
	{Name: "files.thumbnail", Group: "files", Desc: "Thumbnail cache paths, generating what is missing", Params: []ParamSpec{req("paths", "absolute file paths"), opt("size", "normal|large|x-large|xx-large"), opt("watchId", "deliver late results on this watch topic")}},
	{Name: "files.icon", Group: "files", Desc: "Icon name ladders for paths or mime types", Params: []ParamSpec{opt("paths", "absolute paths"), opt("mimes", "mime types")}},

	{Name: "system.colorScheme.get", Group: "system", Desc: "Current portal color scheme (0 no-preference, 1 dark, 2 light)"},
	{Name: "system.openUri", Group: "system", Desc: "Open a URI with the default handler", Params: []ParamSpec{req("uri", "")}},
}

func MethodNames() []string {
	names := make([]string, len(Methods))
	for i, m := range Methods {
		names[i] = m.Name
	}
	sort.Strings(names)
	return names
}

func FindMethod(name string) (MethodSpec, bool) {
	for _, m := range Methods {
		if m.Name == name {
			return m, true
		}
	}
	return MethodSpec{}, false
}
