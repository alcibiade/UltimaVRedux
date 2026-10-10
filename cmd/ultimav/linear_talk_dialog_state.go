package main

import (
	"github.com/bradhannah/UltimaVRedux/internal/conversation"
	"github.com/bradhannah/UltimaVRedux/internal/logging"
	"github.com/hajimehoshi/ebiten/v2"
)

// beginPauseAnimation starts the pause animation sequence and hides UI elements
func (d *LinearTalkDialog) beginPauseAnimation() {
	d.isPausing = true
	d.pauseStartMs = d.gameScene.gameState.ElapsedMs
	d.currentPauseIndex = 0

	// Hide prompt and cursor during pause animation
	d.TextInput.ClearPrompt()
	d.TextInput.ClearPlaceholder()
	d.TextInput.SetCursorVisible(false)
}

// endPauseAnimation completes the pause animation and restores UI elements
func (d *LinearTalkDialog) endPauseAnimation() {
	d.isPausing = false

	// Restore prompt and cursor after all pauses complete
	d.TextInput.SetCursorVisible(true)
	d.commitPresentedKnowledge()
	d.finishResponsePresentation()
}

// finishCurrentPause completes the current pause animation and processes pending output.
// Handles both segment-based (runic) and string-based output.
func (d *LinearTalkDialog) finishCurrentPause() {
	if d.closed || !d.isPausing {
		return
	}
	// Count the consumed control token in the original semantic source stream.
	d.presentedSourceBytes += len(conversation.PauseMarker)
	d.commitPresentedKnowledge()
	logging.Debug(logging.CategoryUI, "Finishing pause %d", d.currentPauseIndex)

	// Check for segment-based pending output first (runic text support)
	if len(d.pendingSegments) > 0 {
		d.finishCurrentPauseWithSegments()

		return
	}

	// String-based pending output (no runic text)
	if d.pendingOutput != "" {
		// Check if pending output contains another pause marker
		if beforePause, afterPause, hasPause := splitAtFirstPauseMarker(d.pendingOutput); hasPause {
			// Split at next marker
			// Display text before next pause - preserve newlines
			d.outputTextToDialog(beforePause, d.pendingContinues)

			// Update pending output for next pause
			d.pendingOutput = afterPause
			d.pendingContinues = outputContinuesAfterText(beforePause, d.pendingContinues)

			// Start next pause animation
			d.pauseStartMs = d.gameScene.gameState.ElapsedMs
			d.currentPauseIndex++

			logging.Debug(logging.CategoryUI, "Starting pause %d, pending: %d chars", d.currentPauseIndex, len(d.pendingOutput))
		} else {
			// No more pauses - display remaining output - preserve newlines
			d.outputTextToDialog(d.pendingOutput, d.pendingContinues)
			d.pendingOutput = ""
			d.pendingContinues = false
			d.endPauseAnimation()

			logging.Debug(logging.CategoryUI, "All pauses complete")
		}
	} else {
		// No more output - done pausing
		d.endPauseAnimation()

		logging.Debug(logging.CategoryUI, "Pause complete, no more output")
	}
}

// finishCurrentPauseWithSegments handles pause completion for segment-based (runic) output.
func (d *LinearTalkDialog) finishCurrentPauseWithSegments() {
	// Check if pending segments contain another pause marker
	if beforePause, afterPause, hasPause := splitSegmentsAtPause(d.pendingSegments); hasPause {
		// Display segments before next pause
		d.outputSegmentsToDialog(beforePause, d.pendingContinues)

		// Update pending segments for next pause
		d.pendingSegments = afterPause
		d.pendingContinues = outputContinuesAfterSegments(beforePause, d.pendingContinues)

		// Start next pause animation
		d.pauseStartMs = d.gameScene.gameState.ElapsedMs
		d.currentPauseIndex++

		logging.Debug(logging.CategoryUI, "Starting segment pause %d, pending: %d segments", d.currentPauseIndex, len(d.pendingSegments))
	} else {
		// No more pauses - display remaining segments
		d.outputSegmentsToDialog(d.pendingSegments, d.pendingContinues)
		d.pendingSegments = nil
		d.pendingContinues = false
		d.endPauseAnimation()

		logging.Debug(logging.CategoryUI, "All segment pauses complete")
	}
}

// enterDismissalWaitState transitions to the dismissal wait state after conversation completion.
// The footer advertises Space to close while the original fresh-key wait remains.
// Used after JoinParty, BYE, or other conversation-ending commands.
//
// Implements proper state-based debouncing:
// Phase 1: Records currently pressed keys (e.g., Enter from submitting BYE)
// Phase 2: Waits for all recorded keys to be released
// Phase 3: Detects NEW keypress after release
func (d *LinearTalkDialog) enterDismissalWaitState() {
	d.waitingForDismissal = true

	// Prepare keyboard for transition with 150ms post-accept delay
	// This prevents the dismissal keypress from bleeding into underlying game input.
	d.gameScene.keyboard.PrepareForTransition(150)

	// Flush legitimate callback feedback before presenting the terminal footer.
	if d.callbacks != nil {
		accumulatedOutput := d.callbacks.GetAccumulatedOutput()
		if accumulatedOutput != "" {
			// Use AddText to properly advance to new line after callback output
			d.Output.AddText(accumulatedOutput)
			d.callbacks.ClearAccumulatedOutput()
		}
	}

	// Hide input prompt, placeholder, and cursor
	d.TextInput.ClearPrompt()
	d.TextInput.ClearPlaceholder()
	d.TextInput.SetCursorVisible(false)

	logging.Debug(logging.CategoryUI, "Entered dismissal wait state")
}

// enterKeyWaitState blocks input; the footer presents the KeyWait (0x8F) hint.
// This is triggered when the conversation engine encounters a KeyWait command in the script,
// which pauses the conversation flow until the player presses any key.
//
// Unlike dismissal state (which ends the conversation), KeyWait continues the conversation
// after the keypress by calling engine.ProcessInput("") to resume script processing.
//
// Implements proper state-based debouncing (same pattern as dismissal):
// Phase 1: Records currently pressed keys (e.g., Enter from submitting keyword)
// Phase 2: Waits for all recorded keys to be released
// Phase 3: Detects NEW keypress after release
//
// UI Changes:
// - Hides input prompt, placeholder, and cursor (user can't type text)
// - Shows Space to continue in the footer without changing conversation output
// - Sets waitingForKeypressContinue flag to block normal input processing
func (d *LinearTalkDialog) enterKeyWaitState() {
	d.waitingForKeypressContinue = true

	// Prepare keyboard for transition with 50ms post-accept delay
	// This prevents the continue keypress from registering as subsequent input
	d.gameScene.keyboard.PrepareForTransition(50)
	if d.callbacks != nil {
		accumulatedOutput := d.callbacks.GetAccumulatedOutput()
		if accumulatedOutput != "" {
			d.Output.AddText(accumulatedOutput)
			d.callbacks.ClearAccumulatedOutput()
		}
	}

	// Hide input UI elements (user can't type during KeyWait)
	d.TextInput.ClearPrompt()
	d.TextInput.ClearPlaceholder()
	// REDUX DEVIATION: TALK-0003: original control byte 0x8F waits without visible output. Redux
	// shows a footer hint without inserting instructions into speech or input.
	d.TextInput.SetCursorVisible(false)

	logging.Debug(logging.CategoryUI, "Entered KeyWait state")
}

// consumeCurrentlyPressedKeys marks all currently pressed keys as consumed in the keyboard state
// This prevents keys from bleeding through to underlying game input handlers
// when the talk dialog closes. By calling TryToRegisterKeyPress on all pressed keys,
// they're added to consumedKeysUntilRelease and won't be processed by subsequent handlers.
func (d *LinearTalkDialog) consumeCurrentlyPressedKeys() {
	// Check all alphanumeric keys
	for i := int(ebiten.KeyA); i <= int(ebiten.KeyZ); i++ {
		if ebiten.IsKeyPressed(ebiten.Key(i)) {
			d.gameScene.keyboard.TryToRegisterKeyPress(ebiten.Key(i))
		}
	}
	for i := int(ebiten.Key0); i <= int(ebiten.Key9); i++ {
		if ebiten.IsKeyPressed(ebiten.Key(i)) {
			d.gameScene.keyboard.TryToRegisterKeyPress(ebiten.Key(i))
		}
	}

	// Check common control keys
	controlKeys := []ebiten.Key{
		ebiten.KeySpace, ebiten.KeyEnter, ebiten.KeyEscape,
		ebiten.KeyBackspace, ebiten.KeyTab,
	}
	for _, key := range controlKeys {
		if ebiten.IsKeyPressed(key) {
			d.gameScene.keyboard.TryToRegisterKeyPress(key)
		}
	}
}
