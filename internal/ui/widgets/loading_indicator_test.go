package widgets

import (
	stdcolor "image/color"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/bradhannah/UltimaVRedux/internal/sprites"
)

func TestNewLoadingIndicator(t *testing.T) {
	t.Run("creates loading indicator with valid config", func(t *testing.T) {
		config := LoadingIndicatorConfig{
			Placement: sprites.PercentBasedPlacement{
				StartPercentX: 0.3,
				EndPercentX:   0.7,
				StartPercentY: 0.4,
				EndPercentY:   0.6,
			},
			Message:        "Loading game data...",
			AnimationStyle: LoadingStyleSpinner,
			AnimationSpeed: 8.0,
			SpinnerSize:    0.4,
			FontSize:       16,
		}

		indicator := NewLoadingIndicator(config)

		assert.NotNil(t, indicator)
		assert.Equal(t, "Loading game data...", indicator.message)
		assert.Equal(t, LoadingStyleSpinner, indicator.animationStyle)
		assert.Equal(t, 8.0, indicator.animationSpeed)
		assert.Equal(t, 0.4, indicator.spinnerSize)
		assert.False(t, indicator.isActive)            // Default not active
		assert.True(t, indicator.isVisible)            // Default visible
		assert.Equal(t, 0.0, indicator.animationFrame) // Starts at 0
	})

	t.Run("applies safe defaults for missing config", func(t *testing.T) {
		config := LoadingIndicatorConfig{
			Placement: sprites.PercentBasedPlacement{
				StartPercentX: 0.2,
				EndPercentX:   0.8,
				StartPercentY: 0.2,
				EndPercentY:   0.8,
			},
			// Message empty - should use default
			// AnimationSpeed 0 - should use default
			// SpinnerSize 0 - should use default
			// FontSize 0 - should use default
		}

		indicator := NewLoadingIndicator(config)

		assert.NotNil(t, indicator)
		assert.Equal(t, "Loading...", indicator.message) // Default message
		assert.Equal(t, 5.0, indicator.animationSpeed)   // Default speed
		assert.Equal(t, 0.3, indicator.spinnerSize)      // Default spinner size
		assert.NotNil(t, indicator.backgroundColor)
		assert.NotNil(t, indicator.spinnerColor)
		assert.NotNil(t, indicator.textColor)
	})

	t.Run("handles invalid spinner size", func(t *testing.T) {
		config := LoadingIndicatorConfig{
			Placement: sprites.PercentBasedPlacement{
				StartPercentX: 0.2,
				EndPercentX:   0.8,
				StartPercentY: 0.2,
				EndPercentY:   0.8,
			},
			SpinnerSize: 1.5, // Invalid - over 100%
		}

		indicator := NewLoadingIndicator(config)
		assert.Equal(t, 0.3, indicator.spinnerSize) // Default value
	})

	t.Run("uses custom colors when provided", func(t *testing.T) {
		customBg := stdcolor.RGBA{R: 255, G: 0, B: 0, A: 255}
		customSpinner := stdcolor.RGBA{R: 0, G: 255, B: 0, A: 255}
		customText := stdcolor.RGBA{R: 0, G: 0, B: 255, A: 255}

		config := LoadingIndicatorConfig{
			Placement: sprites.PercentBasedPlacement{
				StartPercentX: 0.1,
				EndPercentX:   0.9,
				StartPercentY: 0.1,
				EndPercentY:   0.9,
			},
			BackgroundColor: customBg,
			SpinnerColor:    customSpinner,
			TextColor:       customText,
		}

		indicator := NewLoadingIndicator(config)
		assert.Equal(t, customBg, indicator.backgroundColor)
		assert.Equal(t, customSpinner, indicator.spinnerColor)
		assert.Equal(t, customText, indicator.textColor)
	})

	t.Run("handles different animation styles", func(t *testing.T) {
		styles := []LoadingAnimationStyle{
			LoadingStyleSpinner,
			LoadingStyleDots,
			LoadingStyleBars,
		}

		for _, style := range styles {
			config := LoadingIndicatorConfig{
				Placement: sprites.PercentBasedPlacement{
					StartPercentX: 0.1,
					EndPercentX:   0.9,
					StartPercentY: 0.1,
					EndPercentY:   0.9,
				},
				AnimationStyle: style,
			}

			indicator := NewLoadingIndicator(config)
			assert.Equal(t, style, indicator.animationStyle)
		}
	})
}

func TestLoadingIndicatorState(t *testing.T) {
	indicator := NewLoadingIndicator(LoadingIndicatorConfig{
		Placement: sprites.PercentBasedPlacement{
			StartPercentX: 0.2,
			EndPercentX:   0.8,
			StartPercentY: 0.3,
			EndPercentY:   0.7,
		},
		Message: "Test loading",
	})

	t.Run("manages active state correctly", func(t *testing.T) {
		assert.False(t, indicator.IsActive()) // Default not active

		indicator.Start()
		assert.True(t, indicator.IsActive())

		indicator.Stop()
		assert.False(t, indicator.IsActive())
	})

	t.Run("manages visibility correctly", func(t *testing.T) {
		assert.True(t, indicator.IsVisible()) // Default visible

		indicator.SetVisible(false)
		assert.False(t, indicator.IsVisible())

		indicator.SetVisible(true)
		assert.True(t, indicator.IsVisible())
	})

	t.Run("updates message correctly", func(t *testing.T) {
		assert.Equal(t, "Test loading", indicator.GetMessage())

		indicator.SetMessage("New loading message")
		assert.Equal(t, "New loading message", indicator.GetMessage())
	})

	t.Run("updates animation speed correctly", func(t *testing.T) {
		indicator.SetAnimationSpeed(10.0)
		assert.Equal(t, 10.0, indicator.GetAnimationSpeed())

		// Test negative speed handling
		indicator.SetAnimationSpeed(-5.0)
		assert.Equal(t, 5.0, indicator.GetAnimationSpeed()) // Should use absolute value
	})

	t.Run("updates animation style correctly", func(t *testing.T) {
		indicator.SetAnimationStyle(LoadingStyleDots)
		assert.Equal(t, LoadingStyleDots, indicator.GetAnimationStyle())
		assert.Equal(t, 0.0, indicator.animationFrame) // Should reset frame

		indicator.SetAnimationStyle(LoadingStyleBars)
		assert.Equal(t, LoadingStyleBars, indicator.GetAnimationStyle())
		assert.Equal(t, 0.0, indicator.animationFrame) // Should reset frame
	})

	t.Run("resets animation frame", func(t *testing.T) {
		indicator.animationFrame = 180.0 // Set to middle of animation
		indicator.Reset()
		assert.Equal(t, 0.0, indicator.animationFrame)
	})
}

func TestLoadingIndicatorAnimation(t *testing.T) {
	t.Run("advances animation when active", func(t *testing.T) {
		indicator := NewLoadingIndicator(LoadingIndicatorConfig{
			Placement: sprites.PercentBasedPlacement{
				StartPercentX: 0.2,
				EndPercentX:   0.8,
				StartPercentY: 0.3,
				EndPercentY:   0.7,
			},
			AnimationSpeed: 10.0,
		})

		indicator.Start()
		assert.Equal(t, 0.0, indicator.animationFrame)

		indicator.Update()
		assert.Equal(t, 10.0, indicator.animationFrame)

		indicator.Update()
		assert.Equal(t, 20.0, indicator.animationFrame)
	})

	t.Run("doesn't advance animation when inactive", func(t *testing.T) {
		indicator := NewLoadingIndicator(LoadingIndicatorConfig{
			Placement: sprites.PercentBasedPlacement{
				StartPercentX: 0.2,
				EndPercentX:   0.8,
				StartPercentY: 0.3,
				EndPercentY:   0.7,
			},
			AnimationSpeed: 10.0,
		})

		indicator.Stop() // Ensure inactive
		assert.Equal(t, 0.0, indicator.animationFrame)

		indicator.Update()
		assert.Equal(t, 0.0, indicator.animationFrame) // Should not advance
	})

	t.Run("doesn't advance animation when invisible", func(t *testing.T) {
		indicator := NewLoadingIndicator(LoadingIndicatorConfig{
			Placement: sprites.PercentBasedPlacement{
				StartPercentX: 0.2,
				EndPercentX:   0.8,
				StartPercentY: 0.3,
				EndPercentY:   0.7,
			},
			AnimationSpeed: 10.0,
		})

		indicator.Start()
		indicator.SetVisible(false)
		assert.Equal(t, 0.0, indicator.animationFrame)

		indicator.Update()
		assert.Equal(t, 0.0, indicator.animationFrame) // Should not advance
	})

	t.Run("wraps animation frame at 360 degrees", func(t *testing.T) {
		indicator := NewLoadingIndicator(LoadingIndicatorConfig{
			Placement: sprites.PercentBasedPlacement{
				StartPercentX: 0.2,
				EndPercentX:   0.8,
				StartPercentY: 0.3,
				EndPercentY:   0.7,
			},
			AnimationSpeed: 50.0, // Large steps to reach 360 quickly
		})

		indicator.Start()
		indicator.animationFrame = 350.0

		indicator.Update()
		assert.Equal(t, 40.0, indicator.animationFrame) // Should wrap: 350 + 50 - 360 = 40
	})
}

func TestLoadingIndicatorColors(t *testing.T) {
	indicator := NewLoadingIndicator(LoadingIndicatorConfig{
		Placement: sprites.PercentBasedPlacement{
			StartPercentX: 0.2,
			EndPercentX:   0.8,
			StartPercentY: 0.3,
			EndPercentY:   0.7,
		},
	})

	t.Run("updates colors correctly", func(t *testing.T) {
		newBg := stdcolor.RGBA{R: 255, G: 0, B: 0, A: 255}
		newSpinner := stdcolor.RGBA{R: 0, G: 255, B: 0, A: 255}
		newText := stdcolor.RGBA{R: 0, G: 0, B: 255, A: 255}

		indicator.SetColors(newBg, newSpinner, newText)

		assert.Equal(t, newBg, indicator.backgroundColor)
		assert.Equal(t, newSpinner, indicator.spinnerColor)
		assert.Equal(t, newText, indicator.textColor)
	})

	t.Run("handles nil colors in SetColors", func(t *testing.T) {
		originalBg := indicator.backgroundColor
		originalSpinner := indicator.spinnerColor
		originalText := indicator.textColor

		indicator.SetColors(nil, nil, nil)

		// Colors should remain unchanged when nil
		assert.Equal(t, originalBg, indicator.backgroundColor)
		assert.Equal(t, originalSpinner, indicator.spinnerColor)
		assert.Equal(t, originalText, indicator.textColor)
	})
}

func TestLoadingIndicatorUpdate(t *testing.T) {
	indicator := NewLoadingIndicator(LoadingIndicatorConfig{
		Placement: sprites.PercentBasedPlacement{
			StartPercentX: 0.2,
			EndPercentX:   0.8,
			StartPercentY: 0.3,
			EndPercentY:   0.7,
		},
		AnimationSpeed: 15.0,
	})

	t.Run("updates when visible and active", func(t *testing.T) {
		indicator.SetVisible(true)
		indicator.Start()

		initialFrame := indicator.animationFrame
		indicator.Update()
		assert.Greater(t, indicator.animationFrame, initialFrame)
	})

	t.Run("doesn't update when invisible", func(t *testing.T) {
		indicator.SetVisible(false)
		indicator.Start()

		initialFrame := indicator.animationFrame
		indicator.Update()
		assert.Equal(t, initialFrame, indicator.animationFrame)
	})

	t.Run("doesn't update when inactive", func(t *testing.T) {
		indicator.SetVisible(true)
		indicator.Stop()

		initialFrame := indicator.animationFrame
		indicator.Update()
		assert.Equal(t, initialFrame, indicator.animationFrame)
	})
}

func TestLoadingIndicatorEdgeCases(t *testing.T) {
	t.Run("handles zero animation speed", func(t *testing.T) {
		indicator := NewLoadingIndicator(LoadingIndicatorConfig{
			Placement: sprites.PercentBasedPlacement{
				StartPercentX: 0.2,
				EndPercentX:   0.8,
				StartPercentY: 0.3,
				EndPercentY:   0.7,
			},
			AnimationSpeed: 0.0, // Should use default
		})

		assert.Equal(t, 5.0, indicator.animationSpeed) // Default value
	})

	t.Run("handles very high animation speed", func(t *testing.T) {
		indicator := NewLoadingIndicator(LoadingIndicatorConfig{
			Placement: sprites.PercentBasedPlacement{
				StartPercentX: 0.2,
				EndPercentX:   0.8,
				StartPercentY: 0.3,
				EndPercentY:   0.7,
			},
			AnimationSpeed: 500.0, // Very high speed
		})

		indicator.Start()
		indicator.Update()

		// Should handle wrapping correctly even with high speed
		assert.True(t, indicator.animationFrame >= 0.0 && indicator.animationFrame < 360.0)
	})

	t.Run("handles empty message", func(t *testing.T) {
		indicator := NewLoadingIndicator(LoadingIndicatorConfig{
			Placement: sprites.PercentBasedPlacement{
				StartPercentX: 0.2,
				EndPercentX:   0.8,
				StartPercentY: 0.3,
				EndPercentY:   0.7,
			},
			Message: "", // Empty message should use default
		})

		assert.Equal(t, "Loading...", indicator.GetMessage())
	})

	t.Run("handles very long message", func(t *testing.T) {
		longMessage := "This is a very long loading message that might exceed normal display boundaries and should be handled gracefully"

		indicator := NewLoadingIndicator(LoadingIndicatorConfig{
			Placement: sprites.PercentBasedPlacement{
				StartPercentX: 0.2,
				EndPercentX:   0.8,
				StartPercentY: 0.3,
				EndPercentY:   0.7,
			},
			Message: longMessage,
		})

		assert.Equal(t, longMessage, indicator.GetMessage())

	})

	t.Run("handles small display area", func(t *testing.T) {
		indicator := NewLoadingIndicator(LoadingIndicatorConfig{
			Placement: sprites.PercentBasedPlacement{
				StartPercentX: 0.45,
				EndPercentX:   0.55, // Very small width
				StartPercentY: 0.45,
				EndPercentY:   0.55, // Very small height
			},
		})

		indicator.SetVisible(true)
		indicator.Start()
		assert.True(t, indicator.IsVisible())
		assert.True(t, indicator.IsActive())
	})
}
