package audio

import (
	"errors"
	"fmt"
	"sync"

	"github.com/hajimehoshi/ebiten/v2/audio"

	"github.com/bradhannah/UltimaVRedux/internal/config"
	"github.com/bradhannah/UltimaVRedux/internal/game_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
)

// AudioManager is the central coordinator for all audio functionality
// Its facade serializes configuration and voice ownership. Callers must not
// concurrently control subsystems obtained through GetMusicManager directly.
//
//nolint:revive // Preserve the established application facade name during in-place hardening.
type AudioManager struct {
	// Core components
	soundEffectPlayer    *SoundEffectPlayer
	backgroundMusic      *BackgroundMusicSystem
	positionalCalculator *PositionalAudioCalculator
	walkingSounds        *WalkingSoundManager
	musicManager         *MusicManager
	ambientSounds        *AmbientSoundManager
	config               *AudioConfiguration

	// Shared resources (exposed for subsystems)
	sharedContext *audio.Context

	// State management
	enabled           bool
	closed            bool
	ambientSuppressed bool
	mutex             sync.RWMutex

	// Asset management
	assetsLoaded bool

	// Preview audio players
	ambientPreviewPlayer *gainPlayer
	ambientPreviewGain   float64
}

// ErrAudioClosed reports an operation on a terminally closed manager.
var ErrAudioClosed = errors.New("audio manager is closed")

// NewAudioManager creates a new AudioManager with all subsystems initialized using default configuration
func NewAudioManager() (*AudioManager, error) {
	config, err := NewAudioConfiguration()
	if err != nil {
		return nil, fmt.Errorf("failed to create audio configuration: %w", err)
	}

	return newAudioManagerWithConfig(config)
}

// NewAudioManagerWithConfig creates a new AudioManager using provided configuration from Viper config
func NewAudioManagerWithConfig(viperConfig *config.AudioConfiguration) (*AudioManager, error) {
	if viperConfig == nil {
		return NewAudioManager() // Fall back to defaults
	}

	// Convert from Viper config (integer slider values 0-10) to internal audio config (float64 0.0-1.0)
	audioConfig, err := NewAudioConfiguration()
	if err != nil {
		return nil, fmt.Errorf("create audio defaults: %w", err)
	}
	audioConfig.SoundEffectVolume = ConvertSliderToVolume(viperConfig.SoundEffectVolume)
	audioConfig.MusicVolume = ConvertSliderToVolume(viperConfig.MusicVolume)
	audioConfig.AmbientVolume = ConvertSliderToVolume(viperConfig.AmbientVolume)

	return newAudioManagerWithConfig(audioConfig)
}

// newAudioManagerWithConfig is the internal constructor that both public constructors use
func newAudioManagerWithConfig(config *AudioConfiguration) (*AudioManager, error) {
	config = config.Clone()
	// Create shared audio context
	audioContext := getSharedAudioContext()

	soundPlayer, err := NewSoundEffectPlayerWithContext(config, audioContext)
	if err != nil {
		return nil, fmt.Errorf("failed to create sound effect player: %w", err)
	}

	musicSystem, err := NewBackgroundMusicSystemWithContext(config, audioContext)
	if err != nil {
		return nil, fmt.Errorf("failed to create background music system: %w", err)
	}

	positionalCalc := NewPositionalAudioCalculator()
	positionalCalc.SetConfiguration(config)

	manager := new(AudioManager)
	manager.soundEffectPlayer, manager.backgroundMusic = soundPlayer, musicSystem
	manager.positionalCalculator, manager.config = positionalCalc, config
	manager.sharedContext, manager.enabled = audioContext, true

	// Initialize walking sounds manager after main manager is created
	manager.walkingSounds = NewWalkingSoundManager(manager)

	// Initialize music manager for location-based music
	manager.musicManager = NewMusicManager(manager)
	manager.musicManager.SetSuppressed(musicMuted, config.GetEffectiveMusicVolume() == 0)

	// Initialize ambient sound manager for proximity-based audio
	manager.ambientSounds = NewAmbientSoundManager(manager)

	return manager, nil
}

// PlaySoundEffect plays a sound effect using the SystemCallbacks enum system
// This is the one-shot convenience used by presentation-only callers.
func (am *AudioManager) PlaySoundEffect(effect game_state.SoundEffect) error {
	return am.PlaySound(game_state.SoundPlaybackRequest{
		Effect: effect, Mode: game_state.SoundPlaybackOneShot, Volume: 1.0, FadeInMs: 0,
	})
}

// PlaySound handles one complete typed playback request.
func (am *AudioManager) PlaySound(request game_state.SoundPlaybackRequest) error {
	if err := am.ensureAssetsLoaded(); err != nil {
		return err
	}
	am.mutex.RLock()
	defer am.mutex.RUnlock()

	if !am.enabled {
		return nil
	}

	if err := am.soundEffectPlayer.Play(request); err != nil {
		return err
	}
	if isInstrumentEffect(request.Effect) && am.soundEffectPlayer.hasInstrumentMusic() {
		am.musicManager.LocalMusic()
	}

	return nil
}

// StopSound stops or fades every active instance matching one effect.
func (am *AudioManager) StopSound(request game_state.SoundStopRequest) {
	am.mutex.RLock()
	defer am.mutex.RUnlock()

	if !am.enabled {
		return // Silent operation when disabled
	}

	if am.soundEffectPlayer != nil {
		if request.FadeOutMs > 0 {
			am.soundEffectPlayer.FadeOutEffect(request.Effect, request.FadeOutMs)

			return
		}
		am.soundEffectPlayer.StopEffect(request.Effect)
	}
}

// PlaySoundEffectAt plays a sound effect with positional audio
func (am *AudioManager) PlaySoundEffectAt(effect game_state.SoundEffect, playerX, playerY, sourceX, sourceY int) error {
	if err := am.ensureAssetsLoaded(); err != nil {
		return err
	}
	am.mutex.RLock()
	defer am.mutex.RUnlock()

	if !am.enabled {
		return nil
	}

	// Calculate positional audio parameters
	volume, _ := am.positionalCalculator.CalculateAudio(playerX, playerY, sourceX, sourceY)

	return am.soundEffectPlayer.Play(game_state.SoundPlaybackRequest{
		Effect: effect, Mode: game_state.SoundPlaybackOneShot, Volume: volume, FadeInMs: 0,
	})
}

// SetSoundEffectVolume adjusts sound effect volume (0.0 to 1.0)
func (am *AudioManager) SetSoundEffectVolume(volume float64) error {
	am.mutex.Lock()
	defer am.mutex.Unlock()
	if am.closed {
		return ErrAudioClosed
	}

	// Update the configuration
	if err := am.config.SetSoundEffectVolume(volume); err != nil {
		return err
	}

	am.soundEffectPlayer.RefreshVolume()

	return nil
}

// SetMusicVolume adjusts background music volume (0.0 to 1.0)
func (am *AudioManager) SetMusicVolume(volume float64) error {
	am.mutex.Lock()
	defer am.mutex.Unlock()
	if am.closed {
		return ErrAudioClosed
	}

	// Update the configuration
	if err := am.config.SetMusicVolume(volume); err != nil {
		return err
	}
	am.musicManager.SetSuppressed(musicMuted, volume == 0)

	// Apply volume change to currently playing music
	return am.backgroundMusic.RefreshVolume()
}

// SetAmbientVolume adjusts ambient sound volume (0.0 to 1.0)
func (am *AudioManager) SetAmbientVolume(volume float64) error {
	am.mutex.Lock()
	defer am.mutex.Unlock()
	if am.closed {
		return ErrAudioClosed
	}

	// Update the configuration
	if err := am.config.SetAmbientVolume(volume); err != nil {
		return err
	}

	am.ambientSounds.RefreshVolume()
	if am.ambientPreviewPlayer != nil {
		am.ambientPreviewPlayer.SetVolume(am.config.GetEffectiveAmbientVolume(am.ambientPreviewGain))
	}

	return nil
}

// PlayBackgroundMusic starts playing background music
func (am *AudioManager) PlayBackgroundMusic(musicTrack string) error {
	if err := am.ensureAssetsLoaded(); err != nil {
		return err
	}
	am.mutex.RLock()
	defer am.mutex.RUnlock()

	if !am.enabled {
		return nil
	}

	return am.backgroundMusic.Play(musicTrack)
}

// StopBackgroundMusic stops current background music
func (am *AudioManager) StopBackgroundMusic() error {
	am.mutex.RLock()
	defer am.mutex.RUnlock()

	return am.backgroundMusic.Stop()
}

// PlayMusicPreview auditions the authored mix while the scene's menu owner
// suspends scheduling. It never modifies the exploration policy's gain/history.
func (am *AudioManager) PlayMusicPreview(track string) error {
	if err := am.ensureAssetsLoaded(); err != nil {
		return err
	}
	am.mutex.RLock()
	defer am.mutex.RUnlock()
	if !am.enabled {
		return nil
	}
	if err := am.backgroundMusic.SetVolume(1); err != nil {
		return err
	}

	return am.backgroundMusic.Play(track)
}

// PlayWalkingSound plays appropriate walking sound based on tile surface
func (am *AudioManager) PlayWalkingSound(tile *references.Tile) error {
	am.mutex.RLock()
	if am.closed {
		am.mutex.RUnlock()

		return ErrAudioClosed
	}
	if !am.enabled {
		am.mutex.RUnlock()

		return nil
	}
	if am.walkingSounds == nil {
		am.mutex.RUnlock()
		panic("walkingSounds is nil - programming error in AudioManager initialization")
	}
	walkingSounds := am.walkingSounds
	am.mutex.RUnlock()

	return walkingSounds.PlayWalkingSound(tile)
}

// SetEnabled enables or disables the entire audio system
func (am *AudioManager) SetEnabled(enabled bool) {
	am.mutex.Lock()
	defer am.mutex.Unlock()
	if am.closed {
		return
	}

	am.enabled = enabled
	am.musicManager.SetSuppressed(musicSystem, !enabled)
	if !enabled {
		// Stop all currently playing audio
		am.backgroundMusic.Pause()
		am.soundEffectPlayer.StopAll()
		am.ambientSounds.SetEnabled(false)
		am.stopAmbientPreview()
	} else {
		am.ambientSounds.SetEnabled(!am.ambientSuppressed)
	}
}

// SetAmbientSuppressed stops proximity voices immediately while a workflow owns
// the soundscape. The next ordinary scene update rebuilds current sources.
func (am *AudioManager) SetAmbientSuppressed(suppressed bool) {
	am.mutex.Lock()
	defer am.mutex.Unlock()
	am.ambientSuppressed = suppressed
	am.ambientSounds.SetEnabled(am.enabled && !suppressed)
}

// IsEnabled returns whether audio is currently enabled
func (am *AudioManager) IsEnabled() bool {
	am.mutex.RLock()
	defer am.mutex.RUnlock()

	return am.enabled
}

// Update processes audio system updates like fading effects
// Should be called regularly (e.g., every frame) to handle fade-in/fade-out
func (am *AudioManager) Update() {
	am.mutex.RLock()

	if !am.enabled {
		am.mutex.RUnlock()

		return
	}

	if am.soundEffectPlayer != nil {
		am.soundEffectPlayer.UpdateFading()
	}
	if am.soundEffectPlayer.hasInstrumentMusic() || am.ambientSounds.hasBardMusic() {
		am.musicManager.LocalMusic()
	}
	am.mutex.RUnlock()
	// Music may lazily load assets after a request made while audio was off.
	// Never hold the manager lock across that loading boundary.
	am.musicManager.Update()
}

// GetConfiguration returns a detached snapshot. Use setters to change live gains.
func (am *AudioManager) GetConfiguration() *AudioConfiguration {
	am.mutex.RLock()
	defer am.mutex.RUnlock()

	return am.config.Clone()
}

// Close terminally retires owned voices, but never the borrowed process context.
// It is idempotent; SetEnabled cannot reopen a closed manager.
func (am *AudioManager) Close() error {
	am.mutex.Lock()
	defer am.mutex.Unlock()
	if am.closed {
		return nil
	}
	am.closed = true
	am.enabled = false
	am.musicManager.SetSuppressed(musicSystem, true)

	var closeErrors []error

	if am.backgroundMusic != nil {
		if err := am.backgroundMusic.Close(); err != nil {
			closeErrors = append(closeErrors, fmt.Errorf("closing background music: %w", err))
		}
	}

	if am.soundEffectPlayer != nil {
		if err := am.soundEffectPlayer.Close(); err != nil {
			closeErrors = append(closeErrors, fmt.Errorf("closing sound effects: %w", err))
		}
	}

	if am.ambientSounds != nil {
		if err := am.ambientSounds.Close(); err != nil {
			closeErrors = append(closeErrors, fmt.Errorf("closing ambient sounds: %w", err))
		}
	}

	// Clean up ambient preview player
	am.stopAmbientPreview()

	return errors.Join(closeErrors...)
}

// loadAssets performs lazy loading of audio assets
// The caller must hold am.mutex exclusively.
func (am *AudioManager) loadAssets() error {
	// Load sound effect assets
	if err := am.soundEffectPlayer.LoadAssets(); err != nil {
		return fmt.Errorf("failed to load sound effect assets: %w", err)
	}

	// Load music assets
	if err := am.backgroundMusic.LoadAssets(); err != nil {
		return fmt.Errorf("failed to load music assets: %w", err)
	}

	am.assetsLoaded = true

	return nil
}

func (am *AudioManager) ensureAssetsLoaded() error {
	am.mutex.RLock()
	if am.closed {
		am.mutex.RUnlock()

		return ErrAudioClosed
	}
	if !am.enabled || am.assetsLoaded {
		am.mutex.RUnlock()

		return nil
	}
	am.mutex.RUnlock()

	am.mutex.Lock()
	defer am.mutex.Unlock()
	if am.closed {
		return ErrAudioClosed
	}
	if !am.enabled || am.assetsLoaded {
		return nil
	}
	if err := am.loadAssets(); err != nil {
		return fmt.Errorf("failed to load audio assets: %w", err)
	}

	return nil
}

// GetMusicManager returns the music manager for location-based music control
func (am *AudioManager) GetMusicManager() *MusicManager {
	return am.musicManager
}

// PlayMusicForLocation starts playing appropriate music for a specific location
func (am *AudioManager) PlayMusicForLocation(location references.Location) error {
	if am.musicManager == nil {
		panic("musicManager is nil - programming error in AudioManager initialization")
	}

	return am.musicManager.PlayMusicForLocation(location)
}

// PlayMusicForContext starts playing music for a specific game context (intro, victory, etc.)
func (am *AudioManager) PlayMusicForContext(context string) error {
	if am.musicManager == nil {
		panic("musicManager is nil - programming error in AudioManager initialization")
	}

	return am.musicManager.PlayMusicForContext(context)
}

// PauseBackgroundMusic temporarily pauses background music (for camping, dialogs, etc.)
func (am *AudioManager) PauseBackgroundMusic() {
	if am.musicManager == nil {
		panic("musicManager is nil - programming error in AudioManager initialization")
	}
	am.musicManager.PauseBackgroundMusic()
}

// ResumeBackgroundMusic resumes previously paused background music
func (am *AudioManager) ResumeBackgroundMusic() {
	if am.musicManager == nil {
		panic("musicManager is nil - programming error in AudioManager initialization")
	}
	am.musicManager.ResumeBackgroundMusic()
}

// UpdateAmbientSounds updates ambient sounds based on player position and nearby tiles
func (am *AudioManager) UpdateAmbientSounds(playerPos references.Position, nearbyTiles []TileWithPosition) {
	am.UpdateAmbientSoundsWithDynamicSources(playerPos, nearbyTiles, nil)
}

// UpdateAmbientSoundsWithDynamicSources reconciles static tiles and dynamic emitters.
func (am *AudioManager) UpdateAmbientSoundsWithDynamicSources(
	playerPos references.Position, nearbyTiles []TileWithPosition, dynamicSources []AmbientSoundSource,
) {
	am.mutex.RLock()
	defer am.mutex.RUnlock()
	if !am.enabled {
		return
	}
	if am.ambientSounds == nil {
		panic("ambientSounds is nil - programming error in AudioManager initialization")
	}
	am.ambientSounds.UpdatePlayerPositionWithDynamicSources(playerPos, nearbyTiles, dynamicSources)
}

// GetAmbientSoundDefinition returns the base definition for a given ambient sound type
func (am *AudioManager) GetAmbientSoundDefinition(soundType AmbientSoundType) (AmbientSoundDefinition, bool) {
	if am.ambientSounds == nil {
		panic("ambientSounds is nil - programming error in AudioManager initialization")
	}

	return am.ambientSounds.GetSoundTypeDefinition(soundType)
}

// PlayAmbientSoundPreview auditions the currently assigned forest recording.
func (am *AudioManager) PlayAmbientSoundPreview() error {
	am.mutex.Lock()
	defer am.mutex.Unlock()
	if am.closed {
		return ErrAudioClosed
	}

	if !am.enabled {
		return nil // Silent operation when disabled
	}

	if !am.assetsLoaded {
		if err := am.loadAssets(); err != nil {
			return fmt.Errorf("failed to load audio assets: %w", err)
		}
	}

	// Stop any currently playing ambient preview
	am.stopAmbientPreview()

	filePath := libraryCuePath("ambient/AmbientForest")
	if filePath == "" {
		return nil
	}
	fileData, err := soundFiles.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("load ambient preview: %w", err)
	}

	// Decode the audio stream
	cue := productionLibrary().Cues["ambient/AmbientForest"]
	window := productionLibrary().Recordings[cue.RecordingID].PlaybackWindow
	stream, err := decodeRecording(fileData, filePath, am.sharedContext.SampleRate(), window, false)
	if err != nil {
		return fmt.Errorf("decode ambient preview: %w", err)
	}

	// Get stream length for looping
	var streamLength int64
	if lengthStream, ok := stream.(interface{ Length() int64 }); ok {
		streamLength = lengthStream.Length()
	} else {
		streamLength = 44100 // Default 1 second at 44.1kHz
	}

	// Create infinite loop for ambient sound
	loopedStream := audio.NewInfiniteLoop(stream, streamLength)

	// Create audio player
	player, err := newGainPlayer(am.sharedContext, loopedStream)
	if err != nil {
		return fmt.Errorf("failed to create ambient preview player: %w", err)
	}

	// Set volume using ambient volume setting
	am.ambientPreviewGain = cue.VolumeModifier * productionLibrary().recordingCalibration(cue.RecordingID)
	effectiveVolume := am.config.GetEffectiveAmbientVolume(am.ambientPreviewGain)
	player.SetVolume(effectiveVolume)

	// Store player for cleanup
	am.ambientPreviewPlayer = player

	// Start playback
	player.Play()

	return nil
}

// StopAmbientSoundPreview stops the ambient sound preview
func (am *AudioManager) StopAmbientSoundPreview() error {
	am.mutex.Lock()
	defer am.mutex.Unlock()

	am.stopAmbientPreview()

	return nil
}

// stopAmbientPreview is the internal method to stop ambient preview (caller must hold mutex)
func (am *AudioManager) stopAmbientPreview() {
	if am.ambientPreviewPlayer != nil {
		am.ambientPreviewPlayer.PauseAndStopReading()
		am.ambientPreviewPlayer = nil
	}
}
