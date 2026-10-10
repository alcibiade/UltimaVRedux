package widgets

import (
	"github.com/hajimehoshi/ebiten/v2"

	"github.com/bradhannah/UltimaVRedux/internal/visual"
)

// SuspendPresentationClock preserves a covered fade across a full Journal pause.
// The caller resumes exactly once; idle dialogs need no clock compensation.
func (md *MessageDialog) SuspendPresentationClock() func() {
	started := md.animationClock()

	return func() {
		if md.fadeState != fadeNone {
			md.fadeStartTime += md.animationClock() - started
		}
	}
}

// Update handles input dismissal (simple mode) and dissolve animation (advanced mode)
func (md *MessageDialog) Update() {
	// Simple mode: Handle ESC/Enter dismissal
	if md.enableInputDismissal && md.keyboard != nil {
		if md.keyboard.IsKeyPressedAndRegister(ebiten.KeyEscape) ||
			md.keyboard.IsKeyPressedAndRegister(ebiten.KeyEnter) || md.keyboard.IsKeyPressedAndRegister(ebiten.KeyNumpadEnter) {
			md.dismiss()

			return
		}
	}

	// Advanced mode: Handle dissolve animation
	if md.enableDissolveEffects && md.fadeState != fadeNone {
		md.updateDissolveAnimation()
	}
}

func (md *MessageDialog) dismiss() {
	if md.dismissed {
		return
	}
	md.dismissed = true
	if md.hasPromptOutput() {
		md.promptOutput.Clear()
		md.contentDirty = true
	}
	if md.onDismiss != nil {
		md.onDismiss()
	}
}

// updateDissolveAnimation advances the dissolve animation state
func (md *MessageDialog) updateDissolveAnimation() {
	currentTime := md.animationClock()
	elapsedMs := currentTime - md.fadeStartTime

	switch md.fadeState {
	case fadingIn:
		progress := float64(elapsedMs) / float64(md.fadeDurationInMs)
		if progress >= 1.0 {
			md.dissolveLevel = 0.0
			md.fadeState = fadeNone
		} else {
			md.dissolveLevel = 1.0 - progress
		}
		if md.dissolveEffect != nil {
			md.dissolveEffect.UpdateProgress(1.0 - md.dissolveLevel)
		}

	case fadingOut:
		progress := float64(elapsedMs) / float64(md.fadeDurationOutMs)
		if progress >= 1.0 {
			md.dissolveLevel = 1.0
			md.fadeState = fadeNone
			md.visible = false
			md.dissolveEffect = nil
		} else {
			md.dissolveLevel = progress
		}
		if md.dissolveEffect != nil {
			md.dissolveEffect.UpdateProgress(md.dissolveLevel)
		}
	}
}

// Show makes the dialog visible with fade-in animation if effects enabled, otherwise appears instantly.
func (md *MessageDialog) Show() {
	md.visible = true
	md.animationStarted = true
	if md.enableDissolveEffects {
		md.fadeState = fadingIn
		md.dissolveLevel = 1.0
		md.fadeStartTime = md.animationClock()
		md.dissolveEffect = nil
	}
	md.contentDirty = true
}

// ShowWithoutAnimation makes the dialog visible immediately, bypassing any fade-in effects.
func (md *MessageDialog) ShowWithoutAnimation() {
	md.visible = true
	md.animationStarted = true
	md.fadeState = fadeNone
	md.dissolveLevel = 0.0
	md.contentDirty = true
}

// Hide begins fade-out animation if effects enabled, then hides dialog, otherwise hides instantly.
func (md *MessageDialog) Hide() {
	md.animationStarted = true
	if md.enableDissolveEffects {
		md.fadeState = fadingOut
		md.dissolveLevel = 0.0
		md.fadeStartTime = md.animationClock()

		if md.contentBuffer != nil {
			screenWidth, screenHeight := md.contentBuffer.Bounds().Dx(), md.contentBuffer.Bounds().Dy()
			md.dissolveEffect = md.createDissolveEffect(screenWidth, screenHeight, md.getDissolveModeForHide())
			md.dissolveEffect.UpdateProgress(0.0)
		}
	} else {
		md.visible = false
	}
}

// getDissolveModeForHide returns the dissolve mode for hide animation
// This helper allows flexible future enhancement
func (md *MessageDialog) getDissolveModeForHide() visual.DissolveMode {
	return visual.DissolveModeHide
}

// HideWithoutAnimation hides the dialog immediately, bypassing any fade-out effects.
func (md *MessageDialog) HideWithoutAnimation() {
	md.visible = false
	md.fadeState = fadeNone
	md.dissolveLevel = 0.0
	md.dissolveEffect = nil
}

// IsVisible returns true if the dialog is currently visible on screen.
func (md *MessageDialog) IsVisible() bool {
	return md.visible
}
