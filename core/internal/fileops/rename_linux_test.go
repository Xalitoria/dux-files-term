package fileops

import (
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRenameUnlessExistsKeepsExistingTargets(t *testing.T) {
	dir := t.TempDir()
	from, taken, free := filepath.Join(dir, "from"), filepath.Join(dir, "taken"), filepath.Join(dir, "free")
	require.NoError(t, os.WriteFile(from, []byte("new"), 0o644))
	require.NoError(t, os.WriteFile(taken, []byte("old"), 0o644))

	err := renameUnlessExists(from, taken)
	assert.ErrorIs(t, err, fs.ErrExist)
	kept, _ := os.ReadFile(taken)
	assert.Equal(t, "old", string(kept))

	require.NoError(t, renameUnlessExists(from, free))
	moved, _ := os.ReadFile(free)
	assert.Equal(t, "new", string(moved))
}
