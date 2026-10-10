package widgets

import (
	"testing"

	"github.com/bradhannah/UltimaVRedux/internal/sprites"
	"github.com/bradhannah/UltimaVRedux/pkg/input"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSlider_Creation tests that Slider can be created with proper configuration
func TestSlider_Creation(t *testing.T) {
	keyboard := newWidgetTestKeyboard(100)
	placement := sprites.PercentBasedPlacement{
		StartPercentX: 0.2,
		EndPercentX:   0.8,
		StartPercentY: 0.1,
		EndPercentY:   0.2,
	}

	config := SliderConfig{
		Placement:    placement,
		MinValue:     1,
		MaxValue:     10,
		InitialValue: 5,
		Label:        "Master Volume",
		Keyboard:     keyboard,
		OnValueChanged: func(oldValue, newValue int) {
			// Test callback
		},
	}

	slider := NewSlider(config)

	require.NotNil(t, slider, "Slider should be created successfully")
	assert.Equal(t, 5, slider.GetValue(), "Initial value should be set correctly")
	assert.Equal(t, "Master Volume", slider.GetLabel(), "Label should be set correctly")
	assert.Equal(t, 1, slider.GetMinValue(), "Min value should be set correctly")
	assert.Equal(t, 10, slider.GetMaxValue(), "Max value should be set correctly")
}

// TestSlider_KeyboardNavigation tests keyboard navigation for slider value changes
func TestSlider_KeyboardNavigation(t *testing.T) {
	keyboard := newWidgetTestKeyboard(100)
	placement := sprites.PercentBasedPlacement{
		StartPercentX: 0.2,
		EndPercentX:   0.8,
		StartPercentY: 0.1,
		EndPercentY:   0.2,
	}

	valueChanges := []struct {
		oldValue int
		newValue int
	}{}

	config := SliderConfig{
		Placement:    placement,
		MinValue:     1,
		MaxValue:     10,
		InitialValue: 5,
		Label:        "Test Slider",
		Keyboard:     keyboard,
		OnValueChanged: func(oldValue, newValue int) {
			valueChanges = append(valueChanges, struct {
				oldValue int
				newValue int
			}{oldValue, newValue})
		},
	}

	slider := NewSlider(config)

	keyboard.SimulateKeyPress(ebiten.KeyRight)
	slider.Update()
	assert.Equal(t, 6, slider.GetValue(), "Right should increase by one")
	require.Len(t, valueChanges, 1)
	assert.Equal(t, 5, valueChanges[0].oldValue)
	assert.Equal(t, 6, valueChanges[0].newValue)

	slider.Update()
	require.Len(t, valueChanges, 1, "one key press must dispatch once")

	keyboard.SimulateKeyPress(ebiten.KeyLeft)
	slider.Update()
	assert.Equal(t, 5, slider.GetValue(), "Left should decrease by one")
	require.Len(t, valueChanges, 2)
	assert.Equal(t, 6, valueChanges[1].oldValue)
	assert.Equal(t, 5, valueChanges[1].newValue)

	slider.SetValue(slider.GetMinValue())
	changesAtMinimum := len(valueChanges)
	keyboard.SimulateKeyPress(ebiten.KeyLeft)
	slider.Update()
	assert.Equal(t, slider.GetMinValue(), slider.GetValue())
	assert.Len(t, valueChanges, changesAtMinimum, "bounded input must not invoke the callback")

	slider.SetValue(slider.GetMaxValue())
	changesAtMaximum := len(valueChanges)
	keyboard.SimulateKeyPress(ebiten.KeyRight)
	slider.Update()
	assert.Equal(t, slider.GetMaxValue(), slider.GetValue())
	assert.Len(t, valueChanges, changesAtMaximum, "bounded input must not invoke the callback")
}

// TestSlider_SetValueProgrammatically tests setting value programmatically
func TestSlider_SetValueProgrammatically(t *testing.T) {
	keyboard := newWidgetTestKeyboard(100)
	placement := sprites.PercentBasedPlacement{
		StartPercentX: 0.2,
		EndPercentX:   0.8,
		StartPercentY: 0.1,
		EndPercentY:   0.2,
	}

	callbackCalled := false
	config := SliderConfig{
		Placement:    placement,
		MinValue:     1,
		MaxValue:     10,
		InitialValue: 5,
		Label:        "Test Slider",
		Keyboard:     keyboard,
		OnValueChanged: func(oldValue, newValue int) {
			callbackCalled = true
		},
	}

	slider := NewSlider(config)

	// Test setting valid value
	slider.SetValue(7)
	assert.Equal(t, 7, slider.GetValue(), "Value should be updated programmatically")
	assert.True(t, callbackCalled, "Callback should be called when value changes")

	// Test setting value out of bounds (should clamp)
	slider.SetValue(15)
	assert.Equal(t, 10, slider.GetValue(), "Value should be clamped to maximum")

	slider.SetValue(-5)
	assert.Equal(t, 1, slider.GetValue(), "Value should be clamped to minimum")
}

// TestSlider_FocusAndVisibility tests focus and visibility management
func TestSlider_FocusAndVisibility(t *testing.T) {
	pressed := map[ebiten.Key]bool{ebiten.KeyRight: true}
	keyboard := input.NewKeyboardWithKeyStateSource(0, func(key ebiten.Key) bool { return pressed[key] })
	placement := sprites.PercentBasedPlacement{
		StartPercentX: 0.2,
		EndPercentX:   0.8,
		StartPercentY: 0.1,
		EndPercentY:   0.2,
	}

	config := SliderConfig{
		Placement:    placement,
		MinValue:     1,
		MaxValue:     10,
		InitialValue: 5,
		Label:        "Test Slider",
		Keyboard:     keyboard,
	}

	slider := NewSlider(config)

	// Test initial state
	assert.True(t, slider.IsVisible(), "Slider should be visible by default")
	assert.True(t, slider.HasFocus(), "Slider should have focus by default")

	slider.SetFocus(false)
	assert.False(t, slider.HasFocus(), "Slider focus should be updated")
	slider.Update()
	assert.Equal(t, 5, slider.GetValue(), "unfocused slider must ignore keyboard input")

	slider.SetFocus(true)
	slider.SetVisible(false)
	assert.False(t, slider.IsVisible(), "Slider visibility should be updated")
	slider.Update()
	assert.Equal(t, 5, slider.GetValue(), "hidden slider must ignore keyboard input")
}
