package widgets

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/stretchr/testify/assert"
)

// stackTestWidget's field gives it a size, so each instance has a distinct address
// for the stack's duplicate-instance check.
type stackTestWidget struct {
	id int
}

func (*stackTestWidget) Draw(_ *ebiten.Image) {}
func (*stackTestWidget) Update()              {}

func TestDialogStackPeekTopModalSkipsNonBlockingDialogs(t *testing.T) {
	var stack DialogStack
	modal := &stackTestWidget{}
	feedback := &stackTestWidget{}
	stack.PushModalDialog(modal)
	stack.PushNonBlockingDialog(feedback)

	assert.Same(t, modal, stack.PeekTopModalDialog())
}

func TestDialogStackRejectsDuplicateInstance(t *testing.T) {
	var stack DialogStack
	dialog := &stackTestWidget{}
	stack.PushModalDialog(dialog)

	assert.Panics(t, func() { stack.PushNonBlockingDialog(dialog) })
}

func TestInputBoxAutoCloseRemovesExactInstance(t *testing.T) {
	initializeDialogTestDisplay(t)
	var stack DialogStack
	dialog := NewInputBox("Name?", nil, newWidgetTestKeyboard(100)).
		WithRegexValidation(".*").
		WithResultCallback(func(string, bool) {}).
		AutoCloseAndPop(&stack).
		Build()
	above := &stackTestWidget{}
	stack.PushModalDialog(dialog)
	stack.PushModalDialog(above)

	dialog.InvokeOnDismiss()

	assert.Equal(t, 1, stack.Count())
	assert.Same(t, above, stack.PeekTopModalDialog())
}

func TestDialogStackFindsInputBoxBelowNonBlockingDialog(t *testing.T) {
	initializeDialogTestDisplay(t)
	var stack DialogStack
	inputBox := NewInputBox("Name?", nil, newWidgetTestKeyboard(100)).
		WithRegexValidation(".*").
		WithResultCallback(func(string, bool) {}).
		Build()
	stack.PushModalDialog(inputBox)
	stack.PushNonBlockingDialog(&stackTestWidget{})

	assert.Same(t, inputBox, stack.GetOrAssertTopInputBox())
}

func TestMessageDialogDismissCallbackRunsOnce(t *testing.T) {
	initializeDialogTestDisplay(t)
	dismissCount := 0
	dialog := NewMessageDialog("once").
		WithInputDismissal(newWidgetTestKeyboard(100)).
		WithEffects(false).
		WithDismissCallback(func() { dismissCount++ }).
		Build()

	dialog.dismiss()
	dialog.dismiss()

	assert.Equal(t, 1, dismissCount)
}

func initializeDialogTestDisplay(t *testing.T) {
	t.Helper()
	setResponsiveDialogTestResolution(t, 1920, 1080)
}
