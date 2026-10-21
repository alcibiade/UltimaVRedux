package text

import (
	"image"
	"image/color"
	"math"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"

	"github.com/bradhannah/UltimaVRedux/internal/logging"
	ultimacolor "github.com/bradhannah/UltimaVRedux/pkg/color"
)

// centerDivisor is used for centering calculations (divide by 2)
const (
	centerDivisor                    = 2
	commandPromptSourcePixels        = 8
	commandPromptSourceShapeHeight   = 7
	commandPromptSourceTriangleWidth = 5
)

// DrawScrollingBuffer renders retained circular rows without a height limit.
func (o *Output) DrawScrollingBuffer(
	screen *ebiten.Image, point image.Point, textRectWidth int,
	bShowEmptyNewLines bool, horizAlign, vertAlign text.Align,
) {
	o.DrawScrollingBufferWithHeight(screen, point, textRectWidth, 0, bShowEmptyNewLines, horizAlign, vertAlign)
}

// DrawPanelContent renders text using PHYSICAL indices (no circular buffer logic).
// Use this for fresh panel content that should always start at the top of the panel.
// Unlike DrawScrollingBuffer which uses circular buffer indexing for scrolling
// message buffers, this method renders content from physical index 0 upward.
//
// Typical usage pattern:
//
//	output.Reset()  // Clear and reset index to 0
//	output.AppendLine().Yellow("Title").Done()
//	output.AppendLine().White("Content").Done()
//	output.DrawPanelContent(screen, point, width, horizAlign)
func (o *Output) DrawPanelContent(screen *ebiten.Image, point image.Point, textRectWidth int, horizAlign text.Align) {
	o.DrawPanelContentRange(screen, point, textRectWidth, horizAlign, 0, o.GetPanelContentLineCount())
}

// DrawPanelContentRange renders a bounded physical line range from fresh panel content.
// It is intended for capacity-based paging after Reset and AppendLine population.
func (o *Output) DrawPanelContentRange(
	screen *ebiten.Image,
	point image.Point,
	textRectWidth int,
	horizAlign text.Align,
	startLine int,
	lineCount int,
) {
	contentLineCount := o.GetPanelContentLineCount()
	startLine = max(0, startLine)
	lineCount = max(0, lineCount)
	if contentLineCount == 0 || startLine >= contentLineCount || lineCount == 0 {
		return
	}
	lastNonEmptyIndex := min(contentLineCount, startLine+lineCount) - 1

	// DEBUG: Draw red bounding box around the text output area
	if o.debugOptions != nil && o.debugOptions.DialogBoundingBoxes {
		debugHeight := int(float64(lastNonEmptyIndex-startLine+1) * o.lineSpacing)
		strokeWidth := float32(1.0)
		vector.StrokeRect(screen,
			float32(point.X), float32(point.Y),
			float32(textRectWidth), float32(debugHeight),
			strokeWidth, ultimacolor.EGARed, false)

		logging.Debugf(logging.CategoryUI, "=== PANEL CONTENT DEBUG ===")
		logging.Debugf(logging.CategoryUI, "BoundingBox: point=(%d,%d), width=%d, height=%d",
			point.X, point.Y, textRectWidth, debugHeight)
		logging.Debugf(logging.CategoryUI, "Buffer: lastNonEmptyIndex=%d (physical)", lastNonEmptyIndex)
	}

	// Render lines from the requested physical range.
	currentY := float64(point.Y)
	for rowIndex := startLine; rowIndex <= lastNonEmptyIndex; rowIndex++ {
		line := o.lines[rowIndex] // Physical index, no circular offset

		lineAlign := line.Alignment
		if lineAlign == 0 {
			lineAlign = horizAlign
		}

		if len(line.Segments) > 0 {
			o.drawStyledLine(screen, rowIndex, line.Segments, float64(point.X), float64(textRectWidth), currentY, lineAlign)
		} else {
			o.drawSimpleLine(screen, rowIndex, line.Text, line.Color,
				float64(point.X), float64(textRectWidth), currentY, lineAlign)
		}

		currentY += o.lineSpacing
	}
}

// DrawScrollingBufferWithHeight renders text with optional bottom-up alignment
// If textRectHeight > 0 and vertAlign == AlignEnd, text is positioned from bottom and grows upward
// Trailing empty/whitespace-only lines are skipped during rendering (not deleted from buffer)
// Buffer naturally overwrites old lines via circular indexing when new content is added
func (o *Output) DrawScrollingBufferWithHeight(
	screen *ebiten.Image, point image.Point, textRectWidth, textRectHeight int,
	bShowEmptyNewLines bool, horizAlign, vertAlign text.Align,
) {
	_ = bShowEmptyNewLines
	firstNonEmptyIndex, lastNonEmptyIndex, ok := o.scrollingContentRange(textRectHeight)
	if !ok {
		return
	}

	contentLineCount := lastNonEmptyIndex - firstNonEmptyIndex + 1

	o.drawScrollingDebug(screen, point, textRectWidth, textRectHeight, firstNonEmptyIndex, lastNonEmptyIndex)

	currentY := float64(point.Y)
	if textRectHeight > 0 {
		totalTextHeight := o.GetTotalHeightForLines(contentLineCount)

		switch vertAlign {
		case text.AlignStart:
			// Keep the supplied top position.
		case text.AlignCenter:
			currentY += (float64(textRectHeight) - totalTextHeight) / centerDivisor
		case text.AlignEnd:
			currentY = float64(point.Y+textRectHeight) - totalTextHeight
		}
	}

	for rowIndex := firstNonEmptyIndex; rowIndex <= lastNonEmptyIndex; rowIndex++ {
		line := o.lines[(rowIndex+o.nextLineToIndex)%o.maxLines]

		// Use per-line alignment if set, otherwise use parameter default
		lineAlign := line.Alignment
		if lineAlign == 0 {
			lineAlign = horizAlign // Default to parameter if not set
		}

		// Check if this is a styled line (multi-color) or simple line (single color)
		if len(line.Segments) > 0 {
			// Multi-color line - use X-tracking rendering
			row := (rowIndex + o.nextLineToIndex) % o.maxLines
			o.drawStyledLine(screen, row, line.Segments, float64(point.X), float64(textRectWidth), currentY, lineAlign)
		} else {
			// Simple single-color line - use cached rendering
			row := (rowIndex + o.nextLineToIndex) % o.maxLines
			o.drawSimpleLine(screen, row, line.Text, line.Color, float64(point.X), float64(textRectWidth), currentY, lineAlign)
		}

		currentY += o.lineSpacing
	}
}

func (o *Output) drawScrollingDebug(screen *ebiten.Image, point image.Point, width, height, first, last int) {
	if o.debugOptions == nil || !o.debugOptions.DialogBoundingBoxes {
		return
	}
	count := last - first + 1
	if height == 0 {
		height = int(float64(count) * o.lineSpacing)
	}
	vector.StrokeRect(screen, float32(point.X), float32(point.Y), float32(width), float32(height),
		1, ultimacolor.EGARed, false)
	logging.Debugf(logging.CategoryUI, "=== DIALOG BOX DEBUG ===")
	logging.Debugf(logging.CategoryUI, "BoundingBox: point=(%d,%d), width=%d, height=%d", point.X, point.Y, width, height)
	logging.Debugf(logging.CategoryUI, "Buffer: nextLineToIndex=%d, firstNonEmpty=%d, lastNonEmpty=%d, contentCount=%d",
		o.nextLineToIndex, first, last, count)
	const debugRowLimit = 5
	for rowIndex := 0; rowIndex <= last && rowIndex < debugRowLimit; rowIndex++ {
		line := o.lines[(rowIndex+o.nextLineToIndex)%o.maxLines]
		value := line.Text
		if len(line.Segments) > 0 {
			var builder strings.Builder
			builder.WriteString("[styled] ")
			for _, segment := range line.Segments {
				builder.WriteString(segment.Text)
			}
			value = builder.String()
		}
		if value == "" {
			value = "[empty]"
		}
		logging.Debugf(logging.CategoryUI, "  Line[%d]: %s", rowIndex, value)
	}
}

func (o *Output) scrollingContentRange(textRectHeight int) (int, int, bool) {
	firstNonEmptyIndex := o.firstScrollingContentIndex()
	lastNonEmptyIndex := o.lastScrollingContentIndex()
	if firstNonEmptyIndex == -1 || lastNonEmptyIndex == -1 {
		return 0, 0, false
	}
	if textRectHeight > 0 {
		capacity := o.GetLineCapacity(textRectHeight)
		if capacity == 0 {
			return 0, 0, false
		}
		if contentLineCount := lastNonEmptyIndex - firstNonEmptyIndex + 1; contentLineCount > capacity {
			firstNonEmptyIndex = lastNonEmptyIndex - capacity + 1
		}
	}

	return firstNonEmptyIndex, lastNonEmptyIndex, true
}

// GetVisibleScrollingLineCount returns the rows rendered by DrawScrollingBufferWithHeight.
func (o *Output) GetVisibleScrollingLineCount(textRectHeight int) int {
	firstIndex, lastIndex, ok := o.scrollingContentRange(textRectHeight)
	if !ok {
		return 0
	}

	return lastIndex - firstIndex + 1
}

func (o *Output) firstScrollingContentIndex() int {
	for index := range len(o.lines) {
		line := o.lines[(index+o.nextLineToIndex)%o.maxLines]
		if len(line.Segments) > 0 || strings.TrimSpace(line.Text) != "" {
			return index
		}
	}

	return -1
}

func (o *Output) lastScrollingContentIndex() int {
	for index := len(o.lines) - 1; index >= 0; index-- {
		line := o.lines[(index+o.nextLineToIndex)%o.maxLines]
		if len(line.Segments) > 0 || strings.TrimSpace(line.Text) != "" {
			return index
		}
	}

	return -1
}

// ============================================================================
// Private Rendering Helpers
// ============================================================================

// getBitmapScale calculates the scale needed to match an 8px bitmap font to the line height.
func (o *Output) getBitmapScale() float64 {
	return o.lineSpacing / float64(BitmapCharHeight)
}

// getSegmentWidth calculates the rendered width of a segment, handling both font types and font scaling
func (o *Output) getSegmentWidth(seg ColorSegment) float64 {
	switch seg.FontType {
	case FontTypeNormal:
		// Measured below after applying the optional font scale.
	case FontTypeRunic, FontTypeIBM:
		return StringWidthScaled(seg.Text, o.getBitmapScale())
	case FontTypeCommandPrompt:
		width, _ := o.Font.MeasureString("M")

		return width
	}

	// Get effective font scale (default to 1.0 if not set)
	fontScale := seg.FontScale
	if fontScale == 0 {
		fontScale = FontScaleNormal
	}

	// Use scaled font for measurement if scale differs from 1.0
	if fontScale != FontScaleNormal {
		return newTextLayout(seg.Text, o.Font.sdkForScale(fontScale)).Bounds().Width()
	}

	width, _ := o.Font.MeasureString(seg.Text)

	return width
}

func (o *Output) drawCommandPromptSegment(
	screen *ebiten.Image,
	outlineColor color.Color,
	startX, startY float64,
) float64 {
	cellWidth, _ := o.Font.MeasureString("M")
	cellHeight := o.GetFontHeight()
	pixelScale := commandPromptPixelScale(cellWidth, cellHeight)
	shapeHeight := commandPromptSourceShapeHeight * pixelScale
	verticalOffset := math.Round((cellHeight-shapeHeight)/(centerDivisor*pixelScale)) * pixelScale
	topY := startY + verticalOffset
	bottomY := topY + shapeHeight
	middleY := (topY + bottomY) / centerDivisor
	leftX := startX + pixelScale/centerDivisor
	tipX := leftX + commandPromptSourceTriangleWidth*pixelScale
	strokeWidth := float32(pixelScale)

	var promptPath vector.Path
	promptPath.MoveTo(float32(leftX), float32(topY))
	promptPath.LineTo(float32(tipX), float32(middleY))
	promptPath.LineTo(float32(leftX), float32(bottomY))
	promptPath.Close()
	fillOptions := &vector.FillOptions{FillRule: vector.FillRuleNonZero}
	drawOptions := &vector.DrawPathOptions{
		AntiAlias:  false,
		ColorScale: ebiten.ColorScale{},
		Blend:      ebiten.BlendSourceOver,
	}
	drawOptions.ColorScale.ScaleWithColor(ultimacolor.EGALightGray)
	vector.FillPath(screen, &promptPath, fillOptions, drawOptions)

	vector.StrokeLine(screen, float32(leftX), float32(topY), float32(leftX), float32(bottomY),
		strokeWidth, outlineColor, false)
	vector.StrokeLine(screen, float32(leftX), float32(topY), float32(tipX), float32(middleY),
		strokeWidth, outlineColor, false)
	vector.StrokeLine(screen, float32(tipX), float32(middleY), float32(leftX), float32(bottomY),
		strokeWidth, outlineColor, false)

	return cellWidth
}

func commandPromptPixelScale(cellWidth, cellHeight float64) float64 {
	return max(math.Round(min(cellWidth, cellHeight)/commandPromptSourcePixels), 1)
}

// drawStyledLine renders a multi-colored line with X-tracking, supporting mixed TTF/Runic fonts
func (o *Output) drawStyledLine(
	screen *ebiten.Image,
	row int,
	segments []ColorSegment,
	startX, textRectWidth, startY float64,
	horizAlign text.Align,
) {
	geometry := o.layoutLine(row, segments)
	totalWidth := geometry.total

	// Apply alignment offset using proper anchors
	currentX := startX
	switch horizAlign {
	case text.AlignStart:
		// currentX already positioned correctly for left-align (anchor at left edge)
	case text.AlignCenter:
		// Center within the text rectangle
		currentX += (textRectWidth - totalWidth) / centerDivisor
	case text.AlignEnd:
		// Anchor at right edge: startX + textRectWidth - totalWidth
		currentX += textRectWidth - totalWidth
	}

	// Render each segment using appropriate font type
	if geometry.combined != nil {
		for index, segment := range segments {
			ink := segment.Color
			if ink == nil {
				ink = color.White
			}
			geometry.palette[geometry.paintNames[index]] = ink
		}
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(currentX, startY)
		drawPaintedTextLayout(screen, geometry.combined, op, geometry.palette)

		return
	}
	for index, seg := range segments {
		segWidth := geometry.advances[index]

		switch seg.FontType {
		case FontTypeNormal, FontTypeRunic, FontTypeIBM:
			op := &ebiten.DrawImageOptions{}
			op.GeoM.Translate(currentX, startY)
			drawTextLayout(screen, geometry.layouts[index], op, seg.Color)
		case FontTypeCommandPrompt:
			o.drawCommandPromptSegment(screen, seg.Color, currentX, startY)
		}

		// Advance X position - CRITICAL for correct spacing
		currentX += segWidth
	}
}

// drawSimpleLine renders a single-color line using existing cached system
func (o *Output) drawSimpleLine(
	screen *ebiten.Image, row int, lineText string, lineColor color.Color,
	startX, textRectWidth, startY float64, horizAlign text.Align,
) {
	// Check if this is an empty/whitespace-only line and debug mode is active
	if strings.TrimSpace(lineText) == "" {
		// If debug placeholders enabled, render placeholder symbol
		if o.debugOptions != nil && o.debugOptions.ShowEmptyLineDebugPlaceholders {
			lineText = "⏎" // Return symbol to indicate newline
			lineColor = ultimacolor.EGADarkGray
		} else {
			return // No placeholder or debug mode disabled, skip rendering empty line
		}
	}

	segments := []ColorSegment{{Text: lineText, Color: lineColor, FontType: FontTypeNormal, FontScale: 1}}
	geometry := o.layoutLine(row, segments)
	// Preserve the simple-row anchor allowance in the game, not SDK metrics.
	imgWidth := float64(int(geometry.total) + legacyRasterPadding)

	currentX := startX
	switch horizAlign {
	case text.AlignStart:
		// currentX already positioned correctly for left-align (anchor at left edge)
	case text.AlignCenter:
		// Center within the text rectangle
		currentX += (textRectWidth - imgWidth) / centerDivisor
	case text.AlignEnd:
		// Anchor at right edge: startX + textRectWidth - imgWidth
		currentX += textRectWidth - imgWidth
	}

	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(currentX, startY)
	drawTextLayout(screen, geometry.layouts[0], op, lineColor)
}
