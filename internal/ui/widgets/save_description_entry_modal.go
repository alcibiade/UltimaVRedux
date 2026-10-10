package widgets

import (
	"image"
	"math"

	"github.com/bradhannah/UltimaVRedux/internal/sprites"
	"github.com/bradhannah/UltimaVRedux/internal/text"
)

const (
	saveDescriptionSectionGaps   = 5
	saveDescriptionCenterPercent = 0.5
)

func (m *LegacyPathEntryModal) configureDescriptionRendering() {
	_, m.titleOutput = GetScaledFontWithOutput(FontConfig{
		Size: FontSizeDialogTitle, LineSpacing: 0, LineSpacingMultiplier: 0, MaxLines: 1, MaxChars: modalDialogTitleMaxChars,
	})
	m.titleOutput.AppendLine().Text(m.title, DialogHeaderColor).Done()
	padding := m.scaledDimension(legacyPathContentPadding, legacyPathMinimumPadding)
	width := m.border.GetInteriorBounds().Dx() - padding*legacyPathCenterDivisor
	maxChars := m.font.CharsForWidth(max(1, width))
	noticeLines := max(1, text.MeasureText(m.notice, m.font, m.pathOutput.GetLineSpacing(), maxChars).Lines)
	m.noticeOutput = text.NewOutput(m.font, m.pathOutput.GetLineSpacing(), noticeLines, maxChars)
	if m.notice != "" {
		m.noticeOutput.AppendLine().Text(m.notice, DialogStatusWarningColor).WordWrap().Done()
	}
	spacing := m.scaledDimension(legacyPathVerticalSpacing, legacyPathMinimumSpacing)
	fieldPadding := m.scaledDimension(legacyPathFieldPaddingY, legacyPathMinimumPaddingY) + m.pixelUnit()
	// Reserve the complete measured warning and validation space, not a fixed
	// percentage that silently clips long cross-playthrough warnings at 720p.
	contentHeight := m.titleOutput.GetFontHeight() + m.promptOutput.GetFontHeight() +
		m.pathOutput.GetFontHeight() + float64(fieldPadding*legacyPathCenterDivisor) +
		m.noticeOutput.GetTotalHeightForLines(noticeLines) +
		m.messageOutput.GetTotalHeightForLines(legacyPathValidationLines) +
		m.hintOutput.GetTotalHeightForLines(legacyPathHintMaximumLines) + float64(spacing*saveDescriptionSectionGaps)
	height := int(math.Ceil((contentHeight + float64(padding*legacyPathCenterDivisor)) /
		(1 - float64(LayoutSpacingMedium)*legacyPathCenterDivisor)))
	halfHeight := float64(height) / float64(m.lastHeight) / legacyPathCenterDivisor
	m.border = newDialogBorder(sprites.PercentBasedPlacement{
		StartPercentX: legacyPathModalStartX, EndPercentX: legacyPathModalEndX,
		StartPercentY: saveDescriptionCenterPercent - halfHeight, EndPercentY: saveDescriptionCenterPercent + halfHeight,
	}, DialogBackgroundBlack)
}

func (m *LegacyPathEntryModal) calculateDescriptionRects() legacyPathEntryRects {
	padding := m.scaledDimension(legacyPathContentPadding, legacyPathMinimumPadding)
	content := m.border.GetInteriorBounds().Inset(padding)
	spacing := m.scaledDimension(legacyPathVerticalSpacing, legacyPathMinimumSpacing)
	y := content.Min.Y
	next := func(height float64) image.Rectangle {
		rect := image.Rect(content.Min.X, y, content.Max.X, y+int(math.Ceil(height)))
		y = rect.Max.Y + spacing

		return rect
	}
	var rects legacyPathEntryRects
	rects.title = next(m.titleOutput.GetFontHeight())
	rects.prompt = next(m.promptOutput.GetFontHeight())
	rects.field = next(m.pathOutput.GetFontHeight() + float64(legacyPathCenterDivisor*
		(m.pixelUnit()+m.scaledDimension(legacyPathFieldPaddingY, legacyPathMinimumPaddingY))))
	rects.notice = next(m.noticeOutput.GetTotalHeightForLines(max(1, m.noticeOutput.GetPanelContentLineCount())))
	rects.message = next(m.messageOutput.GetTotalHeightForLines(legacyPathValidationLines))
	rects.hint = next(m.hintOutput.GetTotalHeightForLines(legacyPathHintMaximumLines))

	return rects
}
