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
	{Name: "subscribe", Group: "core", Desc: "Subscribe to event topics", Params: []ParamSpec{opt("topics", "ui,colorScheme,jobs,devices,network,sizes,bookmarks,search:<id>")}},
	{Name: "unsubscribe", Group: "core", Desc: "Unsubscribe from event topics", Params: []ParamSpec{opt("topics", "topics to drop")}},

	{Name: "ui.show", Group: "ui", Desc: "Show the file manager window"},
	{Name: "ui.hide", Group: "ui", Desc: "Hide the file manager window"},
	{Name: "ui.toggle", Group: "ui", Desc: "Toggle the file manager window"},
	{Name: "ui.open", Group: "ui", Desc: "Open folders, each in its own tab; a file opens its folder with it selected", Params: []ParamSpec{opt("path", "absolute path or URI"), opt("paths", "absolute paths or URIs, one tab each"), opt("select", "absolute paths or URIs to select, grouped by folder")}},
	{Name: "ui.properties", Group: "ui", Desc: "Show the properties dialog for paths", Params: []ParamSpec{req("paths", "absolute paths or file:// URIs")}},
	{Name: "ui.quit", Group: "ui", Desc: "Quit the running daemon"},

	{Name: "files.list", Group: "files", Desc: "List a directory, or page a live watch when watchId is given", Params: []ParamSpec{req("path", "absolute directory path"), opt("watchId", "page from this watch instead of re-reading"), opt("includeHidden", "include dot-prefixed and .hidden entries"), opt("sort", "name|size|mtime|type"), opt("desc", "reverse the sort"), opt("dirsFirst", "directories before files (default true)"), opt("limit", "page size (default 500)"), opt("cursor", "cursor from the previous page"), opt("filters", "file name globs, case-insensitive; directories always pass")}},
	{Name: "files.watch", Group: "files", Desc: "List a directory and stream changes on files:<watchId>", Params: []ParamSpec{req("path", "absolute directory path"), opt("includeHidden", "include dot-prefixed and .hidden entries"), opt("sort", "name|size|mtime|type"), opt("desc", "reverse the sort"), opt("dirsFirst", "directories before files (default true)"), opt("limit", "page size (default 500)"), opt("filters", "file name globs, case-insensitive; directories always pass")}},
	{Name: "files.unwatch", Group: "files", Desc: "Close a watch and its event topic", Params: []ParamSpec{req("watchId", "id from files.watch")}},
	{Name: "files.stat", Group: "files", Desc: "Metadata for a single path", Params: []ParamSpec{req("path", "absolute path")}},
	{Name: "files.count", Group: "files", Desc: "Child counts for directories, capped at 10000", Params: []ParamSpec{req("paths", "absolute directory paths"), opt("includeHidden", "count dot-prefixed entries")}},
	{Name: "files.thumbnail", Group: "files", Desc: "Thumbnail cache paths, generating what is missing", Params: []ParamSpec{req("paths", "absolute file paths"), opt("size", "normal|large|x-large|xx-large"), opt("watchId", "deliver late results on this watch topic")}},
	{Name: "files.userDirs", Group: "files", Desc: "XDG user directories, home first"},
	{Name: "files.icon", Group: "files", Desc: "Icon name ladders for paths or mime types", Params: []ParamSpec{opt("paths", "absolute paths"), opt("mimes", "mime types")}},
	{Name: "files.size", Group: "files", Desc: "Scan paths recursively, streaming totals on the sizes topic", Params: []ParamSpec{req("paths", "absolute paths")}},
	{Name: "files.sizeCancel", Group: "files", Desc: "Stop a running recursive scan", Params: []ParamSpec{req("scanId", "id from files.size")}},
	{Name: "files.mkdir", Group: "files", Desc: "Create a directory whose parent exists, without undo", Params: []ParamSpec{req("path", "absolute path of the new directory")}},
	{Name: "files.rename", Group: "files", Desc: "Rename a path in place, refusing an existing target, without undo", Params: []ParamSpec{req("path", "absolute path"), req("name", "new name")}},
	{Name: "files.trash", Group: "files", Desc: "Move paths to the trash, reporting each failure, without undo", Params: []ParamSpec{req("paths", "absolute paths")}},

	{Name: "files.op.copy", Group: "files.op", Desc: "Copy paths into a directory", Params: []ParamSpec{req("paths", "absolute source paths"), req("dest", "absolute target directory"), opt("scope", "undo stack key, one per window")}},
	{Name: "files.op.move", Group: "files.op", Desc: "Move paths into a directory", Params: []ParamSpec{req("paths", "absolute source paths"), req("dest", "absolute target directory"), opt("scope", "undo stack key, one per window")}},
	{Name: "files.op.rename", Group: "files.op", Desc: "Rename a single path", Params: []ParamSpec{req("path", "absolute path"), req("name", "new name"), opt("scope", "undo stack key, one per window")}},
	{Name: "files.op.newFolder", Group: "files.op", Desc: "Create a directory", Params: []ParamSpec{req("path", "absolute parent directory"), req("name", "new directory name"), opt("scope", "undo stack key, one per window")}},
	{Name: "files.op.newFile", Group: "files.op", Desc: "Create a file, optionally from a template", Params: []ParamSpec{req("path", "absolute parent directory"), req("name", "new file name"), opt("template", "absolute template path"), opt("scope", "undo stack key, one per window")}},
	{Name: "files.op.link", Group: "files.op", Desc: "Create symlinks to paths in a directory", Params: []ParamSpec{req("paths", "absolute source paths"), req("dest", "absolute target directory"), opt("scope", "undo stack key, one per window")}},
	{Name: "files.op.trash", Group: "files.op", Desc: "Move paths to the trash", Params: []ParamSpec{req("paths", "absolute paths"), opt("scope", "undo stack key, one per window")}},
	{Name: "files.op.restore", Group: "files.op", Desc: "Restore trashed items to their original location", Params: []ParamSpec{req("names", "names inside the trash files dir"), req("trashDir", "absolute trash directory"), opt("scope", "undo stack key, one per window")}},
	{Name: "files.op.delete", Group: "files.op", Desc: "Delete paths permanently", Params: []ParamSpec{req("paths", "absolute paths")}},
	{Name: "files.op.emptyTrash", Group: "files.op", Desc: "Delete every trashed item permanently"},
	{Name: "files.op.setPermissions", Group: "files.op", Desc: "Change mode bits", Params: []ParamSpec{req("paths", "absolute paths"), req("mode", "octal mode, for example 644"), opt("recursive", "apply to the whole tree")}},
	{Name: "files.op.setOwner", Group: "files.op", Desc: "Change owner and/or group", Params: []ParamSpec{req("paths", "absolute paths"), opt("owner", "user name or uid"), opt("group", "group name or gid"), opt("recursive", "apply to the whole tree")}},
	{Name: "files.op.respond", Group: "files.op", Desc: "Answer a paused job", Params: []ParamSpec{req("jobId", "job id"), req("id", "conflict or error id"), req("action", "skip|replace|keepBoth|merge|retry|retryElevated|cancel"), opt("applyToAll", "apply to the rest of the job")}},
	{Name: "files.op.cancel", Group: "files.op", Desc: "Cancel a running or queued job", Params: []ParamSpec{req("jobId", "job id")}},
	{Name: "files.op.jobs", Group: "files.op", Desc: "List running, queued and recently finished jobs"},
	{Name: "files.op.dropAction", Group: "files.op", Desc: "What a plain drag of paths onto dest would do", Params: []ParamSpec{req("paths", "absolute source paths"), req("dest", "absolute target directory")}},
	{Name: "files.op.undo", Group: "files.op", Desc: "Undo the last undoable operation in a scope", Params: []ParamSpec{opt("scope", "undo stack key, one per window")}},
	{Name: "files.op.compress", Group: "files.op", Desc: "Pack paths into a new archive in a directory", Params: []ParamSpec{req("paths", "absolute source paths"), req("dest", "absolute target directory"), req("name", "archive name without extension"), opt("format", "zip|tar.gz (default zip)"), opt("scope", "undo stack key, one per window")}},
	{Name: "files.op.extract", Group: "files.op", Desc: "Unpack zip, tar and tar.gz archives into folders named after them", Params: []ParamSpec{req("paths", "absolute archive paths"), req("dest", "absolute target directory"), opt("scope", "undo stack key, one per window")}},

	{Name: "files.open", Group: "files.apps", Desc: "Open paths with their default application", Params: []ParamSpec{req("paths", "absolute paths"), opt("cwd", "working directory for the launched app"), opt("activationToken", "xdg activation token from the compositor"), opt("terminal", "terminal to use for Terminal=true entries")}},
	{Name: "files.openWith", Group: "files.apps", Desc: "Open paths with a named application", Params: []ParamSpec{req("paths", "absolute paths"), req("desktopId", "desktop entry id, for example org.gnome.gedit.desktop"), opt("cwd", "working directory for the launched app"), opt("activationToken", "xdg activation token from the compositor")}},
	{Name: "files.run", Group: "files.apps", Desc: "Run an executable file or a trusted desktop entry", Params: []ParamSpec{req("path", "absolute path"), opt("cwd", "working directory"), opt("activationToken", "xdg activation token from the compositor")}},
	{Name: "files.openTerminal", Group: "files.apps", Desc: "Open a terminal in a directory", Params: []ParamSpec{req("path", "absolute directory path"), opt("terminal", "terminal command, defaults to $TERMINAL then autodetection")}},
	{Name: "files.apps.forMime", Group: "files.apps", Desc: "Applications that handle a mime type, most preferred first", Params: []ParamSpec{opt("mime", "mime type"), opt("path", "absolute path to type instead")}},
	{Name: "files.apps.all", Group: "files.apps", Desc: "Every launchable application, for the Other application picker"},
	{Name: "files.apps.setDefault", Group: "files.apps", Desc: "Set the default application for a mime type", Params: []ParamSpec{req("mime", "mime type"), req("desktopId", "desktop entry id")}},
	{Name: "files.apps.terminals", Group: "files.apps", Desc: "Terminal emulators installed on this system"},

	{Name: "search.start", Group: "search", Desc: "Search a tree by name, streaming results on search:<searchId>", Params: []ParamSpec{req("root", "absolute directory to search"), opt("query", "case-insensitive name substring; optional when a filter is set"), opt("includeHidden", "descend into and report dot-prefixed entries"), opt("crossDevice", "follow into other filesystems"), opt("types", "documents|images|video|audio|archives|folders"), opt("modifiedAfterMs", "only entries modified after this unix time"), opt("minSize", "bytes"), opt("maxSize", "bytes"), opt("limit", "result cap, at most 10000")}},
	{Name: "search.cancel", Group: "search", Desc: "Stop a running search", Params: []ParamSpec{req("searchId", "id from search.start")}},

	{Name: "bookmarks.list", Group: "bookmarks", Desc: "Saved folders in sidebar order, plus whether a GTK import is on offer"},
	{Name: "bookmarks.add", Group: "bookmarks", Desc: "Save a folder", Params: []ParamSpec{req("path", "absolute directory path"), opt("label", "display name, defaults to the folder name"), opt("icon", "icon name")}},
	{Name: "bookmarks.remove", Group: "bookmarks", Desc: "Forget a saved folder", Params: []ParamSpec{req("path", "absolute directory path")}},
	{Name: "bookmarks.rename", Group: "bookmarks", Desc: "Relabel a saved folder", Params: []ParamSpec{req("path", "absolute directory path"), req("label", "display name")}},
	{Name: "bookmarks.reorder", Group: "bookmarks", Desc: "Set the sidebar order; every bookmark must be listed once", Params: []ParamSpec{req("paths", "every bookmark path in the new order")}},
	{Name: "bookmarks.importGtk", Group: "bookmarks", Desc: "Import the GTK file chooser bookmarks that are not saved yet"},

	{Name: "devices.list", Group: "devices", Desc: "Local drives and volumes udisks2 knows about"},
	{Name: "devices.usage", Group: "devices", Desc: "Disk usage for mounted devices, cached for 30 seconds", Params: []ParamSpec{opt("paths", "mount points, default every mounted device"), opt("force", "ignore the cache")}},
	{Name: "devices.mount", Group: "devices", Desc: "Mount a device", Params: []ParamSpec{req("id", "device id from devices.list")}},
	{Name: "devices.unmount", Group: "devices", Desc: "Unmount a device", Params: []ParamSpec{req("id", "device id from devices.list"), opt("force", "unmount even while busy")}},
	{Name: "devices.eject", Group: "devices", Desc: "Unmount and eject removable media", Params: []ParamSpec{req("id", "device id from devices.list")}},
	{Name: "devices.powerOff", Group: "devices", Desc: "Power off a drive", Params: []ParamSpec{req("id", "device id from devices.list")}},
	{Name: "devices.unlock", Group: "devices", Desc: "Unlock a LUKS device", Params: []ParamSpec{req("id", "device id from devices.list"), req("passphrase", "passphrase, never stored"), opt("mount", "mount the cleartext device afterwards")}},
	{Name: "devices.lock", Group: "devices", Desc: "Lock a LUKS device", Params: []ParamSpec{req("id", "device id from devices.list")}},

	{Name: "network.mounts", Group: "network", Desc: "gvfs mounts, connectable schemes and portable devices waiting to be mounted"},
	{Name: "network.connect", Group: "network", Desc: "Mount a network location, asking on the network topic when gvfs needs credentials", Params: []ParamSpec{req("uri", "for example sftp://host/path")}},
	{Name: "network.disconnect", Group: "network", Desc: "Unmount a gvfs mount", Params: []ParamSpec{req("id", "mount id, uri or path from network.mounts")}},
	{Name: "network.answer", Group: "network", Desc: "Answer a question gvfs asked during a connect", Params: []ParamSpec{req("id", "question id from the network topic"), opt("password", ""), opt("user", ""), opt("domain", ""), opt("anonymous", "connect anonymously"), opt("save", "0 never, 1 session, 2 permanently"), opt("choice", "index for a question"), opt("abort", "give up on the mount")}},
	{Name: "network.recent", Group: "network", Desc: "Servers connected to before"},
	{Name: "network.forget", Group: "network", Desc: "Drop a server from the recent list", Params: []ParamSpec{req("uri", "server uri")}},
	{Name: "network.mountPortable", Group: "network", Desc: "Mount a phone or camera gvfs has detected", Params: []ParamSpec{req("monitor", "volume monitor bus name"), req("id", "volume id")}},

	{Name: "trash.list", Group: "trash", Desc: "Trashed items across the home trash and every mounted volume's trash"},
	{Name: "trash.count", Group: "trash", Desc: "Number of trashed items"},

	{Name: "clipboard.copy", Group: "clipboard", Desc: "Put paths on the system clipboard as a file selection", Params: []ParamSpec{req("paths", "absolute paths"), opt("cut", "mark the selection for a move")}},
	{Name: "clipboard.paths", Group: "clipboard", Desc: "Read the file selection on the system clipboard"},
	{Name: "clipboard.clear", Group: "clipboard", Desc: "Release the system clipboard"},

	{Name: "system.capabilities", Group: "system", Desc: "What the environment permits: sandbox state, host access, bus policies, portal versions and the per-feature table with reason codes"},
	{Name: "system.colorScheme.get", Group: "system", Desc: "Current portal color scheme (0 no-preference, 1 dark, 2 light)"},
	{Name: "system.elevation", Group: "system", Desc: "Whether failed operations can be retried as administrator, and why not"},
	{Name: "system.fileManager.get", Group: "system", Desc: "Whether this instance owns org.freedesktop.FileManager1, and who does otherwise"},
	{Name: "system.fileManager.set", Group: "system", Desc: "Request or release org.freedesktop.FileManager1; another owner is never replaced", Params: []ParamSpec{req("enabled", "true to request the name, false to release it")}},
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
