package widgets

import (
	"image"
	"testing"

	"github.com/bradhannah/UltimaVRedux/internal/text"
	"github.com/stretchr/testify/require"
)

func TestRunicReferencePanelFitsRightRail(t *testing.T) {
	var panel RunicReferencePanel
	for _, size := range []image.Point{{640, 360}, {1280, 720}, {1920, 1080}, {3840, 2160}} {
		rail := image.Rect(size.X*3/4, 0, size.X, size.Y)
		font := text.NewUltimaFont(float64(FontSizeMainScreen) * float64(size.X) / 1920)
		lineSpacing := font.TextFace.Size * 1.087
		panel.ensureLayout(rail, font, lineSpacing)
		for index, output := range panel.cells {
			require.Same(t, font, output.Font)
			require.Equal(t, lineSpacing, output.GetLineSpacing())
			cell := panel.cellBounds[index]
			require.True(t, cell.In(panel.content), "cell %d at %v", index, size)
			labelWidth, _ := output.Font.MeasureString(" " + runicReferenceEntries[index].label)
			require.LessOrEqual(t, labelWidth+output.GetLineSpacing(), float64(cell.Dx())+0.01)
			require.GreaterOrEqual(t, output.GetLineCapacity(cell.Dy()), 1)
			require.LessOrEqual(t, output.GetLineSpacing(), float64(cell.Dy()))
			require.Equal(t, 1, output.GetPanelContentLineCount())
		}
		for _, output := range []*text.Output{panel.header, panel.footer} {
			width, _ := output.Font.MeasureString(output.GetOutputStr(false))
			require.LessOrEqual(t, width, float64(panel.content.Dx())+0.01)
		}
		last := panel.cellBounds[len(panel.cells)-1]
		require.LessOrEqual(t, last.Max.Y, panel.content.Max.Y-int(panel.footer.GetTotalHeightForLines(1)))
		cached := panel.cells[0]
		panel.ensureLayout(rail, font, lineSpacing)
		require.Same(t, cached, panel.cells[0], "reuse outputs until resized")
	}
}

func TestRunicReferenceIncludesAlphabetAndAuthoredCompounds(t *testing.T) {
	var panel RunicReferencePanel
	panel.ensureLayout(image.Rect(1440, 0, 1920, 1080), text.NewUltimaFont(float64(FontSizeMainScreen)), 25)
	for index, letter := range "ABCDEFGHIJKLMNOPQRSTUVWXYZ" {
		require.Equal(t, string(letter)+" "+string(letter), panel.cells[index].GetOutputStr(false))
	}
	for index, expected := range []string{"[ TH", "\\ EE", "] NG", "_ ST"} {
		require.Equal(t, expected, panel.cells[26+index].GetOutputStr(false))
	}
}
