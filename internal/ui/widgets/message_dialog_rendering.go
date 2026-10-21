package widgets

import (
	"image"

	"github.com/hajimehoshi/ebiten/v2"
	etext "github.com/hajimehoshi/ebiten/v2/text/v2"

	"github.com/bradhannah/UltimaVRedux/internal/visual"
	ucolor "github.com/bradhannah/UltimaVRedux/pkg/color"
)

// Draw renders the message dialog with optional dissolve effects
func (md *MessageDialog) Draw(screen *ebiten.Image) {
	if !md.visible {
		return
	}

	md.resolvePopupLayout(screen.Bounds())
	md.drawPrepared(screen)
}

func (md *MessageDialog) drawPrepared(screen *ebiten.Image) {
	screenWidth, screenHeight := screen.Bounds().Dx(), screen.Bounds().Dy()

	// Ensure buffers are created and properly sized
	md.ensureBuffersCreated(screenWidth, screenHeight)

	// Initialize dissolve effect if needed
	md.ensureDissolveEffectInitialized(screenWidth, screenHeight)

	// Re-render content if dirty
	if md.contentDirty || md.disableDirtyOptimization {
		md.renderContent()
		md.contentDirty = false
	}

	// Render to screen with or without dissolve effect (mutually exclusive paths)
	if md.enableDissolveEffects && md.dissolveEffect != nil &&
		(md.fadeState == fadingIn || md.fadeState == fadingOut) {
		md.renderWithDissolveEffect(screen)
	} else {
		md.renderDirect(screen)
	}
}

// renderContent draws border, background, and text using height-based alignment
func (md *MessageDialog) renderContent() {
	md.contentBuffer.Clear()
	borderBounds := md.frameBounds
	pixelUnit := getClassicUIPixelUnit(md.contentBuffer.Bounds().Dx(), md.contentBuffer.Bounds().Dy())
	FillClassicPixelRoundedRect(md.contentBuffer, borderBounds, pixelUnit, ucolor.DialogBackground.RGBA())
	DrawClassicPixelRoundedFrame(md.contentBuffer, borderBounds, pixelUnit, DialogRuleColor)
	scaledLineSpacing := md.getScaledLineSpacing()

	topMargin := scaledLineSpacing
	bottomMargin := scaledLineSpacing

	// Calculate horizontal text area using helper
	textRectX, textRectWidth := md.calculateTextAreaBounds()

	// Get horizontal text alignment from config
	horizAlign := md.getTextHorizontalAlignment()

	if md.hasPromptOutput() {
		// Dual output mode: content + prompt
		layout := md.calculateDualOutputLayout(borderBounds, topMargin, bottomMargin, scaledLineSpacing)

		// Determine content vertical alignment using helper
		contentVertAlign := md.getContentVerticalAlignment()

		// Draw content with height-based alignment
		md.renderOutput.DrawScrollingBufferWithHeight(md.contentBuffer, image.Point{
			X: textRectX,
			Y: int(layout.ContentStartY),
		}, textRectWidth, layout.ContentHeight, false, horizAlign, contentVertAlign)

		// Draw prompt at bottom (no height/alignment - direct positioning)
		md.renderPrompt.DrawScrollingBuffer(md.contentBuffer, image.Point{
			X: textRectX,
			Y: int(layout.PromptStartY),
		}, textRectWidth, false, horizAlign, etext.AlignStart)
	} else {
		// Single output mode: use configured content alignment
		contentStartY := float64(borderBounds.Min.Y) + topMargin
		totalHeight := borderBounds.Max.Y - borderBounds.Min.Y - int(topMargin+bottomMargin)
		// Use configured vertical alignment (not hardcoded)
		contentVertAlign := md.getContentVerticalAlignment()
		md.renderOutput.DrawScrollingBufferWithHeight(md.contentBuffer, image.Point{
			X: textRectX,
			Y: int(contentStartY),
		}, textRectWidth, totalHeight, false, horizAlign, contentVertAlign)
	}
}

// ensureBuffersCreated checks if content buffer needs recreation and creates buffers if necessary.
// This handles screen dimension changes and initial buffer creation.
func (md *MessageDialog) ensureBuffersCreated(screenWidth, screenHeight int) {
	if !md.needsBufferRecreation(screenWidth, screenHeight) {
		return
	}

	// These buffers are dialog-owned, not borrowed from the text/sprite caches.
	// Release them before replacement instead of leaving native memory to GC.
	if md.contentBuffer != nil {
		md.contentBuffer.Deallocate()
	}
	if md.maskedBuffer != nil {
		md.maskedBuffer.Deallocate()
		md.maskedBuffer = nil
	}
	if md.dissolveEffect != nil {
		md.dissolveEffect.Dispose()
	}
	bounds := image.Rect(0, 0, screenWidth, screenHeight)
	md.contentBuffer = ebiten.NewImageWithOptions(bounds, &ebiten.NewImageOptions{Unmanaged: true})
	if md.enableDissolveEffects {
		md.maskedBuffer = ebiten.NewImageWithOptions(bounds, &ebiten.NewImageOptions{Unmanaged: true})
	}
	// A dissolve mask is dimension-specific. Recreate it lazily while retaining
	// the current visual progress when a screen resize occurs mid-fade.
	md.dissolveEffect = nil
	md.contentDirty = true
}

// ensureDissolveEffectInitialized initializes the dissolve effect for either
// active fade, including a first draw that occurs after Hide and resize-mid-fade.
func (md *MessageDialog) ensureDissolveEffectInitialized(screenWidth, screenHeight int) {
	if !md.enableDissolveEffects || (md.fadeState != fadingIn && md.fadeState != fadingOut) {
		return
	}

	// Early exit if dissolve effect already initialized
	if md.dissolveEffect != nil {
		return
	}

	mode := visual.DissolveModeReveal
	progress := 1.0 - md.dissolveLevel
	if md.fadeState == fadingOut {
		mode = md.getDissolveModeForHide()
		progress = md.dissolveLevel
	}
	md.dissolveEffect = md.createDissolveEffect(screenWidth, screenHeight, mode)
	if md.dissolveEffect != nil {
		md.dissolveEffect.UpdateProgress(progress)
	}
}

// renderWithDissolveEffect applies the dissolve effect mask and draws to screen with blending.
// Handles GPU-accelerated fade animations during active fade-in/fade-out states.
func (md *MessageDialog) renderWithDissolveEffect(screen *ebiten.Image) {
	// Early exit if no active dissolve effect
	if md.dissolveEffect == nil || (md.fadeState != fadingIn && md.fadeState != fadingOut) {
		return
	}

	md.maskedBuffer.Clear()
	md.maskedBuffer.DrawImage(md.contentBuffer, nil)

	maskOp := &ebiten.DrawImageOptions{}
	maskOp.Blend = ebiten.BlendDestinationIn
	md.maskedBuffer.DrawImage(md.dissolveEffect.GetMask(), maskOp)

	md.drawComposedBuffer(screen, md.maskedBuffer)
}

// renderDirect draws the content buffer directly to screen without dissolve effects.
// Used when effects are disabled or no fade animation is active.
func (md *MessageDialog) renderDirect(screen *ebiten.Image) {
	md.drawComposedBuffer(screen, md.contentBuffer)
}

func (md *MessageDialog) drawComposedBuffer(screen, buffer *ebiten.Image) {
	target := screen
	if md.completeCardLayout && !md.presentationClip.Empty() {
		clip := md.presentationClip.Intersect(screen.Bounds())
		if clip.Empty() {
			return
		}
		clipped, ok := screen.SubImage(clip).(*ebiten.Image)
		if !ok {
			panic("MessageDialog.drawComposedBuffer: unexpected clipped image type")
		}
		target = clipped
	}
	drawOptions := &ebiten.DrawImageOptions{}
	if md.completeCardLayout {
		drawOptions.GeoM.Translate(
			md.presentationPosition.X-float64(md.bakedBounds.Min.X),
			md.presentationPosition.Y-float64(md.bakedBounds.Min.Y),
		)
	}
	target.DrawImage(buffer, drawOptions)
}
