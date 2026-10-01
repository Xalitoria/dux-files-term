package fileops

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"context"
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

const (
	FormatZip   = "zip"
	FormatTarGz = "tar.gz"
)

var errUnsupportedArchive = errors.New("unsupported archive format")

type archiveWriter interface {
	dir(name string, info fs.FileInfo) error
	symlink(name, target string, info fs.FileInfo) error
	file(name string, info fs.FileInfo) (io.Writer, error)
	Close() error
}

func (r *runner) compress() error {
	sources := r.job.spec.sources
	count, bytes, complete := r.scan(sources)
	r.setTotals(count, bytes, !complete)

	target := filepath.Join(r.job.spec.dest, r.job.spec.name+"."+r.job.spec.format)
	if _, err := os.Lstat(target); err == nil {
		target = uniquePath(target)
	}
	tmp := r.tempPath(target)

	var out *os.File
	if err := r.retryLocal(tmp, func() error {
		created, err := os.OpenFile(tmp, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
		out = created
		return err
	}); err != nil {
		return skipToNil(err)
	}
	discard := func() {
		_ = out.Close()
		_ = os.Remove(tmp)
	}

	writer := newArchiveWriter(out, r.job.spec.format)
	for _, src := range sources {
		if err := r.pack(writer, src); err != nil {
			discard()
			return skipToNil(err)
		}
	}
	if err := writer.Close(); err != nil {
		discard()
		return err
	}
	if err := out.Close(); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	if err := os.Rename(tmp, target); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	r.remember(KindDelete, target, "")
	return nil
}

func newArchiveWriter(out io.Writer, format string) archiveWriter {
	if format == FormatZip {
		return &zipWriter{inner: zip.NewWriter(out)}
	}
	gz := gzip.NewWriter(out)
	return &tarWriter{inner: tar.NewWriter(gz), gz: gz}
}

// pack adds one source tree under its own base name, so extracting the
// archive reproduces what was selected.
func (r *runner) pack(w archiveWriter, src string) error {
	base := filepath.Dir(src)
	return filepath.WalkDir(src, func(path string, entry fs.DirEntry, walkErr error) error {
		if r.job.closed() {
			return context.Canceled
		}
		if walkErr != nil {
			if entry == nil {
				return walkErr
			}
			r.warn(path, walkErr)
			r.skip()
			if entry.IsDir() {
				return fs.SkipDir
			}
			return nil
		}
		name, _ := filepath.Rel(base, path)
		name = filepath.ToSlash(name)
		info, err := entry.Info()
		if err != nil {
			return err
		}

		switch {
		case info.IsDir():
			if err := w.dir(name+"/", info); err != nil {
				return err
			}
			r.advance(path, 1, 0)
			return nil
		case info.Mode()&fs.ModeSymlink != 0:
			target, err := os.Readlink(path)
			if err != nil {
				return err
			}
			if err := w.symlink(name, target, info); err != nil {
				return err
			}
			r.advance(path, 1, 0)
			return nil
		case !info.Mode().IsRegular():
			r.skip()
			return nil
		}

		var in *os.File
		if err := r.retryLocal(path, func() error {
			opened, err := os.Open(path)
			in = opened
			return err
		}); err != nil {
			if errors.Is(err, errSkipped) {
				return nil
			}
			return err
		}
		defer in.Close()

		dst, err := w.file(name, info)
		if err != nil {
			return err
		}
		if err := r.stream(path, in, dst); err != nil {
			return err
		}
		r.advance(path, 1, 0)
		return nil
	})
}

func (r *runner) stream(path string, in io.Reader, out io.Writer) error {
	buf := make([]byte, copyChunk)
	for {
		if r.job.closed() {
			return context.Canceled
		}
		n, readErr := in.Read(buf)
		if n > 0 {
			if _, err := out.Write(buf[:n]); err != nil {
				return err
			}
			r.advance(path, 0, int64(n))
		}
		if readErr == io.EOF {
			return nil
		}
		if readErr != nil {
			return readErr
		}
	}
}

type zipWriter struct{ inner *zip.Writer }

func (z *zipWriter) header(name string, info fs.FileInfo) *zip.FileHeader {
	header := &zip.FileHeader{Name: name, Method: zip.Deflate, Modified: info.ModTime()}
	header.SetMode(info.Mode())
	return header
}

func (z *zipWriter) dir(name string, info fs.FileInfo) error {
	header := z.header(name, info)
	header.Method = zip.Store
	_, err := z.inner.CreateHeader(header)
	return err
}

func (z *zipWriter) symlink(name, target string, info fs.FileInfo) error {
	header := z.header(name, info)
	header.Method = zip.Store
	w, err := z.inner.CreateHeader(header)
	if err != nil {
		return err
	}
	_, err = io.WriteString(w, target)
	return err
}

func (z *zipWriter) file(name string, info fs.FileInfo) (io.Writer, error) {
	return z.inner.CreateHeader(z.header(name, info))
}

func (z *zipWriter) Close() error { return z.inner.Close() }

type tarWriter struct {
	inner *tar.Writer
	gz    *gzip.Writer
}

func (t *tarWriter) dir(name string, info fs.FileInfo) error {
	header, err := tar.FileInfoHeader(info, "")
	if err != nil {
		return err
	}
	header.Name = name
	return t.inner.WriteHeader(header)
}

func (t *tarWriter) symlink(name, target string, info fs.FileInfo) error {
	header, err := tar.FileInfoHeader(info, target)
	if err != nil {
		return err
	}
	header.Name = name
	return t.inner.WriteHeader(header)
}

func (t *tarWriter) file(name string, info fs.FileInfo) (io.Writer, error) {
	header, err := tar.FileInfoHeader(info, "")
	if err != nil {
		return nil, err
	}
	header.Name = name
	if err := t.inner.WriteHeader(header); err != nil {
		return nil, err
	}
	return t.inner, nil
}

func (t *tarWriter) Close() error {
	if err := t.inner.Close(); err != nil {
		return err
	}
	return t.gz.Close()
}

func (r *runner) extract() error {
	sources := r.job.spec.sources
	r.setTotals(0, 0, true)

	for _, archive := range sources {
		if r.job.closed() {
			return nil
		}
		target := filepath.Join(r.job.spec.dest, archiveStem(filepath.Base(archive)))
		if _, err := os.Lstat(target); err == nil {
			target = uniquePath(target)
		}
		if err := os.Mkdir(target, 0o755); err != nil {
			return err
		}
		err := r.unpack(archive, target)
		switch {
		case errors.Is(err, errSkipped):
			_ = os.RemoveAll(target)
			continue
		case err != nil:
			_ = os.RemoveAll(target)
			return skipToNil(err)
		}
		r.remember(KindDelete, target, "")
	}
	return nil
}

func (r *runner) unpack(archive, target string) error {
	format, err := detectFormat(archive)
	if err != nil {
		return err
	}
	var in *os.File
	if err := r.retryLocal(archive, func() error {
		opened, err := os.Open(archive)
		in = opened
		return err
	}); err != nil {
		return err
	}
	defer in.Close()

	root, err := os.OpenRoot(target)
	if err != nil {
		return err
	}
	defer root.Close()

	switch format {
	case FormatZip:
		return r.unzip(in, root)
	case FormatTarGz:
		gz, err := gzip.NewReader(in)
		if err != nil {
			return err
		}
		defer gz.Close()
		return r.untar(gz, root)
	default:
		return r.untar(in, root)
	}
}

// Extraction goes through os.Root so an entry can never land outside the
// target, not even through a symlink an earlier entry planted.
func (r *runner) unzip(in *os.File, root *os.Root) error {
	info, err := in.Stat()
	if err != nil {
		return err
	}
	reader, err := zip.NewReader(in, info.Size())
	if err != nil {
		return err
	}
	var bytes int64
	for _, f := range reader.File {
		bytes += int64(f.UncompressedSize64)
	}
	r.growTotals(len(reader.File))
	r.job.mu.Lock()
	r.job.progress.BytesTotal += bytes
	r.job.mu.Unlock()

	for _, f := range reader.File {
		if r.job.closed() {
			return context.Canceled
		}
		name, err := entryName(f.Name)
		if err != nil {
			return err
		}
		mode := f.Mode()
		switch {
		case strings.HasSuffix(f.Name, "/") || mode.IsDir():
			if err := root.MkdirAll(name, dirPerm(mode)); err != nil {
				return err
			}
		case mode&fs.ModeSymlink != 0:
			link, err := f.Open()
			if err != nil {
				return err
			}
			linkTarget, err := io.ReadAll(link)
			_ = link.Close()
			if err != nil {
				return err
			}
			if err := makeLink(root, name, string(linkTarget)); err != nil {
				return err
			}
		default:
			body, err := f.Open()
			if err != nil {
				return err
			}
			err = r.writeEntry(root, name, body, filePerm(mode))
			_ = body.Close()
			if err != nil {
				return err
			}
			_ = root.Chtimes(name, f.Modified, f.Modified)
		}
		r.advance(filepath.Join(root.Name(), name), 1, 0)
	}
	return nil
}

func (r *runner) untar(in io.Reader, root *os.Root) error {
	reader := tar.NewReader(in)
	for {
		if r.job.closed() {
			return context.Canceled
		}
		header, err := reader.Next()
		if errors.Is(err, io.EOF) {
			return nil
		}
		if err != nil {
			return err
		}
		name, err := entryName(header.Name)
		if err != nil {
			return err
		}
		mode := header.FileInfo().Mode()
		switch header.Typeflag {
		case tar.TypeDir:
			if err := root.MkdirAll(name, dirPerm(mode)); err != nil {
				return err
			}
		case tar.TypeSymlink:
			if err := makeLink(root, name, header.Linkname); err != nil {
				return err
			}
		case tar.TypeLink:
			old, err := entryName(header.Linkname)
			if err != nil {
				return err
			}
			if err := root.MkdirAll(filepath.Dir(name), 0o755); err != nil {
				return err
			}
			if err := root.Link(old, name); err != nil {
				return err
			}
		case tar.TypeReg:
			if err := r.writeEntry(root, name, reader, filePerm(mode)); err != nil {
				return err
			}
			_ = root.Chtimes(name, header.ModTime, header.ModTime)
		default:
			r.skip()
			continue
		}
		r.advance(filepath.Join(root.Name(), name), 1, 0)
	}
}

func makeLink(root *os.Root, name, target string) error {
	if err := root.MkdirAll(filepath.Dir(name), 0o755); err != nil {
		return err
	}
	return root.Symlink(target, name)
}

func (r *runner) writeEntry(root *os.Root, name string, body io.Reader, perm fs.FileMode) error {
	if err := root.MkdirAll(filepath.Dir(name), 0o755); err != nil {
		return err
	}
	out, err := root.OpenFile(name, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, perm)
	if err != nil {
		return err
	}
	if err := r.stream(filepath.Join(root.Name(), name), body, out); err != nil {
		_ = out.Close()
		return err
	}
	return out.Close()
}

// entryName rejects entries that would land outside the extraction folder
// by name; os.Root catches the symlink routes.
func entryName(name string) (string, error) {
	clean := filepath.Clean(filepath.FromSlash(name))
	if clean == "." || !filepath.IsLocal(clean) {
		return "", &archiveEntryError{name}
	}
	return clean, nil
}

type archiveEntryError struct{ name string }

func (e *archiveEntryError) Error() string {
	return "archive entry escapes the target folder: " + e.name
}

func dirPerm(mode fs.FileMode) fs.FileMode {
	perm := mode.Perm() | 0o700
	if perm == 0o700 {
		return 0o755
	}
	return perm
}

func filePerm(mode fs.FileMode) fs.FileMode {
	perm := mode.Perm() | 0o600
	if perm == 0o600 {
		return 0o644
	}
	return perm
}

var archiveSuffixes = []string{".tar.gz", ".tgz", ".tar", ".zip"}

func archiveStem(name string) string {
	lower := strings.ToLower(name)
	for _, suffix := range archiveSuffixes {
		if strings.HasSuffix(lower, suffix) && len(name) > len(suffix) {
			return name[:len(name)-len(suffix)]
		}
	}
	return name
}

// detectFormat trusts the name first, then the magic bytes, so an archive
// saved without an extension still opens.
func detectFormat(path string) (string, error) {
	lower := strings.ToLower(path)
	switch {
	case strings.HasSuffix(lower, ".zip"):
		return FormatZip, nil
	case strings.HasSuffix(lower, ".tar.gz"), strings.HasSuffix(lower, ".tgz"):
		return FormatTarGz, nil
	case strings.HasSuffix(lower, ".tar"):
		return "tar", nil
	}

	head := make([]byte, 265)
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	n, _ := io.ReadFull(f, head)
	_ = f.Close()
	head = head[:n]
	switch {
	case bytes.HasPrefix(head, []byte("PK\x03\x04")):
		return FormatZip, nil
	case bytes.HasPrefix(head, []byte{0x1f, 0x8b}):
		return FormatTarGz, nil
	case len(head) >= 262 && bytes.Equal(head[257:262], []byte("ustar")):
		return "tar", nil
	}
	return "", errUnsupportedArchive
}

// Extractable reports whether a mime type names an archive this package can
// unpack itself; anything else is opened with the default application.
func Extractable(mime string) bool {
	switch mime {
	case "application/zip", "application/x-tar", "application/gzip", "application/x-gzip", "application/x-compressed-tar":
		return true
	}
	return false
}
