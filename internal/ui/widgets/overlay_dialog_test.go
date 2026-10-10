package widgets

import (
	"image"
	"math"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =====================================
// VALIDATION TESTS - Required Config
// =====================================

// TestOverlayDialog_Build_PanicsIfSizingMissing verifies that Build() panics
// when sizing is not configured.
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
		WithDuration(1000).
		Build()
}

// TestOverlayDialog_Build_PanicsIfPositionMissing verifies that Build() panics
// when position is not configured.
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
		WithDuration(1000).
		Build()
}

// TestOverlayDialog_Build_PanicsIfEffectsMissing verifies that Build() panics
// when effects are not configured.
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
		WithDuration(1000).
		Build()
}

// TestOverlayDialog_Build_PanicsIfContentAlignmentMissing verifies that Build() panics
// when content alignment is not configured.
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
		WithDuration(1000).
		Build()
}

// TestOverlayDialog_Build_PanicsIfTextAlignmentMissing verifies that Build() panics
// when text alignment is not configured.
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
		WithDuration(1000).
		Build()
}

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

// TestOverlayDialog_Build_PanicsIfBothDurationMethodsCalled verifies that Build() panics
// when both WithDuration() and WithUserDismissed() are called.
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
		WithDuration(1000).
		WithUserDismissed(keyboard).
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

// TestOverlayDialogBuilder_TracksConfiguredState verifies that the builder
// correctly tracks which configuration has been set.
func TestOverlayDialogBuilder_TracksConfiguredState(t *testing.T) {
	keyboard := newWidgetTestKeyboard(100)

	// Fresh builder should have nothing set
	b := NewOverlayDialog("Test")
	assert.False(t, b.IsSizingSet(), "Fresh builder should not have sizing set")
	assert.False(t, b.IsPositionSet(), "Fresh builder should not have position set")
	assert.False(t, b.IsEffectsSet(), "Fresh builder should not have effects set")
	assert.False(t, b.IsContentAlignmentSet(), "Fresh builder should not have content alignment set")
	assert.False(t, b.IsTextAlignmentSet(), "Fresh builder should not have text alignment set")
	assert.False(t, b.IsDurationSet(), "Fresh builder should not have duration set")
	assert.False(t, b.IsExternalDurationSet(), "Fresh builder should not have external duration set")
	assert.False(t, b.IsUserDismissedSet(), "Fresh builder should not have user dismissed set")

	// Set each and verify
	b.WithSizing(SizingAutoFitDefault)
	assert.True(t, b.IsSizingSet(), "Sizing should be set after WithSizing()")

	b.WithPosition(PositionMapCenterLower)
	assert.True(t, b.IsPositionSet(), "Position should be set after WithPosition()")

	b.WithEffects(EffectsNone)
	assert.True(t, b.IsEffectsSet(), "Effects should be set after WithEffects()")

	b.WithContentAlignment(ContentAlignCenter)
	assert.True(t, b.IsContentAlignmentSet(), "Content alignment should be set after WithContentAlignment()")

	b.WithTextAlignment(TextAlignCenter)
	assert.True(t, b.IsTextAlignmentSet(), "Text alignment should be set after WithTextAlignment()")

	b.WithDuration(1000)
	assert.True(t, b.IsDurationSet(), "Duration should be set after WithDuration()")
	assert.False(t, b.IsExternalDurationSet(), "External duration should not be set after WithDuration()")
	assert.False(t, b.IsUserDismissedSet(), "User dismissed should not be set after WithDuration()")

	// Test user dismissed separately
	b2 := NewOverlayDialog("Test2").WithUserDismissed(keyboard)
	assert.True(t, b2.IsUserDismissedSet(), "User dismissed should be set after WithUserDismissed()")
	assert.False(t, b2.IsDurationSet(), "Duration should not be set after WithUserDismissed()")
	assert.False(t, b2.IsExternalDurationSet(), "External duration should not be set after WithUserDismissed()")
}

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

func TestOverlayDialog_SelfManagedDurationStillAdvancesInUpdate(t *testing.T) {
	dialog := NewOverlayDialog("Widget owned").
		AsOverlayFeedback().
		WithDuration(32).
		Build()
	assert.Equal(t, "Widget owned", dialog.GetMessage())
	assert.Equal(t, 32, dialog.GetDurationMs())
	assert.True(t, dialog.IsTimed())
	assert.False(t, dialog.IsUserDismissed())
	assert.False(t, dialog.IsVisible())
	assert.False(t, dialog.IsDismissed())
	dialog.ShowWithoutAnimation()

	dialog.Update()
	assert.True(t, dialog.IsVisible())
	elapsed, fade, dismissed := dialog.elapsedMs, dialog.dialog.fadeState, dialog.IsDismissed()
	for range 3 {
		dialog.PrepareLayout(image.Rect(0, 0, 1280, 720))
	}
	assert.Equal(t, elapsed, dialog.elapsedMs, "preparation does not spend timed lifetime")
	assert.Equal(t, fade, dialog.dialog.fadeState)
	assert.Equal(t, dismissed, dialog.IsDismissed())
	dialog.Update()

	assert.False(t, dialog.IsDurationExternallyManaged())
	assert.True(t, dialog.IsDismissed())
	assert.False(t, dialog.IsVisible())
}

func TestOverlayDialog_BuildRejectsBothTimedDurationModes(t *testing.T) {
	assert.PanicsWithValue(t,
		"OverlayDialog.Build(): cannot combine duration modes - choose exactly one",
		func() {
			NewOverlayDialog("Test").
				AsOverlayFeedback().
				WithDuration(1000).
				WithExternallyManagedDuration(1000).
				Build()
		})
}

// =====================================
// TEMPLATE TESTS
// =====================================

// TestOverlayDialog_AsOverlayFeedback_SetsAllAppearanceConfig verifies that
// the AsOverlayFeedback template sets all required appearance configuration.
func TestOverlayDialog_AsOverlayFeedback_SetsAllAppearanceConfig(t *testing.T) {
	b := NewOverlayDialog("Test").AsOverlayFeedback()

	assert.True(t, b.IsSizingSet(), "AsOverlayFeedback should set sizing")
	assert.True(t, b.IsPositionSet(), "AsOverlayFeedback should set position")
	assert.True(t, b.IsEffectsSet(), "AsOverlayFeedback should set effects")
	assert.True(t, b.IsContentAlignmentSet(), "AsOverlayFeedback should set content alignment")
	assert.True(t, b.IsTextAlignmentSet(), "AsOverlayFeedback should set text alignment")

	// Duration should NOT be set by template
	assert.False(t, b.IsDurationSet(), "AsOverlayFeedback should NOT set duration")
	assert.False(t, b.IsUserDismissedSet(), "AsOverlayFeedback should NOT set user dismissed")
}

// TestOverlayDialog_AsConfirmation_SetsAllAppearanceConfig verifies that
// the AsConfirmation template sets all required appearance configuration.
func TestOverlayDialog_AsConfirmation_SetsAllAppearanceConfig(t *testing.T) {
	b := NewOverlayDialog("Test").AsConfirmation()

	assert.True(t, b.IsSizingSet(), "AsConfirmation should set sizing")
	assert.True(t, b.IsPositionSet(), "AsConfirmation should set position")
	assert.True(t, b.IsEffectsSet(), "AsConfirmation should set effects")
	assert.True(t, b.IsContentAlignmentSet(), "AsConfirmation should set content alignment")
	assert.True(t, b.IsTextAlignmentSet(), "AsConfirmation should set text alignment")

	// Duration should NOT be set by template
	assert.False(t, b.IsDurationSet(), "AsConfirmation should NOT set duration")
	assert.False(t, b.IsUserDismissedSet(), "AsConfirmation should NOT set user dismissed")
}

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

func TestTimedConfirmationRemainsFooterless(t *testing.T) {
	setResponsiveDialogTestResolution(t, 1280, 720)
	for _, builder := range []*OverlayDialogBuilder{
		NewOverlayDialog("Brief notice").AsConfirmation().WithDuration(1000),
		NewOverlayDialog("Brief notice").AsOverlayFeedback().WithDuration(1000),
		NewOverlayDialog("Scene notice").AsConfirmation().WithExternallyManagedDuration(1000),
		NewOverlayDialog("Brief notice").AsBriefFeedback(),
		NewOverlayDialog("Custom message").WithSizing(FixedLinesSizing{Lines: 4, CharsPerLine: 50}).
			WithPosition(PositionScreenCenter).WithEffects(EffectsConfig{FadeInMs: 500, FadeOutMs: 200}).
			WithContentAlignment(ContentAlignTop).WithTextAlignment(TextAlignLeft).WithDuration(2000),
	} {
		dialog := builder.Build()
		require.True(t, dialog.IsTimed())
		require.False(t, dialog.IsUserDismissed())
		require.False(t, dialog.dialog.hasPromptOutput())
	}
}

// TestOverlayDialog_TemplateCanBeOverridden verifies that template settings
// can be overridden by subsequent builder calls.
func TestOverlayDialog_TemplateCanBeOverridden(t *testing.T) {
	b := NewOverlayDialog("Test").
		AsOverlayFeedback().                // Sets MapCenterLower
		WithPosition(PositionScreenCenter). // Override to ScreenCenter
		WithEffects(EffectsSlow)            // Override effects

	assert.Equal(t, PositionScreenCenter, b.GetPosition(), "Position should be overridden")
	assert.Equal(t, EffectsSlow, b.GetEffects(), "Effects should be overridden")
}

func TestOverlayDialogHorizontalCenterOverridePreservesVerticalPositionAndResize(t *testing.T) {
	setResponsiveDialogTestResolution(t, 1280, 720)
	builder := NewOverlayDialog("Cured!").AsConfirmation().
		WithHorizontalCenter(0.24).WithUserDismissed(newWidgetTestKeyboard(0))
	dialog := builder.Build()
	dialog.ShowWithoutAnimation()
	for _, width := range []int{1280, 1920} {
		height := width * 9 / 16
		setResponsiveDialogTestResolution(t, width, height)
		dialog.PrepareLayout(image.Rect(0, 0, width, height))
		bounds := dialog.dialog.frameBounds
		assert.InDelta(t, 0.24*float64(width), float64(bounds.Min.X+bounds.Max.X)/2, 1)
		assert.InDelta(t, 0.70*float64(height), float64(bounds.Min.Y+bounds.Max.Y)/2, 1)
	}
	reset := builder.WithPosition(PositionScreenCenter).Build()
	assert.InDelta(t, 0.5, (reset.dialog.config.StartPercentX+reset.dialog.config.EndPercentX)/2, 0.0001)
	assert.InDelta(t, 0.5, (reset.dialog.config.StartPercentY+reset.dialog.config.EndPercentY)/2, 0.0001)
}

func TestOverlayDialogHorizontalCenterRejectsInvalidFractions(t *testing.T) {
	for _, x := range []float64{-0.1, 1.1, math.NaN(), math.Inf(1)} {
		assert.Panics(t, func() { NewOverlayDialog("Cured!").AsConfirmation().WithHorizontalCenter(x) })
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
	assert.True(t, EffectsQuick.HasFadeIn(), "EffectsQuick should have fade-in")
	assert.True(t, EffectsSlow.HasFadeIn(), "EffectsSlow should have fade-in")
}

// TestEffectsConfig_HasFadeOut verifies fade-out detection.
func TestEffectsConfig_HasFadeOut(t *testing.T) {
	assert.False(t, EffectsNone.HasFadeOut(), "EffectsNone should not have fade-out")
	assert.True(t, EffectsQuick.HasFadeOut(), "EffectsQuick should have fade-out")
	assert.True(t, EffectsSlow.HasFadeOut(), "EffectsSlow should have fade-out")
}

// TestEffectsConfig_HasAnyEffect verifies any-effect detection.
func TestEffectsConfig_HasAnyEffect(t *testing.T) {
	assert.False(t, EffectsNone.HasAnyEffect(), "EffectsNone should not have any effect")
	assert.True(t, EffectsQuick.HasAnyEffect(), "EffectsQuick should have effects")
	assert.True(t, EffectsSlow.HasAnyEffect(), "EffectsSlow should have effects")

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

	// SizingMedium
	lines, chars, autoFit = SizingMedium.GetConfig()
	assert.False(t, autoFit, "SizingMedium should not be auto-fit")
	assert.Equal(t, 6, lines, "SizingMedium should have 6 lines")
	assert.Equal(t, 40, chars, "SizingMedium should have 40 chars")

	// SizingLarge
	lines, chars, autoFit = SizingLarge.GetConfig()
	assert.False(t, autoFit, "SizingLarge should not be auto-fit")
	assert.Equal(t, 10, lines, "SizingLarge should have 10 lines")
	assert.Equal(t, 50, chars, "SizingLarge should have 50 chars")
}

// TestPositionPresets verifies preset position values.
func TestPositionPresets(t *testing.T) {
	assert.Equal(t, HorizontalMapCenter, PositionMapCenterLower.Horizontal)
	assert.Equal(t, VerticalLower, PositionMapCenterLower.Vertical)

	assert.Equal(t, HorizontalScreenCenter, PositionScreenCenter.Horizontal)
	assert.Equal(t, VerticalCenter, PositionScreenCenter.Vertical)

	assert.Equal(t, HorizontalMapCenter, PositionMapCenterTop.Horizontal)
	assert.Equal(t, VerticalTop, PositionMapCenterTop.Vertical)
}

// TestEffectsPresets verifies preset effects values.
func TestEffectsPresets(t *testing.T) {
	assert.Equal(t, 0, EffectsNone.FadeInMs)
	assert.Equal(t, 0, EffectsNone.FadeOutMs)

	assert.Equal(t, 300, EffectsQuick.FadeInMs)
	assert.Equal(t, 300, EffectsQuick.FadeOutMs)

	assert.Equal(t, 500, EffectsStandard.FadeInMs)
	assert.Equal(t, 500, EffectsStandard.FadeOutMs)

	assert.Equal(t, 3000, EffectsSlow.FadeInMs)
	assert.Equal(t, 1000, EffectsSlow.FadeOutMs)
}

// =====================================
// INTEGRATION TESTS - Dialog Creation
// =====================================

// TestOverlayDialog_ShowHide_WorksCorrectly verifies show/hide functionality.
func TestOverlayDialog_ShowHide_WorksCorrectly(t *testing.T) {
	setResponsiveDialogTestResolution(t, 1920, 1080)

	dialog := NewOverlayDialog("Test").
		AsOverlayFeedback().
		WithDuration(1000).
		Build()

	assert.False(t, dialog.IsVisible(), "Dialog should start invisible")

	dialog.ShowWithoutAnimation()
	assert.True(t, dialog.IsVisible(), "Dialog should be visible after Show")

	dialog.HideWithoutAnimation()
	assert.False(t, dialog.IsVisible(), "Dialog should be invisible after Hide")
}
