package audio

import (
	"errors"
	"fmt"
	"io"
	"math"
	"sync"
	"time"

	"github.com/hajimehoshi/ebiten/v2/audio"

	"github.com/bradhannah/UltimaVRedux/internal/logging"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
)

// Use soundFiles from sound_effects.go (already declared there)

// AmbientSoundType represents different categories of ambient sounds
type AmbientSoundType string

// Built-in cue identities remain useful to code-owned emitters and tests. New
// authored ambient cues use their string identity directly, without a constant.
const (
	AmbientFire       AmbientSoundType = "ambient/AmbientFire"
	AmbientWater      AmbientSoundType = "ambient/AmbientWater"
	AmbientWind       AmbientSoundType = "ambient/AmbientWind"
	AmbientMechanical AmbientSoundType = "ambient/AmbientMechanical"
	AmbientMagical    AmbientSoundType = "ambient/AmbientMagical"
	AmbientStream     AmbientSoundType = "ambient/AmbientStream"
	AmbientOcean      AmbientSoundType = "ambient/AmbientOcean"
	AmbientLighthouse AmbientSoundType = "ambient/AmbientLighthouse"
	AmbientForest     AmbientSoundType = "ambient/AmbientForest"
	AmbientHum        AmbientSoundType = "ambient/AmbientHum"
	AmbientBard       AmbientSoundType = "ambient/AmbientBard"
	AmbientFountain   AmbientSoundType = "ambient/AmbientFountain"
)

// AmbientSoundDefinition defines how a specific tile produces ambient sound
type AmbientSoundDefinition struct {
	SoundType   AmbientSoundType
	FileName    string  // Audio file name
	MaxDistance float64 // Maximum distance to hear the sound
	BaseVolume  float64 // Contextual amplitude with recording calibration; may exceed unity before category gain.
	LoopSound   bool    // Whether the sound should loop
	FadeInTime  time.Duration
	FadeOutTime time.Duration
}

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

	// Base definitions for each sound type
	soundTypeDefinitions map[AmbientSoundType]AmbientSoundDefinition
}

// AmbientSoundInstance represents an active ambient sound at a specific location
type AmbientSoundInstance struct {
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

const (
	AmbientFadeNone AmbientFadeState = iota
	AmbientFadeIn
	AmbientFadeOut
	AmbientFadeStopping
)

// NewAmbientSoundManager creates a new ambient sound manager
func NewAmbientSoundManager(audioManager *AudioManager) *AmbientSoundManager {
	asm := &AmbientSoundManager{
		audioManager:         audioManager,
		audioContext:         getSharedAudioContext(),
		activeSounds:         make(map[AmbientSoundType]*AmbientSoundInstance),
		soundDefinitions:     make(map[indexes.SpriteIndex]AmbientSoundDefinition),
		enabled:              true,
		now:                  time.Now,
		soundTypeDefinitions: defaultAmbientSoundDefinitions(),
	}

	asm.initializeAllAmbientSounds()

	return asm
}

// defaultAmbientSoundDefinitions is shared with the read-only audio catalog.
// Inspecting definitions must not construct players or an audio context.
func defaultAmbientSoundDefinitions() map[AmbientSoundType]AmbientSoundDefinition {
	definitions := make(map[AmbientSoundType]AmbientSoundDefinition)
	library := productionLibrary()
	for id, profile := range library.Routing.Ambient {
		cue := library.Cues[id]
		kind := AmbientSoundType(id)
		definitions[kind] = AmbientSoundDefinition{
			SoundType: kind, FileName: library.recordingPath(cue.RecordingID),
			MaxDistance: profile.RadiusTiles, BaseVolume: cue.VolumeModifier * library.recordingCalibration(cue.RecordingID),
			LoopSound: true, FadeInTime: time.Duration(profile.FadeInMs) * time.Millisecond,
			FadeOutTime: time.Duration(profile.FadeOutMs) * time.Millisecond,
		}
	}

	return definitions
}

// initializeAllAmbientSounds sets up all ambient sound mappings using a generic system
func (asm *AmbientSoundManager) initializeAllAmbientSounds() {
	routing := productionLibrary().Routing
	for spriteIndex, soundType := range ambientSpriteMappings() {
		if definition, exists := asm.soundTypeDefinitions[soundType]; exists {
			profile, gain := routing.TileAmbientSettings(tileAudioKey(spriteIndex))
			definition.MaxDistance = profile.RadiusTiles
			definition.BaseVolume *= gain
			asm.soundDefinitions[spriteIndex] = definition
		}
	}
}

func ambientSpriteMappings() map[indexes.SpriteIndex]AmbientSoundType {
	mappings := make(map[indexes.SpriteIndex]AmbientSoundType)
	for sprite := range *references.NewTileReferences() {
		assignment := productionLibrary().Routing.Tiles[tileAudioKey(sprite)]
		if assignment.AmbientCue != "" {
			mappings[sprite] = AmbientSoundType(assignment.AmbientCue)
		}
	}

	return mappings
}

// UpdatePlayerPosition updates the player's position and manages ambient sounds
func (asm *AmbientSoundManager) UpdatePlayerPosition(playerPos references.Position, nearbyTiles []TileWithPosition) {
	asm.UpdatePlayerPositionWithDynamicSources(playerPos, nearbyTiles, nil)
}

// UpdatePlayerPositionWithDynamicSources updates the player's position and manages both static and dynamic ambient sounds
func (asm *AmbientSoundManager) UpdatePlayerPositionWithDynamicSources(playerPos references.Position, nearbyTiles []TileWithPosition, dynamicSources []AmbientSoundSource) {
	if !asm.enabled {
		return
	}

	asm.mu.Lock()
	defer asm.mu.Unlock()

	asm.lastPlayerPosition = &playerPos

	// Group by cue, retaining the strongest audible static source for each loop.
	soundTypeDistances := make(map[AmbientSoundType]float64)
	soundTypeDefinitions := make(map[AmbientSoundType]AmbientSoundDefinition)
	soundTypePositions := make(map[AmbientSoundType]references.Position)
	soundTypeModifiers := make(map[AmbientSoundType]float64) // Track volume modifiers separately

	// A quiet nearby torch must not mask a louder brazier using the same cue.
	for _, tilePos := range nearbyTiles {
		definition, exists := asm.soundDefinitions[tilePos.Tile.Index]
		if !exists {
			continue
		}
		distance := asm.calculateDistance(playerPos, tilePos.Position)
		soundType := definition.SoundType
		existingDistance, existing := soundTypeDistances[soundType]
		if strongerAmbientSource(definition, distance, soundTypeDefinitions[soundType], existingDistance, existing) {
			soundTypeDistances[soundType] = distance
			soundTypeDefinitions[soundType] = definition
			soundTypePositions[soundType] = tilePos.Position
			soundTypeModifiers[soundType] = 1
		}
	}

	// Process dynamic sources (like moongates)
	for _, source := range dynamicSources {
		if !source.IsActive {
			continue
		}

		distance := asm.calculateDistance(playerPos, source.Position)

		// Start with base definition and apply explicit overrides from source
		baseDef := asm.soundTypeDefinitions[source.SoundType]

		// Keep volume modifier separate - NEVER modify BaseVolume!
		// Note: VolumeModifier can legitimately be 0 for fade-in effects
		// Only default to 1.0 if the field wasn't explicitly set (we check if it's a dynamic source)
		volumeModifier := source.VolumeModifier
		baseVolume := source.GetBaseVolume() // Use getter to access private field
		if !source.IsDynamic && volumeModifier == 0 && baseVolume > 0 {
			volumeModifier = 1.0 // Default modifier for static sources only
		}

		// Create definition with base volume kept unchanged
		definition := AmbientSoundDefinition{
			SoundType:   source.SoundType,
			FileName:    source.SoundFile,
			MaxDistance: source.MaxDistance,
			BaseVolume:  baseVolume, // Use the value from getter
			LoopSound:   baseDef.LoopSound,
			FadeInTime:  source.FadeInTime,
			FadeOutTime: source.FadeOutTime,
		}

		if distance <= definition.MaxDistance {
			// Debug logging for moongate sounds
			if source.SoundType == AmbientHum {
				logging.Debug(logging.CategoryAudio, "Moongate: distance=%.2f, MaxDistance=%.2f, BaseVolume=%.2f, VolumeModifier=%.2f, Final=%.2f, File=%s",
					distance, definition.MaxDistance, baseVolume, volumeModifier, baseVolume*volumeModifier, definition.FileName)
			}

			// Create unique key for dynamic sources based on position to avoid conflicts
			// Use large offset to avoid conflicts with static sound types
			dynamicKey := AmbientSoundType(fmt.Sprintf("source/%s/%s/%d/%d",
				source.Emitter, source.SoundType, source.Position.X, source.Position.Y))
			if definition.MaxDistance == 0 {
				dynamicKey = AmbientSoundType("background/" + string(source.SoundType))
			}
			// The location bed is the same lighthouse voice as the coastal tile,
			// so entering/leaving never overlaps two loops or restarts the recording.
			if source.Emitter == lighthouseEmitter {
				dynamicKey = source.SoundType
			}

			// Use the closest distance for this specific dynamic source
			if existingDistance, exists := soundTypeDistances[dynamicKey]; !exists ||
				distance < existingDistance || source.Emitter == lighthouseEmitter {
				soundTypeDistances[dynamicKey] = distance
				soundTypeDefinitions[dynamicKey] = definition
				soundTypePositions[dynamicKey] = source.Position
				soundTypeModifiers[dynamicKey] = volumeModifier // Store the actual modifier for this source
			}
		}
	}

	// Update or create sounds for each active sound type
	for soundType, distance := range soundTypeDistances {
		definition := soundTypeDefinitions[soundType]
		position := soundTypePositions[soundType]
		volumeModifier := soundTypeModifiers[soundType] // Get the volume modifier for this sound

		if instance, exists := asm.activeSounds[soundType]; exists {
			// Update existing sound with closest distance and volume modifier
			if instance.Definition.FileName != definition.FileName {
				if err := instance.Player.Close(); err != nil {
					logging.Warn(logging.CategoryAudio, "Close replaced ambient sound: %v", err)
				}
				delete(asm.activeSounds, soundType)
				asm.createAmbientSoundWithModifier(definition, position, distance, soundType, volumeModifier)

				continue
			}
			if instance.FadeState == AmbientFadeOut {
				asm.updateFade(instance)
				instance.FadeState = AmbientFadeIn
				instance.FadeFrom = instance.FadeGain
				instance.FadeStart = asm.now()
			}
			instance.VolumeModifier = volumeModifier // Update the modifier in case it changed
			instance.Definition = definition         // Source overrides change gain/range, never loop identity.
			instance.Position = position
			asm.updateAmbientSound(instance, distance)
		} else {
			// Create new ambient sound for this type
			if definition.SoundType == AmbientHum {
				logging.Debug(logging.CategoryAudio, "Creating ambient source: type=%s, baseVolume=%.2f, modifier=%.2f, file=%s",
					soundType, definition.BaseVolume, volumeModifier, definition.FileName)
			}
			asm.createAmbientSoundWithModifier(definition, position, distance, soundType, volumeModifier)
		}
	}

	// Fade out sounds that are no longer in range
	for soundType, instance := range asm.activeSounds {
		if _, stillActive := soundTypeDistances[soundType]; !stillActive {
			asm.fadeOutAmbientSound(instance)
		}
	}

	// Clean up fully stopped sounds
	asm.cleanupStoppedSounds()
}

func strongerAmbientSource(candidate AmbientSoundDefinition, distance float64,
	previous AmbientSoundDefinition, previousDistance float64, existing bool,
) bool {
	volume := candidate.BaseVolume * AmbientDistanceGain(distance, candidate.MaxDistance)
	if volume <= 0 {
		return false
	}
	priorVolume := previous.BaseVolume * AmbientDistanceGain(previousDistance, previous.MaxDistance)

	return !existing || volume > priorVolume || (volume == priorVolume && distance < previousDistance)
}

// TileWithPosition represents a tile at a specific position for ambient sound processing
type TileWithPosition struct {
	Tile     *references.Tile
	Position references.Position
}

// AmbientSoundSource represents a generic sound source for ambient sound processing
type AmbientSoundSource struct {
	Emitter   string              // Code-owned role; independent of the selected cue.
	Position  references.Position // Where the sound is located
	SoundType AmbientSoundType    // Type of ambient sound to play
	IsActive  bool                // Whether this source should produce sound
	IsDynamic bool                // true for dynamic sources, false for tiles

	// Explicit sound parameters (protected to prevent modification)
	SoundFile      string        // Sound file to play
	MaxDistance    float64       // Max distance to hear the sound
	baseVolume     float64       // Base volume level (PRIVATE - never modify!)
	VolumeModifier float64       // Volume modifier for dynamic changes (0.0 to 1.0, default 1.0)
	FadeInTime     time.Duration // Fade in duration
	FadeOutTime    time.Duration // Fade out duration
}

// GetBaseVolume returns the base volume (read-only to prevent modification)
func (a *AmbientSoundSource) GetBaseVolume() float64 {
	return a.baseVolume
}

// SetBaseVolume sets the source gain (use with caution - should only be set during initialization).
func (a *AmbientSoundSource) SetBaseVolume(volume float64) {
	const maxCalibratedSourceGain = 4.0 // +12 dB calibration over a unit cue is below 4x.
	if math.IsNaN(volume) || math.IsInf(volume, 0) {
		logging.Warn(logging.CategoryAudio, "Invalid non-finite ambient base volume")
		volume = 0
	} else if volume < 0.0 || volume > maxCalibratedSourceGain {
		logging.Warn(logging.CategoryAudio, "Attempted to set invalid base volume %.2f, clamping to [0, 4]", volume)
		volume = math.Max(0, math.Min(maxCalibratedSourceGain, volume))
	}
	a.baseVolume = volume
}

// SetVolumeModifier sets the volume modifier for dynamic changes
func (a *AmbientSoundSource) SetVolumeModifier(modifier float64) {
	if math.IsNaN(modifier) || math.IsInf(modifier, 0) {
		logging.Warn(logging.CategoryAudio, "Invalid non-finite ambient modifier")
		modifier = 0
	} else if modifier < 0.0 || modifier > 1.0 {
		logging.Warn(logging.CategoryAudio, "Attempted to set invalid volume modifier %.2f, clamping to [0.0, 1.0]", modifier)
		modifier = math.Max(0.0, math.Min(1.0, modifier))
	}
	a.VolumeModifier = modifier
}

// createAmbientSound creates and starts a new ambient sound instance (backward compatibility wrapper)
//
//nolint:unused // Internal helper for ambient sound creation, may be used for future enhancements
func (asm *AmbientSoundManager) createAmbientSound(definition AmbientSoundDefinition, position references.Position, distance float64, soundType AmbientSoundType) {
	asm.createAmbientSoundWithModifier(definition, position, distance, soundType, 1.0) // Default modifier of 1.0
}

// createAmbientSoundWithModifier creates and starts a new ambient sound instance with a volume modifier
func (asm *AmbientSoundManager) createAmbientSoundWithModifier(definition AmbientSoundDefinition, position references.Position, distance float64, soundType AmbientSoundType, volumeModifier float64) {
	if definition.FileName == "" {
		return // Known cue with an intentionally empty recording slot.
	}

	// Load the actual audio file from embedded assets
	audioStream, err := asm.loadAmbientAudioFile(definition.FileName)
	if err != nil {
		// Log the error but don't crash the game
		logging.Warn(logging.CategoryAudio, "Failed to load ambient sound %s: %v", definition.FileName, err)

		return
	}

	cue := productionLibrary().Cues[string(definition.SoundType)]
	window := productionLibrary().Recordings[cue.RecordingID].PlaybackWindow
	audioStream, err = boundedRecording(audioStream, asm.audioContext.SampleRate(), window)
	if err != nil {
		logging.Warn(logging.CategoryAudio, "Invalid ambient selection %s: %v", definition.FileName, err)

		return
	}
	if definition.LoopSound {
		length, lengthErr := decodedStreamLength(audioStream)
		if lengthErr != nil {
			logging.Warn(logging.CategoryAudio, "Invalid ambient loop %s: %v", definition.FileName, lengthErr)

			return
		}
		audioStream = audio.NewInfiniteLoop(audioStream, length)
	}
	// Create audio player with the loaded stream
	player, err := newGainPlayer(asm.audioContext, audioStream)
	if err != nil {
		logging.Warn(logging.CategoryAudio, "Failed to create ambient sound player for %s: %v", definition.FileName, err)

		return
	}

	// Calculate initial volume based on distance and apply volume modifier
	baseVolume := asm.calculateVolumeFromDistance(distance, definition.MaxDistance, definition.BaseVolume)
	volume := baseVolume * volumeModifier // Apply the modifier to get final volume

	instance := &AmbientSoundInstance{
		Definition:     definition,
		Position:       position,
		Player:         player,
		Volume:         volume,
		VolumeModifier: volumeModifier, // Store the modifier separately
		FadeState:      AmbientFadeIn,
		FadeStart:      asm.now(),
		FadeGain:       0,
		LastDistance:   distance,
	}

	if !definition.LoopSound || definition.FadeInTime <= 0 {
		instance.FadeState = AmbientFadeNone
		instance.FadeGain = 1
	}
	player.SetVolume(asm.effectiveVolume(volume * instance.FadeGain))
	player.Play()

	asm.activeSounds[soundType] = instance
}

// updateAmbientSound updates an existing ambient sound based on distance
func (asm *AmbientSoundManager) updateAmbientSound(instance *AmbientSoundInstance, distance float64) {
	// Calculate base volume from distance, then apply the instance's volume modifier
	baseVolume := asm.calculateVolumeFromDistance(distance, instance.Definition.MaxDistance, instance.Definition.BaseVolume)
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

// cleanupStoppedSounds removes fully stopped ambient sound instances
func (asm *AmbientSoundManager) cleanupStoppedSounds() {
	for soundType, instance := range asm.activeSounds {
		if instance.FadeState == AmbientFadeOut {
			asm.updateFade(instance)
			if instance.FadeGain == 0 {
				instance.Player.Pause()
				if err := instance.Player.Close(); err != nil {
					logging.Warn(logging.CategoryAudio, "Close faded ambient sound: %v", err)
				}
				delete(asm.activeSounds, soundType)
			} else {
				instance.Player.SetVolume(asm.effectiveVolume(instance.Volume * instance.FadeGain))
			}
		}
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

// calculateDistance calculates the Euclidean distance between two positions
func (asm *AmbientSoundManager) calculateDistance(pos1, pos2 references.Position) float64 {
	dx := float64(pos1.X - pos2.X)
	dy := float64(pos1.Y - pos2.Y)

	return math.Sqrt(dx*dx + dy*dy)
}

// calculateVolumeFromDistance calculates volume based on distance using inverse square law
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
	asm.enabled = false

	return errors.Join(closeErrors...)
}

// loadAmbientAudioFile loads an audio file from embedded assets and returns a readable stream
func (asm *AmbientSoundManager) loadAmbientAudioFile(filename string) (io.ReadSeeker, error) {
	fileData, err := soundFiles.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("read exact ambient file %s: %w", filename, err)
	}

	// Use multi-format decoder (supports WAV, MP3, OGG)
	stream, err := DecodeAudioStream(fileData, filename, asm.audioContext.SampleRate())
	if err != nil {
		return nil, fmt.Errorf("failed to decode ambient audio file %s: %w", filename, err)
	}

	return stream, nil
}

// AddCustomAmbientSound allows adding custom ambient sound definitions for specific sprites
func (asm *AmbientSoundManager) AddCustomAmbientSound(spriteIndex indexes.SpriteIndex, definition AmbientSoundDefinition) {
	asm.mu.Lock()
	defer asm.mu.Unlock()

	asm.soundDefinitions[spriteIndex] = definition
}
