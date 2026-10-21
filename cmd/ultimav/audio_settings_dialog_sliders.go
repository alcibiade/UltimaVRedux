package main

import (
	"github.com/bradhannah/UltimaVRedux/internal/audio"
	"github.com/bradhannah/UltimaVRedux/internal/game_state"
	"github.com/bradhannah/UltimaVRedux/internal/logging"
	"github.com/bradhannah/UltimaVRedux/internal/ui/widgets"
)

// previewMusicVolume starts background music for preview (called on focus change)
func (d *ConfigurationDialog) previewMusicVolume() {
	if d.audioManager == nil {
		return
	}

	// Only start music if not already playing preview
	if d.previewMusicPlaying {
		return
	}

	d.previewAudioChanges() // Apply volume first

	// Try multiple tracks to see what works
	tracksToTry := []string{"town", "overworld", "title", "combat", "ambient"}

	for _, track := range tracksToTry {
		if err := d.audioManager.PlayMusicPreview(track); err == nil {
			d.previewMusicPlaying = true

			return // Success - stop trying other tracks
		}
	}
}

// adjustMusicPreviewVolume adjusts volume of currently playing music without restarting
func (d *ConfigurationDialog) adjustMusicPreviewVolume() {
	if d.audioManager == nil {
		return
	}

	d.previewAudioChanges() // Apply new volume to currently playing music

	// If no music is currently playing, start it
	if !d.previewMusicPlaying {
		d.previewMusicVolume()
	}
}

// previewSoundEffectVolume plays a fresh sample sound effect at the current SFX volume level
func (d *ConfigurationDialog) previewSoundEffectVolume() {
	if d.audioManager == nil {
		return
	}

	d.previewAudioChanges() // Apply volume first

	// Use a simple sound effect that should work - door opening is basic and common
	effect := game_state.SoundOpenDoor
	if err := d.audioManager.PlaySoundEffect(effect); err != nil {
		// Fallback to push object sound
		if err := d.audioManager.PlaySoundEffect(game_state.SoundPushObject); err != nil {
			logging.Warnf(logging.CategoryAudio, "Playing sound settings preview: %v", err)
		}
	}
}

// previewAmbientVolume starts looping ambient sound for preview (called on focus change)
func (d *ConfigurationDialog) previewAmbientVolume() {
	if d.audioManager == nil {
		return
	}

	// Only start ambient sound if not already playing preview
	if d.previewAmbientPlaying {
		d.adjustAmbientPreviewVolume()

		return
	}

	// Use the dedicated ambient preview method that plays AmbientNatureOutside.ogg directly
	if err := d.audioManager.PlayAmbientSoundPreview(); err != nil {
		return
	}

	d.previewAmbientPlaying = true
}

// adjustAmbientPreviewVolume adjusts volume of currently playing ambient sounds without restarting
func (d *ConfigurationDialog) adjustAmbientPreviewVolume() {
	if d.audioManager == nil {
		return
	}

	// Update the AudioManager's ambient volume configuration
	ambientVolume := audio.ConvertSliderToVolume(d.stagedAudio.AmbientVolume)
	if err := d.audioManager.SetAmbientVolume(ambientVolume); err != nil {
		logging.Warnf(logging.CategoryAudio, "Failed to set ambient volume: %v", err)
	}

	// The ambient preview player volume needs to be updated directly
	// Since we can't directly access the player, restart the ambient preview
	if d.previewAmbientPlaying {
		// Stop current preview
		if err := d.audioManager.StopAmbientSoundPreview(); err != nil {
			logging.Warnf(logging.CategoryAudio, "Failed to stop ambient preview: %v", err)
		}
		// Start new preview with updated volume
		if err := d.audioManager.PlayAmbientSoundPreview(); err != nil {
			logging.Warnf(logging.CategoryAudio, "Failed to restart ambient preview: %v", err)
			d.previewAmbientPlaying = false
		}
	} else {
		// If no ambient sound is currently playing, start it
		d.previewAmbientVolume()
	}
}

// stopPreviewAudio stops any audio preview that might be playing
func (d *ConfigurationDialog) stopPreviewAudio() {
	if d.audioManager == nil {
		return
	}

	// Stop background music preview
	if err := d.audioManager.StopBackgroundMusic(); err != nil {
		logging.Warnf(logging.CategoryAudio, "Failed to stop preview music: %v", err)
	}

	// Stop ambient sound preview
	if err := d.audioManager.StopAmbientSoundPreview(); err != nil {
		logging.Warnf(logging.CategoryAudio, "Failed to stop ambient preview: %v", err)
	}

	// Reset preview flags
	d.previewMusicPlaying = false
	d.previewAmbientPlaying = false
}

func (d *ConfigurationDialog) leaveAudioTab() {
	d.stopPreviewAudio()
	d.musicSlider.SetFocus(false)
	d.sfxSlider.SetFocus(false)
	d.ambientSlider.SetFocus(false)
	d.restorePausedAudio()
}

// Slider positions in the audio settings dialog; focusedSliderIndex takes one of these.
const (
	musicSliderIndex = iota
	sfxSliderIndex
	ambientSliderIndex
)

// updateSliderFocus updates the focus state of all sliders based on focusedSliderIndex
func (d *ConfigurationDialog) updateSliderFocus() {
	prevFocusedIndex := -1
	// Find previously focused slider
	switch {
	case d.musicSlider.HasFocus():
		prevFocusedIndex = musicSliderIndex
	case d.sfxSlider.HasFocus():
		prevFocusedIndex = sfxSliderIndex
	case d.ambientSlider.HasFocus():
		prevFocusedIndex = ambientSliderIndex
	}

	// Update focus states
	d.musicSlider.SetFocus(d.focusedSliderIndex == musicSliderIndex)
	d.sfxSlider.SetFocus(d.focusedSliderIndex == sfxSliderIndex)
	d.ambientSlider.SetFocus(d.focusedSliderIndex == ambientSliderIndex)

	// Handle focus change audio preview
	if prevFocusedIndex != d.focusedSliderIndex {
		d.handleSliderFocusChange(prevFocusedIndex, d.focusedSliderIndex)
	}
}

// handleSliderFocusChange handles audio preview when focus changes between sliders
func (d *ConfigurationDialog) handleSliderFocusChange(oldFocus, newFocus int) {
	d.stopMusicPreviewOnFocusChange(oldFocus, newFocus)
	d.stopAmbientPreviewOnFocusChange(oldFocus, newFocus)

	// Start audio preview for newly focused slider
	switch newFocus {
	case musicSliderIndex: // Music slider focused
		d.previewMusicVolume()
	case sfxSliderIndex: // SFX slider focused - no need to play immediately, will play on adjustment
		// Don't play sound effect immediately on focus, only on value change
	case ambientSliderIndex: // Ambient slider focused - apply current ambient volume
		d.previewAmbientVolume()
	}
}

func (d *ConfigurationDialog) stopMusicPreviewOnFocusChange(oldFocus, newFocus int) {
	if oldFocus != musicSliderIndex || newFocus == musicSliderIndex || !d.previewMusicPlaying || d.audioManager == nil {
		return
	}
	if err := d.audioManager.StopBackgroundMusic(); err != nil {
		logging.Warnf(logging.CategoryAudio, "Failed to stop preview music: %v", err)
	}
	d.previewMusicPlaying = false
}

func (d *ConfigurationDialog) stopAmbientPreviewOnFocusChange(oldFocus, newFocus int) {
	if oldFocus != ambientSliderIndex || newFocus == ambientSliderIndex ||
		!d.previewAmbientPlaying || d.audioManager == nil {
		return
	}
	if err := d.audioManager.StopAmbientSoundPreview(); err != nil {
		logging.Warnf(logging.CategoryAudio, "Failed to stop ambient preview: %v", err)
	}
	d.previewAmbientPlaying = false
}

// adjustFocusedSlider adjusts the value of the currently focused slider
func (d *ConfigurationDialog) adjustFocusedSlider(delta int) {
	var slider *widgets.Slider
	switch d.focusedSliderIndex {
	case musicSliderIndex:
		slider = d.musicSlider
	case sfxSliderIndex:
		slider = d.sfxSlider
	case ambientSliderIndex:
		slider = d.ambientSlider
	}

	if slider != nil {
		slider.SetValue(slider.GetValue() + delta)
	}
}

// Slider accessor methods for testing
