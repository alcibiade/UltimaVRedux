package widgets

import (
	"bytes"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/display"
	"github.com/bradhannah/UltimaVRedux/internal/sprites"
	ultimacolor "github.com/bradhannah/UltimaVRedux/pkg/color"
)

// Test new button states and enhancements
func TestButtonStateEnhancement(t *testing.T) {
	centerPoint := sprites.PercentBasedCenterPoint{X: 0.5, Y: 0.5}

	t.Run("creates button with default enabled state", func(t *testing.T) {
		button := NewButton("Test", func() {}, centerPoint, MediumButton)

		require.NotNil(t, button)
		assert.True(t, button.IsEnabled())
		assert.Equal(t, ButtonNormal, button.GetButtonStatus())
		assert.Equal(t, "Test", button.GetText())
	})

	t.Run("handles enabled/disabled state changes", func(t *testing.T) {
		button := NewButton("Test", nil, centerPoint, MediumButton)

		// Start enabled
		assert.True(t, button.IsEnabled())
		assert.Equal(t, ButtonNormal, button.GetButtonStatus())

		// Disable button
		button.SetEnabled(false)
		assert.False(t, button.IsEnabled())
		assert.Equal(t, ButtonDisabled, button.GetButtonStatus())

		// Re-enable button
		button.SetEnabled(true)
		assert.True(t, button.IsEnabled())
		assert.Equal(t, ButtonNormal, button.GetButtonStatus())
	})

	t.Run("handles button status changes", func(t *testing.T) {
		button := NewButton("Test", nil, centerPoint, MediumButton)

		// Test different status changes
		button.SetButtonStatus(ButtonSelected)
		assert.Equal(t, ButtonSelected, button.GetButtonStatus())
		assert.True(t, button.IsEnabled())

		button.SetButtonStatus(ButtonDisabled)
		assert.Equal(t, ButtonDisabled, button.GetButtonStatus())
		assert.False(t, button.IsEnabled())

		button.SetButtonStatus(ButtonNormal)
		assert.Equal(t, ButtonNormal, button.GetButtonStatus())
		assert.True(t, button.IsEnabled())
	})

	t.Run("updates text correctly", func(t *testing.T) {
		button := NewButton("Original", nil, centerPoint, MediumButton)

		assert.Equal(t, "Original", button.GetText())

		button.SetText("Updated")
		assert.Equal(t, "Updated", button.GetText())
	})

	t.Run("updates callback correctly", func(t *testing.T) {
		called1 := false
		called2 := false

		button := NewButton("Test", func() { called1 = true }, centerPoint, MediumButton)

		// Set new callback
		button.SetCallback(func() { called2 = true })

		// Updating callback storage does not invoke either callback.
		assert.False(t, called1)
		assert.False(t, called2)
	})
}

func TestButtonVisualStates(t *testing.T) {
	centerPoint := sprites.PercentBasedCenterPoint{X: 0.5, Y: 0.5}

	t.Run("has correct colors for each state", func(t *testing.T) {
		button := NewButton("Test", nil, centerPoint, MediumButton)

		// Test normal state colors
		button.SetButtonStatus(ButtonNormal)
		button.updateVisualState()
		assert.Equal(t, ultimacolor.LighterBlackSemi, button.interiorColor)
		assert.Equal(t, ultimacolor.White, button.textColor)

		// Test selected state colors
		button.SetButtonStatus(ButtonSelected)
		assert.Equal(t, ultimacolor.UltimaBlue, button.interiorColor)
		assert.Equal(t, ultimacolor.White, button.textColor)

		// Test disabled state colors
		button.SetButtonStatus(ButtonDisabled)
		assert.Equal(t, ultimacolor.LighterBlackSemi, button.interiorColor)
		assert.Equal(t, ultimacolor.ButtonTextDisabledGray.RGBA(), button.textColor)
	})
}

func TestButtonFocusableInterface(t *testing.T) {
	centerPoint := sprites.PercentBasedCenterPoint{X: 0.5, Y: 0.5}

	t.Run("implements FocusableItem interface", func(t *testing.T) {
		button := NewButton("Test", nil, centerPoint, MediumButton)

		// Test GetPlacement
		placement := button.GetPlacement()
		assert.Equal(t, button.pbp, placement)

		// Test CanReceiveFocus when enabled
		assert.True(t, button.CanReceiveFocus())

		// Test CanReceiveFocus when disabled
		button.SetEnabled(false)
		assert.False(t, button.CanReceiveFocus())

		// Re-enable for focus testing
		button.SetEnabled(true)
		assert.True(t, button.CanReceiveFocus())
	})

	t.Run("handles focus gained correctly", func(t *testing.T) {
		button := NewButton("Test", nil, centerPoint, MediumButton)

		// Start in normal state
		assert.Equal(t, ButtonNormal, button.GetButtonStatus())

		// Gain focus
		button.OnFocusGained()
		assert.Equal(t, ButtonSelected, button.GetButtonStatus())

		// Focus gained restores the selected visual state.
		button.buttonStatus = ButtonNormal
		button.OnFocusGained()
		assert.Equal(t, ButtonSelected, button.GetButtonStatus())

		button.SetEnabled(false)
		button.OnFocusGained()
		assert.Equal(t, ButtonDisabled, button.GetButtonStatus())
	})

	t.Run("handles focus lost correctly", func(t *testing.T) {
		button := NewButton("Test", nil, centerPoint, MediumButton)

		// Set to selected state (as if focused)
		button.SetButtonStatus(ButtonSelected)
		assert.Equal(t, ButtonSelected, button.GetButtonStatus())

		// Lose focus
		button.OnFocusLost()
		assert.Equal(t, ButtonNormal, button.GetButtonStatus())
	})
}

func TestButtonUpdate(t *testing.T) {
	centerPoint := sprites.PercentBasedCenterPoint{X: 0.5, Y: 0.5}

	callbackCount := 0
	button := NewButton("Test", func() { callbackCount++ }, centerPoint, MediumButton)
	button.SetButtonStatus(ButtonSelected)

	button.Update()
	button.Update()

	assert.Equal(t, ButtonSelected, button.GetButtonStatus())
	assert.Equal(t, 0, callbackCount, "button owners are responsible for activation")
}

func TestButtonActivate(t *testing.T) {
	centerPoint := sprites.PercentBasedCenterPoint{X: 0.5, Y: 0.5}
	callbackCount := 0
	button := NewButton("Test", func() { callbackCount++ }, centerPoint, MediumButton)

	button.Activate()
	assert.Equal(t, 1, callbackCount)

	button.SetEnabled(false)
	button.Activate()
	assert.Equal(t, 1, callbackCount, "disabled buttons must not activate")
}

func TestButtonDisabledTextUsesDisabledColor(t *testing.T) {
	button := NewButton(
		"Disabled",
		nil,
		sprites.PercentBasedCenterPoint{X: 0.5, Y: 0.5},
		MediumButton,
	)
	width, height := display.GetManager().GetCurrentSize()
	normalScreen := ebiten.NewImage(width, height)
	button.Draw(normalScreen)
	normalPixels := make([]byte, width*height*4)
	normalScreen.ReadPixels(normalPixels)

	button.SetEnabled(false)
	disabledScreen := ebiten.NewImage(width, height)
	button.Draw(disabledScreen)
	disabledPixels := make([]byte, width*height*4)
	disabledScreen.ReadPixels(disabledPixels)

	assert.False(t, bytes.Equal(normalPixels, disabledPixels), "disabled text should use a different rendered color")
}

func TestButtonLegacyCompatibility(t *testing.T) {
	centerPoint := sprites.PercentBasedCenterPoint{X: 0.5, Y: 0.5}

	t.Run("legacy constants work correctly", func(t *testing.T) {
		button := NewButton("Test", nil, centerPoint, MediumButton)

		// Test legacy constants map to new ones
		assert.Equal(t, ButtonNormal, NotSelected)
		assert.Equal(t, ButtonSelected, Selected)
		assert.Equal(t, ButtonDisabled, Disabled)

		// Test legacy constants work with SetButtonStatus
		button.SetButtonStatus(NotSelected)
		assert.Equal(t, ButtonNormal, button.GetButtonStatus())

		button.SetButtonStatus(Selected)
		assert.Equal(t, ButtonSelected, button.GetButtonStatus())

		button.SetButtonStatus(Disabled)
		assert.Equal(t, ButtonDisabled, button.GetButtonStatus())
		assert.False(t, button.IsEnabled())
	})
}

func TestButtonStateTransitions(t *testing.T) {
	centerPoint := sprites.PercentBasedCenterPoint{X: 0.5, Y: 0.5}

	t.Run("state transitions work correctly", func(t *testing.T) {
		button := NewButton("Test", nil, centerPoint, MediumButton)

		// Normal -> Selected
		button.SetButtonStatus(ButtonNormal)
		button.SetButtonStatus(ButtonSelected)
		assert.Equal(t, ButtonSelected, button.GetButtonStatus())
		assert.True(t, button.IsEnabled())

		// Selected -> Disabled
		button.SetButtonStatus(ButtonDisabled)
		assert.Equal(t, ButtonDisabled, button.GetButtonStatus())
		assert.False(t, button.IsEnabled())

		// Disabled -> Normal
		button.SetButtonStatus(ButtonNormal)
		assert.Equal(t, ButtonNormal, button.GetButtonStatus())
		assert.True(t, button.IsEnabled())
	})
}

func TestButtonEdgeCases(t *testing.T) {
	centerPoint := sprites.PercentBasedCenterPoint{X: 0.5, Y: 0.5}

	t.Run("handles nil callback gracefully", func(t *testing.T) {
		button := NewButton("Test", nil, centerPoint, MediumButton)

		// Should not panic when callback is nil
		button.Update()

		// Set callback to nil explicitly
		button.SetCallback(nil)
		button.Update()
	})

	t.Run("handles empty text", func(t *testing.T) {
		button := NewButton("", nil, centerPoint, MediumButton)

		assert.Equal(t, "", button.GetText())

	})

	t.Run("handles rapid state changes", func(t *testing.T) {
		button := NewButton("Test", nil, centerPoint, MediumButton)

		// Rapid state changes should not cause issues
		for i := 0; i < 100; i++ {
			button.SetButtonStatus(ButtonNormal)
			button.SetButtonStatus(ButtonSelected)
			button.SetButtonStatus(ButtonDisabled)
			button.SetEnabled(true)
			button.SetEnabled(false)
			button.SetEnabled(true)
		}

		// Should end up in a valid state
		assert.True(t, button.IsEnabled())
	})
}
