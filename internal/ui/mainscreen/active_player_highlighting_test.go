package mainscreen

import (
	"testing"

	ucolor "github.com/bradhannah/UltimaVRedux/pkg/color"
)

func TestCharacterSummary_ColorConstantValidation(t *testing.T) {
	t.Run("YellowColorValid", func(t *testing.T) {
		yellowColor := ucolor.From(ucolor.EGAYellow).Intensity(1.3).RGBA()
		if yellowColor.R == 0 && yellowColor.G == 0 && yellowColor.B == 0 {
			t.Error("Yellow color should not be completely black")
		}
		if yellowColor.A == 0 {
			t.Error("Yellow color should not be transparent")
		}
	})
	t.Run("WhiteColorValid", func(t *testing.T) {
		whiteColor := ucolor.White
		if whiteColor.R < 200 || whiteColor.G < 200 || whiteColor.B < 200 {
			t.Error("White color should have high RGB values")
		}
		if whiteColor.A == 0 {
			t.Error("White color should not be transparent")
		}
	})
}
