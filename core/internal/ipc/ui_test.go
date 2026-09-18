package ipc

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUIOpenRejectsRelativePath(t *testing.T) {
	deps := Deps{Bus: NewEventBus(), Pending: &PendingOpen{}}
	out := routeAndRead(t, Request{ID: 1, Method: "ui.open", Params: map[string]any{"path": "docs"}}, deps)

	errMsg, ok := out["error"].(string)
	require.True(t, ok)
	assert.Contains(t, errMsg, "absolute path")
	assert.Nil(t, deps.Pending.Take())
}

func TestUIOpenStashesWhenNoSubscriber(t *testing.T) {
	deps := Deps{Bus: NewEventBus(), Pending: &PendingOpen{}}
	out := routeAndRead(t, Request{ID: 2, Method: "ui.open", Params: map[string]any{"path": "/tmp"}}, deps)

	result, ok := out["result"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, true, result["ok"])
	assert.Equal(t, map[string]any{"action": "open", "path": "/tmp"}, deps.Pending.Take())
	assert.Nil(t, deps.Pending.Take())
}
