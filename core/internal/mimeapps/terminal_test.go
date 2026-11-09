package mimeapps

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func fakeTerminal(t *testing.T, dir, name string) string {
	t.Helper()
	require.NoError(t, os.MkdirAll(dir, 0o755))
	path := filepath.Join(dir, name)
	require.NoError(t, os.WriteFile(path, []byte("#!/bin/sh\n"), 0o755))
	return path
}

func TestTerminalsAreFoundOutsideTheDaemonPath(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("PATH", t.TempDir())
	t.Setenv("TERMINAL", "")
	ghostty := fakeTerminal(t, filepath.Join(home, ".nix-profile", "bin"), "ghostty")
	alacritty := fakeTerminal(t, filepath.Join(home, ".local", "state", "nix", "profiles", "home-manager", "home-manager-path", "bin"), "alacritty")

	assert.Equal(t, []string{"ghostty", "alacritty"}, Terminals())

	spec, ok := lookupTerminal("alacritty")
	require.True(t, ok)
	assert.Equal(t, alacritty, spec.command, "launched by full path")
	assert.Equal(t, []string{"-e"}, spec.exec)

	picked, err := (&Service{}).terminal("ghostty")
	require.NoError(t, err)
	assert.Equal(t, ghostty, picked.command)
	assert.Contains(t, picked.argv("/tmp"), "--working-directory=/tmp")
}

func TestPathWinsOverTheExtraDirectories(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	bin := t.TempDir()
	t.Setenv("PATH", bin)
	fakeTerminal(t, bin, "kitty")
	fakeTerminal(t, filepath.Join(home, ".local", "bin"), "kitty")

	found, ok := findCommand("kitty")
	require.True(t, ok)
	assert.Equal(t, "kitty", found, "a command on PATH keeps its name")
	_, ok = findCommand("missing-term")
	assert.False(t, ok)
}
