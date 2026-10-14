package mimeapps

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"slices"
	"sort"
	"strings"
)

type Error struct {
	Code    string
	Message string
}

func (e *Error) Error() string { return e.Message }

func fail(code, format string, args ...any) error {
	return &Error{Code: code, Message: fmt.Sprintf(format, args...)}
}

func CodeOf(err error) string {
	if coded, ok := err.(*Error); ok {
		return coded.Code
	}
	return "EIO"
}

type Service struct {
	detector *Detector
	entries  *entryIndex
}

func NewService() *Service { return NewServiceForDirs(MimeDirs()) }

func NewServiceForDirs(mimeDirs []string) *Service {
	return &Service{detector: NewDetector(LoadDB(mimeDirs)), entries: newEntryIndex()}
}

func (s *Service) Detector() *Detector { return s.detector }

func (s *Service) EntryByID(id string) *Entry { return s.entries.byIdentifier(id) }

// Apps lists every launchable application a picker should offer, by name.
func (s *Service) Apps() []*Entry {
	var out []*Entry
	for _, entry := range s.entries.all() {
		if entry.NoDisplay || !entry.Launchable() {
			continue
		}
		out = append(out, entry)
	}
	sort.SliceStable(out, func(i, j int) bool {
		return strings.ToLower(out[i].Name) < strings.ToLower(out[j].Name)
	})
	return out
}

// MimeOf resolves a path's type without a listing to hand it to us.
func (s *Service) MimeOf(path string) (string, error) {
	info, err := os.Stat(path)
	if err != nil {
		return "", fail(codeForOS(err), "%s", err)
	}
	return s.detector.Detect(path, info.IsDir()), nil
}

func (s *Service) Open(ctx context.Context, paths []string, opts LaunchOptions) error {
	if len(paths) == 0 {
		return fail("EINVAL", "no paths given")
	}
	if Sandboxed() {
		return openViaPortal(ctx, paths, false)
	}

	groups, unhandled, err := s.groupByApp(paths)
	if err != nil {
		return err
	}

	for _, group := range groups {
		batch := opts
		batch.Paths = group.paths
		if err := s.Launch(s.EntryByID(group.app), batch); err != nil {
			return fail("EIO", "%s", err)
		}
	}
	if len(unhandled) > 0 {
		return fail("NOTSUPPORTED", "no application is registered for %s", strings.Join(unhandled, ", "))
	}
	return nil
}

// OpenWith starts one named application on the paths. Inside the sandbox the
// portal owns the choice, so this falls back to its chooser.
func (s *Service) OpenWith(ctx context.Context, desktopID string, paths []string, opts LaunchOptions) error {
	if len(paths) == 0 {
		return fail("EINVAL", "no paths given")
	}
	if Sandboxed() {
		return openViaPortal(ctx, paths, true)
	}

	entry := s.EntryByID(desktopID)
	if !entry.Launchable() {
		return fail("ENOENT", "desktop entry %q is not installed", desktopID)
	}
	opts.Paths = paths
	if err := s.Launch(entry, opts); err != nil {
		return fail("EIO", "%s", err)
	}
	return nil
}

// Run executes a file directly. It is refused without the execute bit, which
// is also what marks a desktop file trusted.
func (s *Service) Run(path string, opts LaunchOptions) error {
	info, err := os.Stat(path)
	if err != nil {
		return fail(codeForOS(err), "%s", err)
	}
	if info.IsDir() {
		return fail("EINVAL", "%s is a directory", path)
	}
	if info.Mode()&0o111 == 0 {
		return fail("EACCES", "%s is not marked executable", path)
	}

	if s.detector.Detect(path, false) == TypeDesktop {
		entry := parseDesktopFile(path, baseName(path))
		if entry == nil {
			return fail("EINVAL", "%s is not a valid desktop entry", path)
		}
		opts.Paths = nil
		return s.Launch(entry, opts)
	}

	if Sandboxed() {
		return hostSpawn([]string{path}, opts.WorkingDir)
	}
	return spawn([]string{path}, opts.WorkingDir, opts.ActivationToken, false)
}

func (s *Service) OpenURI(ctx context.Context, uri string) error {
	scheme, _, ok := strings.Cut(uri, ":")
	if !ok || scheme == "" {
		return fail("EINVAL", "invalid uri %q", uri)
	}
	if scheme == "file" {
		path := fileURIPath(uri)
		if path == "" {
			return fail("EINVAL", "invalid file uri %q", uri)
		}
		return s.Open(ctx, []string{path}, LaunchOptions{})
	}
	if Sandboxed() {
		return openURIViaPortal(ctx, uri)
	}

	id := s.Default("x-scheme-handler/" + strings.ToLower(scheme))
	if id == "" {
		return fail("NOTSUPPORTED", "no handler registered for %q", scheme)
	}
	if err := s.Launch(s.EntryByID(id), LaunchOptions{Paths: []string{uri}}); err != nil {
		return fail("EIO", "%s", err)
	}
	return nil
}

type appGroup struct {
	app   string
	paths []string
}

func (s *Service) groupByApp(paths []string) ([]appGroup, []string, error) {
	var groups []appGroup
	var unhandled []string
	at := map[string]int{}
	defaults := map[string]string{}

	for _, path := range paths {
		mimeType, err := s.MimeOf(path)
		if err != nil {
			return nil, nil, err
		}
		app, known := defaults[mimeType]
		if !known {
			app = s.Default(mimeType)
			defaults[mimeType] = app
		}
		if app == "" {
			if !slices.Contains(unhandled, mimeType) {
				unhandled = append(unhandled, mimeType)
			}
			continue
		}
		index, ok := at[app]
		if !ok {
			index = len(groups)
			at[app] = index
			groups = append(groups, appGroup{app: app})
		}
		groups[index].paths = append(groups[index].paths, path)
	}
	return groups, unhandled, nil
}

func fileURIPath(uri string) string {
	parsed, err := url.Parse(uri)
	if err != nil || parsed.Scheme != "file" {
		return ""
	}
	return parsed.Path
}

func codeForOS(err error) string {
	switch {
	case os.IsNotExist(err):
		return "ENOENT"
	case os.IsPermission(err):
		return "EACCES"
	default:
		return "EIO"
	}
}
