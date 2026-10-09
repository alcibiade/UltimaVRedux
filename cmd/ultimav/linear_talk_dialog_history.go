package main

import (
	"image"

	"github.com/bradhannah/UltimaVRedux/internal/display"
	"github.com/bradhannah/UltimaVRedux/internal/text"
	"github.com/bradhannah/UltimaVRedux/internal/ui/widgets"
	"github.com/hajimehoshi/ebiten/v2"
	etext "github.com/hajimehoshi/ebiten/v2/text/v2"
)

func (d *LinearTalkDialog) ensureTalkLayout() {
	width, height := display.GetManager().GetCurrentSize()
	if width != d.lastScreenWidth || height != d.lastScreenHeight {
		d.initializeResizeableVisualElements()
	}
}

// History owns only presentation input. Pauses retain their remaining duration;
// no pending speech, engine continuation, or callback output is flushed here.
func (d *LinearTalkDialog) updateHistoryInput() bool {
	keyboard := d.gameScene.keyboard
	if d.historyPanel == nil {
		return d.openHistoryFromKeyboard()
	}
	if keyboard.IsKeyPressedAndRegister(ebiten.KeyEscape) || keyboard.IsKeyPressedAndRegister(ebiten.KeyEnd) {
		if d.isPausing {
			d.pauseStartMs += d.gameScene.gameState.ElapsedMs - d.historyStartedMs
		}
		d.historyPanel = nil
		keyboard.PrepareForTransition(widgets.KeyboardTransitionDelayMs)

		return true
	}
	key, _ := d.historyNavigation.Poll(keyboard,
		ebiten.KeyUp, ebiten.KeyDown, ebiten.KeyPageUp, ebiten.KeyPageDown, ebiten.KeyHome)
	if key == nil {
		return true
	}
	if len(d.gameScene.lastConversation.Exchanges) > 0 {
		switch *key { //nolint:exhaustive // Arrows select exchanges; page keys scroll the selected text.
		case ebiten.KeyUp:
			d.historyExchange = max(0, d.historyExchange-1)
			d.showJournalHistoryExchange()
		case ebiten.KeyDown:
			d.historyExchange = min(len(d.gameScene.lastConversation.Exchanges)-1, d.historyExchange+1)
			d.showJournalHistoryExchange()
		}
	}
	layout := d.layoutWithFooter(image.Rect(0, 0, d.lastScreenWidth, d.lastScreenHeight))
	page := max(1, d.historyPanel.GetLineCapacity(layout.continuousTranscriptRect.Dy()))
	last := max(0, d.historyPanel.GetPanelContentLineCount()-page)
	switch *key { //nolint:exhaustive // Exchange navigation is handled above.
	case ebiten.KeyHome:
		d.historyRow = 0
	case ebiten.KeyPageUp:
		d.historyRow = max(0, d.historyRow-page)
	case ebiten.KeyPageDown:
		d.historyRow = min(last, d.historyRow+page)
	}

	return true
}

func (d *LinearTalkDialog) openHistoryFromKeyboard() bool {
	keyboard := d.gameScene.keyboard
	key, owned := d.historyNavigation.Poll(keyboard,
		ebiten.KeyUp, ebiten.KeyDown, ebiten.KeyPageUp, ebiten.KeyPageDown, ebiten.KeyHome)
	if key == nil || (*key != ebiten.KeyPageUp && *key != ebiten.KeyHome) {
		return owned
	}
	pageUp := *key == ebiten.KeyPageUp
	d.historyPanel = d.Output.RetainedHistoryPanel()
	d.historyStartedMs = d.gameScene.gameState.ElapsedMs
	d.historyRow = 0
	if pageUp {
		layout := d.layoutWithFooter(image.Rect(0, 0, d.lastScreenWidth, d.lastScreenHeight))
		page := max(1, d.historyPanel.GetLineCapacity(layout.continuousTranscriptRect.Dy()))
		d.historyRow = max(0, d.historyPanel.GetPanelContentLineCount()-2*page)
	}
	if len(d.gameScene.lastConversation.Exchanges) > 0 {
		d.historyExchange = 0
		if pageUp {
			d.historyExchange = len(d.gameScene.lastConversation.Exchanges) - 1
		}
		d.showJournalHistoryExchange()
	}
	keyboard.PrepareForTransition(widgets.KeyboardTransitionDelayMs)

	return true
}

func (d *LinearTalkDialog) showJournalHistoryExchange() {
	exchange := d.gameScene.lastConversation.Exchanges[d.historyExchange]
	d.historyPanel = text.NewOutput(d.font, d.Output.GetLineSpacing(), talkHistoryMaxBytes, talkMaxCharsInput)
	d.historyPanel.AddText(exchange.Title())
	segments := make([]text.ColorSegment, 0, len(exchange.Passages))
	for _, passage := range exchange.Passages {
		font := text.FontTypeNormal
		if passage.Runic {
			font = text.FontTypeRunic
		}
		segments = append(segments, text.ColorSegment{
			Text: passage.Text, Color: widgets.DialogValueColor, FontType: font, FontScale: text.FontScaleNormal,
		})
	}
	d.historyPanel.AddStyledText(segments, etext.AlignStart)
	d.historyRow = 0
}
