package widgets

import (
	"fmt"
	"image"
	"math"
	"runtime"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/exp/textinput"
	etext "github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"

	"github.com/bradhannah/UltimaVRedux/internal/display"
	"github.com/bradhannah/UltimaVRedux/internal/logging"
	platformclipboard "github.com/bradhannah/UltimaVRedux/internal/platform/clipboard"
	"github.com/bradhannah/UltimaVRedux/internal/sprites"
	"github.com/bradhannah/UltimaVRedux/internal/text"
	ucolor "github.com/bradhannah/UltimaVRedux/pkg/color"
	"github.com/bradhannah/UltimaVRedux/pkg/input"
)

const (
	legacyPathModalStartX        = 0.15
	legacyPathModalEndX          = 0.85
	legacyPathModalStartY        = 0.38
	legacyPathModalEndY          = 0.62
	legacyPathReferenceWidth     = 1920
	legacyPathContentPadding     = 25
	legacyPathMinimumPadding     = 8
	legacyPathFieldPaddingX      = 16
	legacyPathFieldPaddingY      = 8
	legacyPathVerticalSpacing    = 16
	legacyPathMinimumPaddingX    = 6
	legacyPathMinimumPaddingY    = 4
	legacyPathMinimumSpacing     = 6
	legacyPathHintMaximumLines   = 2
	legacyPathCenterDivisor      = 2
	legacyPathTailMarker         = "..."
	legacyPathValidationLines    = 2
	legacyEntryBackspaceDelay    = 21 // 350ms at Ebitengine's default 60 update ticks/second.
	legacyEntryBackspaceInterval = 3  // 50ms; no global keyboard repeat changes.
	legacyEntryInputErrorMessage = "Text input failed. Press Esc to go back."
)

var (
	_ ModalDialog        = (*LegacyPathEntryModal)(nil)
	_ modalFocusListener = (*LegacyPathEntryModal)(nil)
)

// LegacyPathEntryCallback receives the current path and whether Enter submitted it.
// A false submitted value means the modal was cancelled.
type LegacyPathEntryCallback func(text string, submitted bool)

// LegacyPathEntryModalConfig configures a LegacyPathEntryModal. The widget only
// collects text; path and file validation remain the caller's responsibility.
type LegacyPathEntryModalConfig struct {
	Prompt            string
	Guidance          string
	SubmitLabel       string
	CancelLabel       string
	InitialPath       string
	Keyboard          *input.Keyboard
	OnResult          LegacyPathEntryCallback
	Validate          func(string) string
	ReadClipboardText func() (string, error)
	ClipboardEnabled  bool
	IsKeyPressed      func(ebiten.Key) bool
}

// SaveDescriptionEntryModalConfig configures optional checkpoint text and the
// reviewed overwrite warning. Persistence remains the caller's responsibility.
type SaveDescriptionEntryModalConfig struct {
	Title        string
	Notice       string
	InitialText  string
	Keyboard     *input.Keyboard
	OnResult     LegacyPathEntryCallback
	Validate     func(string) string
	MaxRunes     int
	IsKeyPressed func(ebiten.Key) bool
}

// LegacyPathEntryResult is the typed value returned through ModalDialog.GetResult.
type LegacyPathEntryResult struct {
	Text      string
	Submitted bool
}

type legacyPathTextField interface {
	Focus()
	Blur()
	IsFocused() bool
	HandleInputWithBounds(bounds image.Rectangle) (handled bool, err error)
	Text() string
	TextForRendering() string
	Selection() (startInBytes, endInBytes int)
	CompositionSelection() (startInBytes, endInBytes int, ok bool)
	SetTextAndSelection(text string, selectionStartInBytes, selectionEndInBytes int)
}

// LegacyPathEntryModal is a centered, single-line platform text input modal.
// Push it with DialogStack.PushModalDialog so platform input focus follows the
// active modal. Use AutoCloseAndPop when the modal should remove itself.
type LegacyPathEntryModal struct {
	prompt   string
	keyboard *input.Keyboard
	onResult LegacyPathEntryCallback
	validate func(string) string
	field    legacyPathTextField

	readClipboardText           func() (string, error)
	clipboardEnabled            bool
	isKeyPressed                func(ebiten.Key) bool
	isWindowFocused             func() bool
	validationMessage           string
	lastSingleLineText          string
	clearLineHeld               bool
	backspaceHeld               bool
	backspaceTicks              int
	compositionBackspaceBlocked bool
	compositionClearBlocked     bool
	selectionReversed           bool
	enterBlocked                bool
	saveDescription             bool
	maxRunes                    int
	title                       string
	notice                      string
	guidance                    string
	submitLabel                 string
	cancelLabel                 string
	titleOutput                 *text.Output
	noticeOutput                *text.Output

	autoCloseStack *DialogStack
	finished       bool
	result         LegacyPathEntryResult

	border        *Border
	font          *text.UltimaFont
	promptOutput  *text.Output
	pathOutput    *text.Output
	hintOutput    *text.Output
	messageOutput *text.Output
	fieldRect     image.Rectangle
	lastWidth     int
	lastHeight    int
}

// NewLegacyPathEntryModal creates a path-entry modal. OnResult and Keyboard are
// mandatory initialization contracts. The field gains focus when this widget
// becomes the top modal in a DialogStack.
func NewLegacyPathEntryModal(config LegacyPathEntryModalConfig) *LegacyPathEntryModal {
	if config.ReadClipboardText == nil {
		config.ReadClipboardText = platformclipboard.ReadText
		config.ClipboardEnabled = platformclipboard.Supported()
	}

	return newLegacyPathEntryModal(config, new(textinput.Field))
}

// NewSaveDescriptionEntryModal reuses the platform editor and modal lifecycle.
func NewSaveDescriptionEntryModal(config SaveDescriptionEntryModalConfig) *LegacyPathEntryModal {
	legacyConfig := new(LegacyPathEntryModalConfig)
	legacyConfig.Prompt = "Description (optional)"
	legacyConfig.Keyboard = config.Keyboard
	legacyConfig.OnResult = config.OnResult
	legacyConfig.Validate = config.Validate
	legacyConfig.IsKeyPressed = config.IsKeyPressed
	modal := NewLegacyPathEntryModal(*legacyConfig)
	modal.saveDescription = true
	modal.maxRunes = config.MaxRunes
	modal.title = config.Title
	if modal.title == "" {
		modal.title = "Save Game"
	}
	modal.notice = strings.TrimSpace(config.Notice)
	modal.SetText(config.InitialText)

	return modal
}

func newLegacyPathEntryModal(config LegacyPathEntryModalConfig, field legacyPathTextField) *LegacyPathEntryModal {
	if config.Keyboard == nil {
		panic("LegacyPathEntryModal: keyboard is nil")
	}
	if config.OnResult == nil {
		panic("LegacyPathEntryModal: OnResult is nil")
	}
	if field == nil {
		panic("LegacyPathEntryModal: text field is nil")
	}

	prompt := config.Prompt
	if prompt == "" {
		prompt = "Legacy SAVED.GAM path:"
	}
	initialPath := legacyPathSingleLine(config.InitialPath)
	field.SetTextAndSelection(initialPath, len(initialPath), len(initialPath))

	modal := new(LegacyPathEntryModal)
	modal.prompt = prompt
	modal.guidance = strings.TrimSpace(config.Guidance)
	modal.submitLabel = strings.TrimSpace(config.SubmitLabel)
	if modal.submitLabel == "" {
		modal.submitLabel = "Submit"
	}
	modal.cancelLabel = strings.TrimSpace(config.CancelLabel)
	if modal.cancelLabel == "" {
		modal.cancelLabel = defaultCancelActionLabel
	}
	modal.keyboard = config.Keyboard
	modal.onResult = config.OnResult
	modal.validate = config.Validate
	modal.field = field
	modal.lastSingleLineText = initialPath
	modal.readClipboardText = config.ReadClipboardText
	modal.clipboardEnabled = config.ClipboardEnabled
	modal.isKeyPressed = config.IsKeyPressed
	modal.isWindowFocused = ebiten.IsFocused
	if modal.isKeyPressed == nil {
		modal.isKeyPressed = ebiten.IsKeyPressed
	}

	return modal
}

// AutoCloseAndPop removes this exact widget instance after its result callback.
func (m *LegacyPathEntryModal) AutoCloseAndPop(stack *DialogStack) *LegacyPathEntryModal {
	if stack == nil {
		panic("LegacyPathEntryModal.AutoCloseAndPop: dialog stack is nil")
	}
	m.autoCloseStack = stack

	return m
}

// Draw renders the modal, path field, and keyboard hints.
func (m *LegacyPathEntryModal) Draw(screen *ebiten.Image) {
	if screen == nil {
		return
	}

	width, height := screen.Bounds().Dx(), screen.Bounds().Dy()
	rects := m.preparePresentation(width, height)
	FillClassicPixelRoundedRect(screen, m.border.GetBounds(), m.pixelUnit(), ucolor.DialogBackground.RGBA())
	m.drawPathField(screen, rects.field)

	m.promptOutput.DrawPanelContent(screen, rects.prompt.Min, rects.prompt.Dx(), etext.AlignStart)
	if m.saveDescription {
		m.titleOutput.DrawPanelContent(screen,
			image.Pt(rects.title.Min.X, centeredGlyphOutputY(rects.title, m.titleOutput, m.title)),
			rects.title.Dx(), etext.AlignCenter)
		m.noticeOutput.DrawPanelContent(screen, rects.notice.Min, rects.notice.Dx(), etext.AlignStart)
	}

	if m.validationMessage != "" || (!m.saveDescription && m.guidance != "") {
		m.messageOutput.DrawPanelContent(screen, rects.message.Min, rects.message.Dx(), etext.AlignStart)
	}
	m.hintOutput.DrawPanelContent(screen, rects.hint.Min, rects.hint.Dx(), etext.AlignCenter)
	DrawClassicPixelRoundedFrame(screen, m.border.GetBounds(), m.pixelUnit(), DialogRuleColor)
}

func (m *LegacyPathEntryModal) preparePresentation(width, height int) legacyPathEntryRects {
	m.ensureRendering(width, height)
	rects := m.calculateRects()
	m.fieldRect = rects.field
	m.promptOutput.Reset()
	m.promptOutput.AppendLine().Text(m.prompt, DialogHeaderColor).Done()
	message := m.validationMessage
	messageColor := DialogStatusWarningColor
	if message == "" && !m.saveDescription {
		message = m.guidance
		messageColor = DialogControlHintColor
	}
	if message != "" {
		m.messageOutput.Reset()
		m.messageOutput.AppendLine().Text(message, messageColor).WordWrap().Done()
	}

	m.hintOutput.Reset()
	shortcutLine := m.hintOutput.AppendLine()
	if m.clipboardEnabled && !m.saveDescription {
		pasteKey := "Ctrl+V"
		if runtime.GOOS == "darwin" {
			pasteKey = "Cmd+V"
		}
		shortcutLine.Text(pasteKey, DialogControlKeyColor).
			Text(": Paste    ", DialogControlHintColor)
	}
	clearHint := ": Clear"
	submitHint := ": " + m.submitLabel + "    "
	cancelHint := ": " + m.cancelLabel
	if m.saveDescription {
		clearHint, submitHint, cancelHint = ": Clear description", ": Save    ", ": Back"
	}
	shortcutLine.Text("Ctrl+U", DialogControlKeyColor).
		Text(clearHint, DialogControlHintColor).
		Done()
	m.hintOutput.AppendLine().
		Text("Enter", DialogControlKeyColor).
		Text(submitHint, DialogControlHintColor).
		Text("Esc", DialogControlKeyColor).
		Text(cancelHint, DialogControlHintColor).
		Done()

	return rects
}

func (m *LegacyPathEntryModal) drawPathField(screen *ebiten.Image, fieldRect image.Rectangle) {
	innerRect := m.drawPathFieldFrame(screen, fieldRect)
	contentRect := m.pathFieldContentRect(innerRect)
	pathWidth := max(1, contentRect.Dx())
	_, selectionEnd := m.directedSelection()
	start, end := orderedLegacyPathSelection(m.field)
	if compositionStart, compositionEnd, composing := m.field.CompositionSelection(); composing {
		start, end = start+compositionStart, start+compositionEnd
		selectionEnd = end
	}
	value := m.field.TextForRendering()
	selectionEnd = min(max(selectionEnd, 0), len(value))
	start, end = min(max(start, 0), len(value)), min(max(end, start), len(value))
	visiblePath, cursorRune := legacyPathVisibleWindow(
		value,
		selectionEnd,
		m.font.CharsForWidth(pathWidth),
	)
	pathY := contentRect.Min.Y + max(
		0,
		(contentRect.Dy()-int(m.pathOutput.GetFontHeight()))/legacyPathCenterDivisor,
	)
	runes := []rune(visiblePath)
	offset := utf8.RuneCountInString(value[:selectionEnd]) - cursorRune
	first := min(len(runes), max(0, utf8.RuneCountInString(value[:start])-offset))
	last := min(len(runes), max(first, utf8.RuneCountInString(value[:end])-offset))
	if !m.field.IsFocused() {
		first, last = 0, 0
	}
	prefix, selected, suffix := string(runes[:first]), string(runes[first:last]), string(runes[last:])
	prefixWidth, _ := m.font.MeasureString(prefix)
	selectedWidth, _ := m.font.MeasureString(selected)
	vector.FillRect(screen, float32(contentRect.Min.X)+float32(prefixWidth), float32(contentRect.Min.Y),
		float32(selectedWidth), float32(contentRect.Dy()), DialogSelectedBackground, false)
	m.pathOutput.Reset()
	m.pathOutput.AppendLine().Text(prefix, ucolor.DialogInputTextColor).
		Text(selected, DialogSelectedText).Text(suffix, ucolor.DialogInputTextColor).Done()
	m.pathOutput.DrawPanelContent(
		screen,
		image.Pt(contentRect.Min.X, pathY),
		pathWidth,
		etext.AlignStart,
	)
	m.drawPathCursor(screen, contentRect, visiblePath, cursorRune)
}

func (m *LegacyPathEntryModal) drawPathFieldFrame(
	screen *ebiten.Image,
	fieldRect image.Rectangle,
) image.Rectangle {
	frameWidth := m.pixelUnit()
	fieldBorderColor := DialogRuleLightColor
	if m.field.IsFocused() && !m.finished {
		fieldBorderColor = ucolor.EGACyan
	}
	vector.FillRect(
		screen,
		float32(fieldRect.Min.X),
		float32(fieldRect.Min.Y),
		float32(fieldRect.Dx()),
		float32(fieldRect.Dy()),
		fieldBorderColor,
		false,
	)
	innerRect := fieldRect.Inset(frameWidth)
	vector.FillRect(
		screen,
		float32(innerRect.Min.X),
		float32(innerRect.Min.Y),
		float32(innerRect.Dx()),
		float32(innerRect.Dy()),
		ucolor.EGABlack,
		false,
	)

	return innerRect
}

func (m *LegacyPathEntryModal) pathFieldContentRect(innerRect image.Rectangle) image.Rectangle {
	paddingX := m.scaledDimension(legacyPathFieldPaddingX, legacyPathMinimumPaddingX)
	paddingY := m.scaledDimension(legacyPathFieldPaddingY, legacyPathMinimumPaddingY)

	return image.Rect(
		min(innerRect.Max.X, innerRect.Min.X+paddingX),
		min(innerRect.Max.Y, innerRect.Min.Y+paddingY),
		max(innerRect.Min.X, innerRect.Max.X-paddingX),
		max(innerRect.Min.Y, innerRect.Max.Y-paddingY),
	)
}

func (m *LegacyPathEntryModal) drawPathCursor(
	screen *ebiten.Image,
	contentRect image.Rectangle,
	visiblePath string,
	cursorRune int,
) {
	if !m.field.IsFocused() || m.finished {
		return
	}
	visibleRunes := []rune(visiblePath)
	cursorRune = min(max(cursorRune, 0), len(visibleRunes))
	pathAdvance, _ := m.font.MeasureString(string(visibleRunes[:cursorRune]))
	cursorX := min(contentRect.Max.X, contentRect.Min.X+int(pathAdvance))
	vector.StrokeLine(
		screen,
		float32(cursorX),
		float32(contentRect.Min.Y),
		float32(cursorX),
		float32(contentRect.Max.Y),
		float32(m.pixelUnit()),
		ucolor.DialogInputTextColor,
		false,
	)
}

// Update consumes platform text input before handling modal control keys.
func (m *LegacyPathEntryModal) Update() {
	if m.finished || !m.field.IsFocused() || !m.isWindowFocused() {
		m.resetEditingRepeat()
		m.enterBlocked = true

		return
	}
	if !m.anyKeyPressed(ebiten.KeyEnter, ebiten.KeyNumpadEnter) {
		m.enterBlocked = false
	}
	width, height := display.GetManager().GetCurrentSize()
	m.ensureRendering(width, height)
	if m.fieldRect.Empty() {
		m.fieldRect = m.calculateRects().field
	}
	previous := m.field.Text()
	start, end := m.directedSelection()
	_, _, wasComposing := m.field.CompositionSelection()
	handled, err := m.field.HandleInputWithBounds(m.fieldRect)
	if err != nil {
		// The native Field retains input errors; keep the text and explanation
		// visible until Escape rather than silently returning to the browser.
		if m.validationMessage != legacyEntryInputErrorMessage {
			logging.Warn(logging.CategoryUI, "Text entry input failed: %v", err)
		}
		m.validationMessage = legacyEntryInputErrorMessage
		if m.keyboard.IsKeyPressedAndRegister(ebiten.KeyEscape) {
			m.finish(false)
		}

		return
	}
	if handled {
		m.selectionReversed = false
	}
	_, _, composing := m.field.CompositionSelection()
	if wasComposing || composing || handled && !m.isNativeShortcutEdit(previous, start, end) {
		// Preedit changes (including deletion of its last character) belong to
		// the IME. SetTextAndSelection would terminate that native session.
		m.resetEditingRepeat()
		m.compositionBackspaceBlocked = m.isKeyPressed(ebiten.KeyBackspace)
		m.compositionClearBlocked = m.controlModifierPressed() && m.isKeyPressed(ebiten.KeyU)
		m.enterBlocked = m.enterBlocked || m.anyKeyPressed(ebiten.KeyEnter, ebiten.KeyNumpadEnter)
		if m.acceptPlatformText(previous, start, end) && handled {
			m.validationMessage = ""
		}

		return
	}
	if m.keyboard.IsKeyPressedAndRegister(ebiten.KeyEscape) {
		m.finish(false)

		return
	}
	// Drain platform events first, then arbitrate owned physical edit keys. A
	// non-composition deletion or typed shortcut letter must not duplicate the local edit.
	if m.handlePhysicalEditing(previous, start, end) {
		m.rememberSingleLineText()

		return
	}
	if !m.acceptPlatformText(previous, start, end) {
		return
	}
	m.handleControlKey()
	m.rememberSingleLineText()
}

func (m *LegacyPathEntryModal) acceptPlatformText(previous string, anchor, caret int) bool {
	if m.saveDescription && !m.acceptFieldText(m.field.Text()) {
		m.setFieldTextAndSelection(previous, anchor, caret)

		return false
	}

	return m.enforceSingleLine()
}

// Only known non-composition edits may use local chord debounce/repeat instead
// of the platform's cadence. All other handled events retain native ownership.
func (m *LegacyPathEntryModal) isNativeShortcutEdit(previous string, anchor, caret int) bool {
	start, end := min(anchor, caret), max(anchor, caret)
	if m.controlModifierPressed() && m.isKeyPressed(ebiten.KeyU) {
		return m.field.Text() == previous[:start]+"u"+previous[end:]
	}
	if !m.isKeyPressed(ebiten.KeyBackspace) {
		return false
	}
	if start == end && start > 0 {
		_, runeLength := utf8.DecodeLastRuneInString(previous[:start])
		start -= runeLength
	}

	return start != end && m.field.Text() == previous[:start]+previous[end:]
}

func (m *LegacyPathEntryModal) handlePhysicalEditing(previous string, start, end int) bool {
	clearPressed := m.controlModifierPressed() && m.isKeyPressed(ebiten.KeyU)
	backspacePressed := m.isKeyPressed(ebiten.KeyBackspace)
	if !backspacePressed {
		m.backspaceHeld = false
		m.backspaceTicks = 0
		m.compositionBackspaceBlocked = false
	}
	if !clearPressed {
		m.compositionClearBlocked = false
	}
	if m.compositionBackspaceBlocked || m.compositionClearBlocked {
		return true
	}
	if clearPressed {
		m.setFieldTextAndSelection(previous, start, end)

		return m.handleClearLineShortcut()
	}
	m.clearLineHeld = false
	if !backspacePressed {
		return false
	}
	currentStart, currentEnd := m.directedSelection()
	if previous != m.field.Text() || start != currentStart || end != currentEnd {
		m.setFieldTextAndSelection(previous, start, end)
	}
	if !m.backspaceHeld {
		m.backspaceHeld = true
		m.backspaceTicks = legacyEntryBackspaceDelay
	} else {
		m.backspaceTicks--
		if m.backspaceTicks > 0 {
			return true
		}
		m.backspaceTicks = legacyEntryBackspaceInterval
	}
	m.validationMessage = ""
	m.deleteBackward()

	return true
}

func (m *LegacyPathEntryModal) resetEditingRepeat() {
	m.backspaceHeld = false
	m.backspaceTicks = 0
	m.clearLineHeld = false
	m.compositionBackspaceBlocked = false
	m.compositionClearBlocked = false
}

func (m *LegacyPathEntryModal) handleControlKey() {
	if m.handleClearLineShortcut() || m.handlePrimaryShortcut() || m.handleCompletionKey() {
		return
	}
	m.handleEditingKey()
}

func (m *LegacyPathEntryModal) handleClearLineShortcut() bool {
	pressed := m.controlModifierPressed() && m.isKeyPressed(ebiten.KeyU)
	if !pressed {
		m.clearLineHeld = false

		return false
	}
	if m.clearLineHeld {
		return true
	}

	m.clearLineHeld = true
	m.SetText("")

	return true
}

func (m *LegacyPathEntryModal) handlePrimaryShortcut() bool {
	if !m.primaryModifierPressed() {
		return false
	}
	switch {
	case m.keyboard.IsKeyPressedAndRegister(ebiten.KeyV):
		m.pasteClipboardText()
	case m.keyboard.IsKeyPressedAndRegister(ebiten.KeyA):
		valueLength := len(m.field.Text())
		m.setFieldTextAndSelection(m.field.Text(), 0, valueLength)
	default:
		return false
	}

	return true
}

func (m *LegacyPathEntryModal) handleCompletionKey() bool {
	if m.enterBlocked {
		return true
	}
	switch {
	case m.keyboard.IsKeyPressedAndRegister(ebiten.KeyEnter),
		m.keyboard.IsKeyPressedAndRegister(ebiten.KeyNumpadEnter):
		m.submit()
	default:
		return false
	}

	return true
}

func (m *LegacyPathEntryModal) handleEditingKey() {
	switch {
	case m.keyboard.IsKeyPressedAndRegister(ebiten.KeyBackspace):
		m.validationMessage = ""
		m.deleteBackward()
	case m.keyboard.IsKeyPressedAndRegister(ebiten.KeyDelete):
		m.validationMessage = ""
		m.deleteForward()
	case m.keyboard.IsKeyPressedAndRegister(ebiten.KeyLeft):
		m.moveCursorLeft(m.shiftPressed())
	case m.keyboard.IsKeyPressedAndRegister(ebiten.KeyRight):
		m.moveCursorRight(m.shiftPressed())
	case m.keyboard.IsKeyPressedAndRegister(ebiten.KeyHome):
		m.moveCursorHome(m.shiftPressed())
	case m.keyboard.IsKeyPressedAndRegister(ebiten.KeyEnd):
		m.moveCursorEnd(m.shiftPressed())
	}
}

func (m *LegacyPathEntryModal) submit() {
	if !m.acceptFieldText(m.field.Text()) {
		return
	}
	if m.validate != nil {
		if message := strings.TrimSpace(m.validate(m.field.Text())); message != "" {
			m.validationMessage = message

			return
		}
	}
	m.validationMessage = ""
	m.finish(true)
}

func (m *LegacyPathEntryModal) pasteClipboardText() {
	if !m.clipboardEnabled || m.readClipboardText == nil {
		return
	}
	value, err := m.readClipboardText()
	if err != nil {
		m.validationMessage = "Clipboard text is unavailable."

		return
	}
	if !m.saveDescription {
		value = strings.TrimRight(value, "\r\n")
	}
	if strings.ContainsAny(value, "\r\n") {
		m.validationMessage = "Clipboard contains more than one line."

		return
	}
	if value == "" {
		m.validationMessage = "Clipboard does not contain a path."
		if m.saveDescription {
			m.validationMessage = "Clipboard does not contain text."
		}

		return
	}

	text := m.field.Text()
	start, end := orderedLegacyPathSelection(m.field)
	text = text[:start] + value + text[end:]
	if !m.acceptFieldText(text) {
		return
	}
	cursor := start + len(value)
	m.setFieldTextAndSelection(text, cursor, cursor)
	m.validationMessage = ""
}

func (m *LegacyPathEntryModal) primaryModifierPressed() bool {
	return m.controlModifierPressed() || m.anyKeyPressed(
		ebiten.KeyMeta,
		ebiten.KeyMetaLeft,
		ebiten.KeyMetaRight,
	)
}

func (m *LegacyPathEntryModal) controlModifierPressed() bool {
	return m.anyKeyPressed(ebiten.KeyControl, ebiten.KeyControlLeft, ebiten.KeyControlRight)
}

func (m *LegacyPathEntryModal) shiftPressed() bool {
	return m.anyKeyPressed(ebiten.KeyShift, ebiten.KeyShiftLeft, ebiten.KeyShiftRight)
}

func (m *LegacyPathEntryModal) anyKeyPressed(keys ...ebiten.Key) bool {
	for _, key := range keys {
		if m.isKeyPressed(key) {
			return true
		}
	}

	return false
}

// Dismiss cancels the modal programmatically. Repeated dismissal is harmless.
func (m *LegacyPathEntryModal) Dismiss() {
	m.finish(false)
}

// Submit validates and submits the current text programmatically.
func (m *LegacyPathEntryModal) Submit() {
	if m.finished {
		return
	}
	m.submit()
}

// GetText returns the entered path without applying path or file validation.
func (m *LegacyPathEntryModal) GetText() string {
	return m.field.Text()
}

// GetValidationMessage returns the current inline validation or clipboard error.
func (m *LegacyPathEntryModal) GetValidationMessage() string {
	return m.validationMessage
}

// SetValidationMessage replaces the current inline validation message.
func (m *LegacyPathEntryModal) SetValidationMessage(message string) {
	m.validationMessage = strings.TrimSpace(message)
}

// SetText replaces the entered path and places the cursor at the end.
func (m *LegacyPathEntryModal) SetText(value string) {
	if !m.acceptFieldText(value) {
		return
	}
	if !m.saveDescription {
		value = legacyPathSingleLine(value)
	}
	m.setFieldTextAndSelection(value, len(value), len(value))
	m.lastSingleLineText = value
	m.validationMessage = ""
}

// IsFinished reports whether the modal has submitted or cancelled.
func (m *LegacyPathEntryModal) IsFinished() bool {
	return m.finished
}

// GetResult returns a LegacyPathEntryResult for the ModalDialog contract.
func (m *LegacyPathEntryModal) GetResult() interface{} {
	return m.result
}

func (m *LegacyPathEntryModal) onDialogFocusChanged(focused bool) {
	m.resetEditingRepeat()
	if focused && !m.finished {
		m.field.Focus()
		m.enterBlocked = m.anyKeyPressed(ebiten.KeyEnter, ebiten.KeyNumpadEnter)

		return
	}
	m.field.Blur()
}

func (m *LegacyPathEntryModal) finish(submitted bool) {
	if m.finished {
		return
	}

	m.finished = true
	m.resetEditingRepeat()
	m.result = LegacyPathEntryResult{
		Text:      m.field.Text(),
		Submitted: submitted,
	}
	m.field.Blur()
	m.onResult(m.result.Text, submitted)
	if m.autoCloseStack != nil {
		m.autoCloseStack.MustRemoveWidgetInstance(m)
	}
}

func (m *LegacyPathEntryModal) acceptFieldText(value string) bool {
	if !m.saveDescription {
		return true
	}
	if !utf8.ValidString(value) || strings.ContainsFunc(value, unicode.IsControl) {
		m.validationMessage = "Description must be a single line without control characters."

		return false
	}
	if m.maxRunes > 0 && utf8.RuneCountInString(value) > m.maxRunes {
		m.validationMessage = fmt.Sprintf("Description must be at most %d characters.", m.maxRunes)

		return false
	}

	return true
}

func (m *LegacyPathEntryModal) enforceSingleLine() bool {
	value := m.field.Text()
	cleaned := strings.TrimRight(value, "\r\n")
	if strings.ContainsAny(cleaned, "\r\n") {
		cursor := len(m.lastSingleLineText)
		m.setFieldTextAndSelection(m.lastSingleLineText, cursor, cursor)
		m.validationMessage = "Only one path can be entered."

		return false
	}
	if cleaned == value {
		m.lastSingleLineText = value

		return true
	}

	selectionStart, selectionEnd := m.directedSelection()
	selectionStart = min(max(selectionStart, 0), len(value))
	selectionEnd = min(max(selectionEnd, 0), len(value))
	cleanedStart := len(legacyPathSingleLine(value[:selectionStart]))
	cleanedEnd := len(legacyPathSingleLine(value[:selectionEnd]))
	m.setFieldTextAndSelection(cleaned, cleanedStart, cleanedEnd)
	m.lastSingleLineText = cleaned

	return true
}

func (m *LegacyPathEntryModal) rememberSingleLineText() {
	if !strings.ContainsAny(m.field.Text(), "\r\n") {
		m.lastSingleLineText = m.field.Text()
	}
}

func (m *LegacyPathEntryModal) deleteBackward() {
	value := m.field.Text()
	start, end := orderedLegacyPathSelection(m.field)
	if start == end && start > 0 {
		_, runeLength := utf8.DecodeLastRuneInString(value[:start])
		start -= runeLength
	}
	if start == end {
		return
	}
	value = value[:start] + value[end:]
	m.setFieldTextAndSelection(value, start, start)
}

func (m *LegacyPathEntryModal) deleteForward() {
	value := m.field.Text()
	start, end := orderedLegacyPathSelection(m.field)
	if start == end && end < len(value) {
		_, runeLength := utf8.DecodeRuneInString(value[end:])
		end += runeLength
	}
	if start == end {
		return
	}
	value = value[:start] + value[end:]
	m.setFieldTextAndSelection(value, start, start)
}

func (m *LegacyPathEntryModal) moveCursorLeft(selecting bool) {
	value := m.field.Text()
	selectionStart, selectionEnd := m.directedSelection()
	if !selecting {
		start, end := orderedLegacyPathSelection(m.field)
		if start == end && start > 0 {
			_, runeLength := utf8.DecodeLastRuneInString(value[:start])
			start -= runeLength
		}
		m.setFieldTextAndSelection(value, start, start)

		return
	}
	selectionEnd = min(max(selectionEnd, 0), len(value))
	if selectionEnd > 0 {
		_, runeLength := utf8.DecodeLastRuneInString(value[:selectionEnd])
		selectionEnd -= runeLength
	}
	m.setFieldTextAndSelection(value, selectionStart, selectionEnd)
}

func (m *LegacyPathEntryModal) moveCursorRight(selecting bool) {
	value := m.field.Text()
	selectionStart, selectionEnd := m.directedSelection()
	if !selecting {
		start, end := orderedLegacyPathSelection(m.field)
		if start == end && end < len(value) {
			_, runeLength := utf8.DecodeRuneInString(value[end:])
			end += runeLength
		}
		m.setFieldTextAndSelection(value, end, end)

		return
	}
	selectionEnd = min(max(selectionEnd, 0), len(value))
	if selectionEnd < len(value) {
		_, runeLength := utf8.DecodeRuneInString(value[selectionEnd:])
		selectionEnd += runeLength
	}
	m.setFieldTextAndSelection(value, selectionStart, selectionEnd)
}

func (m *LegacyPathEntryModal) moveCursorHome(selecting bool) {
	if selecting {
		selectionStart, _ := m.directedSelection()
		m.setFieldTextAndSelection(m.field.Text(), selectionStart, 0)

		return
	}
	m.setFieldTextAndSelection(m.field.Text(), 0, 0)
}

func (m *LegacyPathEntryModal) moveCursorEnd(selecting bool) {
	textLength := len(m.field.Text())
	if selecting {
		selectionStart, _ := m.directedSelection()
		m.setFieldTextAndSelection(m.field.Text(), selectionStart, textLength)

		return
	}
	m.setFieldTextAndSelection(m.field.Text(), textLength, textLength)
}

// The native field requires ordered replacement ranges. Keep the active end
// locally so Shift navigation can shrink, reverse, and extend a selection.
func (m *LegacyPathEntryModal) setFieldTextAndSelection(value string, anchor, caret int) {
	m.selectionReversed = anchor > caret
	m.field.SetTextAndSelection(value, min(anchor, caret), max(anchor, caret))
}

func (m *LegacyPathEntryModal) directedSelection() (int, int) {
	start, end := orderedLegacyPathSelection(m.field)
	if m.selectionReversed {
		return end, start
	}

	return start, end
}

func (m *LegacyPathEntryModal) ensureRendering(width, height int) {
	if m.border != nil && width == m.lastWidth && height == m.lastHeight {
		return
	}
	m.lastWidth = width
	m.lastHeight = height
	m.border = newDialogBorder(legacyPathEntryPlacement(), DialogBackgroundBlack)
	m.font, m.pathOutput = GetScaledFontWithOutput(FontConfig{
		Size:                  FontSizeMainScreen,
		LineSpacing:           0,
		LineSpacingMultiplier: 0,
		MaxLines:              1,
		MaxChars:              1,
	})
	_, m.promptOutput = GetScaledFontWithOutput(FontConfig{
		Size:                  FontSizeMainScreen,
		LineSpacing:           0,
		LineSpacingMultiplier: 0,
		MaxLines:              1,
		MaxChars:              1,
	})
	_, m.hintOutput = GetScaledFontWithOutput(FontConfig{
		Size:                  FontSizeMainScreen,
		LineSpacing:           0,
		LineSpacingMultiplier: 0,
		MaxLines:              legacyPathHintMaximumLines,
		MaxChars:              1,
	})
	_, m.messageOutput = GetScaledFontWithOutput(FontConfig{
		Size:                  FontSizeMainScreen,
		LineSpacing:           0,
		LineSpacingMultiplier: 0,
		MaxLines:              legacyPathValidationLines,
		MaxChars:              1,
	})
	if m.saveDescription {
		m.configureDescriptionRendering()
	}

	rects := m.calculateRects()
	m.fieldRect = rects.field
	m.promptOutput.SetMaxCharsPerLine(m.promptOutput.Font.CharsForWidth(max(1, rects.prompt.Dx())))
	pathInset := legacyPathCenterDivisor *
		(m.pixelUnit() + m.scaledDimension(legacyPathFieldPaddingX, legacyPathMinimumPaddingX))
	m.pathOutput.SetMaxCharsPerLine(m.font.CharsForWidth(max(1, rects.field.Dx()-pathInset)))
	m.hintOutput.SetMaxCharsPerLine(m.hintOutput.Font.CharsForWidth(max(1, rects.hint.Dx())))
	m.messageOutput.SetMaxCharsPerLine(m.messageOutput.Font.CharsForWidth(max(1, rects.message.Dx())))
}

type legacyPathEntryRects struct {
	title   image.Rectangle
	notice  image.Rectangle
	prompt  image.Rectangle
	field   image.Rectangle
	message image.Rectangle
	hint    image.Rectangle
}

func (m *LegacyPathEntryModal) calculateRects() legacyPathEntryRects {
	if m.saveDescription {
		return m.calculateDescriptionRects()
	}
	bounds := m.border.GetInteriorBounds()
	padding := m.scaledDimension(legacyPathContentPadding, legacyPathMinimumPadding)
	content := bounds.Inset(padding)
	spacing := m.scaledDimension(legacyPathVerticalSpacing, legacyPathMinimumSpacing)
	promptHeight := max(1, int(math.Ceil(m.promptOutput.GetFontHeight())))
	frameWidth := m.pixelUnit()
	fieldPaddingY := m.scaledDimension(legacyPathFieldPaddingY, legacyPathMinimumPaddingY)
	fieldHeight := max(1, int(math.Ceil(m.pathOutput.GetFontHeight()))+2*(frameWidth+fieldPaddingY))
	hintLines := legacyPathHintMaximumLines
	hintHeight := max(1, int(math.Ceil(m.hintOutput.GetTotalHeightForLines(hintLines))))

	prompt := image.Rect(content.Min.X, content.Min.Y, content.Max.X, min(content.Max.Y, content.Min.Y+promptHeight))
	fieldTop := min(content.Max.Y, prompt.Max.Y+spacing)
	field := image.Rect(content.Min.X, fieldTop, content.Max.X, min(content.Max.Y, fieldTop+fieldHeight))
	hintTop := max(field.Max.Y, content.Max.Y-hintHeight)
	hint := image.Rect(content.Min.X, hintTop, content.Max.X, content.Max.Y)
	messageTop := min(hint.Min.Y, field.Max.Y+spacing)
	messageBottom := max(messageTop, hint.Min.Y-spacing)

	return legacyPathEntryRects{
		title:   image.Rect(0, 0, 0, 0),
		notice:  image.Rect(0, 0, 0, 0),
		prompt:  prompt,
		field:   field,
		message: image.Rect(content.Min.X, messageTop, content.Max.X, messageBottom),
		hint:    hint,
	}
}

func (m *LegacyPathEntryModal) scaledDimension(value, minimum int) int {
	return max(minimum, int(math.Round(float64(value)*float64(max(1, m.lastWidth))/legacyPathReferenceWidth)))
}

func (m *LegacyPathEntryModal) pixelUnit() int {
	return getClassicUIPixelUnit(m.lastWidth, m.lastHeight)
}

func legacyPathEntryPlacement() sprites.PercentBasedPlacement {
	return sprites.PercentBasedPlacement{
		StartPercentX: legacyPathModalStartX,
		EndPercentX:   legacyPathModalEndX,
		StartPercentY: legacyPathModalStartY,
		EndPercentY:   legacyPathModalEndY,
	}
}

func legacyPathSingleLine(value string) string {
	return strings.NewReplacer("\r", "", "\n", "").Replace(value)
}

func legacyPathVisibleWindow(value string, cursorByte, maxRunes int) (string, int) {
	if maxRunes <= 0 {
		return "", 0
	}
	runes := []rune(value)
	cursorByte = min(max(cursorByte, 0), len(value))
	cursorRune := utf8.RuneCountInString(value[:cursorByte])
	if len(runes) <= maxRunes {
		return value, cursorRune
	}
	markerLength := utf8.RuneCountInString(legacyPathTailMarker)
	if maxRunes <= markerLength {
		start := min(max(cursorRune-maxRunes/legacyPathCenterDivisor, 0), len(runes)-maxRunes)

		return string(runes[start : start+maxRunes]), cursorRune - start
	}
	contentCapacity := maxRunes - markerLength
	if cursorRune <= contentCapacity {
		return string(runes[:contentCapacity]) + legacyPathTailMarker, cursorRune
	}
	if len(runes)-cursorRune <= contentCapacity {
		start := len(runes) - contentCapacity

		return legacyPathTailMarker + string(runes[start:]), markerLength + cursorRune - start
	}
	if maxRunes <= markerLength*legacyPathCenterDivisor {
		start := min(max(cursorRune-contentCapacity/legacyPathCenterDivisor, 0), len(runes)-contentCapacity)

		return legacyPathTailMarker + string(runes[start:start+contentCapacity]), markerLength + cursorRune - start
	}

	contentCapacity = maxRunes - markerLength*legacyPathCenterDivisor
	start := cursorRune - contentCapacity/legacyPathCenterDivisor
	start = min(max(start, 0), len(runes)-contentCapacity)

	return legacyPathTailMarker + string(runes[start:start+contentCapacity]) + legacyPathTailMarker,
		markerLength + cursorRune - start
}

func orderedLegacyPathSelection(field legacyPathTextField) (int, int) {
	start, end := field.Selection()
	start = min(max(start, 0), len(field.Text()))
	end = min(max(end, 0), len(field.Text()))
	if start > end {
		return end, start
	}

	return start, end
}
