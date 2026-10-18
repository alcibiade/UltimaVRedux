package main

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/config"
	"github.com/bradhannah/UltimaVRedux/internal/debugging"
)

type fakeDebugDisplayManager struct {
	actualMode        config.Mode
	requestedMode     config.Mode
	transitionPending bool
	normalWidth       int
	normalHeight      int
	windowWidth       int
	windowHeight      int
	monitorWidth      int
	monitorHeight     int
	containerWidth    int
	containerHeight   int
	logicalWidth      int
	logicalHeight     int
	requestMode       config.Mode
	requestWidth      int
	requestHeight     int
	requestErr        error
}

func (f *fakeDebugDisplayManager) Mode() config.Mode          { return f.actualMode }
func (f *fakeDebugDisplayManager) RequestedMode() config.Mode { return f.requestedMode }
func (f *fakeDebugDisplayManager) IsTransitionPending() bool  { return f.transitionPending }
func (f *fakeDebugDisplayManager) LastNormalWindowSize() (int, int) {
	return f.normalWidth, f.normalHeight
}
func (f *fakeDebugDisplayManager) WindowSize() (int, int) {
	return f.windowWidth, f.windowHeight
}
func (f *fakeDebugDisplayManager) MonitorSize() (int, int) {
	return f.monitorWidth, f.monitorHeight
}
func (f *fakeDebugDisplayManager) ContainerSize() (int, int) {
	return f.containerWidth, f.containerHeight
}
func (f *fakeDebugDisplayManager) LogicalSize() (int, int) {
	return f.logicalWidth, f.logicalHeight
}
func (f *fakeDebugDisplayManager) DeviceScaleFactor() float64 { return 1.5 }
func (f *fakeDebugDisplayManager) OutputSize() (int, int)     { return 2880, 1560 }
func (f *fakeDebugDisplayManager) RequestMode(mode config.Mode, width, height int) error {
	f.requestMode, f.requestWidth, f.requestHeight = mode, width, height
	if f.requestErr == nil {
		f.requestedMode = mode
		f.transitionPending = true
	}

	return f.requestErr
}

func TestDisplayStatusReportsManagerTelemetryAndPresets(t *testing.T) {
	manager := newFakeDebugDisplayManager()
	manager.actualMode = config.ModeMaximized
	manager.requestedMode = config.ModeFullscreen
	manager.transitionPending = true

	result, handled := executeDisplayDebugCommandWithManager(debugging.Request{
		SessionID: "test", ID: "status", CommandID: debugging.CommandDisplayStatus,
	}, manager)

	require.True(t, handled)
	require.True(t, result.Success)
	var status displayStatusResult
	require.NoError(t, json.Unmarshal(result.Data, &status))
	assert.Equal(t, config.ModeFullscreen, status.RequestedMode)
	assert.Equal(t, config.ModeMaximized, status.ActualMode)
	assert.True(t, status.TransitionPending)
	assert.Equal(t, 1600, status.NormalWindowWidth)
	assert.Equal(t, 900, status.NormalWindowHeight)
	assert.Equal(t, 1920, status.ActualWindowWidth)
	assert.Equal(t, 1040, status.ActualWindowHeight)
	assert.Equal(t, 3440, status.MonitorWidth)
	assert.Equal(t, 1440, status.MonitorHeight)
	assert.Equal(t, 1920, status.ContainerWidth)
	assert.Equal(t, 1040, status.ContainerHeight)
	assert.Equal(t, 1848, status.LogicalWidth)
	assert.Equal(t, 1040, status.LogicalHeight)
	assert.Equal(t, 1.5, status.DeviceScaleFactor)
	assert.Equal(t, 2880, status.OutputWidth)
	assert.Equal(t, 1560, status.OutputHeight)
	require.Len(t, status.PresetResolutions, 5)
	assert.Equal(t, displayResolutionResult{Index: 0, Width: 1280, Height: 720}, status.PresetResolutions[0])
}

func TestDisplayModeSetRequestsRuntimeModeWithNormalWindowSize(t *testing.T) {
	manager := newFakeDebugDisplayManager()
	request := debugging.Request{
		SessionID: "test", ID: "mode", CommandID: debugging.CommandDisplayModeSet,
		Arguments: map[string]json.RawMessage{"mode": json.RawMessage(`"maximized"`)},
	}

	result, handled := executeDisplayDebugCommandWithManager(request, manager)

	require.True(t, handled)
	require.True(t, result.Success)
	assert.Equal(t, config.ModeMaximized, manager.requestMode)
	assert.Equal(t, 1600, manager.requestWidth)
	assert.Equal(t, 900, manager.requestHeight)
	assert.Equal(t, debugging.ExecutionPresentation, result.EffectiveExecutionMode)
	assert.Contains(t, result.Summary, "player settings unchanged")
	require.Len(t, result.Changes, 1)
	assert.Equal(t, "presentation.display.requestedMode", result.Changes[0].Path)
}

func TestDisplayCompatibilityCommandsRouteThroughManagerWithoutPersistence(t *testing.T) {
	manager := newFakeDebugDisplayManager()
	manager.requestedMode = config.ModeMaximized
	resolutionRequest := debugging.Request{
		SessionID: "test", ID: "resolution", CommandID: debugging.CommandDisplayResolutionSet,
		Arguments: map[string]json.RawMessage{"index": json.RawMessage(`2`)},
	}

	resolution, handled := executeDisplayDebugCommandWithManager(resolutionRequest, manager)

	require.True(t, handled)
	require.True(t, resolution.Success)
	assert.Equal(t, config.ModeMaximized, manager.requestMode)
	assert.Equal(t, 1920, manager.requestWidth)
	assert.Equal(t, 1080, manager.requestHeight)
	assert.Contains(t, resolution.Summary, "player settings unchanged")

	fullscreenRequest := debugging.Request{
		SessionID: "test", ID: "fullscreen", CommandID: debugging.CommandDisplayFullscreenSet,
		Arguments: map[string]json.RawMessage{"enabled": json.RawMessage(`false`)},
	}
	fullscreen, handled := executeDisplayDebugCommandWithManager(fullscreenRequest, manager)

	require.True(t, handled)
	require.True(t, fullscreen.Success)
	assert.Equal(t, config.ModeWindowed, manager.requestMode)
	assert.Equal(t, 1600, manager.requestWidth)
	assert.Equal(t, 900, manager.requestHeight)
}

func TestDisplayRequestFailureIsRecoverable(t *testing.T) {
	manager := newFakeDebugDisplayManager()
	manager.requestErr = errors.New("window backend rejected request")
	request := debugging.Request{
		SessionID: "test", ID: "mode", CommandID: debugging.CommandDisplayModeSet,
		Arguments: map[string]json.RawMessage{"mode": json.RawMessage(`"fullscreen"`)},
	}

	result, handled := executeDisplayDebugCommandWithManager(request, manager)

	require.True(t, handled)
	assert.False(t, result.Success)
	assert.Equal(t, "display_request_failed", result.ErrorCode)
}

func newFakeDebugDisplayManager() *fakeDebugDisplayManager {
	return &fakeDebugDisplayManager{
		actualMode:      config.ModeWindowed,
		requestedMode:   config.ModeWindowed,
		normalWidth:     1600,
		normalHeight:    900,
		windowWidth:     1920,
		windowHeight:    1040,
		monitorWidth:    3440,
		monitorHeight:   1440,
		containerWidth:  1920,
		containerHeight: 1040,
		logicalWidth:    1848,
		logicalHeight:   1040,
	}
}
