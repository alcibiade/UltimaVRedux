package widgets

import (
	"image"
	"math"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMessageDialogControlledClockFreezesAndResumesFade(t *testing.T) {
	setResponsiveDialogTestResolution(t, 1280, 720)
	now := int64(0)
	dialog := NewActionFeedbackDialog("Unlocked!", PositionMapCenterLower)
	dialog.UseAnimationClock(func() int64 { return now })
	dialog.Show()

	now = 200
	dialog.Update()
	assert.InDelta(t, 0.5, dialog.DissolveLevel(), 0.000001)
	dialog.Update()
	assert.InDelta(t, 0.5, dialog.DissolveLevel(), 0.000001)
	now = 400
	dialog.Update()
	assert.Zero(t, dialog.DissolveLevel())

	now = 2000
	dialog.Hide()
	now = 2150
	dialog.Update()
	assert.InDelta(t, 0.5, dialog.DissolveLevel(), 0.000001)
	assert.Panics(t, func() { dialog.UseAnimationClock(func() int64 { return 0 }) })
}

func TestMessageDialogCompleteCardMeasuresAllRowsAndTranslatesCachedComposition(t *testing.T) {
	setResponsiveDialogTestResolution(t, 1280, 720)
	dialog := NewSearchFeedbackDialog(
		"A styled-width row that wraps inside the safe pane.\n\nThe complete final row remains visible.",
		PositionMapCenterLower,
	)
	screenBounds := image.Rect(0, 0, 1280, 720)
	available := image.Rect(120, 80, 640, 650)
	bounds := dialog.MeasureCompleteCard(screenBounds, available, 400)
	requiredHeight := int(math.Ceil(float64(dialog.renderOutput.GetPanelContentLineCount()+messagePopupMarginLines) *
		dialog.getScaledLineSpacing()))
	assert.GreaterOrEqual(t, bounds.Dy(), requiredHeight)
	assert.GreaterOrEqual(t, bounds.Min.X, available.Min.X)
	assert.LessOrEqual(t, bounds.Max.X, available.Max.X)
	assert.Equal(t, available.Min.Y, bounds.Min.Y)

	dialog.ShowWithoutAnimation()
	dialog.SetPresentationPosition(float64(bounds.Min.X), 180, available)
	screen := ebiten.NewImage(1280, 720)
	dialog.Draw(screen)
	firstBounds := renderedNontransparentBounds(screen, screen.Bounds())
	require.False(t, firstBounds.Empty())
	assert.Equal(t, 180, firstBounds.Min.Y)
	buffer, bounds := dialog.contentBuffer, dialog.frameBounds

	screen.Clear()
	dialog.SetPresentationPosition(float64(bounds.Min.X), 260, available)
	dialog.Draw(screen)
	secondBounds := renderedNontransparentBounds(screen, screen.Bounds())
	assert.Equal(t, 260, secondBounds.Min.Y)
	assert.Same(t, buffer, dialog.contentBuffer, "pure movement reuses the rendered card")
	assert.Equal(t, bounds, dialog.frameBounds, "pure movement does not change the baked frame")
	screen.Deallocate()
}

func TestMessageDialogCompleteCardClipsOversizedTailOnlyAtPresentationBoundary(t *testing.T) {
	setResponsiveDialogTestResolution(t, 1280, 720)
	dialog := NewSearchFeedbackDialog(
		"one\ntwo\nthree\nfour\nfive\nsix\nseven\neight\nnine\nten\neleven\ntwelve\nthirteen\nfourteen\nfifteen",
		PositionMapCenterLower,
	)
	available := image.Rect(100, 100, 700, 300)
	bounds := dialog.MeasureCompleteCard(image.Rect(0, 0, 1280, 720), available, 400)
	require.Greater(t, bounds.Dy(), available.Dy())
	dialog.SetPresentationPosition(float64(bounds.Min.X), float64(available.Min.Y), available)
	dialog.ShowWithoutAnimation()
	screen := ebiten.NewImage(1280, 720)
	dialog.Draw(screen)
	rendered := renderedNontransparentBounds(screen, screen.Bounds())
	assert.Equal(t, available.Min.Y, rendered.Min.Y)
	assert.LessOrEqual(t, rendered.Max.Y, available.Max.Y)
	assert.Greater(t, dialog.PresentationBounds().Max.Y, available.Max.Y)
	screen.Deallocate()
}

func TestMessageDialogTranslatedFadeUsesSameDissolvedComposition(t *testing.T) {
	setResponsiveDialogTestResolution(t, 1280, 720)
	now := int64(0)
	dialog := NewActionFeedbackDialog("Moving while fading", PositionMapCenterLower)
	dialog.UseAnimationClock(func() int64 { return now })
	available := image.Rect(80, 40, 700, 680)
	bounds := dialog.MeasureCompleteCard(image.Rect(0, 0, 1280, 720), available, 400)
	dialog.ShowWithoutAnimation()
	now = 2000
	dialog.Hide()
	now = 2150
	dialog.Update()
	require.InDelta(t, 0.5, dialog.DissolveLevel(), 0.000001)

	dialog.SetPresentationPosition(float64(bounds.Min.X), 180, available)
	firstScreen := ebiten.NewImage(1280, 720)
	dialog.Draw(firstScreen)
	firstBounds := renderedNontransparentBounds(firstScreen, firstScreen.Bounds())
	require.False(t, firstBounds.Empty())
	effect, buffer, bounds := dialog.dissolveEffect, dialog.contentBuffer, dialog.frameBounds

	dialog.SetPresentationPosition(float64(bounds.Min.X), 260, available)
	secondScreen := ebiten.NewImage(1280, 720)
	dialog.Draw(secondScreen)
	secondBounds := renderedNontransparentBounds(secondScreen, secondScreen.Bounds())
	require.False(t, secondBounds.Empty())
	assert.Equal(t, firstBounds.Add(image.Pt(0, 80)), secondBounds)
	assert.Same(t, effect, dialog.dissolveEffect)
	assert.Same(t, buffer, dialog.contentBuffer)
	assert.Equal(t, bounds, dialog.frameBounds)
	firstScreen.Deallocate()
	secondScreen.Deallocate()
}

func TestMessageDialogDissolveRebuildsDimensionValidMaskWithPreservedProgress(t *testing.T) {
	setResponsiveDialogTestResolution(t, 1280, 720)
	now := int64(0)
	dialog := NewActionFeedbackDialog("Resize", PositionMapCenterLower)
	dialog.UseAnimationClock(func() int64 { return now })
	available := image.Rect(80, 40, 700, 680)
	dialog.MeasureCompleteCard(image.Rect(0, 0, 1280, 720), available, 400)
	dialog.Show()
	now = 200
	dialog.Update()
	screen := ebiten.NewImage(1280, 720)
	dialog.Draw(screen)
	oldEffect := dialog.dissolveEffect
	require.NotNil(t, oldEffect)

	now = 2000
	dialog.Hide()
	now = 2150
	dialog.Update()
	assert.InDelta(t, 0.5, dialog.DissolveLevel(), 0.000001)
	setResponsiveDialogTestResolution(t, 1920, 1080)
	largeAvailable := image.Rect(120, 60, 1000, 1000)
	dialog.MeasureCompleteCard(image.Rect(0, 0, 1920, 1080), largeAvailable, 600)
	largeScreen := ebiten.NewImage(1920, 1080)
	assert.NotPanics(t, func() { dialog.Draw(largeScreen) })
	require.NotNil(t, dialog.dissolveEffect)
	assert.NotSame(t, oldEffect, dialog.dissolveEffect)
	assert.InDelta(t, 0.5, dialog.DissolveLevel(), 0.000001)
	assert.True(t, dialog.IsVisible())
	screen.Deallocate()
	largeScreen.Deallocate()
}

func TestMessageDialogFirstDrawAfterFadeOutInitialization(t *testing.T) {
	setResponsiveDialogTestResolution(t, 1280, 720)
	now := int64(0)
	dialog := NewActionFeedbackDialog("Late draw", PositionMapCenterLower)
	dialog.UseAnimationClock(func() int64 { return now })
	dialog.Show()
	now = 2000
	dialog.Hide()
	now = 2150
	dialog.Update()
	screen := ebiten.NewImage(1280, 720)
	assert.NotPanics(t, func() { dialog.Draw(screen) })
	require.NotNil(t, dialog.dissolveEffect)
	assert.InDelta(t, 0.5, dialog.DissolveLevel(), 0.000001)
	screen.Deallocate()
}
