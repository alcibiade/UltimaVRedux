package main

import (
	"reflect"

	"github.com/bradhannah/UltimaVRedux/internal/ui/widgets"
)

type merchantWorkflowOwnedDialog interface {
	widgets.Widget
	Dismiss()
	IsDismissed() bool
}

type merchantDialogRegistry struct {
	owned     []merchantWorkflowOwnedDialog
	seen      map[merchantWorkflowOwnedDialog]struct{}
	finalized bool
}

func (r *merchantDialogRegistry) registerAndPush(
	stack *widgets.DialogStack,
	dialog merchantWorkflowOwnedDialog,
) {
	if stack == nil {
		panic("merchantDialogRegistry: dialog stack is required")
	}
	if r.finalized {
		panic("merchantDialogRegistry: cannot register after cleanup")
	}
	value := reflect.ValueOf(dialog)
	if !value.IsValid() || value.Kind() != reflect.Pointer || value.IsNil() {
		panic("merchantDialogRegistry: owned dialog must be a non-nil pointer")
	}
	if r.seen == nil {
		r.seen = make(map[merchantWorkflowOwnedDialog]struct{})
	}
	if _, exists := r.seen[dialog]; exists {
		panic("merchantDialogRegistry: dialog instance registered twice")
	}
	r.seen[dialog] = struct{}{}
	r.owned = append(r.owned, dialog)
	stack.PushModalDialog(dialog)
}

func (r *merchantDialogRegistry) dismissAndRemoveAll(stack *widgets.DialogStack) {
	if stack == nil {
		panic("merchantDialogRegistry: dialog stack is required")
	}
	if r.finalized {
		return
	}
	for i := len(r.owned) - 1; i >= 0; i-- {
		dialog := r.owned[i]
		if !dialog.IsDismissed() {
			dialog.Dismiss()
		}
		stack.RemoveWidgetInstance(dialog)
	}
	r.owned = nil
	r.seen = nil
	r.finalized = true
}

func (s *merchantTalkWorkflowState) pushOwnedDialog(
	stack *widgets.DialogStack,
	dialog merchantWorkflowOwnedDialog,
) {
	s.ownedDialogs.registerAndPush(stack, dialog)
}
