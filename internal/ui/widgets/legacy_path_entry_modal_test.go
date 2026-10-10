package widgets

import (
	"errors"
	"image"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/stretchr/testify/require"
)

type fakeLegacyPathTextField struct {
	text           string
	selectionStart int
	selectionEnd   int
	focused        bool
	focusCalls     int
	blurCalls      int
	handleText     string
	handleErr      error
	handleInput    bool
	handleCalls    int
}

func (f *fakeLegacyPathTextField) Focus() {
	f.focused = true
	f.focusCalls++
}

func (f *fakeLegacyPathTextField) Blur() {
	f.focused = false
	f.blurCalls++
}

func (f *fakeLegacyPathTextField) IsFocused() bool {
	return f.focused
}

func (f *fakeLegacyPathTextField) HandleInputWithBounds(_ image.Rectangle) (bool, error) {
	f.handleCalls++
	if f.handleErr != nil {
		return false, f.handleErr
	}
	if f.handleText == "" && !f.handleInput {
		return false, nil
	}
	f.text = f.handleText
	f.selectionStart = len(f.text)
	f.selectionEnd = len(f.text)
	f.handleText = ""
	f.handleInput = false

	return true, nil
}

func (f *fakeLegacyPathTextField) Text() string {
	return f.text
}

func (f *fakeLegacyPathTextField) TextForRendering() string {
	return f.text
}

func (f *fakeLegacyPathTextField) Selection() (int, int) {
	return f.selectionStart, f.selectionEnd
}

func (*fakeLegacyPathTextField) CompositionSelection() (int, int, bool) {
	return 0, 0, false
}

func (f *fakeLegacyPathTextField) SetTextAndSelection(text string, selectionStart, selectionEnd int) {
	f.text = text
	f.selectionStart = selectionStart
	f.selectionEnd = selectionEnd
}

func TestLegacyPathEntryModalFollowsDialogStackFocus(t *testing.T) {
	field := new(fakeLegacyPathTextField)
	modal := newTestLegacyPathEntryModal(field, func(string, bool) {})
	coveringModal := &stackTestWidget{}
	var stack DialogStack

	require.False(t, field.focused)
	stack.PushModalDialog(modal)
	require.True(t, field.focused)
	require.Equal(t, 1, field.focusCalls)

	stack.PushModalDialog(coveringModal)
	require.False(t, field.focused)
	require.Equal(t, 1, field.blurCalls)

	stack.RemoveWidgetInstance(coveringModal)
	require.True(t, field.focused)
	require.Equal(t, 2, field.focusCalls)

	stack.RemoveWidgetInstance(modal)
	require.False(t, field.focused)
	require.Equal(t, 2, field.blurCalls)
}

func TestLegacyPathEntryModalAcceptsPlatformTextUnchanged(t *testing.T) {
	field := &fakeLegacyPathTextField{
		handleText: `/Users/avatar/Ultima V (DOS)/SAVED.GAM`,
	}
	modal := newTestLegacyPathEntryModal(field, func(string, bool) {})
	var stack DialogStack
	stack.PushModalDialog(modal)

	modal.Update()

	require.Equal(t, `/Users/avatar/Ultima V (DOS)/SAVED.GAM`, modal.GetText())
}

func TestLegacyPathEntryModalSubmitCallsBackOnceAndAutoCloses(t *testing.T) {
	field := new(fakeLegacyPathTextField)
	var results []LegacyPathEntryResult
	modal := newTestLegacyPathEntryModal(field, func(text string, submitted bool) {
		results = append(results, LegacyPathEntryResult{Text: text, Submitted: submitted})
	})
	modal.SetText(`C:\Program Files\Ultima V\SAVED.GAM`)
	var stack DialogStack
	modal.AutoCloseAndPop(&stack)
	stack.PushModalDialog(modal)
	stack.PushNonBlockingDialog(&stackTestWidget{})
	modal.keyboard.SimulateKeyPress(ebiten.KeyEnter)

	modal.Update()
	modal.Update()
	modal.Dismiss()

	require.Equal(t, []LegacyPathEntryResult{{
		Text:      `C:\Program Files\Ultima V\SAVED.GAM`,
		Submitted: true,
	}}, results)
	require.True(t, modal.IsFinished())
	require.False(t, field.focused)
	require.Equal(t, 1, stack.Count())
	require.Equal(t, results[0], modal.GetResult())
}

func TestLegacyPathEntryModalEscapeReturnsEnteredTextAsCancelled(t *testing.T) {
	field := new(fakeLegacyPathTextField)
	var result LegacyPathEntryResult
	modal := newTestLegacyPathEntryModal(field, func(text string, submitted bool) {
		result = LegacyPathEntryResult{Text: text, Submitted: submitted}
	})
	modal.SetText("./legacy saves/SAVED.GAM")
	var stack DialogStack
	stack.PushModalDialog(modal)
	modal.keyboard.SimulateKeyPress(ebiten.KeyEscape)

	modal.Update()

	require.Equal(t, LegacyPathEntryResult{
		Text:      "./legacy saves/SAVED.GAM",
		Submitted: false,
	}, result)
	require.False(t, field.focused)
}

func TestLegacyPathEntryModalInputErrorStaysVisibleUntilCancelled(t *testing.T) {
	field := &fakeLegacyPathTextField{handleErr: errors.New("platform input failed")}
	callbackCalls := 0
	modal := newTestLegacyPathEntryModal(field, func(_ string, submitted bool) {
		callbackCalls++
		require.False(t, submitted)
	})
	var stack DialogStack
	stack.PushModalDialog(modal)
	modal.SetText("b_skiff")
	modal.keyboard.SimulateKeyPress(ebiten.KeyEnter)

	modal.Update()
	modal.Update()

	require.False(t, modal.IsFinished())
	require.Zero(t, callbackCalls)
	require.Equal(t, "b_skiff", modal.GetText())
	require.Equal(t, "Text input failed. Press Esc to go back.", modal.GetValidationMessage())
	modal.keyboard.SimulateKeyPress(ebiten.KeyEscape)
	modal.Update()

	require.True(t, modal.IsFinished())
	require.False(t, field.focused)
	require.Equal(t, 1, callbackCalls)
}

func TestLegacyPathEntryModalRejectsMultilinePlatformInput(t *testing.T) {
	field := &fakeLegacyPathTextField{handleText: "one\r\ntwo three"}
	modal := newTestLegacyPathEntryModal(field, func(string, bool) {})
	var stack DialogStack
	stack.PushModalDialog(modal)

	modal.Update()

	require.Empty(t, modal.GetText())
	require.Equal(t, "Only one path can be entered.", modal.GetValidationMessage())
}

func TestLegacyPathVisibleWindowFollowsCaret(t *testing.T) {
	visible, cursor := legacyPathVisibleWindow("short", len("short"), 10)
	require.Equal(t, "short", visible)
	require.Equal(t, 5, cursor)

	visible, cursor = legacyPathVisibleWindow("a/very/long/SAVED.GAM", len("a/very/long/SAVED.GAM"), 9)
	require.Equal(t, "...ED.GAM", visible)
	require.Equal(t, 9, cursor)

	visible, cursor = legacyPathVisibleWindow("a/very/long/SAVED.GAM", 0, 9)
	require.Equal(t, "a/very...", visible)
	require.Zero(t, cursor)

	visible, cursor = legacyPathVisibleWindow("SAVED.GAM", len("SAVED.GAM"), 0)
	require.Empty(t, visible)
	require.Zero(t, cursor)
}

func TestLegacyPathEntryModalPastesOverSelection(t *testing.T) {
	field := new(fakeLegacyPathTextField)
	keyboard := newWidgetTestKeyboard(0)
	modifierHeld := true
	modal := newFocusedLegacyPathEntryModal(LegacyPathEntryModalConfig{
		Keyboard:         keyboard,
		OnResult:         func(string, bool) {},
		ClipboardEnabled: true,
		ReadClipboardText: func() (string, error) {
			return "/Users/avatar/Ultima V/SAVED.GAM\n", nil
		},
		IsKeyPressed: func(key ebiten.Key) bool {
			return modifierHeld && (key == ebiten.KeyMeta || key == ebiten.KeyControl)
		},
	}, field)
	field.SetTextAndSelection("old path", 0, len("old path"))
	var stack DialogStack
	stack.PushModalDialog(modal)
	keyboard.SimulateKeyPress(ebiten.KeyV)

	modal.Update()

	require.Equal(t, "/Users/avatar/Ultima V/SAVED.GAM", modal.GetText())
	start, end := field.Selection()
	require.Equal(t, len(modal.GetText()), start)
	require.Equal(t, start, end)
}

func TestLegacyPathEntryModalPrimaryModifierSelectsAll(t *testing.T) {
	field := new(fakeLegacyPathTextField)
	keyboard := newWidgetTestKeyboard(0)
	modal := newFocusedLegacyPathEntryModal(LegacyPathEntryModalConfig{
		InitialPath: "select this",
		Keyboard:    keyboard,
		OnResult:    func(string, bool) {},
		IsKeyPressed: func(key ebiten.Key) bool {
			return key == ebiten.KeyMeta
		},
	}, field)
	var stack DialogStack
	stack.PushModalDialog(modal)
	keyboard.SimulateKeyPress(ebiten.KeyA)

	modal.Update()

	start, end := field.Selection()
	require.Zero(t, start)
	require.Equal(t, len("select this"), end)
}

func TestLegacyPathEntryModalControlUClearsOnceUntilRelease(t *testing.T) {
	field := new(fakeLegacyPathTextField)
	keyboard := newWidgetTestKeyboard(0)
	pressed := map[ebiten.Key]bool{
		ebiten.KeyControl: true,
		ebiten.KeyU:       true,
	}
	modal := newFocusedLegacyPathEntryModal(LegacyPathEntryModalConfig{
		InitialPath: "erase this",
		Keyboard:    keyboard,
		OnResult:    func(string, bool) {},
		IsKeyPressed: func(key ebiten.Key) bool {
			return pressed[key]
		},
	}, field)
	var stack DialogStack
	stack.PushModalDialog(modal)

	modal.Update()
	require.Empty(t, modal.GetText())

	modal.SetText("keep while held")
	modal.Update()
	require.Equal(t, "keep while held", modal.GetText())

	pressed[ebiten.KeyU] = false
	modal.Update()
	pressed[ebiten.KeyU] = true
	modal.Update()
	require.Empty(t, modal.GetText())
}

func TestLegacyPathEntryModalRejectsMultilineClipboard(t *testing.T) {
	field := new(fakeLegacyPathTextField)
	keyboard := newWidgetTestKeyboard(0)
	modal := newFocusedLegacyPathEntryModal(LegacyPathEntryModalConfig{
		Keyboard:         keyboard,
		OnResult:         func(string, bool) {},
		ClipboardEnabled: true,
		ReadClipboardText: func() (string, error) {
			return "first\nsecond", nil
		},
		IsKeyPressed: func(key ebiten.Key) bool {
			return key == ebiten.KeyControl
		},
	}, field)
	field.SetTextAndSelection("keep", 4, 4)
	var stack DialogStack
	stack.PushModalDialog(modal)
	keyboard.SimulateKeyPress(ebiten.KeyV)

	modal.Update()

	require.Equal(t, "keep", modal.GetText())
	require.Equal(t, "Clipboard contains more than one line.", modal.validationMessage)
	require.False(t, modal.IsFinished())
}

func TestLegacyPathEntryModalValidationKeepsFieldOpen(t *testing.T) {
	field := new(fakeLegacyPathTextField)
	keyboard := newWidgetTestKeyboard(0)
	valid := false
	results := 0
	modal := newFocusedLegacyPathEntryModal(LegacyPathEntryModalConfig{
		Keyboard: keyboard,
		OnResult: func(string, bool) { results++ },
		Validate: func(string) string {
			if !valid {
				return "SAVED.OOL is missing."
			}

			return ""
		},
	}, field)
	var stack DialogStack
	stack.PushModalDialog(modal)
	keyboard.SimulateKeyPress(ebiten.KeyEnter)

	modal.Update()

	require.False(t, modal.IsFinished())
	require.True(t, field.focused)
	require.Equal(t, "SAVED.OOL is missing.", modal.validationMessage)
	require.Zero(t, results)

	valid = true
	keyboard.SimulateKeyPress(ebiten.KeyEnter)
	modal.Update()
	require.True(t, modal.IsFinished())
	require.Equal(t, 1, results)
}

func TestLegacyPathEntryModalCanShowDeferredValidation(t *testing.T) {
	modal := NewLegacyPathEntryModal(LegacyPathEntryModalConfig{
		Prompt: "Path", Guidance: "", SubmitLabel: "Check", CancelLabel: "Cancel",
		InitialPath: "/games/ultima-v", Keyboard: newWidgetTestKeyboard(0),
		OnResult: func(string, bool) {}, Validate: nil, ReadClipboardText: nil,
		ClipboardEnabled: false, IsKeyPressed: noWidgetTestKeyHeld,
	})

	modal.SetValidationMessage("Required game files are missing.")

	require.Equal(t, "Required game files are missing.", modal.GetValidationMessage())
	require.Equal(t, "/games/ultima-v", modal.GetText())
}

func TestLegacyPathEntryModalRequiresInitializationContracts(t *testing.T) {
	validConfig := LegacyPathEntryModalConfig{
		Keyboard: newWidgetTestKeyboard(0),
		OnResult: func(string, bool) {},
	}
	require.Panics(t, func() {
		newLegacyPathEntryModal(LegacyPathEntryModalConfig{
			OnResult: validConfig.OnResult,
		}, new(fakeLegacyPathTextField))
	})
	require.Panics(t, func() {
		newLegacyPathEntryModal(LegacyPathEntryModalConfig{
			Keyboard: validConfig.Keyboard,
		}, new(fakeLegacyPathTextField))
	})
	require.Panics(t, func() {
		newLegacyPathEntryModal(validConfig, nil)
	})
}

func newTestLegacyPathEntryModal(
	field legacyPathTextField,
	onResult LegacyPathEntryCallback,
) *LegacyPathEntryModal {
	return newFocusedLegacyPathEntryModal(LegacyPathEntryModalConfig{
		Keyboard: newWidgetTestKeyboard(0),
		OnResult: onResult,
	}, field)
}

func newFocusedLegacyPathEntryModal(config LegacyPathEntryModalConfig, field legacyPathTextField) *LegacyPathEntryModal {
	if config.IsKeyPressed == nil {
		config.IsKeyPressed = config.Keyboard.IsKeyHeld
	}
	modal := newLegacyPathEntryModal(config, field)
	modal.isWindowFocused = func() bool { return true }
	return modal
}
