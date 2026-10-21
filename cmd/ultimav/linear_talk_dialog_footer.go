package main

import (
	"image"
	"math"

	"github.com/bradhannah/UltimaVRedux/internal/ui/widgets"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

const (
	talkFooterPaddingUnits  = 2
	excerptNoticeDurationMs = 3000
)

// Add the footer beneath the existing continuous transcript. Keep its height
// fixed across input, key waits, history and capture confirmation so inline
// typing never jumps when a hint changes. Small windows shift the whole card
// upward rather than reducing the transcript to make room for controls.
func (d *LinearTalkDialog) layoutWithFooter(bounds image.Rectangle) linearTalkDialogLayout {
	layout := calculateLinearTalkDialogLayout(bounds)
	padding := layout.pixelUnit * talkFooterPaddingUnits
	height := int(math.Ceil(d.footerOutput.GetTotalHeightForLines(talkFooterLines))) + padding*2 + layout.pixelUnit
	layout.footerDividerY = layout.frameRect.Max.Y - layout.pixelUnit
	layout.frameRect.Max.Y += height
	layout.footerRect = image.Rect(layout.continuousTranscriptRect.Min.X, layout.footerDividerY+layout.pixelUnit+padding,
		layout.continuousTranscriptRect.Max.X, layout.frameRect.Max.Y-padding-layout.pixelUnit)
	shift := image.Pt(0, min(0, bounds.Max.Y-layout.pixelUnit-layout.frameRect.Max.Y))
	layout.frameRect = layout.frameRect.Add(shift)
	layout.continuousTranscriptRect = layout.continuousTranscriptRect.Add(shift)
	layout.footerRect = layout.footerRect.Add(shift)
	layout.footerDividerY += shift.Y

	return layout
}

func (d *LinearTalkDialog) conversationFooterGuidance(width int) string {
	acknowledge := ""
	if d.historyPanel == nil && d.waitingForKeypressContinue {
		acknowledge = "[Space] Continue  "
	} else if d.historyPanel == nil && d.waitingForDismissal {
		acknowledge = "[Space] Close  "
	}
	if d.excerptNotice != "" && d.excerptNotice != "Excerpt kept" {
		// Keep the failure's prefix readable without allowing an OS path or other
		// long diagnostic to overflow the frame. Retry retains the interaction.
		notice := widgets.FitDialogLine(d.excerptNotice, d.footerOutput.Font, width)

		return notice + "\n[Tab] Retry"
	}
	keep := "[Tab] Keep excerpt"
	if d.excerptNotice == "Excerpt kept" {
		keep = "[Tab] Excerpt kept"
	}
	if d.historyPanel != nil {
		return "[Up/Dn] Exchange  [Shift+Up/Dn/Pg] Read  " + keep + "  [Esc] Back"
	}

	return acknowledge + keep + "  [Shift+Up/PgUp] History  [Shift+J] Journal"
}

func (d *LinearTalkDialog) drawConversationFooter(screen *ebiten.Image, layout linearTalkDialogLayout) {
	vector.FillRect(screen, float32(layout.frameRect.Min.X), float32(layout.footerDividerY),
		float32(layout.frameRect.Dx()), float32(layout.pixelUnit), widgets.DialogRuleColor, false)
	widgets.DrawKeyboardFooter(screen, layout.footerRect, d.conversationFooterGuidance(layout.footerRect.Dx()),
		talkFooterLines, d.footerOutput)
}

func (d *LinearTalkDialog) setExcerptNotice(message string, brief bool) {
	d.excerptNotice = message
	d.excerptNoticeUntilMs = 0
	if brief {
		d.excerptNoticeUntilMs = d.gameScene.gameState.ElapsedMs + excerptNoticeDurationMs
	}
}

func (d *LinearTalkDialog) expireExcerptNotice() {
	if d.excerptNoticeUntilMs > 0 && d.gameScene.gameState.ElapsedMs >= d.excerptNoticeUntilMs {
		d.setExcerptNotice("", false)
	}
}
