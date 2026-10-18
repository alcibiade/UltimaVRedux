package display

import (
	"testing"

	"github.com/bradhannah/UltimaVRedux/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPixelPresetsQuantizeWithoutChangingLegacyIndices(t *testing.T) {
	assert.Equal(t, config.ScreenResolution{X: 1920, Y: 1080}, config.SupportedWindowResolutions()[2])
	assert.Contains(t, config.WindowPixelPresets(), config.ScreenResolution{X: 1600, Y: 900})
	for _, scale := range []float64{1, 1.25, 1.5, 1.75, 2} {
		for _, preset := range config.WindowPixelPresets() {
			width, height := WindowUnits(preset.X, preset.Y, scale)
			pixelsW, pixelsH := WindowPixels(width, height, scale)
			assert.InDelta(t, preset.X, pixelsW, scale/2+0.5)
			assert.InDelta(t, preset.Y, pixelsH, scale/2+0.5)
		}
	}
}

func TestHighDPIWindowMinimumAndStartupFit(t *testing.T) {
	for _, test := range []struct {
		monitorW, monitorH int
		scale              float64
	}{
		{1707, 960, 1.5}, {1280, 720, 2}, {1024, 600, 1},
	} {
		width, height := FitWindowSize(2560, 1440, test.monitorW, test.monitorH, test.scale)
		assert.LessOrEqual(t, float64(width), float64(test.monitorW)*WindowFitFraction)
		assert.LessOrEqual(t, float64(height), float64(test.monitorH)*WindowFitFraction)
		assert.GreaterOrEqual(t, width, config.MinimumWindowUnitsWidth)
		assert.GreaterOrEqual(t, height, config.MinimumWindowUnitsHeight)
	}
	backend := newFakeBackend()
	backend.scale = 2
	manager := newDisplayManager(backend)
	manager.ConfigureResizable()
	assert.Equal(t, [4]int{640, 360, -1, -1}, backend.limits)
	require.NoError(t, manager.RequestMode(config.ModeWindowed, 960, 540))
}

func TestRenderSizeUsesFractionalOutputDensityAndBounds(t *testing.T) {
	for _, test := range []struct {
		width, height, scale float64
		wantW, wantH         int
	}{
		{1280, 720, 1.5, 1920, 1080},
		{2560.0 / 1.5, 960, 1.5, 2560, 1440},
		{1280, 800, 2, 2560, 1440},
		{3440, 1440, 1, 2560, 1440},
		{320, 180, 1, 1280, 720},
		{7680, 4320, 2, 3840, 2160},
	} {
		assert.Equal(t, []int{test.wantW, test.wantH}, pair(RenderSize(test.width, test.height, test.scale)))
	}
}

func TestRestoreRetainsCapturedCustomGeometryUntilDesktopRequiresFitting(t *testing.T) {
	// The preset headroom must not shrink an already valid custom window on Revert.
	assert.Equal(t, []int{1800, 1000}, pair(RestoreWindowSize(1800, 1000, 1920, 1080, 1)))
	width, height := RestoreWindowSize(1800, 1000, 1280, 720, 2)
	assert.LessOrEqual(t, width, 1280)
	assert.LessOrEqual(t, height, 720)
	assert.Equal(t, []int{1280, 720}, pair(RestoreWindowSize(640, 360, 1920, 1080, 1)))
}

func TestScaleOnlyChangeInvalidatesWithoutMovingOrPersistingWindow(t *testing.T) {
	backend := newFakeBackend()
	manager := newDisplayManager(backend)
	manager.LayoutSize(1280, 720)
	manager.Update()
	backend.scale = 1.5
	assert.True(t, manager.Update())
	assert.Equal(t, 1.5, manager.DeviceScaleFactor())
	assert.Zero(t, backend.positionCalls)
	assert.Equal(t, []int{1280, 720}, pair(manager.PersistableWindowSize()))
	assert.Equal(t, []int{1920, 1080}, pair(manager.LayoutSize(1280, 720)))
	manager.PublishOutputSize(1920, 1080)
	assert.Equal(t, []int{1920, 1080}, pair(manager.OutputSize()))
}
