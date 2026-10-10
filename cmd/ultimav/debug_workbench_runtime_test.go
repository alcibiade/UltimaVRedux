package main

import (
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestShutdownDebugWorkbenchAllowsDisabledServer(t *testing.T) {
	err := shutdownDebugWorkbench(nil)

	require.NoError(t, err)
}

func TestStartDebugWorkbenchRejectsInvalidPort(t *testing.T) {
	game := NewGame(&noDebugDispatchTestScene{})

	server, err := startDebugWorkbench(game, 0)

	require.Error(t, err)
	assert.ErrorIs(t, err, errInvalidDebugWorkbenchPort)
	assert.Nil(t, server)
	assert.False(t, game.debugDiscoveryPolling.Load())
}

func TestWorkbenchLifetimeOwnsContinuousDiscovery(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	port := listener.Addr().(*net.TCPAddr).Port
	scene := &debugDispatchTestScene{}
	game := NewGame(scene)
	_, err = startDebugWorkbench(game, port)
	require.Error(t, err, "occupied listener must reject startup")
	assert.False(t, game.debugDiscoveryPolling.Load())
	require.NoError(t, listener.Close())
	runtime, err := startDebugWorkbench(game, port)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, shutdownDebugWorkbench(runtime)) })
	assert.True(t, game.debugDiscoveryPolling.Load())
	reads := scene.optionReads
	require.NoError(t, game.Update())
	assert.Greater(t, scene.optionReads, reads, "listening server publishes fresh choices even before browser connection")
	require.NoError(t, shutdownDebugWorkbench(runtime))
	assert.False(t, game.debugDiscoveryPolling.Load())
	reads = scene.optionReads
	require.NoError(t, game.Update())
	assert.Equal(t, reads, scene.optionReads)
}
