package widgets

import (
	"image/color"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/sprites"
)

// Mock focusable item for testing
type mockFocusableItem struct {
	placement   sprites.PercentBasedPlacement
	canFocus    bool
	hasFocus    bool
	focusGained int
	focusLost   int
}

func (m *mockFocusableItem) GetPlacement() sprites.PercentBasedPlacement {
	return m.placement
}

func (m *mockFocusableItem) CanReceiveFocus() bool {
	return m.canFocus
}

func (m *mockFocusableItem) OnFocusGained() {
	m.hasFocus = true
	m.focusGained++
}

func (m *mockFocusableItem) OnFocusLost() {
	m.hasFocus = false
	m.focusLost++
}

func TestNewFocusIndicator(t *testing.T) {
	t.Run("creates with valid config", func(t *testing.T) {
		config := FocusIndicatorConfig{
			Style:          FocusStyleBorder,
			PrimaryColor:   color.RGBA{R: 255, G: 0, B: 0, A: 255},
			AnimationSpeed: 3.0,
			BorderWidth:    2.5,
			GlowIntensity:  0.8,
			Padding:        1.5,
		}

		indicator := NewFocusIndicator(config)
		require.NotNil(t, indicator)
		assert.Equal(t, FocusStyleBorder, indicator.config.Style)
		assert.Equal(t, color.RGBA{R: 255, G: 0, B: 0, A: 255}, indicator.config.PrimaryColor)
		assert.Equal(t, 3.0, indicator.config.AnimationSpeed)
		assert.Equal(t, 2.5, indicator.config.BorderWidth)
		assert.Equal(t, 0.8, indicator.config.GlowIntensity)
		assert.Equal(t, 1.5, indicator.config.Padding)
		assert.False(t, indicator.isVisible)
	})

	t.Run("applies defaults for invalid values", func(t *testing.T) {
		config := FocusIndicatorConfig{
			AnimationSpeed: -1.0, // Invalid
			BorderWidth:    0,    // Invalid
			GlowIntensity:  2.0,  // Invalid (> 1.0)
			Padding:        -5.0, // Invalid
		}

		indicator := NewFocusIndicator(config)
		require.NotNil(t, indicator)
		assert.Equal(t, DefaultFocusConfig.AnimationSpeed, indicator.config.AnimationSpeed)
		assert.Equal(t, DefaultFocusConfig.BorderWidth, indicator.config.BorderWidth)
		assert.Equal(t, 1.0, indicator.config.GlowIntensity) // Clamped to max
		assert.Equal(t, 0.0, indicator.config.Padding)       // Clamped to min
	})

	t.Run("creates with default config", func(t *testing.T) {
		indicator := NewDefaultFocusIndicator()
		require.NotNil(t, indicator)
		assert.Equal(t, DefaultFocusConfig.Style, indicator.config.Style)
		assert.Equal(t, DefaultFocusConfig.AnimationSpeed, indicator.config.AnimationSpeed)
	})

	t.Run("creates accessibility variant", func(t *testing.T) {
		indicator := NewAccessibilityFocusIndicator()
		require.NotNil(t, indicator)
		assert.Equal(t, AccessibilityFocusConfig.Style, indicator.config.Style)
		assert.Equal(t, AccessibilityFocusConfig.PrimaryColor, indicator.config.PrimaryColor)
		assert.False(t, indicator.config.PulseEffect)
	})
}

func TestFocusIndicatorVisibility(t *testing.T) {
	indicator := NewDefaultFocusIndicator()

	t.Run("starts invisible", func(t *testing.T) {
		assert.False(t, indicator.IsVisible())
	})

	t.Run("show makes visible", func(t *testing.T) {
		indicator.Show()
		assert.True(t, indicator.IsVisible())
		assert.Equal(t, 0.0, indicator.animationTime)
	})

	t.Run("hide makes invisible", func(t *testing.T) {
		indicator.Show()
		indicator.Hide()
		assert.False(t, indicator.IsVisible())
	})
}

func TestFocusIndicatorTarget(t *testing.T) {
	indicator := NewDefaultFocusIndicator()

	t.Run("sets target with padding", func(t *testing.T) {
		target := sprites.PercentBasedPlacement{
			StartPercentX: 0.1,
			EndPercentX:   0.5,
			StartPercentY: 0.2,
			EndPercentY:   0.4,
		}

		indicator.SetFocusTarget(target)

		// Should expand by padding (2.0 pixels = 0.02 percent assuming 100 pixel screen)
		expectedPadding := indicator.config.Padding / 100.0
		assert.InDelta(t, 0.1-expectedPadding, indicator.placement.StartPercentX, 0.001)
		assert.InDelta(t, 0.5+expectedPadding, indicator.placement.EndPercentX, 0.001)
		assert.InDelta(t, 0.2-expectedPadding, indicator.placement.StartPercentY, 0.001)
		assert.InDelta(t, 0.4+expectedPadding, indicator.placement.EndPercentY, 0.001)
	})

	t.Run("clamps to screen bounds", func(t *testing.T) {
		target := sprites.PercentBasedPlacement{
			StartPercentX: -0.1, // Out of bounds
			EndPercentX:   1.1,  // Out of bounds
			StartPercentY: -0.05,
			EndPercentY:   1.05,
		}

		indicator.SetFocusTarget(target)

		// Should be clamped to [0, 1] range
		assert.GreaterOrEqual(t, indicator.placement.StartPercentX, 0.0)
		assert.LessOrEqual(t, indicator.placement.EndPercentX, 1.0)
		assert.GreaterOrEqual(t, indicator.placement.StartPercentY, 0.0)
		assert.LessOrEqual(t, indicator.placement.EndPercentY, 1.0)
	})
}

func TestFocusIndicatorUpdate(t *testing.T) {
	t.Run("updates animation time when visible and pulsing", func(t *testing.T) {
		config := FocusIndicatorConfig{
			AnimationSpeed: 2.0,
			PulseEffect:    true,
		}
		indicator := NewFocusIndicator(config)
		indicator.Show()

		indicator.Update(500) // 0.5 seconds
		assert.InDelta(t, 0.5, indicator.animationTime, 0.001)

		indicator.Update(1000) // 1.0 second more
		assert.InDelta(t, 1.5, indicator.animationTime, 0.001)
	})

	t.Run("does not update when invisible", func(t *testing.T) {
		config := FocusIndicatorConfig{
			PulseEffect: true,
		}
		indicator := NewFocusIndicator(config)
		// indicator starts invisible

		indicator.Update(1000)
		assert.Equal(t, 0.0, indicator.animationTime)
	})

	t.Run("does not update when pulse disabled", func(t *testing.T) {
		config := FocusIndicatorConfig{
			PulseEffect: false,
		}
		indicator := NewFocusIndicator(config)
		indicator.Show()

		indicator.Update(1000)
		assert.Equal(t, 0.0, indicator.animationTime) // Should remain 0
	})
}

func TestNewFocusManager(t *testing.T) {
	t.Run("creates with provided indicator", func(t *testing.T) {
		indicator := NewDefaultFocusIndicator()
		manager := NewFocusManager(indicator)

		require.NotNil(t, manager)
		assert.Same(t, indicator, manager.focusIndicator)
		assert.Equal(t, -1, manager.currentFocus)
		assert.True(t, manager.enabled)
		assert.Len(t, manager.focusableItems, 0)
	})

	t.Run("creates default indicator when nil provided", func(t *testing.T) {
		manager := NewFocusManager(nil)

		require.NotNil(t, manager)
		require.NotNil(t, manager.focusIndicator)
		assert.Equal(t, -1, manager.currentFocus)
		assert.True(t, manager.enabled)
	})
}

func TestFocusManagerItems(t *testing.T) {
	t.Run("adds focusable items", func(t *testing.T) {
		manager := NewFocusManager(nil)
		item1 := &mockFocusableItem{canFocus: true}
		item2 := &mockFocusableItem{canFocus: true}

		manager.AddFocusableItem(item1)
		manager.AddFocusableItem(item2)

		assert.Len(t, manager.focusableItems, 2)
		assert.Same(t, item1, manager.focusableItems[0])
		assert.Same(t, item2, manager.focusableItems[1])
	})

	t.Run("removes focusable items", func(t *testing.T) {
		manager := NewFocusManager(nil)
		item1 := &mockFocusableItem{canFocus: true}
		item2 := &mockFocusableItem{canFocus: true}
		item3 := &mockFocusableItem{canFocus: true}

		manager.AddFocusableItem(item1)
		manager.AddFocusableItem(item2)
		manager.AddFocusableItem(item3)

		manager.RemoveFocusableItem(item2)

		assert.Len(t, manager.focusableItems, 2)
		assert.Same(t, item1, manager.focusableItems[0])
		assert.Same(t, item3, manager.focusableItems[1])
	})

	t.Run("updates focus when removing current item", func(t *testing.T) {
		manager := NewFocusManager(nil)
		item1 := &mockFocusableItem{canFocus: true}
		item2 := &mockFocusableItem{canFocus: true}

		manager.AddFocusableItem(item1)
		manager.AddFocusableItem(item2)

		// Focus first item
		manager.FocusNext()
		assert.Equal(t, 0, manager.currentFocus)

		// Remove focused item
		manager.RemoveFocusableItem(item1)

		assert.Equal(t, -1, manager.currentFocus) // Focus cleared
		assert.False(t, manager.focusIndicator.IsVisible())
	})

	t.Run("adjusts focus index when removing earlier item", func(t *testing.T) {
		manager := NewFocusManager(nil)
		item1 := &mockFocusableItem{canFocus: true}
		item2 := &mockFocusableItem{canFocus: true}
		item3 := &mockFocusableItem{canFocus: true}

		manager.AddFocusableItem(item1)
		manager.AddFocusableItem(item2)
		manager.AddFocusableItem(item3)

		// Focus third item
		manager.FocusNext()
		manager.FocusNext()
		manager.FocusNext()
		assert.Equal(t, 2, manager.currentFocus)

		// Remove first item
		manager.RemoveFocusableItem(item1)

		assert.Equal(t, 1, manager.currentFocus) // Adjusted down by 1
	})
}

func TestFocusManagerNavigation(t *testing.T) {
	t.Run("focuses next item", func(t *testing.T) {
		manager := NewFocusManager(nil)
		item1 := &mockFocusableItem{
			placement: sprites.PercentBasedPlacement{StartPercentX: 0.1, EndPercentX: 0.2, StartPercentY: 0.1, EndPercentY: 0.2},
			canFocus:  true,
		}
		item2 := &mockFocusableItem{
			placement: sprites.PercentBasedPlacement{StartPercentX: 0.3, EndPercentX: 0.4, StartPercentY: 0.3, EndPercentY: 0.4},
			canFocus:  true,
		}

		manager.AddFocusableItem(item1)
		manager.AddFocusableItem(item2)

		manager.FocusNext()

		assert.Equal(t, 0, manager.currentFocus)
		assert.True(t, item1.hasFocus)
		assert.False(t, item2.hasFocus)
		assert.Equal(t, 1, item1.focusGained)
		assert.Equal(t, 0, item1.focusLost)
		assert.True(t, manager.focusIndicator.IsVisible())

		manager.FocusNext()

		assert.Equal(t, 1, manager.currentFocus)
		assert.False(t, item1.hasFocus)
		assert.True(t, item2.hasFocus)
		assert.Equal(t, 1, item1.focusLost)
		assert.Equal(t, 1, item2.focusGained)
	})

	t.Run("wraps around to first item", func(t *testing.T) {
		manager := NewFocusManager(nil)
		item1 := &mockFocusableItem{
			placement: sprites.PercentBasedPlacement{StartPercentX: 0.1, EndPercentX: 0.2, StartPercentY: 0.1, EndPercentY: 0.2},
			canFocus:  true,
		}
		item2 := &mockFocusableItem{
			placement: sprites.PercentBasedPlacement{StartPercentX: 0.3, EndPercentX: 0.4, StartPercentY: 0.3, EndPercentY: 0.4},
			canFocus:  true,
		}

		manager.AddFocusableItem(item1)
		manager.AddFocusableItem(item2)

		// Focus to second item
		manager.FocusNext()
		manager.FocusNext()
		assert.Equal(t, 1, manager.currentFocus)

		// Should wrap to first
		manager.FocusNext()
		assert.Equal(t, 0, manager.currentFocus)
		assert.True(t, item1.hasFocus)
		assert.False(t, item2.hasFocus)
	})

	t.Run("focuses previous item", func(t *testing.T) {
		manager := NewFocusManager(nil)
		item1 := &mockFocusableItem{
			placement: sprites.PercentBasedPlacement{StartPercentX: 0.1, EndPercentX: 0.2, StartPercentY: 0.1, EndPercentY: 0.2},
			canFocus:  true,
		}
		item2 := &mockFocusableItem{
			placement: sprites.PercentBasedPlacement{StartPercentX: 0.3, EndPercentX: 0.4, StartPercentY: 0.3, EndPercentY: 0.4},
			canFocus:  true,
		}

		manager.AddFocusableItem(item1)
		manager.AddFocusableItem(item2)

		// Start at second item
		manager.FocusNext()
		manager.FocusNext()
		assert.Equal(t, 1, manager.currentFocus)

		manager.FocusPrevious()

		assert.Equal(t, 0, manager.currentFocus)
		assert.True(t, item1.hasFocus)
		assert.False(t, item2.hasFocus)
	})

	t.Run("wraps around to last item", func(t *testing.T) {
		manager := NewFocusManager(nil)
		item1 := &mockFocusableItem{
			placement: sprites.PercentBasedPlacement{StartPercentX: 0.1, EndPercentX: 0.2, StartPercentY: 0.1, EndPercentY: 0.2},
			canFocus:  true,
		}
		item2 := &mockFocusableItem{
			placement: sprites.PercentBasedPlacement{StartPercentX: 0.3, EndPercentX: 0.4, StartPercentY: 0.3, EndPercentY: 0.4},
			canFocus:  true,
		}

		manager.AddFocusableItem(item1)
		manager.AddFocusableItem(item2)

		// Start with no focus, then go previous
		manager.FocusPrevious()

		assert.Equal(t, 1, manager.currentFocus)
		assert.False(t, item1.hasFocus)
		assert.True(t, item2.hasFocus)
	})

	t.Run("skips non-focusable items", func(t *testing.T) {
		manager := NewFocusManager(nil)
		item1 := &mockFocusableItem{
			placement: sprites.PercentBasedPlacement{StartPercentX: 0.1, EndPercentX: 0.2, StartPercentY: 0.1, EndPercentY: 0.2},
			canFocus:  true,
		}
		item2 := &mockFocusableItem{
			placement: sprites.PercentBasedPlacement{StartPercentX: 0.3, EndPercentX: 0.4, StartPercentY: 0.3, EndPercentY: 0.4},
			canFocus:  false, // Not focusable
		}
		item3 := &mockFocusableItem{
			placement: sprites.PercentBasedPlacement{StartPercentX: 0.5, EndPercentX: 0.6, StartPercentY: 0.5, EndPercentY: 0.6},
			canFocus:  true,
		}

		manager.AddFocusableItem(item1)
		manager.AddFocusableItem(item2)
		manager.AddFocusableItem(item3)

		manager.FocusNext()
		assert.Equal(t, 0, manager.currentFocus) // First item

		manager.FocusNext()
		assert.Equal(t, 2, manager.currentFocus) // Skip to third item
		assert.False(t, item2.hasFocus)          // Second item never focused
	})

	t.Run("handles no focusable items", func(t *testing.T) {
		manager := NewFocusManager(nil)
		item1 := &mockFocusableItem{canFocus: false}
		item2 := &mockFocusableItem{canFocus: false}

		manager.AddFocusableItem(item1)
		manager.AddFocusableItem(item2)

		manager.FocusNext()

		assert.Equal(t, -1, manager.currentFocus)
		assert.False(t, manager.focusIndicator.IsVisible())
	})
}

func TestFocusManagerState(t *testing.T) {
	t.Run("clears focus", func(t *testing.T) {
		manager := NewFocusManager(nil)
		item := &mockFocusableItem{
			placement: sprites.PercentBasedPlacement{StartPercentX: 0.1, EndPercentX: 0.2, StartPercentY: 0.1, EndPercentY: 0.2},
			canFocus:  true,
		}

		manager.AddFocusableItem(item)
		manager.FocusNext()
		assert.True(t, item.hasFocus)

		manager.ClearFocus()

		assert.Equal(t, -1, manager.currentFocus)
		assert.False(t, item.hasFocus)
		assert.Equal(t, 1, item.focusLost)
		assert.False(t, manager.focusIndicator.IsVisible())
	})

	t.Run("gets current focus", func(t *testing.T) {
		manager := NewFocusManager(nil)
		item1 := &mockFocusableItem{
			placement: sprites.PercentBasedPlacement{StartPercentX: 0.1, EndPercentX: 0.2, StartPercentY: 0.1, EndPercentY: 0.2},
			canFocus:  true,
		}
		item2 := &mockFocusableItem{
			placement: sprites.PercentBasedPlacement{StartPercentX: 0.3, EndPercentX: 0.4, StartPercentY: 0.3, EndPercentY: 0.4},
			canFocus:  true,
		}

		manager.AddFocusableItem(item1)
		manager.AddFocusableItem(item2)

		// No focus initially
		assert.Nil(t, manager.GetCurrentFocus())

		manager.FocusNext()
		assert.Same(t, item1, manager.GetCurrentFocus())

		manager.FocusNext()
		assert.Same(t, item2, manager.GetCurrentFocus())
	})

	t.Run("enables and disables", func(t *testing.T) {
		manager := NewFocusManager(nil)
		item := &mockFocusableItem{
			placement: sprites.PercentBasedPlacement{StartPercentX: 0.1, EndPercentX: 0.2, StartPercentY: 0.1, EndPercentY: 0.2},
			canFocus:  true,
		}

		manager.AddFocusableItem(item)
		manager.FocusNext()
		assert.True(t, item.hasFocus)

		manager.SetEnabled(false)

		assert.False(t, manager.IsEnabled())
		assert.False(t, item.hasFocus) // Focus cleared when disabled

		// Should not respond to navigation when disabled
		manager.FocusNext()
		assert.Equal(t, -1, manager.currentFocus)

		manager.SetEnabled(true)
		assert.True(t, manager.IsEnabled())
	})
}

func TestGlobalFocusManager(t *testing.T) {
	// Reset global state
	globalFocusManager = nil

	t.Run("creates global instance", func(t *testing.T) {
		manager := GetGlobalFocusManager()
		require.NotNil(t, manager)
		assert.True(t, manager.IsEnabled())

		// Should return same instance
		manager2 := GetGlobalFocusManager()
		assert.Same(t, manager, manager2)
	})

	t.Run("sets global instance", func(t *testing.T) {
		customManager := NewFocusManager(NewAccessibilityFocusIndicator())
		SetGlobalFocusManager(customManager)

		retrieved := GetGlobalFocusManager()
		assert.Same(t, customManager, retrieved)
	})
}
