package color

import (
	"image/color"
	"testing"

	"github.com/stretchr/testify/assert"
)

// ============================================================================
// EGA COLOR TYPE TESTS
// ============================================================================

func TestEGAColor_ImplementsColorInterface(t *testing.T) {
	// Verify EGAColor implements color.Color interface
	var _ color.Color = EGARed
	var _ color.Color = EGABlue
	var _ color.Color = EGAYellow
}

func TestEGAColor_RGBA(t *testing.T) {
	// Test that RGBA() returns correct 16-bit premultiplied values
	r, g, b, a := EGARed.RGBA()

	// EGARed is {170, 0, 0, 255}
	// 170 * 0x101 = 170 * 257 = 43690
	// 255 * 0x101 = 255 * 257 = 65535
	assert.Equal(t, uint32(43690), r)
	assert.Equal(t, uint32(0), g)
	assert.Equal(t, uint32(0), b)
	assert.Equal(t, uint32(65535), a)
}

func TestEGAColor_Matches_ExactMatch(t *testing.T) {
	assert.True(t, EGARed.Matches(170, 0, 0))
	assert.True(t, EGABlue.Matches(0, 0, 170))
	assert.True(t, EGAYellow.Matches(255, 255, 85))
	assert.True(t, EGALightBlue.Matches(85, 85, 255))
}

func TestEGAColor_Matches_NoMatch(t *testing.T) {
	assert.False(t, EGARed.Matches(171, 0, 0))  // Off by 1
	assert.False(t, EGARed.Matches(170, 1, 0))  // Off by 1
	assert.False(t, EGARed.Matches(170, 0, 1))  // Off by 1
	assert.False(t, EGARed.Matches(0, 0, 170))  // Blue, not red
	assert.False(t, EGABlue.Matches(170, 0, 0)) // Red, not blue
}

func TestEGAColor_MatchesWithTolerance(t *testing.T) {
	const tolerance = 10

	// Within tolerance
	assert.True(t, EGARed.MatchesWithTolerance(175, 5, 5, tolerance))
	assert.True(t, EGARed.MatchesWithTolerance(165, 0, 0, tolerance))
	assert.True(t, EGARed.MatchesWithTolerance(170, 10, 10, tolerance))

	// At exact boundary (should pass)
	assert.True(t, EGARed.MatchesWithTolerance(180, 10, 10, tolerance))

	// Outside tolerance
	assert.False(t, EGARed.MatchesWithTolerance(181, 0, 0, tolerance))  // R too high
	assert.False(t, EGARed.MatchesWithTolerance(158, 0, 0, tolerance))  // R too low
	assert.False(t, EGARed.MatchesWithTolerance(170, 11, 0, tolerance)) // G too high
	assert.False(t, EGARed.MatchesWithTolerance(170, 0, 11, tolerance)) // B too high
}

func TestEGAColor_MatchesRGBA(t *testing.T) {
	redRGBA := color.RGBA{R: 170, G: 0, B: 0, A: 255}
	blueRGBA := color.RGBA{R: 0, G: 0, B: 170, A: 255}

	assert.True(t, EGARed.MatchesRGBA(redRGBA))
	assert.False(t, EGARed.MatchesRGBA(blueRGBA))

	// Alpha is ignored
	redDiffAlpha := color.RGBA{R: 170, G: 0, B: 0, A: 128}
	assert.True(t, EGARed.MatchesRGBA(redDiffAlpha))
}

func TestEGAColor_ToRGBA(t *testing.T) {
	rgba := EGARed.ToRGBA()

	assert.Equal(t, uint8(170), rgba.R)
	assert.Equal(t, uint8(0), rgba.G)
	assert.Equal(t, uint8(0), rgba.B)
	assert.Equal(t, uint8(255), rgba.A)
}

func TestEGAColor_UsableWithColorBuilder(t *testing.T) {
	// Verify EGAColor works with From() since it implements color.Color
	builder := From(EGARed)
	result := builder.SemiTransparent().RGBA()

	assert.Equal(t, uint8(170), result.R)
	assert.Equal(t, uint8(0), result.G)
	assert.Equal(t, uint8(0), result.B)
	assert.Equal(t, uint8(128), result.A) // SemiTransparent = 128
}

// ============================================================================
// EGA PALETTE CORRECTNESS TESTS
// ============================================================================

func TestEGAPalette_CorrectValues(t *testing.T) {
	tests := []struct {
		name    string
		color   EGAColor
		r, g, b uint8
	}{
		{"Black", EGABlack, 0, 0, 0},
		{"Blue", EGABlue, 0, 0, 170},
		{"Green", EGAGreen, 0, 170, 0},
		{"Cyan", EGACyan, 0, 170, 170},
		{"Red", EGARed, 170, 0, 0},
		{"Magenta", EGAMagenta, 170, 0, 170},
		{"Brown", EGABrown, 170, 85, 0},
		{"LightGray", EGALightGray, 170, 170, 170},
		{"DarkGray", EGADarkGray, 85, 85, 85},
		{"LightBlue", EGALightBlue, 85, 85, 255},
		{"LightGreen", EGALightGreen, 85, 255, 85},
		{"LightCyan", EGALightCyan, 85, 255, 255},
		{"LightRed", EGALightRed, 255, 85, 85},
		{"LightMagenta", EGALightMagenta, 255, 85, 255},
		{"Yellow", EGAYellow, 255, 255, 85},
		{"White", EGAWhite, 255, 255, 255},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.r, tt.color.R, "R mismatch")
			assert.Equal(t, tt.g, tt.color.G, "G mismatch")
			assert.Equal(t, tt.b, tt.color.B, "B mismatch")
			assert.Equal(t, uint8(AlphaOpaque), tt.color.A, "A should be opaque")
		})
	}
}

func TestEGATransparent_IsFullyTransparent(t *testing.T) {
	assert.Equal(t, uint8(0), EGATransparent.R)
	assert.Equal(t, uint8(0), EGATransparent.G)
	assert.Equal(t, uint8(0), EGATransparent.B)
	assert.Equal(t, uint8(0), EGATransparent.A)
}
