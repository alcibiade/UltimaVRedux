package widgets

import (
	"image"
	"math"
	"strings"

	"github.com/bradhannah/UltimaVRedux/internal/display"
	"github.com/bradhannah/UltimaVRedux/internal/text"
	ucolor "github.com/bradhannah/UltimaVRedux/pkg/color"
	"github.com/hajimehoshi/ebiten/v2"
	etext "github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// AppendControlHints styles existing "Key: Action" hints separated by two or
// more spaces without changing their wording, spacing, or line ownership.
func AppendControlHints(line *text.LineBuilder, guidance string) *text.LineBuilder {
	for index, hint := range strings.Split(guidance, "  ") {
		if index > 0 {
			line.Text("  ", DialogControlHintColor)
		}
		key, action, found := strings.Cut(hint, ":")
		if found {
			line.Text(key, DialogControlKeyColor).Text(":"+action, DialogControlHintColor)
		} else {
			line.Text(hint, DialogControlHintColor)
		}
	}

	return line
}

// staticViewerHint keeps return guidance outside the original artwork.
type staticViewerHint struct {
	output *text.Output
	width  int
}

const (
	staticViewerHintLines       = 4
	staticViewerHintChars       = 80
	staticViewerHintGapDivisor  = 2
	staticViewerHintLineSpacing = 1.2
)

func (h *staticViewerHint) draw(screen *ebiten.Image, artwork image.Rectangle, guidance string) {
	width, _ := display.GetManager().GetCurrentSize()
	if h.output == nil || h.width != width {
		h.width = width
		_, h.output = GetScaledFontWithOutputForWidth(FontConfig{Size: FontSizeMainScreen,
			MaxLines: staticViewerHintLines, MaxChars: staticViewerHintChars, LineSpacing: 0,
			LineSpacingMultiplier: staticViewerHintLineSpacing}, width)
	}
	h.output.Reset()
	h.output.SetMaxCharsPerLine(max(1, h.output.Font.CharsForWidth(artwork.Dx())))
	AppendControlHints(h.output.AppendLine(), guidance).WordWrap().Done()
	height := int(math.Ceil(h.output.GetTotalHeightForLines(h.output.GetPanelContentLineCount())))
	gap := max(1, height/staticViewerHintGapDivisor)
	top := artwork.Max.Y + gap
	if top+height > screen.Bounds().Max.Y {
		top = artwork.Min.Y - gap - height
	}
	if top < screen.Bounds().Min.Y {
		return // Never cover source art in unsupported full-screen placements.
	}
	vector.FillRect(screen, float32(artwork.Min.X), float32(top),
		float32(artwork.Dx()), float32(height), ucolor.EGABlack, false)
	h.output.DrawPanelContent(screen, image.Pt(artwork.Min.X, top), artwork.Dx(), etext.AlignCenter)
}
