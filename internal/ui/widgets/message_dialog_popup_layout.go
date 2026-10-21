package widgets

import (
	"image"
	"math"

	"github.com/bradhannah/UltimaVRedux/internal/display"
)

const (
	messagePopupCenterDivisor = 2
	messagePopupMarginLines   = 2
)

// resolvePopupLayout retains the original output and placement so a dialog can
// cross viewport boundaries or resize without losing styled text or input state.
func (md *MessageDialog) resolvePopupLayout(screen image.Rectangle) {
	if md.completeCardLayout {
		return
	}
	viewport := image.Rect(0, 0, 0, 0)
	if !md.screenRelative {
		viewport = display.GetManager().MapPopupViewport(screen)
	}
	if md.lastScreenWidth != screen.Dx() || md.lastScreenHeight != screen.Dy() {
		md.config = md.baseConfig
		md.initializeVisualElements()
		md.contentDirty = true
	}
	if md.popupViewport != viewport {
		md.popupViewport = viewport
		md.contentDirty = true
	}
	if !md.contentDirty && md.renderOutput != nil {
		return
	}

	previousConfig := md.config
	md.config = md.baseConfig
	md.renderOutput = md.output
	md.renderPrompt = md.promptOutput
	if !viewport.Empty() {
		width := min(md.baseConfig.EndPercentX-md.baseConfig.StartPercentX,
			float64(viewport.Dx())/float64(screen.Dx()))
		centerX := float64(viewport.Min.X+viewport.Max.X-messagePopupCenterDivisor*screen.Min.X) /
			(messagePopupCenterDivisor * float64(screen.Dx()))
		md.config.StartPercentX = centerX - width/messagePopupCenterDivisor
		md.config.EndPercentX = centerX + width/messagePopupCenterDivisor
		textWidth := max(1, int(math.Floor((width-2*messagePercentTextIndentFromBorder)*float64(screen.Dx()))))
		md.renderOutput = md.output.WrappedCopyForWidth(textWidth)
		lines := md.renderOutput.GetPanelContentLineCount()
		if md.hasPromptOutput() {
			md.renderPrompt = md.promptOutput.WrappedCopyForWidth(textWidth)
			md.config.PromptLines = max(md.baseConfig.PromptLines, md.renderPrompt.GetPanelContentLineCount())
			lines += md.config.PromptLines
		}
		// Grow vertically for wrapped feedback while retaining its vertical anchor.
		height := max(md.baseConfig.EndPercentY-md.baseConfig.StartPercentY,
			float64(lines+messagePopupMarginLines)*md.getScaledLineSpacing()/float64(screen.Dy()))
		centerY := (md.baseConfig.StartPercentY + md.baseConfig.EndPercentY) / messagePopupCenterDivisor
		height = min(height, messagePopupCenterDivisor*min(centerY, 1-centerY))
		md.config.StartPercentY = centerY - height/messagePopupCenterDivisor
		md.config.EndPercentY = centerY + height/messagePopupCenterDivisor
	}
	if md.config != previousConfig {
		md.updateFrameBounds()
	}
}

// MeasureCompleteCard reflows retained content to the available width and sizes
// the border for every resulting row. It is opt-in and used by transient action
// feedback; ordinary MessageDialogs retain their anchor-limited layout policy.
func (md *MessageDialog) MeasureCompleteCard(
	screen, available image.Rectangle,
	preferredCenterX int,
) image.Rectangle {
	if screen.Empty() || available.Empty() {
		panic("MessageDialog.MeasureCompleteCard: screen and available bounds are required")
	}
	available = available.Intersect(screen)
	if available.Empty() {
		panic("MessageDialog.MeasureCompleteCard: available bounds must intersect the screen")
	}

	if md.lastScreenWidth != screen.Dx() || md.lastScreenHeight != screen.Dy() {
		md.config = md.baseConfig
		md.initializeVisualElements()
	}

	width := int(math.Round((md.baseConfig.EndPercentX - md.baseConfig.StartPercentX) * float64(screen.Dx())))
	width = min(max(1, width), available.Dx())
	left := preferredCenterX - width/messagePopupCenterDivisor
	left = max(available.Min.X, min(left, available.Max.X-width))
	textWidth := max(1, width-int(math.Round(2*messagePercentTextIndentFromBorder*float64(screen.Dx()))))
	md.renderOutput = md.output.WrappedCopyForWidth(textWidth)
	lines := max(md.baseConfig.TextLines, md.renderOutput.GetPanelContentLineCount())
	md.renderPrompt = md.promptOutput
	promptLines := md.baseConfig.PromptLines
	if md.hasPromptOutput() {
		md.renderPrompt = md.promptOutput.WrappedCopyForWidth(textWidth)
		promptLines = max(promptLines, md.renderPrompt.GetPanelContentLineCount())
		lines += promptLines
	}
	height := int(math.Round((md.baseConfig.EndPercentY - md.baseConfig.StartPercentY) * float64(screen.Dy())))
	height = max(height, int(math.Ceil(float64(lines+messagePopupMarginLines)*md.getScaledLineSpacing())))
	top := available.Min.Y

	md.config = md.baseConfig
	md.config.PromptLines = promptLines
	md.config.StartPercentX = float64(left-screen.Min.X) / float64(screen.Dx())
	md.config.EndPercentX = float64(left+width-screen.Min.X) / float64(screen.Dx())
	md.config.StartPercentY = float64(top-screen.Min.Y) / float64(screen.Dy())
	md.config.EndPercentY = float64(top+height-screen.Min.Y) / float64(screen.Dy())
	md.updateFrameBounds()
	md.bakedBounds = md.frameBounds
	md.presentationPosition.X = float64(md.bakedBounds.Min.X)
	md.presentationPosition.Y = float64(md.bakedBounds.Min.Y)
	md.presentationClip = available
	md.completeCardLayout = true
	md.popupViewport = image.Rect(0, 0, 0, 0)
	md.contentDirty = true

	return md.bakedBounds
}
