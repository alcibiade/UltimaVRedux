package widgets

import (
	"testing"

	"github.com/bradhannah/UltimaVRedux/internal/sprites"
)

// TestButtonSizeImplementation verifies that all button sizes return proper dimensions
func TestButtonSizeImplementation(t *testing.T) {
	tests := []struct {
		size           ButtonSize
		expectedWidth  float64
		expectedHeight float64
	}{
		{SmallButton, 0.15, 0.05},
		{MediumButton, 0.2, 0.065},
		{LargeButton, 0.3, 0.08},
	}

	for _, test := range tests {
		width := GetButtonWidthPercent(test.size)
		height := GetButtonHeightPercent(test.size)

		if width != test.expectedWidth {
			t.Errorf("GetButtonWidthPercent(%v) = %v, expected %v", test.size, width, test.expectedWidth)
		}
		if width == 0 {
			t.Errorf("GetButtonWidthPercent(%v) should not return 0 (was not implemented)", test.size)
		}

		if height != test.expectedHeight {
			t.Errorf("GetButtonHeightPercent(%v) = %v, expected %v", test.size, height, test.expectedHeight)
		}
		if height == 0 {
			t.Errorf("GetButtonHeightPercent(%v) should not return 0 (was not implemented)", test.size)
		}
	}
}

// TestButtonSizeFontPoints verifies that button font points are properly implemented
func TestButtonSizeFontPoints(t *testing.T) {
	// Test that different button sizes use appropriate font points
	smallBtn := NewButton("Small", func() {}, sprites.PercentBasedCenterPoint{X: 0.5, Y: 0.5}, SmallButton)
	mediumBtn := NewButton("Medium", func() {}, sprites.PercentBasedCenterPoint{X: 0.5, Y: 0.5}, MediumButton)
	largeBtn := NewButton("Large", func() {}, sprites.PercentBasedCenterPoint{X: 0.5, Y: 0.5}, LargeButton)

	// Verify that each button has the correct size assigned
	if smallBtn.size != SmallButton {
		t.Errorf("Small button should have SmallButton size")
	}
	if mediumBtn.size != MediumButton {
		t.Errorf("Medium button should have MediumButton size")
	}
	if largeBtn.size != LargeButton {
		t.Errorf("Large button should have LargeButton size")
	}

	// Verify that font sizes are retrieved correctly via getFontSize()
	// Font sizes use StandardFontSize but preserve original button values
	if smallBtn.getFontSize() != StandardFontSize(smallButtonFontPoint) {
		t.Errorf("Small button font size = %v, expected %v", smallBtn.getFontSize(), StandardFontSize(smallButtonFontPoint))
	}
	if mediumBtn.getFontSize() != StandardFontSize(mediumButtonFontPoint) {
		t.Errorf("Medium button font size = %v, expected %v", mediumBtn.getFontSize(), StandardFontSize(mediumButtonFontPoint))
	}
	if largeBtn.getFontSize() != StandardFontSize(largeButtonFontPoint) {
		t.Errorf("Large button font size = %v, expected %v", largeBtn.getFontSize(), StandardFontSize(largeButtonFontPoint))
	}
}
