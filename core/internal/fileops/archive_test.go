package fileops

import (
	"archive/zip"
	"os"
	"path/filepath"
	"testing"

	"github.com/AvengeMedia/dankgo/ipc"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func sampleTree(t *testing.T) (root, project string) {
	t.Helper()
	root = t.TempDir()
	project = filepath.Join(root, "project")
	write(t, filepath.Join(project, "README.md"), "hello")
	write(t, filepath.Join(project, "src", "main.go"), "package main")
	require.NoError(t, os.Chmod(filepath.Join(project, "src", "main.go"), 0o755))
	require.NoError(t, os.Symlink("README.md", filepath.Join(project, "link")))
	require.NoError(t, os.MkdirAll(filepath.Join(project, "empty"), 0o755))
	return root, project
}

func TestCompressAndExtractRoundTrip(t *testing.T) {
	for _, format := range []string{FormatZip, FormatTarGz} {
		t.Run(format, func(t *testing.T) {
			root, project := sampleTree(t)
			manager, bus := newHarness(t)

			view := run(t, manager, bus, spec{kind: KindCompress, sources: []string{project, filepath.Join(project, "README.md")}, dest: root, name: "bundle", format: format})
			require.Equal(t, StateDone, view.State, view.Error)
			archive := filepath.Join(root, "bundle."+format)
			require.FileExists(t, archive)
			assert.True(t, view.Undoable)
			assert.Equal(t, 0, view.Progress.FilesTotal-view.Progress.FilesDone)
			assert.Empty(t, tempsIn(t, root))

			out := filepath.Join(root, "out")
			require.NoError(t, os.Mkdir(out, 0o755))
			view = run(t, manager, bus, spec{kind: KindExtract, sources: []string{archive}, dest: out})
			require.Equal(t, StateDone, view.State, view.Error)

			base := filepath.Join(out, "bundle")
			assert.Equal(t, "hello", read(t, filepath.Join(base, "project", "README.md")))
			assert.Equal(t, "package main", read(t, filepath.Join(base, "project", "src", "main.go")))
			assert.Equal(t, "hello", read(t, filepath.Join(base, "README.md")), "a second top-level source lands beside the first")
			info, err := os.Stat(filepath.Join(base, "project", "src", "main.go"))
			require.NoError(t, err)
			assert.Equal(t, os.FileMode(0o755), info.Mode().Perm())
			assert.DirExists(t, filepath.Join(base, "project", "empty"))
			target, err := os.Readlink(filepath.Join(base, "project", "link"))
			require.NoError(t, err)
			assert.Equal(t, "README.md", target)
			assert.True(t, view.Undoable)

			undo, err := manager.Undo("")
			require.NoError(t, err)
			bus.wait(t, undo.ID)
			assert.NoDirExists(t, base)
		})
	}
}

func TestCompressKeepsBothWhenTheNameIsTaken(t *testing.T) {
	root, project := sampleTree(t)
	write(t, filepath.Join(root, "project.zip"), "old")
	manager, bus := newHarness(t)

	view := run(t, manager, bus, spec{kind: KindCompress, sources: []string{project}, dest: root, name: "project", format: FormatZip})
	require.Equal(t, StateDone, view.State, view.Error)
	assert.Equal(t, "old", read(t, filepath.Join(root, "project.zip")))
	assert.FileExists(t, filepath.Join(root, "project (copy).zip"))
}

func TestExtractIntoAnExistingFolderNameKeepsBoth(t *testing.T) {
	root, project := sampleTree(t)
	manager, bus := newHarness(t)
	run(t, manager, bus, spec{kind: KindCompress, sources: []string{project}, dest: root, name: "project", format: FormatTarGz})
	require.NoError(t, os.Mkdir(filepath.Join(root, "out"), 0o755))
	require.NoError(t, os.Mkdir(filepath.Join(root, "out", "project"), 0o755))

	view := run(t, manager, bus, spec{kind: KindExtract, sources: []string{filepath.Join(root, "project.tar.gz")}, dest: filepath.Join(root, "out")})
	require.Equal(t, StateDone, view.State, view.Error)
	assert.FileExists(t, filepath.Join(root, "out", "project (copy)", "project", "README.md"))
}

func TestExtractRefusesEntriesThatEscape(t *testing.T) {
	root := t.TempDir()
	archive := filepath.Join(root, "evil.zip")
	f, err := os.Create(archive)
	require.NoError(t, err)
	w := zip.NewWriter(f)
	entry, err := w.Create("../escaped.txt")
	require.NoError(t, err)
	_, err = entry.Write([]byte("nope"))
	require.NoError(t, err)
	require.NoError(t, w.Close())
	require.NoError(t, f.Close())

	manager, bus := newHarness(t)
	out := filepath.Join(root, "out")
	require.NoError(t, os.Mkdir(out, 0o755))
	view := run(t, manager, bus, spec{kind: KindExtract, sources: []string{archive}, dest: out})
	assert.Equal(t, StateFailed, view.State)
	assert.NoFileExists(t, filepath.Join(root, "escaped.txt"))
	assert.NoDirExists(t, filepath.Join(out, "evil"), "a failed extraction leaves nothing behind")
}

func TestExtractDetectsFormatWithoutAnExtension(t *testing.T) {
	root, project := sampleTree(t)
	manager, bus := newHarness(t)
	run(t, manager, bus, spec{kind: KindCompress, sources: []string{project}, dest: root, name: "project", format: FormatZip})
	bare := filepath.Join(root, "bare")
	require.NoError(t, os.Rename(filepath.Join(root, "project.zip"), bare))
	out := filepath.Join(root, "out")
	require.NoError(t, os.Mkdir(out, 0o755))

	view := run(t, manager, bus, spec{kind: KindExtract, sources: []string{bare}, dest: out})
	require.Equal(t, StateDone, view.State, view.Error)
	assert.FileExists(t, filepath.Join(out, "bare", "project", "README.md"))

	write(t, filepath.Join(root, "plain.bin"), "not an archive")
	view = run(t, manager, bus, spec{kind: KindExtract, sources: []string{filepath.Join(root, "plain.bin")}, dest: out})
	assert.Equal(t, StateFailed, view.State)
	assert.NoDirExists(t, filepath.Join(out, "plain.bin"))
}

func TestArchiveStem(t *testing.T) {
	assert.Equal(t, "a", archiveStem("a.tar.gz"))
	assert.Equal(t, "a.b", archiveStem("a.b.tgz"))
	assert.Equal(t, "A", archiveStem("A.ZIP"))
	assert.Equal(t, "a", archiveStem("a.tar"))
	assert.Equal(t, "plain", archiveStem("plain"))
	assert.Equal(t, ".zip", archiveStem(".zip"))
}

func TestCompressHandlerValidation(t *testing.T) {
	manager, _ := newHarness(t)
	cases := []map[string]any{
		{"paths": "/tmp/a", "dest": "/tmp", "name": "x", "format": "rar"},
		{"paths": "/tmp/a", "dest": "/tmp", "name": "x/y"},
		{"paths": "/tmp/a", "dest": "relative", "name": "x"},
	}
	for _, params := range cases {
		out := call(t, manager, ipcRequest("files.op.compress", params))
		require.Contains(t, out, "error")
	}
	out := call(t, manager, ipcRequest("files.op.extract", map[string]any{"paths": "/tmp/a"}))
	require.Contains(t, out, "error")
	require.Empty(t, manager.Jobs())
}

func ipcRequest(method string, params map[string]any) ipc.Request {
	return ipc.Request{Method: method, Params: params}
}

func TestExtractRefusesWritingThroughAPlantedSymlink(t *testing.T) {
	root := t.TempDir()
	outside := filepath.Join(root, "outside")
	require.NoError(t, os.Mkdir(outside, 0o755))
	archive := filepath.Join(root, "sneaky.zip")
	f, err := os.Create(archive)
	require.NoError(t, err)
	w := zip.NewWriter(f)
	link := &zip.FileHeader{Name: "evil"}
	link.SetMode(os.ModeSymlink | 0o777)
	entry, err := w.CreateHeader(link)
	require.NoError(t, err)
	_, err = entry.Write([]byte(outside))
	require.NoError(t, err)
	entry, err = w.Create("evil/planted.txt")
	require.NoError(t, err)
	_, err = entry.Write([]byte("nope"))
	require.NoError(t, err)
	require.NoError(t, w.Close())
	require.NoError(t, f.Close())

	manager, bus := newHarness(t)
	out := filepath.Join(root, "out")
	require.NoError(t, os.Mkdir(out, 0o755))
	view := run(t, manager, bus, spec{kind: KindExtract, sources: []string{archive}, dest: out})
	assert.Equal(t, StateFailed, view.State)
	assert.NoFileExists(t, filepath.Join(outside, "planted.txt"))
}

func TestCompressSkipsAnUnreadableSubdirectory(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("root reads everything")
	}
	root, project := sampleTree(t)
	locked := filepath.Join(project, "locked")
	require.NoError(t, os.Mkdir(locked, 0o755))
	write(t, filepath.Join(locked, "secret.txt"), "x")
	require.NoError(t, os.Chmod(locked, 0o000))
	t.Cleanup(func() { _ = os.Chmod(locked, 0o755) })

	manager, bus := newHarness(t)
	view := run(t, manager, bus, spec{kind: KindCompress, sources: []string{project}, dest: root, name: "project", format: FormatZip})
	require.Equal(t, StateDone, view.State, view.Error)
	assert.FileExists(t, filepath.Join(root, "project.zip"))
	assert.Equal(t, 1, view.Skipped)
}
