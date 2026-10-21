package main

import (
	"github.com/bradhannah/UltimaVRedux/internal/logging"
	"github.com/hajimehoshi/ebiten/v2"
)

// Update method (simplified from original since no channel handling needed)
func (d *LinearTalkDialog) Update() {
	if d.closed || (d.callbacks != nil && !d.mounted) {
		return
	}
	if d.mounted && !d.isMounted() {
		d.discardPendingKnowledge()

		return
	}
	d.ensureTalkLayout()
	d.expireExcerptNotice()
	if d.gameScene.keyboard.IsKeyPressedAndRegister(ebiten.KeyTab) {
		d.keepPresentedExchange()
		d.gameScene.keyboard.PrepareForTransition(0)

		return
	}
	if d.updateHistoryInput() {
		return
	}
	// Handle KeyWait state (0x8F) - blocks input until any key pressed, then continues conversation
	// PRIORITY: Process before dismissal since both block input, but KeyWait continues conversation
	if d.waitingForKeypressContinue {
		// Phase 1: Wait for all initially-pressed keys to be released
		// Call TryAcceptKey with dummy key to trigger release state check
		if d.gameScene.keyboard.IsWaitingForRelease() {
			d.gameScene.keyboard.TryAcceptKey(ebiten.KeySpace) // Triggers state update
			if d.gameScene.keyboard.IsWaitingForRelease() {
				return // Still waiting for keys to release
			}
			logging.Debugf(logging.CategoryUI, "All keys released, ready for KeyWait keypress")
		}

		// Phase 2: Detect NEW keypress (after all initial keys released)
		if d.gameScene.keyboard.IsAnyKeyPressed() {
			logging.Debugf(logging.CategoryUI, "New keypress detected in KeyWait state, continuing conversation")

			// Mark all currently pressed keys as consumed to prevent bleeding to underlying systems
			d.consumeCurrentlyPressedKeys()

			// Clear KeyWait state
			// Release-based consumption tracking will automatically clean up when key is released
			d.waitingForKeypressContinue = false

			// Continue conversation by calling engine.ProcessInput("")
			// The semantic engine resumes its suspended action frames.
			d.currentResponse = d.engine.ProcessInput("")
			d.processCurrentResponse()

			return
		}

		return // Don't process other input while waiting for KeyWait keypress
	}

	// Handle dismissal state with proper 2-phase debouncing
	if d.waitingForDismissal {
		// Phase 1: Wait for all initially-pressed keys to be released
		// Call TryAcceptKey with dummy key to trigger release state check
		if d.gameScene.keyboard.IsWaitingForRelease() {
			d.gameScene.keyboard.TryAcceptKey(ebiten.KeySpace) // Triggers state update
			if d.gameScene.keyboard.IsWaitingForRelease() {
				return // Still waiting for keys to release
			}
			logging.Debugf(logging.CategoryUI, "All keys released, ready for new keypress")
		}

		// Phase 2: Detect NEW keypress (after all initial keys released)
		if d.gameScene.keyboard.IsAnyKeyPressed() {
			logging.Debugf(logging.CategoryUI, "New keypress detected in dismissal state, closing dialog")

			// Mark all currently pressed keys as consumed to prevent bleeding to underlying input handlers.
			d.consumeCurrentlyPressedKeys()

			// Release-based consumption tracking will automatically handle cleanup when key is released
			// This prevents the dismissal key from bleeding into the underlying game input.
			// Remove self from dialog stack (explicit reference instead of blind pop)
			d.Close()

			return
		}

		return // Don't process other input while waiting for dismissal
	}

	// Block input during pause animation
	if !d.isPausing {
		d.TextInput.Update()
	}

	// Handle pause animation timing (no visual feedback during pause)
	if d.isPausing {
		elapsedMs := d.gameScene.gameState.ElapsedMs - d.pauseStartMs

		// Wait 1.5 seconds per pause (half of original 3 seconds)
		if elapsedMs >= 1500 {
			// Pause complete - finish it and process next marker if any
			d.finishCurrentPause()

			return // Exit early to avoid processing accumulated output during pause
		}
	}

	// Check if engine has any pending output to display (only when not pausing)
	if !d.isPausing && d.callbacks != nil {
		accumulatedOutput := d.callbacks.GetAccumulatedOutput()
		if accumulatedOutput != "" {
			d.Output.AppendText(accumulatedOutput)
			d.callbacks.ClearAccumulatedOutput()
		}
	}
}
