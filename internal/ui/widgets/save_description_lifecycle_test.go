package widgets

import (
	"errors"
	"image"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/stretchr/testify/require"
)

type fakeSaveDescriptionTextField struct {
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

func (f *fakeSaveDescriptionTextField) Focus() {
	f.focused = true
	f.focusCalls++
}

func (f *fakeSaveDescriptionTextField) Blur() {
	f.focused = false
	f.blurCalls++
}

func (f *fakeSaveDescriptionTextField) IsFocused() bool {
	return f.focused
}

func (f *fakeSaveDescriptionTextField) HandleInputWithBounds(_ image.Rectangle) (bool, error) {
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

func (f *fakeSaveDescriptionTextField) Text() string {
	return f.text
}

func (f *fakeSaveDescriptionTextField) TextForRendering() string {
	return f.text
}

func (f *fakeSaveDescriptionTextField) Selection() (int, int) {
	return f.selectionStart, f.selectionEnd
}

func (*fakeSaveDescriptionTextField) CompositionSelection() (int, int, bool) {
	return 0, 0, false
}

func (f *fakeSaveDescriptionTextField) SetTextAndSelection(text string, selectionStart, selectionEnd int) {
	f.text = text
	f.selectionStart = selectionStart
	f.selectionEnd = selectionEnd
}

func TestSaveDescriptionEntryModalFollowsDialogStackFocus(t *testing.T) {
	field := new(fakeSaveDescriptionTextField)
	modal := newTestSaveDescriptionEntryModal(field, func(string, bool) {})
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

func TestSaveDescriptionEntryModalAcceptsPlatformTextUnchanged(t *testing.T) {
	field := &fakeSaveDescriptionTextField{
		handleText: "Before Doom (with the full party)!",
	}
	modal := newTestSaveDescriptionEntryModal(field, func(string, bool) {})
	var stack DialogStack
	stack.PushModalDialog(modal)

	modal.Update()

	require.Equal(t, "Before Doom (with the full party)!", modal.GetText())
}

func TestSaveDescriptionEntryModalSubmitCallsBackOnceAndAutoCloses(t *testing.T) {
	field := new(fakeSaveDescriptionTextField)
	var results []SaveDescriptionEntryResult
	modal := newTestSaveDescriptionEntryModal(field, func(text string, submitted bool) {
		results = append(results, SaveDescriptionEntryResult{Text: text, Submitted: submitted})
	})
	modal.SetText("Before entering Doom")
	var stack DialogStack
	modal.AutoCloseAndPop(&stack)
	stack.PushModalDialog(modal)
	stack.PushNonBlockingDialog(&stackTestWidget{})
	modal.keyboard.SimulateKeyPress(ebiten.KeyEnter)

	modal.Update()
	modal.Update()
	modal.Dismiss()

	require.Equal(t, []SaveDescriptionEntryResult{{
		Text:      "Before entering Doom",
		Submitted: true,
	}}, results)
	require.True(t, modal.IsFinished())
	require.False(t, field.focused)
	require.Equal(t, 1, stack.Count())
	require.Equal(t, results[0], modal.GetResult())
}

func TestSaveDescriptionEntryModalEscapeReturnsEnteredTextAsCancelled(t *testing.T) {
	field := new(fakeSaveDescriptionTextField)
	var result SaveDescriptionEntryResult
	modal := newTestSaveDescriptionEntryModal(field, func(text string, submitted bool) {
		result = SaveDescriptionEntryResult{Text: text, Submitted: submitted}
	})
	modal.SetText("At the shrine")
	var stack DialogStack
	stack.PushModalDialog(modal)
	modal.keyboard.SimulateKeyPress(ebiten.KeyEscape)

	modal.Update()

	require.Equal(t, SaveDescriptionEntryResult{
		Text:      "At the shrine",
		Submitted: false,
	}, result)
	require.False(t, field.focused)
}

func TestSaveDescriptionEntryModalInputErrorStaysVisibleUntilCancelled(t *testing.T) {
	field := &fakeSaveDescriptionTextField{handleErr: errors.New("platform input failed")}
	callbackCalls := 0
	modal := newTestSaveDescriptionEntryModal(field, func(_ string, submitted bool) {
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
	require.Equal(t, "Text input failed.", modal.GetValidationMessage())
	modal.preparePresentation(1280, 720)
	require.Contains(t, modal.hintOutput.GetOutputStr(false), "Esc: Back")
	_, hasControlInk := modal.hintOutput.FirstPanelLineWithColor(0, modal.hintOutput.GetPanelContentLineCount(), DialogControlKeyColor)
	require.True(t, hasControlInk, "the recovery control remains in the styled hint area")
	modal.keyboard.SimulateKeyPress(ebiten.KeyEscape)
	modal.Update()

	require.True(t, modal.IsFinished())
	require.False(t, field.focused)
	require.Equal(t, 1, callbackCalls)
}

func TestSaveDescriptionEntryModalRejectsMultilinePlatformInput(t *testing.T) {
	field := &fakeSaveDescriptionTextField{handleText: "one\r\ntwo three"}
	modal := newTestSaveDescriptionEntryModal(field, func(string, bool) {})
	var stack DialogStack
	stack.PushModalDialog(modal)

	modal.Update()

	require.Empty(t, modal.GetText())
	require.Equal(t, "Description must be a single line without control characters.", modal.GetValidationMessage())
}

func TestSaveDescriptionVisibleWindowFollowsCaret(t *testing.T) {
	visible, cursor := saveDescriptionVisibleWindow("short", len("short"), 10)
	require.Equal(t, "short", visible)
	require.Equal(t, 5, cursor)

	visible, cursor = saveDescriptionVisibleWindow("Before entering Doom", len("Before entering Doom"), 9)
	require.Equal(t, "...g Doom", visible)
	require.Equal(t, 9, cursor)

	visible, cursor = saveDescriptionVisibleWindow("Before entering Doom", 0, 9)
	require.Equal(t, "Before...", visible)
	require.Zero(t, cursor)

	visible, cursor = saveDescriptionVisibleWindow("Doom", len("Doom"), 0)
	require.Empty(t, visible)
	require.Zero(t, cursor)
}

func TestSaveDescriptionEntryModalPrimaryModifierSelectsAll(t *testing.T) {
	for _, modifier := range []ebiten.Key{ebiten.KeyMeta, ebiten.KeyControl} {
		t.Run(modifier.String(), func(t *testing.T) {
			field := new(fakeSaveDescriptionTextField)
			keyboard := newWidgetTestKeyboard(0)
			modal := newFocusedSaveDescriptionEntryModal(SaveDescriptionEntryModalConfig{
				InitialText:  "select this",
				Keyboard:     keyboard,
				OnResult:     func(string, bool) {},
				IsKeyPressed: func(key ebiten.Key) bool { return key == modifier },
			}, field)
			var stack DialogStack
			stack.PushModalDialog(modal)
			keyboard.SimulateKeyPress(ebiten.KeyA)
			modal.Update()
			start, end := field.Selection()
			require.Zero(t, start)
			require.Equal(t, len("select this"), end)
		})
	}
}

func TestSaveDescriptionEntryModalControlUClearsOnceUntilRelease(t *testing.T) {
	field := new(fakeSaveDescriptionTextField)
	keyboard := newWidgetTestKeyboard(0)
	pressed := map[ebiten.Key]bool{
		ebiten.KeyControl: true,
		ebiten.KeyU:       true,
	}
	modal := newFocusedSaveDescriptionEntryModal(SaveDescriptionEntryModalConfig{
		InitialText: "erase this",
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

func TestSaveDescriptionEntryModalValidationKeepsFieldOpen(t *testing.T) {
	field := new(fakeSaveDescriptionTextField)
	keyboard := newWidgetTestKeyboard(0)
	valid := false
	results := 0
	modal := newFocusedSaveDescriptionEntryModal(SaveDescriptionEntryModalConfig{
		Keyboard: keyboard,
		OnResult: func(string, bool) { results++ },
		Validate: func(string) string {
			if !valid {
				return "Review the destination again."
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
	require.Equal(t, "Review the destination again.", modal.validationMessage)
	require.Zero(t, results)

	valid = true
	keyboard.SimulateKeyPress(ebiten.KeyEnter)
	modal.Update()
	require.True(t, modal.IsFinished())
	require.Equal(t, 1, results)
}

func TestSaveDescriptionEntryModalCanShowDeferredValidation(t *testing.T) {
	modal := NewSaveDescriptionEntryModal(SaveDescriptionEntryModalConfig{
		InitialText: "Before Doom", Keyboard: newWidgetTestKeyboard(0),
		OnResult: func(string, bool) {}, IsKeyPressed: noWidgetTestKeyHeld,
	})

	modal.SetValidationMessage("Review the destination again.")

	require.Equal(t, "Review the destination again.", modal.GetValidationMessage())
	require.Equal(t, "Before Doom", modal.GetText())
}

func TestSaveDescriptionEntryModalRequiresInitializationContracts(t *testing.T) {
	validConfig := SaveDescriptionEntryModalConfig{
		Keyboard: newWidgetTestKeyboard(0),
		OnResult: func(string, bool) {},
	}
	require.Panics(t, func() {
		newSaveDescriptionEntryModal(SaveDescriptionEntryModalConfig{
			OnResult: validConfig.OnResult,
		}, new(fakeSaveDescriptionTextField))
	})
	require.Panics(t, func() {
		newSaveDescriptionEntryModal(SaveDescriptionEntryModalConfig{
			Keyboard: validConfig.Keyboard,
		}, new(fakeSaveDescriptionTextField))
	})
	require.Panics(t, func() {
		newSaveDescriptionEntryModal(validConfig, nil)
	})
}

func newTestSaveDescriptionEntryModal(
	field saveDescriptionTextField,
	onResult SaveDescriptionEntryCallback,
) *SaveDescriptionEntryModal {
	return newFocusedSaveDescriptionEntryModal(SaveDescriptionEntryModalConfig{
		Keyboard: newWidgetTestKeyboard(0),
		OnResult: onResult,
	}, field)
}

func newFocusedSaveDescriptionEntryModal(config SaveDescriptionEntryModalConfig, field saveDescriptionTextField) *SaveDescriptionEntryModal {
	if config.IsKeyPressed == nil {
		config.IsKeyPressed = config.Keyboard.IsKeyHeld
	}
	modal := newSaveDescriptionEntryModal(config, field)
	modal.isWindowFocused = func() bool { return true }
	return modal
}
