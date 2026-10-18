package main

import (
	"math"
	"testing"

	"github.com/bradhannah/UltimaVRedux/internal/config"
	"github.com/bradhannah/UltimaVRedux/internal/display"
	"github.com/bradhannah/UltimaVRedux/internal/sprites"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDisplayPixelPresetsRemainUsefulAcrossDesktopScales(t *testing.T) {
	for _, scale := range []float64{1, 1.25, 1.5, 1.75, 2} {
		manager := newConfigurationDisplayManagerStub()
		manager.scale = scale
		manager.monitorWidth = int(math.Round(2560 / scale))
		manager.monitorHeight = int(math.Round(1440 / scale))
		dialog, _ := newDisplayTestDialog(manager, func() {})
		found := false
		for _, choice := range dialog.windowSizeChoices() {
			if choice.pixelWidth == 1920 && choice.pixelHeight == 1080 {
				require.True(t, choice.available, "1080p should fit 1440p at scale %g", scale)
				width, height := display.WindowUnits(1920, 1080, scale)
				assert.Equal(t, width, choice.X)
				assert.Equal(t, height, choice.Y)
				found = true
			}
		}
		require.True(t, found)
	}
}

func TestDisplayLabelsExposeRepresentablePixelsAndRounding(t *testing.T) {
	manager := newConfigurationDisplayManagerStub()
	manager.scale = 1.5
	dialog, _ := newDisplayTestDialog(manager, func() {})
	dialog.stagedDisplay.WindowWidth, dialog.stagedDisplay.WindowHeight = display.WindowUnits(1366, 768, manager.scale)
	assert.Equal(t, "About 1367 x 768 pixels", dialog.displaySizeLabel())
	dialog.stagedDisplay.WindowWidth, dialog.stagedDisplay.WindowHeight = 1280, 720
	assert.Equal(t, "1920 x 1080 pixels", dialog.displaySizeLabel())
}

func TestDisplayConfirmationRollsBackOnScaleChangeWithoutSaving(t *testing.T) {
	manager := newConfigurationDisplayManagerStub()
	dialog, configuration := newDisplayTestDialog(manager, func() {})
	pressAudioSettingsKey(dialog, ebiten.KeyTab)
	pressAudioSettingsKey(dialog, ebiten.KeyRight)
	pressAudioSettingsKey(dialog, ebiten.KeyEnter)
	manager.completeTransition()
	dialog.Update()
	require.Equal(t, configurationDisplayConfirming, dialog.displayPhase)
	manager.scale = 2
	dialog.Update()
	assert.Equal(t, configurationDisplayReverting, dialog.displayPhase)
	assert.Equal(t, config.ModeWindowed, configuration.DisplayConfiguration.Mode)
	assert.Contains(t, dialog.saveError, "Display changed")
}

func TestDisplayRejectsSizeThatStopsFittingBeforeApply(t *testing.T) {
	manager := newConfigurationDisplayManagerStub()
	dialog, _ := newDisplayTestDialog(manager, func() {})
	pressAudioSettingsKey(dialog, ebiten.KeyTab)
	pressAudioSettingsKey(dialog, ebiten.KeyDown)
	pressAudioSettingsKey(dialog, ebiten.KeyRight)
	manager.monitorWidth, manager.monitorHeight = 1024, 600
	pressAudioSettingsKey(dialog, ebiten.KeyEnter)
	assert.Equal(t, configurationDisplayEditing, dialog.displayPhase)
	assert.Contains(t, dialog.saveError, "no longer fits")
	assert.False(t, manager.pending)
}

func TestDisplayHelpAndControlsFitTitleAndGameplayPlacements(t *testing.T) {
	manager := newConfigurationDisplayManagerStub()
	dialog, _ := newDisplayTestDialog(manager, func() {})
	for _, width := range []int{1280, 1920, 3840} {
		for _, gameplay := range []bool{false, true} {
			if gameplay {
				placement := gameScreenPercents
				dialog.placement = &placement
			} else {
				dialog.placement = &sprites.PercentBasedPlacement{EndPercentX: 1, EndPercentY: 1}
			}
			dialog.lastWidth = 0
			dialog.ensureLayout(width, width*9/16)
			for _, focused := range []int{configurationDisplayModeRow, configurationDisplaySizeRow} {
				dialog.focusedDisplayIndex = focused
				screen := ebiten.NewImage(width, width*9/16)
				dialog.drawDisplayContent(screen)
				height := int(math.Ceil(dialog.detailOutput.GetTotalHeightForLines(dialog.detailOutput.GetPanelContentLineCount())))
				assert.LessOrEqual(t, height+dialog.layout.pixelUnit*audioSettingsSectionGapUnits,
					dialog.layout.displayHelp.Dy(), "width=%d gameplay=%v focus=%d", width, gameplay, focused)
				assert.True(t, dialog.layout.displayHelp.In(dialog.layout.bodyRect))
				screen.Deallocate()
			}
		}
	}
}
