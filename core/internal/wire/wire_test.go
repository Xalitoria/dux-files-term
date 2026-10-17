package wire

import (
	"errors"
	"testing"

	"github.com/AvengeMedia/dankgo/files"
	"github.com/stretchr/testify/assert"
)

func TestMessageDropsTheCodePrefix(t *testing.T) {
	assert.Equal(t, "gvfs is not running", message(&files.Error{Code: files.CodeNotSupported, Err: errors.New("gvfs is not running")}))
	assert.Equal(t, "EACCES: /x", message(&files.Error{Code: files.CodeAccess, Path: "/x"}))
	assert.Equal(t, "plain", message(errors.New("plain")))
}

func TestStringListAcceptsAStringOrAList(t *testing.T) {
	assert.Equal(t, []string{"/a"}, StringList(map[string]any{"paths": "/a"}, "paths"))
	assert.Equal(t, []string{"/a", "/b"}, StringList(map[string]any{"paths": []any{"/a", "/b"}}, "paths"))
	assert.Nil(t, StringList(map[string]any{}, "paths"))
}
