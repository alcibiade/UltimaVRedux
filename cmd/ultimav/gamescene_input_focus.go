package main

import "github.com/bradhannah/UltimaVRedux/internal/ui/widgets"

// Native editors release their session even when a paused reader returns before
// modal Update. This optional widget capability cannot accept keys or intents.
func (g *GameScene) suspendNativeDialogInput() {
	if editor, ok := g.dialogStack.PeekTopModalDialog().(interface{ SuspendTextInput() }); ok {
		editor.SuspendTextInput()
	}
}

// updateDialogsWithoutInput advances the animation-bearing passive primitives
// while the scene keyboard is suspended. Other modal selectors/readers can poll
// their own physical adapters, so their input Update must not run without focus.
// Their static presentation/layout continues through Draw.
func (g *GameScene) updateDialogsWithoutInput() bool {
	dialog := g.dialogStack.PeekTopModalDialog()
	switch animated := dialog.(type) {
	case *widgets.MessageDialog:
		animated.Update()
	case *widgets.OverlayDialog:
		animated.Update()
	}

	return dialog != nil
}
