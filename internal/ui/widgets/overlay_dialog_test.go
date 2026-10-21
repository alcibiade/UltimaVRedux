package widgets

import (
	"image"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =====================================
// VALIDATION TESTS - Required Config
// =====================================

// TestOverlayDialog_Build_PanicsIfDurationMissing verifies that Build() panics
// when neither duration method is called.
func TestOverlayDialog_Build_PanicsIfDurationMissing(t *testing.T) {
	defer func() {
		r := recover()
		require.NotNil(t, r, "Build() should panic when duration is missing")
		panicMsg := r.(string)
		assert.Contains(t, panicMsg, "Duration", "Panic message should mention Duration")
	}()

	NewOverlayDialog("Test").
		WithSizing(SizingAutoFitDefault).
		WithPosition(PositionMapCenterLower).
		WithEffects(EffectsNone).
		WithContentAlignment(ContentAlignCenter).
		WithTextAlignment(TextAlignCenter).
		Build()
}

// TestOverlayDialog_Build_PanicsWithMultipleMissingFields verifies that the panic message
// lists all missing required fields.
func TestOverlayDialog_Build_PanicsWithMultipleMissingFields(t *testing.T) {
	defer func() {
		r := recover()
		require.NotNil(t, r, "Build() should panic with multiple missing fields")
		panicMsg := r.(string)
		assert.Contains(t, panicMsg, "Sizing", "Panic message should mention Sizing")
		assert.Contains(t, panicMsg, "Position", "Panic message should mention Position")
		assert.Contains(t, panicMsg, "Effects", "Panic message should mention Effects")
		assert.Contains(t, panicMsg, "Duration", "Panic message should mention Duration")
	}()

	NewOverlayDialog("Test").Build()
}

// =====================================
// VALIDATION TESTS - Builder State Tracking
// =====================================

func TestOverlayDialog_ExternallyManagedDurationDoesNotAdvanceInUpdate(t *testing.T) {
	dismissals := 0
	dialog := NewOverlayDialog("Scene owned").
		AsOverlayFeedback().
		WithExternallyManagedDuration(1000).
		OnDismiss(func() { dismissals++ }).
		Build()
	dialog.ShowWithoutAnimation()

	for range 100 {
		dialog.Update()
	}

	assert.True(t, dialog.IsTimed())
	assert.True(t, dialog.IsDurationExternallyManaged())
	assert.True(t, dialog.IsVisible())
	assert.False(t, dialog.IsDismissed())
	assert.Zero(t, dismissals)
}

// =====================================
// TEMPLATE TESTS
// =====================================

func TestOverlayDialogConfirmationShowsFooterOnlyWhileAwaitingInput(t *testing.T) {
	setResponsiveDialogTestResolution(t, 1280, 720)
	keyboard := newWidgetTestKeyboard(0)
	dialog := NewOverlayDialog("Moonstone buried!").
		AsConfirmation().
		WithUserDismissed(keyboard).
		Build()
	dialog.ShowWithoutAnimation()

	require.True(t, dialog.dialog.hasPromptOutput())
	require.Equal(t, "Enter/Esc: Close", dialog.dialog.promptOutput.GetOutputStr(false))
	screen := ebiten.NewImage(1280, 720)
	t.Cleanup(screen.Dispose)
	dialog.Draw(screen)
	requireClassicFrameSilhouette(t, screen, dialog.dialog.frameBounds)

	keyboard.SimulateKeyPress(ebiten.KeyEnter)
	dialog.Update()
	require.True(t, dialog.IsDismissed())
	require.Empty(t, dialog.dialog.promptOutput.GetOutputStr(false))
}

func TestOverlayDialogUserDismissalAlwaysShowsSharedFooter(t *testing.T) {
	builders := map[string]func() *OverlayDialogBuilder{
		"confirmation": func() *OverlayDialogBuilder { return NewOverlayDialog("OK").AsConfirmation() },
		"feedback":     func() *OverlayDialogBuilder { return NewOverlayDialog("OK").AsOverlayFeedback() },
		"custom": func() *OverlayDialogBuilder {
			return NewOverlayDialog("OK").WithSizing(SizingAutoFitDefault).
				WithPosition(PositionScreenCenter).WithEffects(EffectsNone).
				WithContentAlignment(ContentAlignCenter).WithTextAlignment(TextAlignCenter)
		},
	}
	for name, build := range builders {
		t.Run(name, func(t *testing.T) {
			setResponsiveDialogTestResolution(t, 1280, 720)
			keyboard := newWidgetTestKeyboard(0)
			dismissals := 0
			dialog := build().WithUserDismissed(keyboard).OnDismiss(func() { dismissals++ }).Build()
			require.Equal(t, "OK", dialog.GetMessage())
			require.Zero(t, dialog.GetDurationMs())
			require.False(t, dialog.IsTimed())
			require.True(t, dialog.IsUserDismissed())
			require.False(t, dialog.IsVisible())
			dialog.ShowWithoutAnimation()
			for _, width := range []int{1280, 1920} {
				setResponsiveDialogTestResolution(t, width, width*9/16)
				dialog.PrepareLayout(image.Rect(0, 0, width, width*9/16))
				require.False(t, dialog.IsDismissed())
				require.True(t, dialog.IsVisible())
				require.Equal(t, DialogCloseHint, dialog.dialog.promptOutput.GetOutputStr(false))
				require.Equal(t, DialogControlHintColor, dialog.dialog.promptColor)
				require.Equal(t, "OK", dialog.GetContentText(), "footer must not replace the message")
				_, textWidth := dialog.dialog.calculateTextAreaBounds()
				hintWidth, _ := dialog.dialog.font.MeasureString(DialogCloseHint)
				require.LessOrEqual(t, hintWidth, float64(textWidth), "short messages must still fit the full footer")
			}
			require.Zero(t, dismissals, "preparation cannot invoke dismissal callbacks")
			keyboard.SimulateKeyPress(ebiten.KeyF4)
			dialog.PrepareLayout(image.Rect(0, 0, 1920, 1080))
			require.True(t, keyboard.IsKeyPressedAndRegister(ebiten.KeyF4), "preparation does not consume scene input")
			keyboard.SimulateKeyPress(ebiten.KeySpace)
			dialog.Update()
			require.False(t, dialog.IsDismissed(), "acknowledgement remains Enter/Escape, not Space")
			keyboard.SimulateKeyPress(ebiten.KeyEscape)
			dialog.Update()
			require.True(t, dialog.IsDismissed())
			require.Equal(t, 1, dismissals)
			require.Empty(t, dialog.dialog.promptOutput.GetOutputStr(false))
		})
	}
}

// =====================================
// CONFIG TYPE TESTS
// =====================================

// TestSizingStrategy_AutoFitSizing_GetConfig verifies AutoFitSizing behavior.
func TestSizingStrategy_AutoFitSizing_GetConfig(t *testing.T) {
	sizing := AutoFitSizing{MaxWidthPercent: 0.8, MinCharsPerLine: 20}
	lines, charsPerLine, autoFit := sizing.GetConfig()

	assert.Equal(t, 0, lines, "AutoFitSizing should return 0 lines (calculated from content)")
	assert.Equal(t, 20, charsPerLine, "AutoFitSizing should return MinCharsPerLine")
	assert.True(t, autoFit, "AutoFitSizing should return autoFit=true")
}

// TestSizingStrategy_FixedLinesSizing_GetConfig verifies FixedLinesSizing behavior.
func TestSizingStrategy_FixedLinesSizing_GetConfig(t *testing.T) {
	sizing := FixedLinesSizing{Lines: 5, CharsPerLine: 40}
	lines, charsPerLine, autoFit := sizing.GetConfig()

	assert.Equal(t, 5, lines, "FixedLinesSizing should return configured lines")
	assert.Equal(t, 40, charsPerLine, "FixedLinesSizing should return configured charsPerLine")
	assert.False(t, autoFit, "FixedLinesSizing should return autoFit=false")
}

// TestPosition_GetHorizontalPercent verifies horizontal position percentages.
func TestPosition_GetHorizontalPercent(t *testing.T) {
	tests := []struct {
		name     string
		position Position
		expected float64
	}{
		{"MapCenter", Position{HorizontalMapCenter, VerticalCenter}, 0.37},
		{"ScreenCenter", Position{HorizontalScreenCenter, VerticalCenter}, 0.50},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.InDelta(t, tt.expected, tt.position.GetHorizontalPercent(), 0.01)
		})
	}
}

// TestPosition_GetVerticalPercent verifies vertical position percentages.
func TestPosition_GetVerticalPercent(t *testing.T) {
	tests := []struct {
		name     string
		position Position
		expected float64
	}{
		{"Top", Position{HorizontalMapCenter, VerticalTop}, 0.15},
		{"Center", Position{HorizontalMapCenter, VerticalCenter}, 0.50},
		{"Lower", Position{HorizontalMapCenter, VerticalLower}, 0.70},
		{"Bottom", Position{HorizontalMapCenter, VerticalBottom}, 0.85},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.InDelta(t, tt.expected, tt.position.GetVerticalPercent(), 0.01)
		})
	}
}

// TestEffectsConfig_HasFadeIn verifies fade-in detection.
func TestEffectsConfig_HasFadeIn(t *testing.T) {
	assert.False(t, EffectsNone.HasFadeIn(), "EffectsNone should not have fade-in")
	assert.True(t, EffectsConfig{FadeInMs: 300, FadeOutMs: 300}.HasFadeIn(), "a fade-in duration should have fade-in")
	assert.False(t, EffectsConfig{FadeOutMs: 1000}.HasFadeIn(), "fade-out only should not have fade-in")
}

// TestEffectsConfig_HasFadeOut verifies fade-out detection.
func TestEffectsConfig_HasFadeOut(t *testing.T) {
	assert.False(t, EffectsNone.HasFadeOut(), "EffectsNone should not have fade-out")
	assert.True(t, EffectsConfig{FadeInMs: 300, FadeOutMs: 300}.HasFadeOut(), "a fade-out duration should have fade-out")
	assert.False(t, EffectsConfig{FadeInMs: 3000}.HasFadeOut(), "fade-in only should not have fade-out")
}

// TestEffectsConfig_HasAnyEffect verifies any-effect detection.
func TestEffectsConfig_HasAnyEffect(t *testing.T) {
	assert.False(t, EffectsNone.HasAnyEffect(), "EffectsNone should not have any effect")
	assert.True(t, EffectsConfig{FadeInMs: 300, FadeOutMs: 300}.HasAnyEffect(), "both fades should have effects")

	// Custom config with only fade-in
	fadeInOnly := EffectsConfig{FadeInMs: 500, FadeOutMs: 0}
	assert.True(t, fadeInOnly.HasAnyEffect(), "Fade-in only should have effects")

	// Custom config with only fade-out
	fadeOutOnly := EffectsConfig{FadeInMs: 0, FadeOutMs: 500}
	assert.True(t, fadeOutOnly.HasAnyEffect(), "Fade-out only should have effects")
}

// =====================================
// PRESET TESTS
// =====================================

// TestSizingPresets verifies preset sizing values.
func TestSizingPresets(t *testing.T) {
	// SizingAutoFitDefault
	_, chars, autoFit := SizingAutoFitDefault.GetConfig()
	assert.True(t, autoFit, "SizingAutoFitDefault should be auto-fit")
	assert.Equal(t, 15, chars, "SizingAutoFitDefault should have 15 min chars")

	// SizingCompact
	lines, chars, autoFit := SizingCompact.GetConfig()
	assert.False(t, autoFit, "SizingCompact should not be auto-fit")
	assert.Equal(t, 2, lines, "SizingCompact should have 2 lines")
	assert.Equal(t, 35, chars, "SizingCompact should have 35 chars")
}

// TestPositionPresets verifies preset position values.
func TestPositionPresets(t *testing.T) {
	assert.Equal(t, HorizontalMapCenter, PositionMapCenterLower.Horizontal)
	assert.Equal(t, VerticalLower, PositionMapCenterLower.Vertical)

	assert.Equal(t, HorizontalScreenCenter, PositionScreenCenter.Horizontal)
	assert.Equal(t, VerticalCenter, PositionScreenCenter.Vertical)
}

// TestEffectsPresets verifies preset effects values.
func TestEffectsPresets(t *testing.T) {
	assert.Equal(t, 0, EffectsNone.FadeInMs)
	assert.Equal(t, 0, EffectsNone.FadeOutMs)
}

// =====================================
// INTEGRATION TESTS - Dialog Creation
// =====================================

func TestOverlayDialog_Build_PanicsIfSizingMissing(t *testing.T) {
	defer func() {
		r := recover()
		require.NotNil(t, r, "Build() should panic when sizing is missing")
		panicMsg := r.(string)
		assert.Contains(t, panicMsg, "Sizing", "Panic message should mention Sizing")
	}()

	NewOverlayDialog("Test").
		WithPosition(PositionMapCenterLower).
		WithEffects(EffectsNone).
		WithContentAlignment(ContentAlignCenter).
		WithTextAlignment(TextAlignCenter).
		WithExternallyManagedDuration(1000).
		Build()
}

func TestOverlayDialog_Build_PanicsIfPositionMissing(t *testing.T) {
	defer func() {
		r := recover()
		require.NotNil(t, r, "Build() should panic when position is missing")
		panicMsg := r.(string)
		assert.Contains(t, panicMsg, "Position", "Panic message should mention Position")
	}()

	NewOverlayDialog("Test").
		WithSizing(SizingAutoFitDefault).
		WithEffects(EffectsNone).
		WithContentAlignment(ContentAlignCenter).
		WithTextAlignment(TextAlignCenter).
		WithExternallyManagedDuration(1000).
		Build()
}

func TestOverlayDialog_Build_PanicsIfEffectsMissing(t *testing.T) {
	defer func() {
		r := recover()
		require.NotNil(t, r, "Build() should panic when effects are missing")
		panicMsg := r.(string)
		assert.Contains(t, panicMsg, "Effects", "Panic message should mention Effects")
	}()

	NewOverlayDialog("Test").
		WithSizing(SizingAutoFitDefault).
		WithPosition(PositionMapCenterLower).
		WithContentAlignment(ContentAlignCenter).
		WithTextAlignment(TextAlignCenter).
		WithExternallyManagedDuration(1000).
		Build()
}

func TestOverlayDialog_Build_PanicsIfContentAlignmentMissing(t *testing.T) {
	defer func() {
		r := recover()
		require.NotNil(t, r, "Build() should panic when content alignment is missing")
		panicMsg := r.(string)
		assert.Contains(t, panicMsg, "ContentAlignment", "Panic message should mention ContentAlignment")
	}()

	NewOverlayDialog("Test").
		WithSizing(SizingAutoFitDefault).
		WithPosition(PositionMapCenterLower).
		WithEffects(EffectsNone).
		WithTextAlignment(TextAlignCenter).
		WithExternallyManagedDuration(1000).
		Build()
}

func TestOverlayDialog_Build_PanicsIfTextAlignmentMissing(t *testing.T) {
	defer func() {
		r := recover()
		require.NotNil(t, r, "Build() should panic when text alignment is missing")
		panicMsg := r.(string)
		assert.Contains(t, panicMsg, "TextAlignment", "Panic message should mention TextAlignment")
	}()

	NewOverlayDialog("Test").
		WithSizing(SizingAutoFitDefault).
		WithPosition(PositionMapCenterLower).
		WithEffects(EffectsNone).
		WithContentAlignment(ContentAlignCenter).
		WithExternallyManagedDuration(1000).
		Build()
}

func TestOverlayDialog_Build_PanicsIfBothDurationMethodsCalled(t *testing.T) {
	keyboard := newWidgetTestKeyboard(100)

	defer func() {
		r := recover()
		require.NotNil(t, r, "Build() should panic when both duration methods are called")
		panicMsg := r.(string)
		assert.Contains(t, panicMsg, "cannot combine duration modes", "Panic message should explain the conflict")
	}()

	NewOverlayDialog("Test").
		WithSizing(SizingAutoFitDefault).
		WithPosition(PositionMapCenterLower).
		WithEffects(EffectsNone).
		WithContentAlignment(ContentAlignCenter).
		WithTextAlignment(TextAlignCenter).
		WithExternallyManagedDuration(1000).
		WithUserDismissed(keyboard).
		Build()
}

func TestOverlayDialog_ShowHide_WorksCorrectly(t *testing.T) {
	setResponsiveDialogTestResolution(t, 1920, 1080)

	dialog := NewOverlayDialog("Test").
		AsOverlayFeedback().
		WithExternallyManagedDuration(1000).
		Build()

	assert.False(t, dialog.IsVisible(), "Dialog should start invisible")

	dialog.ShowWithoutAnimation()
	assert.True(t, dialog.IsVisible(), "Dialog should be visible after Show")

	dialog.HideWithoutAnimation()
	assert.False(t, dialog.IsVisible(), "Dialog should be invisible after Hide")
}

// The appearance templates must supply every required setting except the duration.
func TestOverlayDialog_TemplatesSupplyAllAppearanceConfig(t *testing.T) {
	for name, template := range map[string]func(*OverlayDialogBuilder) *OverlayDialogBuilder{
		"AsConfirmation":    (*OverlayDialogBuilder).AsConfirmation,
		"AsOverlayFeedback": (*OverlayDialogBuilder).AsOverlayFeedback,
	} {
		t.Run(name, func(t *testing.T) {
			assert.NotPanics(t, func() {
				template(NewOverlayDialog("Test")).WithExternallyManagedDuration(1000).Build()
			})
			assert.PanicsWithValue(t, "OverlayDialog.Build(): missing required configuration:\n  - "+
				"Duration (call WithExternallyManagedDuration(ms) or WithUserDismissed(keyboard))", func() {
				template(NewOverlayDialog("Test")).Build()
			})
		})
	}
}

func TestTimedConfirmationRemainsFooterless(t *testing.T) {
	setResponsiveDialogTestResolution(t, 1280, 720)
	for _, builder := range []*OverlayDialogBuilder{
		NewOverlayDialog("Scene notice").AsConfirmation().WithExternallyManagedDuration(1000),
		NewOverlayDialog("Brief notice").AsOverlayFeedback().WithExternallyManagedDuration(1000),
		NewOverlayDialog("Custom message").WithSizing(FixedLinesSizing{Lines: 4, CharsPerLine: 50}).
			WithPosition(PositionScreenCenter).WithEffects(EffectsConfig{FadeInMs: 500, FadeOutMs: 200}).
			WithContentAlignment(ContentAlignTop).WithTextAlignment(TextAlignLeft).WithExternallyManagedDuration(2000),
	} {
		dialog := builder.Build()
		require.True(t, dialog.IsTimed())
		require.False(t, dialog.IsUserDismissed())
		require.False(t, dialog.IsDismissed(), "the owner, not the dialog, ends a timed overlay")
		require.False(t, dialog.dialog.hasPromptOutput())
	}
}
