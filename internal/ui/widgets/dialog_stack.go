package widgets

import (
	"github.com/hajimehoshi/ebiten/v2"

	"github.com/bradhannah/UltimaVRedux/internal/logging"
	"github.com/bradhannah/UltimaVRedux/pkg/helpers"
)

// dialogEntry represents a dialog in the stack with its modal status.
type dialogEntry struct {
	widget  Widget
	isModal bool // true = blocks input, false = non-blocking (like action feedback popups)
}

// modalFocusListener is implemented by modal widgets that own platform input focus.
// Only the highest modal receives focus; non-blocking dialogs do not affect it.
type modalFocusListener interface {
	onDialogFocusChanged(focused bool)
}

// DialogStack manages a stack of dialogs with LIFO (last-in-first-out) behavior.
// It provides centralized dialog lifecycle management for the game's interface system.
// Dialogs can be modal (blocking input) or non-blocking (allowing game to continue).
type DialogStack struct {
	entries []dialogEntry
}

// PushModalDialog adds a new modal (input-blocking) dialog to the top of the stack.
// The dialog becomes the active modal and will receive input focus.
// Use this for dialogs that require user interaction before continuing (confirmations, menus, conversations).
func (d *DialogStack) PushModalDialog(dialog Widget) {
	d.mustNotContain(dialog)
	previousModal := d.PeekTopModalDialog()
	logging.Info(logging.CategoryUI, "PushModalDialog called, stack size before: %d", len(d.entries))
	d.entries = append(d.entries, dialogEntry{widget: dialog, isModal: true})
	d.updateModalFocus(previousModal, dialog)
	logging.Info(logging.CategoryUI, "PushModalDialog completed, stack size after: %d", len(d.entries))
}

// PushNonBlockingDialog adds a non-blocking dialog to the stack.
// Non-blocking dialogs are displayed but don't prevent game input from being processed.
// Use this for action feedback popups and other informational displays that shouldn't interrupt gameplay.
func (d *DialogStack) PushNonBlockingDialog(dialog Widget) {
	d.mustNotContain(dialog)
	logging.Info(logging.CategoryUI, "PushNonBlockingDialog called, stack size before: %d", len(d.entries))
	d.entries = append(d.entries, dialogEntry{widget: dialog, isModal: false})
	logging.Info(logging.CategoryUI, "PushNonBlockingDialog completed, stack size after: %d", len(d.entries))
}

// PeekTopModalDialog returns the highest input-blocking dialog without removing it.
// Non-blocking dialogs may be physically above it in the stack.
func (d *DialogStack) PeekTopModalDialog() Widget {
	for i := len(d.entries) - 1; i >= 0; i-- {
		if d.entries[i].isModal {
			return d.entries[i].widget
		}
	}

	return nil
}

// HasOpenDialog returns true if there are any dialogs currently on the stack.
func (d *DialogStack) HasOpenDialog() bool {
	return len(d.entries) > 0
}

// HasModalDialog returns true if there are any modal (input-blocking) dialogs on the stack.
// This is used by the game loop to determine if input should be blocked.
func (d *DialogStack) HasModalDialog() bool {
	for _, entry := range d.entries {
		if entry.isModal {
			return true
		}
	}

	return false
}

func (d *DialogStack) HasWidgetTypeOnTop(iface interface{}) bool {
	if !d.HasOpenDialog() {
		return false
	}

	return helpers.IsOfTypeInterface(d.entries[len(d.entries)-1].widget, iface)
}

func (d *DialogStack) GetIndexOfDialogType(iface interface{}) int {
	for i, entry := range d.entries {
		if helpers.IsOfTypeInterface(entry.widget, iface) {
			return i
		}
	}

	return -1
}

func (d *DialogStack) GetOrAssertTopInputBox() *InputBox {
	w := d.PeekTopModalDialog()
	if w == nil {
		return nil
	}

	ib, ok := w.(*InputBox)

	if !ok {
		logging.Warn(logging.CategoryUI, "Expected InputBox on dialog stack but found different type, returning nil")

		return nil // Graceful degradation instead of crash
	}

	return ib
}

func (d *DialogStack) RemoveWidget(iface interface{}) {
	nIndex := d.GetIndexOfDialogType(iface)
	if nIndex == -1 {
		logging.Warn(logging.CategoryUI, "Attempted to remove dialog that wasn't found in stack, ignoring")

		return // Graceful degradation - silently ignore missing widget
	}

	previousModal := d.PeekTopModalDialog()
	d.entries = append(d.entries[:nIndex], d.entries[nIndex+1:]...)
	d.updateModalFocus(previousModal, d.PeekTopModalDialog())
}

// RemoveWidgetInstance removes a specific widget instance from the stack by pointer equality.
// Returns true if the widget was found and removed, false if not found.
// This is safer than RemoveWidget when multiple dialogs of the same type may exist,
// as it matches by instance rather than by type.
func (d *DialogStack) RemoveWidgetInstance(widget Widget) bool {
	for i, entry := range d.entries {
		if entry.widget == widget {
			previousModal := d.PeekTopModalDialog()
			d.entries = append(d.entries[:i], d.entries[i+1:]...)
			d.updateModalFocus(previousModal, d.PeekTopModalDialog())
			logging.Debug(logging.CategoryUI,
				"RemoveWidgetInstance: Removed widget at index %d, stack size now: %d", i, len(d.entries))

			return true
		}
	}

	logging.Debug(logging.CategoryUI, "RemoveWidgetInstance: Widget not found in stack (already removed or never added)")

	return false
}

// ContainsWidgetInstance reports whether the exact widget is still stack-owned.
func (d *DialogStack) ContainsWidgetInstance(widget Widget) bool {
	for _, entry := range d.entries {
		if entry.widget == widget {
			return true
		}
	}

	return false
}

// MustRemoveWidgetInstance removes an owned widget or panics if stack ownership was lost.
func (d *DialogStack) MustRemoveWidgetInstance(widget Widget) {
	if !d.RemoveWidgetInstance(widget) {
		panic("DialogStack.MustRemoveWidgetInstance: required widget is not owned by this stack")
	}
}

// UpdateNonBlockingDialogs advances dialogs that cannot consume modal input.
func (d *DialogStack) UpdateNonBlockingDialogs() {
	for _, entry := range d.entries {
		if !entry.isModal {
			entry.widget.Update()
		}
	}
}

// Draw renders all dialogs from bottom to top without allocating a snapshot.
func (d *DialogStack) Draw(screen *ebiten.Image) {
	for _, entry := range d.entries {
		entry.widget.Draw(screen)
	}
}

func (d *DialogStack) mustNotContain(widget Widget) {
	if widget == nil {
		panic("DialogStack: cannot push a nil widget")
	}
	for _, entry := range d.entries {
		if entry.widget == widget {
			panic("DialogStack: cannot push the same widget instance twice")
		}
	}
}

func (*DialogStack) updateModalFocus(previous, current Widget) {
	if previous == current {
		return
	}
	if listener, ok := previous.(modalFocusListener); ok {
		listener.onDialogFocusChanged(false)
	}
	if listener, ok := current.(modalFocusListener); ok {
		listener.onDialogFocusChanged(true)
	}
}

// Dialogs returns all widgets in the stack for iteration (e.g., drawing all dialogs).
// Returns widgets in order from bottom to top.
func (d *DialogStack) Dialogs() []Widget {
	widgets := make([]Widget, len(d.entries))
	for i, entry := range d.entries {
		widgets[i] = entry.widget
	}

	return widgets
}

// Count returns the number of dialogs currently on the stack.
func (d *DialogStack) Count() int {
	return len(d.entries)
}
