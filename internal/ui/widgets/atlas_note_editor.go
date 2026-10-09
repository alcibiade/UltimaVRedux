package widgets

import (
	"fmt"
	"image"
	"image/color"
	"math"
	"strings"
	"unicode/utf8"

	"github.com/bradhannah/UltimaVRedux/internal/display"
	"github.com/bradhannah/UltimaVRedux/internal/journal"
	"github.com/bradhannah/UltimaVRedux/internal/text"
	ucolor "github.com/bradhannah/UltimaVRedux/pkg/color"
	"github.com/bradhannah/UltimaVRedux/pkg/input"
	"github.com/hajimehoshi/ebiten/v2"
	etext "github.com/hajimehoshi/ebiten/v2/text/v2"
)

// AtlasNoteEditorConfig supplies detached initial text and scene-owned commit/cancel.
type AtlasNoteEditorConfig struct {
	Record   journal.AtlasNoteRecord
	Context  string
	Deleting bool
	Keyboard *input.Keyboard
	Submit   func(label, description string) error
	Cancel   func()
}

// AtlasNoteEditor is a keyboard-only, two-field multiline editor. Its owner
// removes the exact child on successful submit/cancel; failed commits keep focus.
type AtlasNoteEditor struct {
	config  AtlasNoteEditorConfig
	fields  [2][]rune
	cursors [2]int
	// At a shared soft-wrap offset, upstream keeps the caret on the preceding row.
	upstream                  [2]bool
	focus                     int
	focused, finished         bool
	message                   string
	appendChars               func([]rune) []rune
	font                      *text.UltimaFont
	output                    *text.Output
	validation                *text.Output
	width, height, lineHeight int
	pane                      image.Rectangle
	fieldRects                [2]image.Rectangle
	hintRect, validationRect  image.Rectangle
	scroll                    [2]int
}

const (
	atlasEditorPageLines       = 5
	atlasEditorValidationLines = 2
)

// NewAtlasNoteEditor creates a scene-owned authoring or deletion-confirmation child.
func NewAtlasNoteEditor(config AtlasNoteEditorConfig) *AtlasNoteEditor {
	if config.Keyboard == nil || config.Submit == nil || config.Cancel == nil {
		panic("atlas editor requires input and lifecycle owners")
	}
	editor := new(AtlasNoteEditor)
	editor.config, editor.focused, editor.appendChars = config, true, ebiten.AppendInputChars
	editor.fields = [2][]rune{[]rune(config.Record.Label), []rune(config.Record.Description)}
	for i := range editor.fields {
		editor.cursors[i] = len(editor.fields[i])
	}
	editor.ensureLayout()

	return editor
}

func (e *AtlasNoteEditor) onDialogFocusChanged(focused bool) { e.focused = focused }

// Update drains pending text before applying the focused editor's physical controls.
func (e *AtlasNoteEditor) Update() {
	if e.finished || !e.focused || e.config.Keyboard.IsWaitingForRelease() {
		return
	}
	e.ensureLayout()
	keyboard := e.config.Keyboard
	for _, key := range []ebiten.Key{ebiten.KeyEscape, ebiten.KeyTab, ebiten.KeyEnter, ebiten.KeyNumpadEnter,
		ebiten.KeyBackspace, ebiten.KeyDelete, ebiten.KeyLeft, ebiten.KeyRight, ebiten.KeyUp, ebiten.KeyDown,
		ebiten.KeyHome, ebiten.KeyEnd, ebiten.KeyPageUp, ebiten.KeyPageDown} {
		if keyboard.IsTextKeyPressedAndRegister(key) {
			if key != ebiten.KeyEscape && !e.config.Deleting && !e.drainPlatformText() {
				return
			}
			e.handleKey(key)

			return
		}
	}
	if !e.config.Deleting && !keyboard.IsModifierPressed(input.ModifierCtrl) &&
		!keyboard.IsModifierPressed(input.ModifierAlt) && !keyboard.IsModifierPressed(input.ModifierMeta) {
		e.drainPlatformText()
	}
}

// Owned physical controls supply paragraph breaks and field changes themselves.
// Printable platform text belongs to the old field even when Ctrl+Enter is now
// held; unrelated modifier shortcuts remain suppressed by Update's ordinary path.
func (e *AtlasNoteEditor) drainPlatformText() bool {
	runes := e.appendChars(nil)
	printable := runes[:0]
	for _, char := range runes {
		if char >= ' ' && char != '\u007f' {
			printable = append(printable, char)
		}
	}

	return e.insert(printable)
}

func (e *AtlasNoteEditor) handleKey(key ebiten.Key) {
	if key == ebiten.KeyEscape {
		e.finished = true
		e.config.Cancel()

		return
	}
	if e.config.Deleting {
		if key == ebiten.KeyEnter || key == ebiten.KeyNumpadEnter {
			e.submit()
		}

		return
	}
	if key == ebiten.KeyTab {
		e.focus = 1 - e.focus

		return
	}
	if key == ebiten.KeyEnter || key == ebiten.KeyNumpadEnter {
		if e.config.Keyboard.IsModifierPressed(input.ModifierCtrl) {
			e.submit()
		} else if e.focus == 1 {
			e.insert([]rune{'\n'})
		}

		return
	}
	e.editAtCursor(key)
}

//nolint:cyclop // Keep each physical edit beside its caret-affinity reset; vertical moves retain affinity.
func (e *AtlasNoteEditor) editAtCursor(key ebiten.Key) {
	index := e.focus
	cursor, content := e.cursors[index], e.fields[index]
	switch key { //nolint:exhaustive // Only editing/navigation keys enter this dispatcher.
	case ebiten.KeyLeft:
		e.upstream[index] = false
		e.cursors[index] = max(0, cursor-1)
	case ebiten.KeyRight:
		e.upstream[index] = false
		e.cursors[index] = min(len(content), cursor+1)
	case ebiten.KeyBackspace:
		e.upstream[index] = false
		if cursor > 0 {
			e.fields[index] = append(content[:cursor-1], content[cursor:]...)
			e.cursors[index]--
		}
	case ebiten.KeyDelete:
		e.upstream[index] = false
		if cursor < len(content) {
			e.fields[index] = append(content[:cursor], content[cursor+1:]...)
		}
	case ebiten.KeyHome:
		e.upstream[index] = false
		for cursor > 0 && content[cursor-1] != '\n' {
			cursor--
		}
		e.cursors[index] = cursor
	case ebiten.KeyEnd:
		e.upstream[index] = false
		for cursor < len(content) && content[cursor] != '\n' {
			cursor++
		}
		e.cursors[index] = cursor
	case ebiten.KeyUp:
		e.moveVertical(-1)
	case ebiten.KeyDown:
		e.moveVertical(1)
	case ebiten.KeyPageUp:
		e.moveVertical(-atlasEditorPageLines)
	case ebiten.KeyPageDown:
		e.moveVertical(atlasEditorPageLines)
	}
}

func (e *AtlasNoteEditor) insert(runes []rune) bool {
	if len(runes) == 0 {
		return true
	}
	limit := journal.MaxAtlasNoteLabelRunes
	if e.focus == 1 {
		limit = journal.MaxAtlasNoteDescriptionRunes
	}
	for _, r := range runes {
		if r < ' ' || r > '~' {
			if e.focus == 1 && r == '\n' {
				continue
			}
			e.message = "Use printable game-font characters."

			return false
		}
	}
	if len(e.fields[e.focus])+len(runes) > limit {
		e.message = fmt.Sprintf("This field allows at most %d characters.", limit)

		return false
	}
	cursor := e.cursors[e.focus]
	content := append([]rune(nil), e.fields[e.focus][:cursor]...)
	content = append(content, runes...)
	content = append(content, e.fields[e.focus][cursor:]...)
	e.fields[e.focus], e.cursors[e.focus] = content, cursor+len(runes)
	e.upstream[e.focus] = false
	e.message = ""

	return true
}

func (e *AtlasNoteEditor) submit() {
	label, description := strings.TrimSpace(string(e.fields[0])), string(e.fields[1])
	if !e.config.Deleting {
		if err := journal.ValidateAtlasNoteText(label, description); err != nil {
			e.message = err.Error()

			return
		}
	}
	if err := e.config.Submit(label, description); err != nil {
		e.message = err.Error()

		return
	}
	e.finished = true
}

type atlasEditorLine struct{ start, end int }

// Measured wrapping retains original rune offsets, including empty paragraphs,
// so arrow motion, cursor rendering and scroll all agree with the actual text.
func (e *AtlasNoteEditor) lines(field int) []atlasEditorLine {
	content := e.fields[field]
	width := e.fieldLayout(field).textWidth
	start := 0
	var lines []atlasEditorLine
	for index, r := range content {
		if r == '\n' {
			lines = append(lines, atlasEditorLine{start, index})
			start = index + 1

			continue
		}
		measured, _ := e.font.MeasureString(string(content[start : index+1]))
		if measured > float64(width) && index > start {
			lines = append(lines, atlasEditorLine{start, index})
			start = index
		}
	}

	return append(lines, atlasEditorLine{start, len(content)})
}

func atlasEditorCursorLine(lines []atlasEditorLine, cursor int) int {
	for i, line := range lines {
		if cursor <= line.end && (i == len(lines)-1 || cursor < lines[i+1].start) {
			return i
		}
	}

	return len(lines) - 1
}

func (e *AtlasNoteEditor) moveVertical(delta int) {
	lines := e.lines(e.focus)
	row := e.cursorLine(e.focus, lines)
	column := e.cursors[e.focus] - lines[row].start
	targetRow := max(0, min(len(lines)-1, row+delta))
	target := lines[targetRow]
	e.cursors[e.focus] = min(target.end, target.start+column)
	e.upstream[e.focus] = targetRow+1 < len(lines) && target.end == lines[targetRow+1].start &&
		e.cursors[e.focus] == target.end
}

func (e *AtlasNoteEditor) cursorLine(field int, lines []atlasEditorLine) int {
	cursor := e.cursors[field]
	row := atlasEditorCursorLine(lines, cursor)
	if e.upstream[field] && row > 0 && cursor == lines[row].start && cursor == lines[row-1].end {
		return row - 1
	}

	return row
}

//nolint:mnd,varnamelen // Measured field rows and proportional pane margins scale with logical w/h.
func (e *AtlasNoteEditor) ensureLayout() {
	w, h := display.GetManager().GetCurrentSize()
	if e.font != nil && e.width == w && e.height == h {
		e.layoutEditorFooter()

		return
	}
	e.width, e.height = w, h
	config := FontConfig{Size: FontSizeMainScreen, LineSpacing: 0,
		LineSpacingMultiplier: 1.15, MaxLines: 32, MaxChars: 128}
	e.font, e.output = GetScaledFontWithOutput(config)
	_, e.validation = GetScaledFontWithOutput(config)
	e.lineHeight = int(math.Ceil(e.output.GetTotalHeightForLines(1))) + 2
	margin := max(8, w/80)
	e.pane = image.Rect(w/8, h/10, w-w/8, h-h/10)
	left, right := e.pane.Min.X+margin, e.pane.Max.X-margin
	top := e.pane.Min.Y + margin + 4*e.lineHeight
	e.fieldRects[0] = image.Rect(left, top, right, top+2*e.lineHeight)
	e.fieldRects[1] = image.Rect(left, top+3*e.lineHeight, right, e.pane.Max.Y-margin-4*e.lineHeight)
	e.layoutEditorFooter()
}

// Reserve the exact physical hint rows and at least two validation rows. Longer
// operation errors expand their own block instead of painting across the hints.
func (e *AtlasNoteEditor) layoutEditorFooter() {
	left, right := e.fieldRects[0].Min.X, e.fieldRects[0].Max.X
	margin := left - e.pane.Min.X
	width := right - left
	e.buildEditorHints(width)
	hintHeight := int(math.Ceil(e.output.GetTotalHeightForLines(e.output.GetPanelContentLineCount())))
	bottom := e.pane.Max.Y - margin
	e.hintRect = image.Rect(left, bottom-hintHeight, right, bottom)
	e.validation.Reset()
	e.validation.SetMaxCharsPerLine(max(1, e.font.CharsForWidth(width)))
	e.validation.AppendLine().Text(e.message, DialogValueColor).Done()
	validationLines := max(atlasEditorValidationLines, e.validation.GetPanelContentLineCount())
	validationHeight := int(math.Ceil(e.validation.GetTotalHeightForLines(validationLines)))
	e.validationRect = image.Rect(left, e.hintRect.Min.Y-margin-validationHeight, right, e.hintRect.Min.Y-margin)
	e.fieldRects[1].Max.Y = e.validationRect.Min.Y - margin
}

// Pack whole key/action pairs using actual font widths. Output receives explicit
// physical rows with exact rune budgets, so it cannot split Esc from Cancel later.
func (e *AtlasNoteEditor) buildEditorHints(width int) {
	hints := [][2]string{{"Tab/Shift+Tab", "Field"}, {atlasEnterKey, "Paragraph"},
		{"Ctrl+Enter", "Save"}, {atlasEscapeKey, "Cancel"}}
	if e.config.Deleting {
		hints = [][2]string{{atlasEnterKey, "Delete"}, {atlasEscapeKey, "Keep"}}
	}
	const gap = "   "
	e.output.Reset()
	start := 0
	for end := 1; end <= len(hints); end++ {
		measured, _ := e.font.MeasureString(atlasHelpText(hints[start:end], gap))
		if measured > float64(width) && end-start > 1 {
			e.appendEditorHintRow(hints[start:end-1], gap)
			start = end - 1
		}
	}
	e.appendEditorHintRow(hints[start:], gap)
}

func (e *AtlasNoteEditor) appendEditorHintRow(hints [][2]string, gap string) {
	e.output.SetMaxCharsPerLine(max(1, utf8.RuneCountInString(atlasHelpText(hints, gap))))
	line := e.output.AppendLine()
	for index, hint := range hints {
		if index > 0 {
			line.Text(gap, DialogControlHintColor)
		}
		line.Text(hint[0], DialogControlKeyColor).Text(" "+hint[1], DialogControlHintColor)
	}
	line.Done()
}

//nolint:mnd // Right text inset matches the editor pane's interior budget.
func (e *AtlasNoteEditor) drawLine(screen *ebiten.Image, value string, point image.Point, colour color.Color) {
	atlasSingleLine(e.output, value, colour)
	e.output.DrawPanelContent(screen, point, e.pane.Max.X-point.X-8, etext.AlignStart)
}

// Draw renders the editor's fields, validation feedback and keyboard hints.
//
//nolint:mnd // Header rows use the same measured line height as the field captions.
func (e *AtlasNoteEditor) Draw(screen *ebiten.Image) {
	if e.finished {
		return
	}
	e.ensureLayout()
	unit := ClassicUIPixelUnit(e.width, e.height)
	FillClassicPixelRoundedRect(screen, e.pane, unit, ucolor.DialogBackground.RGBA())
	DrawClassicPixelRoundedFrame(screen, e.pane, unit, DialogLabelColor)
	target := atlasSubImage(screen, e.pane.Inset(unit))
	point := e.pane.Min.Add(image.Pt(12, 12))
	title := "MARK CURRENT POSITION"
	if e.config.Record.ID != 0 {
		title = "EDIT MAP NOTE"
	}
	if e.config.Deleting {
		title = "Delete this map note?"
	}
	e.drawLine(target, title, point, DialogHeaderColor)
	point.Y += e.lineHeight
	if e.config.Deleting {
		e.drawLine(target, e.config.Record.Label, point, DialogValueColor)
	} else {
		e.drawFields(target, point)
	}
	e.drawEditorFooter(target)
}

//nolint:mnd // Coordinate and field captions share the pane's paired twelve-pixel insets.
func (e *AtlasNoteEditor) drawFields(target *ebiten.Image, point image.Point) {
	e.output.Reset()
	e.output.SetMaxCharsPerLine(max(1, e.font.CharsForWidth(e.pane.Dx()-24)))
	e.output.AppendLine().Text("Coordinates: ", DialogLabelColor).
		Text(atlasCoordinates(e.config.Record.Position), DialogValueColor).Done()
	e.output.DrawPanelContent(target, point, e.pane.Dx()-24, etext.AlignStart)
	point.Y += e.lineHeight
	e.drawLine(target, e.config.Context, point, DialogValueColor)
	for field, name := range []string{"Map label", atlasNoteDescriptionLabel} {
		limit := journal.MaxAtlasNoteLabelRunes
		if field == 1 {
			limit = journal.MaxAtlasNoteDescriptionRunes
		}
		rect := e.fieldRects[field]
		e.output.Reset()
		e.output.SetMaxCharsPerLine(64)
		e.output.AppendLine().Text(name+": ", DialogLabelColor).
			Textf("%d/%d", DialogValueColor, len(e.fields[field]), limit).Done()
		e.output.DrawPanelContent(target, image.Pt(rect.Min.X, rect.Min.Y-e.lineHeight), rect.Dx(), etext.AlignStart)
		e.drawField(target, field)
	}
}

func (e *AtlasNoteEditor) drawEditorFooter(target *ebiten.Image) {
	e.validation.DrawPanelContent(target, e.validationRect.Min, e.validationRect.Dx(), etext.AlignStart)
	e.buildEditorHints(e.hintRect.Dx())
	e.output.DrawPanelContent(target, e.hintRect.Min, e.hintRect.Dx(), etext.AlignStart)
}

func (e *AtlasNoteEditor) drawField(screen *ebiten.Image, field int) {
	rect := e.fieldRects[field]
	layout := e.fieldLayout(field)
	colour := DialogRuleLightColor
	if field == e.focus {
		colour = DialogLabelColor
	}
	DrawClassicPixelRoundedFrame(screen, rect, layout.pixelUnit, colour)
	if layout.capacity == 0 {
		return
	}
	target := atlasSubImage(screen, layout.content)
	lines, row := e.prepareFieldScroll(field, layout)
	for lineIndex := e.scroll[field]; lineIndex < min(len(lines), e.scroll[field]+layout.capacity); lineIndex++ {
		line := lines[lineIndex]
		point := layout.lineOrigin(lineIndex - e.scroll[field])
		atlasSingleLine(e.output, string(e.fields[field][line.start:line.end]), DialogValueColor)
		e.output.DrawPanelContent(target, point, layout.textWidth, etext.AlignStart)
		if field == e.focus && lineIndex == row {
			width, _ := e.font.MeasureString(string(e.fields[field][line.start:e.cursors[field]]))
			fillAtlasRect(target, layout.caretRect(lineIndex-e.scroll[field], width), DialogLabelColor)
		}
	}
}

func (e *AtlasNoteEditor) prepareFieldScroll(field int, layout atlasEditorFieldLayout) ([]atlasEditorLine, int) {
	lines := e.lines(field)
	row := e.cursorLine(field, lines)
	if layout.capacity > 0 {
		e.scroll[field] = min(max(e.scroll[field], row-layout.capacity+1), row)
	}

	return lines, row
}
