package widgets

import (
	"fmt"
	"image"
	"math"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

var _ modalFocusListener = (*TavernShopDialog)(nil)

// SuspendTextInput releases the native session when the scene skips input Update.
// It retains modal ownership and the committed draft, discards undelivered native
// edits, and requires transition controls to release on the next allowed Update.
// It is safe to call repeatedly while Keyboard input polling is suspended.
func (d *TavernShopDialog) SuspendTextInput() {
	d.topicInputSuspended = true
	d.inputKeysReady = false
	clear(d.topicControlBlocked[:])
	d.Reset()
	d.releaseTopicField()
	d.populateBodyOutput()
}

func (d *TavernShopDialog) onDialogFocusChanged(focused bool) {
	// Preserve list navigation's held-key boundary as well as native focus.
	d.navigation.onDialogFocusChanged(focused)
	d.modalFocused = focused
	d.prepareForTransition()
}

func (d *TavernShopDialog) syncTopicFocus() {
	if d.topicField == nil {
		return
	}
	want := d.modalFocused && d.IsVisible() && d.isWindowFocused() &&
		d.mode == TavernShopConversation && d.phase == TavernShopBrowse &&
		d.state.CustomerEligible && d.inputKeysReady && !d.topicInputSuspended
	if want == d.topicField.IsFocused() {
		return
	}
	if want {
		d.topicField.Focus()
	} else {
		d.releaseTopicField()
	}
	d.populateBodyOutput()
}

func (d *TavernShopDialog) releaseTopicField() {
	if d.topicField == nil || !d.topicField.IsFocused() {
		return
	}
	// Blur may flush pending native commits. Only text already received by
	// Update belongs to this draft; late commits must not change a submitted
	// intent, revive a cancelled subject, or leak across an input suspension.
	value := d.topicField.Text()
	start, end := d.topicField.Selection()
	value, start, end = filterTavernTopic(value, start, end)
	d.topicField.Blur()
	currentStart, currentEnd := d.topicField.Selection()
	if d.topicField.Text() != value || currentStart != start || currentEnd != end {
		d.topicField.SetTextAndSelection(value, start, end)
	}
}

func (d *TavernShopDialog) setTopicInput(value string) {
	if d.topicField == nil {
		d.topicInput = []rune(value)

		return
	}
	// A transition may restore exactly the current draft. Do not restart its
	// native session or reset selection just to copy that same value.
	if d.topicField.Text() != value {
		d.topicField.SetTextAndSelection(value, len(value), len(value))
		d.selectionReversed = false
	}
}

func (d *TavernShopDialog) topicRenderingText() string {
	if d.topicField != nil {
		return d.topicField.TextForRendering()
	}

	return d.GetTopicInput()
}

// updateNativeTopic pumps the native session BEFORE speech paging, mode changes,
// editing keys or submission. A candidate-confirming Enter owns its entire press.
func (d *TavernShopDialog) updateNativeTopic() bool {
	d.syncTopicFocus()
	if d.topicField == nil || !d.topicField.IsFocused() {
		return false
	}
	previous := d.topicField.Text()
	start, end := d.topicField.Selection()
	_, _, wasComposing := d.topicField.CompositionSelection()
	handled, err := d.topicField.HandleInputWithBounds(d.topicCaretBounds)
	if err != nil {
		d.topicField.SetTextAndSelection(previous, start, end)
		d.ApplySessionError(fmt.Sprintf("Subject text input failed: %v. Press Enter or Esc to leave.", err))

		return true
	}
	_, _, composing := d.topicField.CompositionSelection()
	if handled {
		d.selectionReversed = false
	}
	// Preedit is unconstrained. Validate committed text only, without cancelling
	// a live composition on idle frames or at the character limit.
	if !composing {
		d.acceptNativeTopic(previous, start, end)
	}
	d.populateBodyOutput()
	owned := handled || wasComposing || composing
	blocked := d.blockNativeTopicControls(owned)
	if owned || blocked {
		return true
	}

	return false
}

func (d *TavernShopDialog) blockNativeTopicControls(nativeOwned bool) bool {
	blocked := false
	for _, key := range tavernTopicControlKeys() {
		if !d.isKeyPressed(key) {
			d.topicControlBlocked[key] = false
		} else if nativeOwned {
			d.topicControlBlocked[key] = true
		}
		blocked = blocked || d.topicControlBlocked[key]
	}

	return blocked
}

func tavernTopicControlKeys() []ebiten.Key {
	return []ebiten.Key{
		ebiten.KeyEnter, ebiten.KeyNumpadEnter, ebiten.KeyEscape, ebiten.KeyBackspace,
		ebiten.KeyDelete, ebiten.KeyTab, ebiten.KeyLeft, ebiten.KeyRight,
		ebiten.KeyHome, ebiten.KeyEnd, ebiten.KeyUp, ebiten.KeyDown,
		ebiten.KeyPageUp, ebiten.KeyPageDown,
	}
}

func (d *TavernShopDialog) acceptNativeTopic(previous string, start, end int) {
	value := d.topicField.Text()
	if !utf8.ValidString(value) {
		d.topicField.SetTextAndSelection(previous, start, end)

		return
	}
	// Match the rune adapter: retain at most 15 printable runes, including
	// spaces. Preserve byte selection after filtering, including replacements.
	anchor, caret := d.topicField.Selection()
	filtered, filteredAnchor, filteredCaret := filterTavernTopicReplacement(previous, value, start, end, anchor, caret)
	if filtered != value {
		d.topicField.SetTextAndSelection(filtered, filteredAnchor, filteredCaret)
	}
	if value != previous || filtered != value {
		d.inlineFeedback = ""
	}
}

// Limit the inserted/replacement text, not the untouched suffix. Truncating the
// complete native value would erase the last character on insertion at capacity.
func filterTavernTopicReplacement(previous, value string, start, end, anchor, caret int) (string, int, int) {
	prefix, suffix := previous[:start], previous[end:]
	insertEnd := len(value) - len(suffix)
	boundedPrevious, _, _ := filterTavernTopic(previous, start, end)
	if insertEnd < start || !strings.HasPrefix(value, prefix) || !strings.HasSuffix(value, suffix) ||
		boundedPrevious != previous {
		return filterTavernTopic(value, anchor, caret)
	}
	capacity := tavernTopicRuneLimit - utf8.RuneCountInString(prefix) - utf8.RuneCountInString(suffix)
	inserted, insertedAnchor, insertedCaret := filterTavernTopicRunes(
		value[start:insertEnd], anchor-start, caret-start, capacity,
	)
	anchor = tavernTopicReplacementEndpoint(anchor, start, insertEnd, len(inserted), insertedAnchor)
	caret = tavernTopicReplacementEndpoint(caret, start, insertEnd, len(inserted), insertedCaret)

	return prefix + inserted + suffix, anchor, caret
}

func tavernTopicReplacementEndpoint(endpoint, start, end, insertedLength, filteredEndpoint int) int {
	if endpoint <= start {
		return endpoint
	}
	if endpoint >= end {
		return start + insertedLength + endpoint - end
	}

	return start + filteredEndpoint
}

func filterTavernTopic(value string, anchor, caret int) (string, int, int) {
	return filterTavernTopicRunes(value, anchor, caret, tavernTopicRuneLimit)
}

func filterTavernTopicRunes(value string, anchor, caret, capacity int) (string, int, int) {
	runes := make([]rune, 0, capacity)
	bytes, filteredAnchor, filteredCaret := 0, 0, 0
	for offset, char := range value {
		if unicode.IsPrint(char) && len(runes) < capacity {
			runes = append(runes, char)
			bytes += utf8.RuneLen(char)
		}
		if offset < anchor {
			filteredAnchor = bytes
		}
		if offset < caret {
			filteredCaret = bytes
		}
	}

	return string(runes), filteredAnchor, filteredCaret
}

// Physical keys edit selection only; characters always come from the native
// text service. On platforms without native deletion, these are the fallback.
func (d *TavernShopDialog) editNativeTopic() bool {
	value := d.topicField.Text()
	start, end := d.topicField.Selection()
	if d.topicPrimaryModifierPressed() && d.keyboard.IsKeyPressedAndRegister(ebiten.KeyA) {
		d.setTopicSelection(value, 0, len(value))

		return true
	}
	for _, key := range []ebiten.Key{ebiten.KeyBackspace, ebiten.KeyDelete} {
		if d.keyboard.IsKeyPressedAndRegister(key) {
			d.deleteTopicSelection(key == ebiten.KeyBackspace)

			return true
		}
	}

	return d.moveTopicCaret(value, start, end)
}

func (d *TavernShopDialog) deleteTopicSelection(backward bool) {
	value := d.topicField.Text()
	start, end := d.topicField.Selection()
	if start == end {
		if backward && start > 0 {
			_, size := utf8.DecodeLastRuneInString(value[:start])
			start -= size
		} else if !backward && end < len(value) {
			_, size := utf8.DecodeRuneInString(value[end:])
			end += size
		}
	}
	if start != end {
		d.setTopicSelection(value[:start]+value[end:], start, start)
	}
}

func (d *TavernShopDialog) moveTopicCaret(value string, start, end int) bool {
	anchor, caret := start, end
	if d.selectionReversed {
		anchor, caret = end, start
	}
	selecting := d.isKeyPressed(ebiten.KeyShift) || d.isKeyPressed(ebiten.KeyShiftLeft) ||
		d.isKeyPressed(ebiten.KeyShiftRight)
	collapseSelection := !selecting && start != end
	switch {
	case d.keyboard.IsKeyPressedAndRegister(ebiten.KeyLeft):
		if collapseSelection {
			caret = start
		} else if caret > 0 {
			_, size := utf8.DecodeLastRuneInString(value[:caret])
			caret -= size
		}
	case d.keyboard.IsKeyPressedAndRegister(ebiten.KeyRight):
		if collapseSelection {
			caret = end
		} else if caret < len(value) {
			_, size := utf8.DecodeRuneInString(value[caret:])
			caret += size
		}
	case d.keyboard.IsKeyPressedAndRegister(ebiten.KeyHome):
		caret = 0
	case d.keyboard.IsKeyPressedAndRegister(ebiten.KeyEnd):
		caret = len(value)
	default:
		return false
	}
	if !selecting {
		anchor = caret
	}
	d.setTopicSelection(value, anchor, caret)

	return true
}

func (d *TavernShopDialog) setTopicSelection(value string, anchor, caret int) {
	d.selectionReversed = anchor > caret
	d.topicField.SetTextAndSelection(value, min(anchor, caret), max(anchor, caret))
	d.inlineFeedback = ""
	d.populateBodyOutput()
}

func (d *TavernShopDialog) topicPrimaryModifierPressed() bool {
	for _, key := range []ebiten.Key{
		ebiten.KeyControl, ebiten.KeyControlLeft, ebiten.KeyControlRight,
		ebiten.KeyMeta, ebiten.KeyMetaLeft, ebiten.KeyMetaRight,
	} {
		if d.isKeyPressed(key) {
			return true
		}
	}

	return false
}

func (d *TavernShopDialog) populateTopicLine() {
	if d.topicField == nil {
		d.bodyOutput.AppendLine().LightGray("Subject: ").LightCyan(d.GetTopicInput()).White("_").Done()

		return
	}
	value := d.topicRenderingText()
	start, end := d.topicField.Selection()
	caret := end
	if d.selectionReversed {
		caret = start
	}
	if first, last, composing := d.topicField.CompositionSelection(); composing {
		start, end = start+first, start+last
		caret = end
	}
	start, end, caret = min(start, len(value)), min(end, len(value)), min(caret, len(value))
	labelWidth, _ := d.bodyFont.MeasureString("Subject: ")
	width := max(1, d.bodyText.Dx()-int(math.Ceil(labelWidth))-1)
	visible, cursor := saveDescriptionVisibleWindow(value, caret, d.bodyFont.CharsForWidth(width))
	runes := []rune(visible)
	offset := utf8.RuneCountInString(value[:caret]) - cursor
	first := min(len(runes), max(0, utf8.RuneCountInString(value[:start])-offset))
	last := min(len(runes), max(first, utf8.RuneCountInString(value[:end])-offset))
	if !d.topicField.IsFocused() {
		first, last = 0, 0
	}
	prefix, selected, suffix := string(runes[:first]), string(runes[first:last]), string(runes[last:])
	d.bodyOutput.AppendLine().LightGray("Subject: ").LightCyan(prefix).
		Text(selected, DialogSelectedText).LightCyan(suffix).Done()
	topicY := d.bodyText.Min.Y + int(float64(d.bodyOutput.GetPanelContentLineCount()-1)*d.bodyOutput.GetLineSpacing())
	topicY = min(topicY, d.bodyText.Max.Y-int(math.Ceil(d.bodyOutput.GetFontHeight())))
	d.topicBounds = image.Rect(d.bodyText.Min.X+int(labelWidth), topicY, d.bodyText.Max.X,
		topicY+int(math.Ceil(d.bodyOutput.GetFontHeight())))
	prefixWidth, _ := d.bodyFont.MeasureString(prefix)
	selectedWidth, _ := d.bodyFont.MeasureString(selected)
	caretWidth, _ := d.bodyFont.MeasureString(string(runes[:cursor]))
	x := min(d.topicBounds.Max.X-1, d.topicBounds.Min.X+int(caretWidth))
	d.topicCaretBounds = image.Rect(x, topicY, x+1, d.topicBounds.Max.Y)
	x = d.topicBounds.Min.X + int(prefixWidth)
	d.topicSelectionBounds = image.Rect(x, topicY, x+int(selectedWidth), d.topicBounds.Max.Y).Intersect(d.topicBounds)
}

func (d *TavernShopDialog) drawTopicSelection(screen *ebiten.Image) {
	if d.topicField == nil || !d.topicField.IsFocused() || d.topicSelectionBounds.Empty() {
		return
	}
	r := d.topicSelectionBounds
	vector.FillRect(screen, float32(r.Min.X), float32(r.Min.Y), float32(r.Dx()), float32(r.Dy()),
		DialogSelectedBackground, false)
}

func (d *TavernShopDialog) drawTopicCaret(screen *ebiten.Image) {
	if d.topicField == nil || !d.topicField.IsFocused() || d.topicCaretBounds.Empty() {
		return
	}
	r := d.topicCaretBounds
	vector.FillRect(screen, float32(r.Min.X), float32(r.Min.Y), float32(r.Dx()), float32(r.Dy()),
		DialogValueColor, false)
}
