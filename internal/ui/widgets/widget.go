// Package widgets provides reusable UI components for the game interface.
//
// This package implements the Widget interface pattern for consistent UI element
// behavior across the application. Core widgets include dialogs, buttons, text inputs,
// and layout components that integrate with the Ebiten rendering system.
//
// Key interfaces:
//   - Widget: Basic draw/update interface for all UI components
//   - ModalDialog: Extended interface for dismissible dialogs with results
//
// Common usage patterns:
//   - Create widgets via New* constructor functions
//   - Manage dialog lifecycle with DialogStack
//   - Handle user input through keyboard integration
package widgets

import "github.com/hajimehoshi/ebiten/v2"

// Widget defines the basic interface for all UI components that can be rendered and updated.
// All widgets must implement both Draw() for rendering and Update() for handling input/state changes.
type Widget interface {
	// Draw renders the widget to the provided screen image.
	// The screen parameter must not be nil.
	Draw(screen *ebiten.Image)

	// Update processes input events and updates internal widget state.
	// This is called once per frame before Draw().
	Update()
}

// DismissibleDialog provides a unified interface for dialogs with visibility control.
// Both OverlayDialog and MessageDialog implement this interface, allowing them to be
// pushed to DialogStack and managed uniformly.
//
// Use this interface for storing dialog references that may be either type:
//
//	var dialog widgets.DismissibleDialog
//	dialog = overlayBuilder.Build()  // OverlayDialog
//	dialog.ShowWithoutAnimation()
//	dialogStack.PushModalDialog(dialog)
type DismissibleDialog interface {
	Widget
	Show()
	ShowWithoutAnimation()
	Hide()
	HideWithoutAnimation()
	IsVisible() bool
	IsDismissed() bool
}
