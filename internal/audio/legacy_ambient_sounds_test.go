// Package audio provides application audio playback and shared authoring mechanisms.
package audio

import (
	"errors"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/hajimehoshi/ebiten/v2/audio"

	"github.com/bradhannah/UltimaVRedux/internal/logging"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
)

// AmbientSoundManager handles proximity-based ambient sounds
type AmbientSoundManager struct {
	mu                 sync.RWMutex
	audioManager       *AudioManager
	audioContext       *audio.Context
	activeSounds       map[AmbientSoundType]*AmbientSoundInstance // Key: sound type (not position)
	soundDefinitions   map[indexes.SpriteIndex]AmbientSoundDefinition
	lastPlayerPosition *references.Position
	enabled            bool
	now                func() time.Time
	desired            map[AmbientSoundType]ambientCandidate
	attempts           map[AmbientSoundType]ambientAttempt
	openStream         func(string) (io.ReadSeeker, error)

	// Base definitions for each sound type
	soundTypeDefinitions map[AmbientSoundType]AmbientSoundDefinition
}

// AmbientSoundInstance represents an active ambient sound at a specific location
type AmbientSoundInstance struct {
	emitter        string
	Definition     AmbientSoundDefinition
	Position       references.Position
	Player         *gainPlayer
	Volume         float64
	VolumeModifier float64 // Dynamic volume scaling (0.0 to 1.0), separate from BaseVolume
	FadeState      AmbientFadeState
	FadeStart      time.Time
	FadeFrom       float64 // Envelope when the current fade began.
	FadeGain       float64 // Envelope independent of category/position gains.
	LastDistance   float64
}

// AmbientFadeState represents the current fade state of an ambient sound
type AmbientFadeState int

// Ambient envelopes retain a voice through release and reversible re-entry.
const (
	AmbientFadeNone AmbientFadeState = iota
	AmbientFadeIn
	AmbientFadeOut
	AmbientFadeStopping
)

// NewAmbientSoundManager creates a new ambient sound manager
func NewAmbientSoundManager(audioManager *AudioManager) *AmbientSoundManager {
	asm := &AmbientSoundManager{
		mu:                   sync.RWMutex{},
		audioManager:         audioManager,
		audioContext:         audioManager.sharedContext,
		activeSounds:         make(map[AmbientSoundType]*AmbientSoundInstance),
		soundDefinitions:     make(map[indexes.SpriteIndex]AmbientSoundDefinition),
		enabled:              true,
		now:                  audioManager.clock.Now,
		lastPlayerPosition:   nil,
		desired:              make(map[AmbientSoundType]ambientCandidate),
		attempts:             make(map[AmbientSoundType]ambientAttempt),
		openStream:           nil,
		soundTypeDefinitions: ambientSoundDefinitions(audioManager.library),
	}
	asm.openStream = asm.loadAmbientAudioFile

	if audioManager.redux {
		asm.initializeAllAmbientSounds()
	}

	return asm
}

// initializeAllAmbientSounds sets up all ambient sound mappings using a generic system
func (asm *AmbientSoundManager) initializeAllAmbientSounds() {
	routing := asm.audioManager.library.document.Routing
	for spriteIndex, soundType := range ambientSpriteMappingsFor(routing) {
		if definition, exists := asm.soundTypeDefinitions[soundType]; exists {
			profile, gain := routing.TileAmbientSettings(tileAudioKey(spriteIndex))
			definition.MaxDistance = profile.RadiusTiles
			definition.BaseVolume *= gain
			asm.soundDefinitions[spriteIndex] = definition
		}
	}
}

// UpdatePlayerPosition updates the player's position and manages ambient sounds
func (asm *AmbientSoundManager) UpdatePlayerPosition(playerPos references.Position, nearbyTiles []TileWithPosition) {
	asm.UpdatePlayerPositionWithDynamicSources(playerPos, nearbyTiles, nil)
}

// UpdatePlayerPositionWithDynamicSources reconciles an explicit source snapshot.
// Empty source lists mean departure; skipping discovery does not.
func (asm *AmbientSoundManager) UpdatePlayerPositionWithDynamicSources(
	playerPos references.Position, nearbyTiles []TileWithPosition, dynamicSources []AmbientSoundSource,
) {
	asm.mu.Lock()
	defer asm.mu.Unlock()
	if !asm.enabled {
		return
	}
	asm.lastPlayerPosition = &playerPos
	asm.reconcileSourcesLocked(asm.collectSources(playerPos, nearbyTiles, dynamicSources))
	asm.serviceLocked()
}

func (asm *AmbientSoundManager) prepareAmbientVoice(
	definition AmbientSoundDefinition, position references.Position, distance, volumeModifier float64,
) (*AmbientSoundInstance, error) {
	audioStream, err := asm.openStream(definition.FileName)
	if err != nil {
		return nil, fmt.Errorf("load ambient sound %s: %w", definition.FileName, err)
	}
	prepared, err := asm.prepareAmbientStream(audioStream, definition)
	if err != nil {
		return nil, errors.Join(err, closeRuntimeSource(audioStream))
	}
	// Create audio player with the loaded stream
	owned := &ownedRuntimeStream{ReadSeeker: prepared, owner: audioStream}
	player, err := newGainPlayerWithBackend(asm.audioManager.backend, owned)
	if err != nil {
		return nil, fmt.Errorf("ambient player %s: %w", definition.FileName, err)
	}

	// Calculate initial volume based on distance and apply volume modifier
	baseVolume := asm.calculateVolumeFromDistance(distance, definition.MaxDistance, definition.BaseVolume)
	volume := baseVolume * volumeModifier // Apply the modifier to get final volume

	instance := &AmbientSoundInstance{
		emitter:        "",
		Definition:     definition,
		Position:       position,
		Player:         player,
		Volume:         volume,
		VolumeModifier: volumeModifier, // Store the modifier separately
		FadeState:      AmbientFadeIn,
		FadeStart:      asm.now(),
		FadeFrom:       0,
		FadeGain:       0,
		LastDistance:   distance,
	}

	if !definition.LoopSound || definition.FadeInTime <= 0 {
		instance.FadeState = AmbientFadeNone
		instance.FadeGain = 1
	}
	player.SetVolume(asm.effectiveVolume(volume * instance.FadeGain))

	return instance, nil
}

func (asm *AmbientSoundManager) prepareAmbientStream(
	source io.ReadSeeker, definition AmbientSoundDefinition,
) (io.ReadSeeker, error) {
	if source == nil {
		return nil, errSoundStreamUnavailable
	}
	library := asm.audioManager.library.document
	cue := library.Cues[string(definition.SoundType)]
	window := library.Recordings[cue.RecordingID].PlaybackWindow
	stream, err := boundedRecording(source, asm.audioManager.backend.SampleRate(), window)
	if err != nil {
		return nil, fmt.Errorf("ambient selection %s: %w", definition.FileName, err)
	}
	if definition.LoopSound {
		length, err := decodedStreamLength(stream)
		if err != nil {
			return nil, fmt.Errorf("ambient loop %s: %w", definition.FileName, err)
		}

		return audio.NewInfiniteLoop(stream, length), nil
	}

	return stream, nil
}

// updateAmbientSound updates an existing ambient sound based on distance
func (asm *AmbientSoundManager) updateAmbientSound(instance *AmbientSoundInstance, distance float64) {
	// Calculate base volume from distance, then apply the instance's volume modifier
	baseVolume := asm.calculateVolumeFromDistance(
		distance, instance.Definition.MaxDistance, instance.Definition.BaseVolume)
	newVolume := baseVolume * instance.VolumeModifier // Apply the modifier

	asm.updateFade(instance)
	effectiveVolume := asm.effectiveVolume(newVolume * instance.FadeGain)

	// Update player volume with real audio data
	if instance.Player != nil {
		instance.Player.SetVolume(effectiveVolume)
	}
	instance.Volume = newVolume
	instance.LastDistance = distance
}

func (asm *AmbientSoundManager) updateFade(instance *AmbientSoundInstance) {
	if instance.FadeState != AmbientFadeIn && instance.FadeState != AmbientFadeOut {
		return
	}
	duration := instance.Definition.FadeInTime
	if instance.FadeState == AmbientFadeOut {
		duration = instance.Definition.FadeOutTime
	}
	progress := 1.0
	if duration > 0 {
		progress = min(1, max(0, float64(asm.now().Sub(instance.FadeStart))/float64(duration)))
	}
	if instance.FadeState == AmbientFadeIn {
		instance.FadeGain = instance.FadeFrom + (1-instance.FadeFrom)*progress
		if progress == 1 {
			instance.FadeState = AmbientFadeNone
		}
	} else {
		instance.FadeGain = instance.FadeFrom * (1 - progress)
	}
}

// fadeOutAmbientSound begins fading out an ambient sound
func (asm *AmbientSoundManager) fadeOutAmbientSound(instance *AmbientSoundInstance) {
	if instance.FadeState != AmbientFadeOut && instance.FadeState != AmbientFadeStopping {
		asm.updateFade(instance)
		instance.FadeState = AmbientFadeOut
		instance.FadeFrom = instance.FadeGain
		instance.FadeStart = asm.now()
	}
}

func (asm *AmbientSoundManager) effectiveVolume(volume float64) float64 {
	return asm.audioManager.config.GetEffectiveAmbientVolume(volume)
}

// RefreshVolume applies the current user preference to existing ambient voices,
// including those fading out while their source is out of range.
func (asm *AmbientSoundManager) RefreshVolume() {
	asm.mu.Lock()
	defer asm.mu.Unlock()
	for _, instance := range asm.activeSounds {
		asm.updateFade(instance)
		instance.Player.SetVolume(asm.effectiveVolume(instance.Volume * instance.FadeGain))
	}
}

// calculateVolumeFromDistance applies the shared quadratic ambient profile.
func (asm *AmbientSoundManager) calculateVolumeFromDistance(distance, maxDistance, baseVolume float64) float64 {
	return baseVolume * AmbientDistanceGain(distance, maxDistance)
}

// SetEnabled enables or disables the ambient sound system
func (asm *AmbientSoundManager) SetEnabled(enabled bool) {
	asm.mu.Lock()
	defer asm.mu.Unlock()

	if !enabled && asm.enabled {
		// Stop all active sounds
		for _, instance := range asm.activeSounds {
			instance.Player.Pause()
			if err := instance.Player.Close(); err != nil {
				logging.Warn(logging.CategoryAudio, "Disabling ambient sound: %v", err)
			}
		}
		asm.activeSounds = make(map[AmbientSoundType]*AmbientSoundInstance)
		clear(asm.desired)
		clear(asm.attempts)
	}

	asm.enabled = enabled
}

// IsEnabled returns whether the ambient sound system is enabled
func (asm *AmbientSoundManager) IsEnabled() bool {
	asm.mu.RLock()
	defer asm.mu.RUnlock()

	return asm.enabled
}

// GetActiveAmbientSounds returns information about currently active ambient sounds
func (asm *AmbientSoundManager) GetActiveAmbientSounds() map[AmbientSoundType]AmbientSoundInfo {
	asm.mu.RLock()
	defer asm.mu.RUnlock()

	result := make(map[AmbientSoundType]AmbientSoundInfo)
	for soundType, instance := range asm.activeSounds {
		result[soundType] = AmbientSoundInfo{
			Position:  instance.Position,
			SoundType: instance.Definition.SoundType,
			Volume:    instance.Volume,
			Distance:  instance.LastDistance,
			FadeState: instance.FadeState,
		}
	}

	return result
}

// AmbientSoundInfo provides information about an active ambient sound
type AmbientSoundInfo struct {
	Position  references.Position
	SoundType AmbientSoundType
	Volume    float64
	Distance  float64
	FadeState AmbientFadeState
}

// GetSoundTypeDefinition returns the base definition for a given ambient sound type
func (asm *AmbientSoundManager) GetSoundTypeDefinition(soundType AmbientSoundType) (AmbientSoundDefinition, bool) {
	asm.mu.RLock()
	defer asm.mu.RUnlock()

	definition, exists := asm.soundTypeDefinitions[soundType]

	return definition, exists
}

// Close properly shuts down the ambient sound manager
func (asm *AmbientSoundManager) Close() error {
	asm.mu.Lock()
	defer asm.mu.Unlock()

	var closeErrors []error
	// Stop and close all active sounds, preserving every close failure.
	for _, instance := range asm.activeSounds {
		if instance.Player != nil {
			instance.Player.Pause()
			if err := instance.Player.Close(); err != nil {
				closeErrors = append(closeErrors, fmt.Errorf("closing ambient sound: %w", err))
			}
		}
	}

	asm.activeSounds = make(map[AmbientSoundType]*AmbientSoundInstance)
	clear(asm.desired)
	clear(asm.attempts)
	asm.enabled = false

	return errors.Join(closeErrors...)
}

// loadAmbientAudioFile loads an audio file from embedded assets and returns a readable stream
func (asm *AmbientSoundManager) loadAmbientAudioFile(filename string) (io.ReadSeeker, error) {
	fileData, err := asm.audioManager.library.readFile(filename)
	if err != nil {
		return nil, fmt.Errorf("read exact ambient file %s: %w", filename, err)
	}

	// Use multi-format decoder (supports WAV, MP3, OGG)
	stream, err := DecodeAudioStream(fileData, filename, asm.audioManager.backend.SampleRate())
	if err != nil {
		return nil, fmt.Errorf("failed to decode ambient audio file %s: %w", filename, err)
	}

	return stream, nil
}

// AddCustomAmbientSound allows adding custom ambient sound definitions for specific sprites
func (asm *AmbientSoundManager) AddCustomAmbientSound(
	spriteIndex indexes.SpriteIndex, definition AmbientSoundDefinition,
) {
	asm.mu.Lock()
	defer asm.mu.Unlock()

	asm.soundDefinitions[spriteIndex] = definition
}
