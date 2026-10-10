package text

import (
	"image"
	"math"
	"testing"
)

func TestIntroMenuChoicePositionsFitPanel(t *testing.T) {
	const (
		screenHeight = 720
		fontHeight   = 22.0
		choiceCount  = 9
	)

	positions := introMenuChoicePositions(screenHeight, fontHeight, choiceCount)
	if len(positions) != choiceCount {
		t.Fatalf("position count = %d, want %d", len(positions), choiceCount)
	}
	menuTop := screenHeight * introMenuTextStartY
	menuBottom := screenHeight * introMenuTextEndY
	if positions[0] < menuTop {
		t.Fatalf("first choice top = %f, want at least %f", positions[0], menuTop)
	}
	lastBottom := positions[len(positions)-1] + fontHeight
	if lastBottom > menuBottom+1e-9 {
		t.Fatalf("last choice bottom = %f, want at most %f", lastBottom, menuBottom)
	}
	for index := 1; index < len(positions); index++ {
		if positions[index] <= positions[index-1] {
			t.Fatalf("choice %d top = %f, want greater than previous top %f", index, positions[index], positions[index-1])
		}
	}
	regularStep := positions[1] - positions[0]
	for index := 1; index < len(positions); index++ {
		step := positions[index] - positions[index-1]
		if math.Abs(step-regularStep) > 1e-9 {
			t.Fatalf("choice %d step = %f, want uniform step %f", index, step, regularStep)
		}
	}
}

func TestIntroMenuNinthChoicePreservesEightChoiceFontHeight(t *testing.T) {
	if len(IntroChoices) != introMenuMaxChoices {
		t.Fatalf("choice count = %d, want capacity %d", len(IntroChoices), introMenuMaxChoices)
	}
	eightChoiceHeight := introMenuFontHeightPercent(8)
	nineChoiceHeight := introMenuFontHeightPercent(len(IntroChoices))
	if nineChoiceHeight != eightChoiceHeight {
		t.Fatalf("nine-choice font height = %f, want preserved height %f", nineChoiceHeight, eightChoiceHeight)
	}
}

func TestIntroMenuChoiceHighlightsClearFrameBorder(t *testing.T) {
	const (
		screenHeight = 720
		fontHeight   = 22.0
	)
	positions := introMenuChoicePositions(screenHeight, fontHeight, len(IntroChoices))
	highlightPadding := screenHeight * introExtraYHeightPercent
	firstHighlightTop := positions[0] - highlightPadding
	lastHighlightBottom := positions[len(positions)-1] + fontHeight + highlightPadding
	safeTop := screenHeight * (introMenuFrameStartY + introMenuFrameSafeInsetY)
	safeBottom := screenHeight * (introMenuFrameEndY - introMenuFrameSafeInsetY)

	if firstHighlightTop < safeTop-1e-9 {
		t.Fatalf("first highlight top = %f, want at least safe frame top %f", firstHighlightTop, safeTop)
	}
	if lastHighlightBottom > safeBottom+1e-9 {
		t.Fatalf("last highlight bottom = %f, want at most safe frame bottom %f", lastHighlightBottom, safeBottom)
	}
}

func TestIntroMenuRejectsMoreThanNineChoices(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("expected layout-capacity panic")
		}
	}()

	introMenuFontHeightPercent(introMenuMaxChoices + 1)
}

func TestIntroChoiceHighlightUsesSteppedOutsideEdges(t *testing.T) {
	rect := image.Rect(10, 20, 110, 50)
	bands := introChoiceHighlightBands(rect, 3)

	if len(bands) != 4 {
		t.Fatalf("band count = %d, want 4", len(bands))
	}
	if bands[0] != image.Rect(16, 20, 104, 23) {
		t.Fatalf("top band = %v, want stepped inset", bands[0])
	}
	if bands[1] != image.Rect(13, 23, 107, 47) {
		t.Fatalf("middle band = %v, want one-pixel-unit inset", bands[1])
	}
	if bands[3] != image.Rect(10, 26, 110, 44) {
		t.Fatalf("center band = %v, want full-width outside edges", bands[3])
	}
}
