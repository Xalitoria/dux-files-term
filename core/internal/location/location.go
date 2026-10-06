package location

import (
	"errors"
	"fmt"
	"io/fs"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
)

type Kind string

const (
	KindFolder  Kind = "folder"
	KindArchive Kind = "archive"
	KindRemote  Kind = "remote"
	KindServers Kind = "servers"
)

type Location struct {
	Kind   Kind     `json:"kind"`
	Path   string   `json:"path,omitempty"`
	URI    string   `json:"uri,omitempty"`
	Select []string `json:"select"`
}

type Env struct {
	TrashFiles string
	Stat       func(path string) (fs.FileInfo, error)
	Lstat      func(path string) (fs.FileInfo, error)
	IsArchive  func(path string) bool
}

var remoteSchemes = []string{"sftp", "ssh", "smb", "ftp", "ftps", "dav", "davs", "nfs", "afp"}

var uriPrefix = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9+.-]*:/`)

func Absolute(raw, cwd string) string {
	switch {
	case uriPrefix.MatchString(raw):
		return raw
	case filepath.IsAbs(raw):
		return filepath.Clean(raw)
	}
	return filepath.Join(cwd, raw)
}

func Resolve(open, sel []string, env Env) ([]Location, error) {
	var out []Location
	folderAt := map[string]int{}

	addFolder := func(path string) int {
		if index, ok := folderAt[path]; ok {
			return index
		}
		out = append(out, Location{Kind: KindFolder, Path: path, Select: []string{}})
		folderAt[path] = len(out) - 1
		return len(out) - 1
	}
	addSelection := func(path string) {
		if path == "/" {
			addFolder(path)
			return
		}
		index := addFolder(filepath.Dir(path))
		if !slices.Contains(out[index].Select, path) {
			out[index].Select = append(out[index].Select, path)
		}
	}

	for _, raw := range open {
		loc, err := Parse(raw, env)
		if err != nil {
			return nil, err
		}
		if loc.Kind != KindFolder {
			out = append(out, loc)
			continue
		}
		if info, err := env.Stat(loc.Path); err == nil && info.IsDir() {
			addFolder(loc.Path)
			continue
		}
		if _, err := env.Lstat(loc.Path); err != nil {
			return nil, describe(loc.Path, err)
		}
		if env.IsArchive != nil && env.IsArchive(loc.Path) {
			out = append(out, Location{Kind: KindArchive, Path: loc.Path, Select: []string{}})
			continue
		}
		addSelection(loc.Path)
	}

	for _, raw := range sel {
		loc, err := Parse(raw, env)
		if err != nil {
			return nil, err
		}
		if loc.Kind != KindFolder {
			return nil, fmt.Errorf("%s cannot be selected, only opened", raw)
		}
		if _, err := env.Lstat(loc.Path); err != nil {
			return nil, describe(loc.Path, err)
		}
		addSelection(loc.Path)
	}
	return out, nil
}

func LocalPaths(targets []string, env Env) ([]string, error) {
	out := make([]string, 0, len(targets))
	for _, raw := range targets {
		loc, err := Parse(raw, env)
		if err != nil {
			return nil, err
		}
		if loc.Kind != KindFolder {
			return nil, fmt.Errorf("%s is not a local path", raw)
		}
		if _, err := env.Lstat(loc.Path); err != nil {
			return nil, describe(loc.Path, err)
		}
		out = append(out, loc.Path)
	}
	return out, nil
}

func Parse(raw string, env Env) (Location, error) {
	switch {
	case raw == "":
		return Location{}, errors.New("empty location")
	case strings.HasPrefix(raw, "/"):
		return folder(filepath.Clean(raw)), nil
	}

	u, err := url.Parse(raw)
	if err != nil || u.Scheme == "" {
		return Location{}, fmt.Errorf("%s is not an absolute path or a supported URI", raw)
	}

	scheme := strings.ToLower(u.Scheme)
	switch scheme {
	case "file":
		if u.Host != "" && u.Host != "localhost" {
			return Location{}, fmt.Errorf("%s is on another host", raw)
		}
		return absolute(raw, u.Path)
	case "admin":
		return absolute(raw, u.Path)
	case "trash":
		if env.TrashFiles == "" {
			return Location{}, errors.New("the trash location is unknown")
		}
		return folder(filepath.Join(env.TrashFiles, filepath.Clean("/"+u.Path))), nil
	case "computer":
		return folder("/"), nil
	case "network":
		return Location{Kind: KindServers, Select: []string{}}, nil
	}

	if !slices.Contains(remoteSchemes, scheme) {
		return Location{}, fmt.Errorf("%s:// locations are not supported", scheme)
	}
	if u.Host == "" {
		return Location{}, fmt.Errorf("%s needs a host, for example %s://host/path", raw, scheme)
	}
	return Location{Kind: KindRemote, URI: raw, Select: []string{}}, nil
}

func absolute(raw, path string) (Location, error) {
	if !filepath.IsAbs(path) {
		return Location{}, fmt.Errorf("%s does not name an absolute path", raw)
	}
	return folder(filepath.Clean(path)), nil
}

func folder(path string) Location {
	return Location{Kind: KindFolder, Path: path, Select: []string{}}
}

func describe(path string, err error) error {
	switch {
	case errors.Is(err, fs.ErrNotExist):
		return fmt.Errorf("%s does not exist", path)
	case errors.Is(err, fs.ErrPermission):
		return fmt.Errorf("%s is not accessible", path)
	}
	return fmt.Errorf("%s: %w", path, err)
}

func HostEnv(dataHome string, isArchive func(string) bool) Env {
	return Env{TrashFiles: filepath.Join(dataHome, "Trash", "files"), Stat: os.Stat, Lstat: os.Lstat, IsArchive: isArchive}
}
