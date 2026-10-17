package widgets

import (
	"image"
	"math"
	"strconv"
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
	for _, segment := range colonControlHintSegments(guidance) {
		line.Text(segment.Text, segment.Color)
	}

	return line
}

// Only recognized control labels get key ink. Plain status prose containing a
// colon stays prose; parse logical source lines before their descriptions wrap.
func colonControlHintSegments(guidance string) []merchantFooterSegment {
	if !strings.Contains(guidance, ":") {
		return []merchantFooterSegment{{Text: guidance, Color: DialogControlHintColor}}
	}
	var segments []merchantFooterSegment
	for index, hint := range strings.Split(guidance, "  ") {
		if index > 0 {
			segments = append(segments, merchantFooterSegment{Text: "  ", Color: DialogControlHintColor})
		}
		key, action, found := strings.Cut(hint, ":")
		if found && isControlHintLabel(key) {
			segments = append(segments, merchantFooterSegment{Text: key, Color: DialogControlKeyColor},
				merchantFooterSegment{Text: ":" + action, Color: DialogControlHintColor})
		} else {
			segments = append(segments, merchantFooterSegment{Text: hint, Color: DialogControlHintColor})
		}
	}

	return segments
}

func isControlHintLabel(label string) bool {
	if strings.EqualFold(strings.TrimSpace(label), "Any key") {
		return true
	}
	parts := strings.FieldsFunc(label, func(r rune) bool { return r == '/' || r == '+' || r == '-' || r == ' ' })
	if len(parts) == 0 {
		return false
	}
	for _, part := range parts {
		if !isControlHintKey(part) {
			return false
		}
	}

	return true
}

func isControlHintKey(part string) bool {
	if len(part) == 1 && (part[0] >= 'A' && part[0] <= 'Z' || part[0] >= '0' && part[0] <= '9') {
		return true
	}
	switch strings.ToLower(part) {
	case "enter", "esc", "escape", "tab", "space", "up", "dn", "down", "left", "right", "home", "end",
		"pgup", "pgdn", "pageup", "pagedown", "backspace", "delete", "ctrl", "shift", "shft", "shf", "alt", "meta", "or":
		return true
	}
	const lastFunctionKey = 12
	number, err := strconv.Atoi(strings.TrimPrefix(part, "F"))

	return strings.HasPrefix(part, "F") && err == nil && number >= 1 && number <= lastFunctionKey
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
