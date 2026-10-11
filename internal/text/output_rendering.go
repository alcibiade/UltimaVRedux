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

// ============================================================================
// INTERNAL DRAWING METHODS - DO NOT USE DIRECTLY IN EXTERNAL CODE
// ============================================================================
//
// The DrawText, DrawTextCenter, and DrawTextRightToLeft methods below are
// INTERNAL to the Output system. External code must NEVER call these directly.
//
// These methods exist for internal use by:
//   - DrawScrollingBuffer (scrolling buffer rendering)
//   - DrawPanelContent (panel rendering internals)
//
// REQUIRED PATTERNS for external code:
//
//   Panel content (info displays, spell details):
//     output.Reset()
//     output.AppendLine().Yellow("Label: ").White("Value").Done()
//     output.DrawPanelContent(screen, point, width, etext.AlignStart)
//
//   Scrolling buffers (game messages, logs):
//     output.AddText("Message")
//     output.DrawScrollingBuffer(screen, point, width, false, hAlign, vAlign)
//
// Pre-commit checks will FAIL if DrawText* methods are called outside this file.
// See ui-text-rendering skill for complete Output API reference.
// ============================================================================

func (o *Output) DrawText(screen *ebiten.Image, textStr string, op *ebiten.DrawImageOptions) {
	layoutOpts := &text.LayoutOptions{
		LineSpacing:  o.lineSpacing,
		PrimaryAlign: text.AlignStart,
	}

	// Get cached text image with color baked in
	textImg := GetCachedText(textStr, o.Font.TextFace, layoutOpts, o.color)

	// Draw cached image (color already applied)
	screen.DrawImage(textImg, op)
}

func (o *Output) DrawTextCenter(screen *ebiten.Image, textStr string, op *ebiten.DrawImageOptions) {
	// Use cached text with alignment adjustment
	o.drawCachedTextWithAlignment(screen, textStr, op, text.AlignCenter, text.AlignStart)
}

func (o *Output) DrawTextRightToLeft(screen *ebiten.Image, textStr string, op *ebiten.DrawImageOptions) {
	// Use cached text with alignment adjustment
	o.drawCachedTextWithAlignment(screen, textStr, op, text.AlignEnd, text.AlignStart)
}

func (o *Output) DrawScrollingBuffer(screen *ebiten.Image, point image.Point, textRectWidth int, bShowEmptyNewLines bool, horizAlign, vertAlign text.Align) {
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

		logging.Debug(logging.CategoryUI, "=== PANEL CONTENT DEBUG ===")
		logging.Debug(logging.CategoryUI, "BoundingBox: point=(%d,%d), width=%d, height=%d",
			point.X, point.Y, textRectWidth, debugHeight)
		logging.Debug(logging.CategoryUI, "Buffer: lastNonEmptyIndex=%d (physical)", lastNonEmptyIndex)
	}

	// Render lines from the requested physical range.
	currentY := float64(point.Y)
	for i := startLine; i <= lastNonEmptyIndex; i++ {
		line := o.lines[i] // Physical index, no circular offset

		lineAlign := line.Alignment
		if lineAlign == 0 {
			lineAlign = horizAlign
		}

		if len(line.Segments) > 0 {
			o.drawStyledLine(screen, i, line.Segments, float64(point.X), float64(textRectWidth), currentY, lineAlign)
		} else {
			o.drawSimpleLine(screen, line.Text, line.Color, float64(point.X), float64(textRectWidth), currentY, lineAlign)
		}

		currentY += o.lineSpacing
	}
}

// DrawScrollingBufferWithHeight renders text with optional bottom-up alignment
// If textRectHeight > 0 and vertAlign == AlignEnd, text is positioned from bottom and grows upward
// Trailing empty/whitespace-only lines are skipped during rendering (not deleted from buffer)
// Buffer naturally overwrites old lines via circular indexing when new content is added
func (o *Output) DrawScrollingBufferWithHeight(screen *ebiten.Image, point image.Point, textRectWidth, textRectHeight int, bShowEmptyNewLines bool, horizAlign, vertAlign text.Align) {
	_ = bShowEmptyNewLines
	firstNonEmptyIndex, lastNonEmptyIndex, ok := o.scrollingContentRange(textRectHeight)
	if !ok {
		return
	}

	contentLineCount := lastNonEmptyIndex - firstNonEmptyIndex + 1

	// DEBUG: Draw red bounding box around the text output area
	if o.debugOptions != nil && o.debugOptions.DialogBoundingBoxes {
		// Calculate the actual height used for rendering
		debugHeight := textRectHeight
		if debugHeight == 0 {
			// When no height specified, calculate from content
			debugHeight = int(float64(contentLineCount) * o.lineSpacing)
		}
		// Draw red rectangle outline (1 pixel wide)
		strokeWidth := float32(1.0)
		vector.StrokeRect(screen,
			float32(point.X), float32(point.Y),
			float32(textRectWidth), float32(debugHeight),
			strokeWidth, ultimacolor.EGARed, false)

		// Log buffer state for debugging
		logging.Debug(logging.CategoryUI, "=== DIALOG BOX DEBUG ===")
		logging.Debug(logging.CategoryUI, "BoundingBox: point=(%d,%d), width=%d, height=%d",
			point.X, point.Y, textRectWidth, debugHeight)
		logging.Debug(logging.CategoryUI, "Buffer: nextLineToIndex=%d, firstNonEmpty=%d, lastNonEmpty=%d, contentCount=%d",
			o.nextLineToIndex, firstNonEmptyIndex, lastNonEmptyIndex, contentLineCount)
		// Log first few lines of content
		for i := 0; i <= lastNonEmptyIndex && i < 5; i++ {
			line := o.lines[(i+o.nextLineToIndex)%o.maxLines]
			var lineText string
			if len(line.Segments) > 0 {
				var lineTextSb185 strings.Builder
				for _, seg := range line.Segments {
					lineTextSb185.WriteString(seg.Text)
				}
				lineText += lineTextSb185.String()
				lineText = "[styled] " + lineText
			} else {
				lineText = line.Text
			}
			if lineText == "" {
				lineText = "[empty]"
			}
			logging.Debug(logging.CategoryUI, "  Line[%d]: %s", i, lineText)
		}
	}

	currentY := float64(point.Y)
	if textRectHeight > 0 {
		totalTextHeight := o.GetTotalHeightForLines(contentLineCount)

		switch vertAlign {
		case text.AlignCenter:
			currentY += (float64(textRectHeight) - totalTextHeight) / centerDivisor
		case text.AlignEnd:
			currentY = float64(point.Y+textRectHeight) - totalTextHeight
		}
	}

	for i := firstNonEmptyIndex; i <= lastNonEmptyIndex; i++ {
		line := o.lines[(i+o.nextLineToIndex)%o.maxLines]

		// Use per-line alignment if set, otherwise use parameter default
		lineAlign := line.Alignment
		if lineAlign == 0 {
			lineAlign = horizAlign // Default to parameter if not set
		}

		// Check if this is a styled line (multi-color) or simple line (single color)
		if len(line.Segments) > 0 {
			// Multi-color line - use X-tracking rendering
			row := (i + o.nextLineToIndex) % o.maxLines
			o.drawStyledLine(screen, row, line.Segments, float64(point.X), float64(textRectWidth), currentY, lineAlign)
		} else {
			// Simple single-color line - use cached rendering
			o.drawSimpleLine(screen, line.Text, line.Color, float64(point.X), float64(textRectWidth), currentY, lineAlign)
		}

		currentY += o.lineSpacing
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

// getScaledFontFace returns a font face scaled by the given factor.
// The source/configuration are shared read-only; only this derived face's size changes.
func (o *Output) getScaledFontFace(scale float64) *text.GoTextFace {
	face := *o.Font.TextFace
	face.Size *= scale

	return &face
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
		scaledFace := o.getScaledFontFace(fontScale)
		width, _ := text.Measure(seg.Text, scaledFace, 0)

		return width
	}

	width, _ := text.Measure(seg.Text, o.Font.TextFace, 0)

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

// drawBitmapSegment renders a fixed-cell bitmap text segment.
func (o *Output) drawBitmapSegment(
	screen *ebiten.Image,
	font *BitmapFont,
	segText string,
	segColor color.Color,
	startX, startY float64,
) float64 {
	scale := o.getBitmapScale()
	r, g, b := ultimacolor.ToRGBFloats(segColor)

	// Calculate vertical offset to center runic text with TTF baseline
	// TTF text is positioned by baseline, but bitmap font draws from top-left
	// We want the runic text to appear centered within the line height
	// Runic scaled height = BitmapCharHeight * scale = lineSpacing
	// For visual alignment, we shift down slightly (empirical adjustment)
	yOffset := (o.lineSpacing - float64(BitmapCharHeight)*scale) / centerDivisor

	return font.DrawString(screen, segText, startX, startY+yOffset, scale, r, g, b)
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
		currentX += (textRectWidth - totalWidth) / 2
	case text.AlignEnd:
		// Anchor at right edge: startX + textRectWidth - totalWidth
		currentX += textRectWidth - totalWidth
	}

	// Render each segment using appropriate font type
	for index, seg := range segments {
		segWidth := geometry.advances[index]

		switch seg.FontType {
		case FontTypeNormal:
			// Determine font face to use (scaled if FontScale is set)
			fontScale := seg.FontScale
			if fontScale == 0 {
				fontScale = FontScaleNormal
			}

			var fontFace *text.GoTextFace
			if fontScale != FontScaleNormal {
				fontFace = o.getScaledFontFace(fontScale)
			} else {
				fontFace = o.Font.TextFace
			}

			// Render using TTF font (with optional scaling)
			entry := globalCache.getOrRenderEntry(TextRenderParams{
				Content: seg.Text, Face: fontFace,
				Options: &text.LayoutOptions{
					LineSpacing: o.lineSpacing, PrimaryAlign: text.AlignStart, SecondaryAlign: text.AlignStart,
				}, Color: seg.Color,
			})

			op := &ebiten.DrawImageOptions{}
			op.GeoM.Translate(currentX, startY)
			screen.DrawImage(entry.image, op)

			// Use measured width for advancement
			// Advancement uses the retained unpadded layout width, not image bounds.
		case FontTypeRunic:
			o.drawBitmapSegment(screen, GetBitmapFonts().Runic, seg.Text, seg.Color, currentX, startY)
		case FontTypeIBM:
			o.drawBitmapSegment(screen, GetBitmapFonts().IBM, seg.Text, seg.Color, currentX, startY)
		case FontTypeCommandPrompt:
			o.drawCommandPromptSegment(screen, seg.Color, currentX, startY)
		}

		// Advance X position - CRITICAL for correct spacing
		currentX += segWidth
	}
}

// drawSimpleLine renders a single-color line using existing cached system
func (o *Output) drawSimpleLine(screen *ebiten.Image, lineText string, lineColor color.Color, startX, textRectWidth, startY float64, horizAlign text.Align) {
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

	img := GetCachedText(lineText, o.Font.TextFace,
		&text.LayoutOptions{LineSpacing: o.lineSpacing, PrimaryAlign: text.AlignStart},
		lineColor)

	// Calculate alignment offset using proper anchors
	bounds := img.Bounds()
	imgWidth := float64(bounds.Dx())

	currentX := startX
	switch horizAlign {
	case text.AlignStart:
		// currentX already positioned correctly for left-align (anchor at left edge)
	case text.AlignCenter:
		// Center within the text rectangle
		currentX += (textRectWidth - imgWidth) / 2
	case text.AlignEnd:
		// Anchor at right edge: startX + textRectWidth - imgWidth
		currentX += textRectWidth - imgWidth
	}

	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(currentX, startY)
	screen.DrawImage(img, op)
}

// drawCachedTextWithAlignment draws cached text with proper alignment adjustment
func (o *Output) drawCachedTextWithAlignment(
	screen *ebiten.Image,
	textStr string,
	op *ebiten.DrawImageOptions,
	horizAlign, vertAlign text.Align,
) {
	o.drawCachedTextWithAlignmentStr(screen, textStr, op, horizAlign, vertAlign)
}

// drawCachedTextWithAlignmentStr is the core function that handles cached text with alignment
func (o *Output) drawCachedTextWithAlignmentStr(
	screen *ebiten.Image,
	textStr string,
	op *ebiten.DrawImageOptions,
	horizAlign, vertAlign text.Align,
) {
	// Get cached text image (always cached with AlignStart for consistency)
	layoutOpts := &text.LayoutOptions{
		LineSpacing:    o.lineSpacing,
		PrimaryAlign:   text.AlignStart, // Always cache with start alignment
		SecondaryAlign: text.AlignStart,
	}

	textImg := GetCachedText(textStr, o.Font.TextFace, layoutOpts, o.color)

	// Calculate alignment offset based on image dimensions
	bounds := textImg.Bounds()
	imgWidth := float64(bounds.Dx())
	imgHeight := float64(bounds.Dy())

	// Apply horizontal alignment offset
	switch horizAlign {
	case text.AlignCenter:
		op.GeoM.Translate(-imgWidth/2, 0)
	case text.AlignEnd:
		op.GeoM.Translate(-imgWidth, 0)
		// AlignStart needs no offset
	}

	// Apply vertical alignment offset
	switch vertAlign {
	case text.AlignCenter:
		op.GeoM.Translate(0, -imgHeight/2)
	case text.AlignEnd:
		op.GeoM.Translate(0, -imgHeight)
		// AlignStart needs no offset
	}

	// Draw cached image with adjusted position (color already applied)
	screen.DrawImage(textImg, op)
}
