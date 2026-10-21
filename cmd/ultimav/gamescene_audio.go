package main

import (
	"github.com/bradhannah/UltimaVRedux/internal/audio"
	"github.com/bradhannah/UltimaVRedux/internal/game_state"
	"github.com/bradhannah/UltimaVRedux/internal/logging"
	"github.com/bradhannah/UltimaVRedux/internal/map_units"
	"github.com/bradhannah/UltimaVRedux/internal/references"
)

// Audio system methods for GameScene
// Handles sound effects, background music, and audio state management

//nolint:ireturn // Gameplay receives only opaque cleanup capability, never a device/backend dependency.
func (g *GameScene) playSoundVoice(request game_state.SoundPlaybackRequest) game_state.SoundVoice {
	if g.audioManager == nil {
		return game_state.SilentSoundVoice{}
	}
	voice, err := g.audioManager.PlayVoice(request)
	if err != nil {
		logging.Warnf(logging.CategoryAudio, "Start owned sound %s: %v", audio.GetSoundEffectName(request.Effect), err)
	}

	return voice
}

func (g *GameScene) playSound(request game_state.SoundPlaybackRequest) {
	// Use the comprehensive audio system to play sound effects
	if g.audioManager != nil {
		err := g.audioManager.PlaySound(request)
		if err != nil {
			// Log error but don't crash the game - audio failures are not critical
			logging.Warnf(logging.CategoryAudio, "Failed to play sound effect %s: %v", audio.GetSoundEffectName(request.Effect), err)
		}
	}
	// If audioManager is nil, silently continue (audio system not available)
}

func (g *GameScene) stopSound(request game_state.SoundStopRequest) {
	// Stop the specified sound effect using the comprehensive audio system
	if g.audioManager != nil {
		g.audioManager.StopSound(request)
		logging.Debugf(logging.CategoryAudio, "Stopped sound effect %s", audio.GetSoundEffectName(request.Effect))
	}
	// If audioManager is nil, silently continue (audio system not available)
}

// updateMusicForCurrentLocation updates background music based on current player location
func (g *GameScene) updateMusicForCurrentLocation() {
	if g.audioManager != nil {
		currentLocation := g.gameState.MapState.PlayerLocation
		err := g.audioManager.GetMusicManager().PlayMusicForScene(currentLocation)
		if err != nil {
			// Log but don't break gameplay - audio failures are not critical
			logging.Warnf(logging.CategoryAudio, "Failed to update music for location %v: %v", currentLocation, err)
		}
	}
}

// pauseBackgroundMusic temporarily pauses background music (for camping, dialogs, etc.)
func (g *GameScene) pauseBackgroundMusic() {
	if g.audioManager != nil {
		g.audioManager.PauseBackgroundMusic()
	}
}

// resumeBackgroundMusic resumes previously paused background music
func (g *GameScene) resumeBackgroundMusic() {
	if g.audioManager != nil {
		// Workflows can change maps while holding score ownership.
		g.updateMusicForCurrentLocation()
		g.audioManager.ResumeBackgroundMusic()
	}
}

// updateSceneMusic runs once after each scene update, including early-return
// modals/cinematics. Only explicit special presentations own the soundscape;
// ordinary action workflows never duck the score.
func (g *GameScene) updateSceneMusic() {
	if g.audioManager == nil {
		return
	}
	var encounter uint64
	var victory bool
	if combat := g.gameState.CombatScene; combat != nil {
		encounter = uint64(combat.EncounterID)
		victory = combat.Victory
	}
	_, sailsRaised := g.navalAudioSituation()
	g.audioManager.GetMusicManager().SetSceneSoundscape(g.gameState.MapState.PlayerLocation, encounter, sailsRaised, victory)
	cinematic := g.hideAmbientSounds || g.finalePresentation != nil ||
		g.shadowlordPresenceActive || g.shadowlordShardActive || g.shadowlordSummonPresentation != nil
	g.audioManager.GetMusicManager().SetSuppressed(audio.MusicCinematic, cinematic)
	g.audioManager.Update()
}

// navalAudioSituation follows the boarded vehicle and sail state, not automatic
// sailing movement or a ship's current animation frame.
func (g *GameScene) navalAudioSituation() (string, bool) {
	vehicle := g.gameState.PartyVehicle.GetVehicleDetails()
	switch vehicle.VehicleType {
	case references.FrigateVehicle:
		return audio.NavalFrigate, vehicle.SailState == map_units.SailRaised
	case references.SkiffVehicle:
		return audio.NavalSkiff, false
	case references.NoPartyVehicle, references.CarpetVehicle, references.HorseVehicle, references.NPC:
		return "", false
	default:
		return "", false
	}
}
