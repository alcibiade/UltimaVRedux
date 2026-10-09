package audio

import (
	"embed"
	"errors"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/hajimehoshi/ebiten/v2/audio"

	"github.com/bradhannah/UltimaVRedux/internal/game_state"
	"github.com/bradhannah/UltimaVRedux/internal/logging"
)

// Burning is brief damage feedback. The manifest selects its one-second
// excerpt; release the last 200ms rather than abruptly cutting the crackle.
const fireBurnReleaseMs = 200

var (
	errUnsupportedSoundPlaybackMode = errors.New("unsupported sound playback mode")
	errSoundAssetsNotLoaded         = errors.New("sound assets not loaded")
	errSoundEffectNotFound          = errors.New("sound effect not found")
	errSoundStreamUnavailable       = errors.New("sound stream unavailable")
	errSoundVoiceLimit              = errors.New("all sound effect voices are owned loops")
	errInvalidSoundFade             = errors.New("invalid sound fade-in duration")
)

// Embed sound effect files (supports WAV, MP3, OGG)
//
//go:embed assets/*
var soundFiles embed.FS

// SoundEffectPlayer handles individual sound effect playbook with volume and positional audio
type SoundEffectPlayer struct {
	config            *AudioConfiguration
	audioContext      *audio.Context
	activeSounds      map[string]*ActiveSound
	mutex             sync.RWMutex
	nextSoundSequence uint64
	nowMillis         func() int64
	assetsLoaded      bool
	soundAssets       map[game_state.SoundEffect][]*SoundAsset
	nextVariant       map[game_state.SoundEffect]int
}

// ActiveSound represents a currently playing sound effect
type ActiveSound struct {
	Effect      game_state.SoundEffect
	RecordingID string
	Mode        game_state.SoundPlaybackMode
	Player      *gainPlayer
	Volume      float64
	BaseGain    float64 // Request and authored cue gain, before category and fade.
	FadeGain    float64 // Presentation envelope independent of category preference.
	StartTime   int64   // Unix timestamp in milliseconds
	Duration    int64   // Finite decoded duration for diagnostics (not a stop timer).
	Sequence    uint64

	// Fade settings
	FadeInDuration  int64   // Fade-in duration in milliseconds (0 = no fade-in)
	FadeOutStart    int64   // When to start fade-out (0 = no fade-out)
	FadeOutDuration int64   // Fade-out duration in milliseconds
	FadeOutVolume   float64 // Volume captured when fade-out begins
	FadeOutGain     float64 // Fade envelope captured when fade-out begins.
	TargetVolume    float64 // Target volume after fade-in
}

// SoundAsset represents a loaded sound effect asset
type SoundAsset struct {
	Effect        game_state.SoundEffect
	RecordingID   string
	StreamFactory func(mode game_state.SoundPlaybackMode) io.ReadSeeker
	Duration      int64 // Duration in milliseconds
	SampleRate    int   // Sample rate in Hz
}

// NewSoundEffectPlayerWithContext creates a new sound effect player with specific audio context
func NewSoundEffectPlayerWithContext(config *AudioConfiguration, audioContext *audio.Context) (*SoundEffectPlayer, error) {
	if config == nil {
		return nil, fmt.Errorf("configuration cannot be nil")
	}

	if audioContext == nil {
		return nil, fmt.Errorf("audio context cannot be nil")
	}

	return &SoundEffectPlayer{
		config:       config,
		audioContext: audioContext,
		activeSounds: make(map[string]*ActiveSound),
		nowMillis:    getCurrentTimeMillis,
		soundAssets:  make(map[game_state.SoundEffect][]*SoundAsset),
		nextVariant:  make(map[game_state.SoundEffect]int),
		assetsLoaded: false,
	}, nil
}

// LoadAssets loads all sound effect assets from embedded files
func (sep *SoundEffectPlayer) LoadAssets() error {
	sep.mutex.Lock()
	defer sep.mutex.Unlock()

	if sep.assetsLoaded {
		return nil // Already loaded
	}

	// Use data-driven sound file mapping from metadata
	for effect, metadata := range soundEffectMetadataMap {
		sep.loadConfiguredSoundAsset(effect, metadata)
	}

	sep.assetsLoaded = true

	return nil
}

func (sep *SoundEffectPlayer) loadConfiguredSoundAsset(
	effect game_state.SoundEffect,
	metadata SoundEffectMetadata,
) {
	cue := productionLibrary().Cues["sfx/"+catalogEffectNames()[effect]]
	for _, recordingID := range cue.Recordings() {
		filePath := productionLibrary().recordingPath(recordingID)
		if filePath == "" {
			continue
		} // Only single-clip cues may be empty slots.
		window := productionLibrary().Recordings[recordingID].PlaybackWindow
		asset, err := sep.prepareSoundAsset(effect, recordingID, filePath, window)
		if err != nil {
			panic(fmt.Sprintf("Failed to load sound effect %s: %v", metadata.Name, err))
		}
		sep.soundAssets[effect] = append(sep.soundAssets[effect], asset)
	}
}

func (sep *SoundEffectPlayer) prepareSoundAsset(
	effect game_state.SoundEffect, recordingID, filePath string, window *PlaybackWindow,
) (*SoundAsset, error) {
	fileData, err := soundFiles.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("read sound %s: %w", filePath, err)
	}
	stream, err := DecodeAudioStream(fileData, filePath, sep.audioContext.SampleRate())
	if err != nil {
		return nil, fmt.Errorf("decode sound %s: %w", filePath, err)
	}

	if window != nil {
		asset, err := excerptSoundAsset(effect, stream, sep.audioContext.SampleRate(), *window)
		if err != nil {
			return nil, fmt.Errorf("prepare %s: %w", GetSoundEffectName(effect), err)
		}
		asset.RecordingID = recordingID

		return asset, nil
	}

	// Get the stream length for duration calculation
	streamLength, err := decodedStreamLength(stream)
	if err != nil {
		return nil, err
	}

	// Store the original file data for creating fresh streams
	originalData := make([]byte, len(fileData))
	copy(originalData, fileData)

	newStream := func() io.ReadSeeker {
		freshStream, err := DecodeAudioStream(originalData, filePath, sep.audioContext.SampleRate())
		if err != nil {
			return &silentStream{position: 0}
		}

		return freshStream
	}
	asset := &SoundAsset{
		Effect:      effect,
		RecordingID: recordingID,
		StreamFactory: func(mode game_state.SoundPlaybackMode) io.ReadSeeker {
			freshStream := newStream()
			if mode == game_state.SoundPlaybackLoop {
				return audio.NewInfiniteLoop(freshStream, streamLength)
			}

			return freshStream
		},
		Duration:   streamLength * millisecondsPerSecond / (int64(sep.audioContext.SampleRate()) * decodedPCMBytesPerFrame),
		SampleRate: sep.audioContext.SampleRate(),
	}

	return asset, nil
}

// Play starts one explicitly configured sound instance.
func (sep *SoundEffectPlayer) Play(request game_state.SoundPlaybackRequest) error {
	sep.mutex.Lock()
	defer sep.mutex.Unlock()
	asset, skip, err := sep.assetForRequest(request)
	if err != nil {
		return err
	}
	if skip {
		return nil
	}

	// Repeated footfalls and burning damage replace their own preceding voice.
	// Fire ambience and the workflow-owned campfire use different identities.
	if request.Effect == game_state.SoundWalkMetal1 || request.Effect == game_state.SoundFireBurn {
		sep.stopEffectLocked(request.Effect)
	}

	// Retire completed finite voices before applying the shared voice budget.
	sep.retireCompletedLocked()
	// Long-running workflow loops are never evicted by an ordinary one-shot.
	if len(sep.activeSounds) >= 8 {
		if !sep.removeOldestOneShot() {
			if request.Mode == game_state.SoundPlaybackLoop {
				return errSoundVoiceLimit
			}
			logging.Debug(logging.CategoryAudio, "Dropping %s: all SFX voices are owned loops",
				GetSoundEffectName(request.Effect))

			return nil
		}
	}

	// Calculate effective volume with sound-specific volume modifier
	baseVolume := request.Volume
	if metadata, exists := GetSoundEffectMetadata(request.Effect); exists {
		baseVolume *= metadata.VolumeModifier * productionLibrary().recordingCalibration(asset.RecordingID)
	}
	effectiveVolume := sep.config.GetEffectiveVolume(baseVolume)

	// Create a fresh stream for this player to avoid reuse issues
	freshStream := asset.StreamFactory(request.Mode)
	if freshStream == nil {
		return fmt.Errorf("%w: %s", errSoundStreamUnavailable, GetSoundEffectName(request.Effect))
	}

	// Create audio player
	player, err := newGainPlayer(sep.audioContext, freshStream)
	if err != nil {
		return fmt.Errorf("failed to create audio player: %w", err)
	}

	// Set initial volume for fade-in or normal playback
	initialVolume := effectiveVolume
	if request.FadeInMs > 0 {
		initialVolume = 0.0 // Start at silence for fade-in
		logging.Debug(
			logging.CategoryAudio,
			"Starting fade-in for %s: 0.0 -> %.3f over %dms",
			GetSoundEffectName(request.Effect),
			effectiveVolume,
			request.FadeInMs,
		)
	}
	player.SetVolume(initialVolume)

	// TODO: Implement stereo pan when Ebiten supports it
	// pan parameter is calculated but not yet applied

	sep.trackActiveSound(request, player, baseVolume, initialVolume, effectiveVolume, asset)

	// Start playback
	player.Play()
	sep.nextVariant[request.Effect] = (sep.nextVariant[request.Effect] + 1) % len(sep.soundAssets[request.Effect])

	return nil
}

func (sep *SoundEffectPlayer) assetForRequest(
	request game_state.SoundPlaybackRequest,
) (*SoundAsset, bool, error) {
	if request.Mode != game_state.SoundPlaybackOneShot && request.Mode != game_state.SoundPlaybackLoop {
		return nil, false, fmt.Errorf("%w: %d", errUnsupportedSoundPlaybackMode, request.Mode)
	}
	if err := validateVolumeRange(request.Volume); err != nil {
		return nil, false, fmt.Errorf("invalid sound request volume: %w", err)
	}
	if request.FadeInMs < 0 {
		return nil, false, fmt.Errorf("%w: %d", errInvalidSoundFade, request.FadeInMs)
	}
	if !sep.assetsLoaded {
		return nil, false, errSoundAssetsNotLoaded
	}
	assets := sep.soundAssets[request.Effect]
	if len(assets) > 0 {
		return assets[sep.nextVariant[request.Effect]%len(assets)], false, nil
	}
	if isPlaceholderSound(request.Effect) {
		return nil, true, nil
	}

	return nil, false, fmt.Errorf("%w: %d", errSoundEffectNotFound, request.Effect)
}

func isPlaceholderSound(effect game_state.SoundEffect) bool {
	metadata, exists := GetSoundEffectMetadata(effect)

	return exists && productionLibrary().Recordings[metadata.RecordingID].File == nil
}

// nextSoundID returns a collision-free ID while the caller holds sep.mutex.
func (sep *SoundEffectPlayer) nextSoundID(effect game_state.SoundEffect) (string, int64, uint64) {
	sep.nextSoundSequence++
	startTime := sep.nowMillis()

	return fmt.Sprintf("%d_%d_%d", effect, startTime, sep.nextSoundSequence), startTime, sep.nextSoundSequence
}

func (sep *SoundEffectPlayer) trackActiveSound(
	request game_state.SoundPlaybackRequest,
	player *gainPlayer,
	baseGain float64,
	volume float64,
	targetVolume float64,
	asset *SoundAsset,
) {
	soundID, startTime, sequence := sep.nextSoundID(request.Effect)
	sep.activeSounds[soundID] = &ActiveSound{
		Effect:          request.Effect,
		RecordingID:     asset.RecordingID,
		Mode:            request.Mode,
		Player:          player,
		Volume:          volume,
		BaseGain:        baseGain,
		FadeGain:        1,
		StartTime:       startTime,
		Duration:        asset.Duration,
		Sequence:        sequence,
		FadeInDuration:  request.FadeInMs,
		FadeOutStart:    0,
		FadeOutDuration: 0,
		FadeOutVolume:   0,
		TargetVolume:    targetVolume,
	}
	if request.FadeInMs > 0 {
		sep.activeSounds[soundID].FadeGain = 0
	}
	if request.Effect == game_state.SoundFireBurn && request.Mode == game_state.SoundPlaybackOneShot {
		sound := sep.activeSounds[soundID]
		sound.FadeOutDuration = min(fireBurnReleaseMs, asset.Duration)
		sound.FadeOutStart = startTime + asset.Duration - sound.FadeOutDuration
		sound.FadeOutGain = 1
		if request.FadeInMs > 0 {
			sound.FadeOutGain = min(1, float64(asset.Duration-sound.FadeOutDuration)/float64(request.FadeInMs))
		}
		sound.FadeOutVolume = targetVolume * sound.FadeOutGain
	}
}

// StopAll stops all currently playing sound effects
func (sep *SoundEffectPlayer) StopAll() {
	sep.mutex.Lock()
	defer sep.mutex.Unlock()

	for _, sound := range sep.activeSounds {
		if sound.Player != nil {
			if err := sound.Player.Close(); err != nil {
				logging.Warn(logging.CategoryAudio, "Stopping sound %s: %v", GetSoundEffectName(sound.Effect), err)
			}
		}
	}

	sep.activeSounds = make(map[string]*ActiveSound)
}

// StopEffect stops all currently playing instances of a specific sound effect
func (sep *SoundEffectPlayer) StopEffect(effect game_state.SoundEffect) {
	sep.mutex.Lock()
	defer sep.mutex.Unlock()
	sep.stopEffectLocked(effect)
}

// stopEffectLocked requires the caller to hold sep.mutex exclusively.
func (sep *SoundEffectPlayer) stopEffectLocked(effect game_state.SoundEffect) {
	for soundID, sound := range sep.activeSounds {
		if sound.Effect == effect {
			if sound.Player != nil {
				if err := sound.Player.Close(); err != nil {
					logging.Warn(logging.CategoryAudio, "Stopping sound %s: %v", GetSoundEffectName(sound.Effect), err)
				}
			}
			delete(sep.activeSounds, soundID)
		}
	}
}

// FadeOutEffect starts a fade-out for all currently playing instances of a specific sound effect
func (sep *SoundEffectPlayer) FadeOutEffect(effect game_state.SoundEffect, fadeOutDurationMs int64) {
	sep.mutex.Lock()
	defer sep.mutex.Unlock()

	currentTime := sep.nowMillis()
	for _, sound := range sep.activeSounds {
		if sound.Effect == effect {
			sep.updateEnvelopeLocked(sound, currentTime)
			sound.FadeOutStart = currentTime
			sound.FadeOutDuration = fadeOutDurationMs
			sound.FadeOutVolume = sound.Volume
			sound.FadeOutGain = sound.FadeGain
			sound.FadeInDuration = 0
			logging.Debug(logging.CategoryAudio, "Starting fade-out for %s: %.3f -> 0.0 over %dms", GetSoundEffectName(effect), sound.Volume, fadeOutDurationMs)
		}
	}
}

// Update method to handle fading - should be called regularly (e.g., every frame)
func (sep *SoundEffectPlayer) UpdateFading() {
	sep.mutex.Lock()
	defer sep.mutex.Unlock()

	currentTime := sep.nowMillis()
	for soundID, sound := range sep.activeSounds {
		if sound.Player == nil {
			continue
		}

		sep.updateEnvelopeLocked(sound, currentTime)
		if sound.FadeOutDuration > 0 && currentTime-sound.FadeOutStart >= sound.FadeOutDuration {
			sep.closeSoundLocked(soundID)
		}
	}
	sep.retireCompletedLocked()
}

func (sep *SoundEffectPlayer) updateEnvelopeLocked(sound *ActiveSound, now int64) {
	if sound.FadeInDuration > 0 {
		sound.FadeGain = min(1, float64(max(0, now-sound.StartTime))/float64(sound.FadeInDuration))
		if sound.FadeGain == 1 {
			sound.FadeInDuration = 0
		}
	}
	if sound.FadeOutDuration > 0 && now >= sound.FadeOutStart {
		progress := min(1, float64(max(0, now-sound.FadeOutStart))/float64(sound.FadeOutDuration))
		sound.FadeGain = sound.FadeOutGain * (1 - progress)
	}
	sound.TargetVolume = sep.config.GetEffectiveVolume(sound.BaseGain)
	volume := sep.config.GetEffectiveVolume(sound.BaseGain * sound.FadeGain)
	if volume != sound.Volume {
		sound.Player.SetVolume(volume)
		sound.Volume = volume
	}
}

// RefreshVolume applies category changes to every active voice at its current
// envelope position, without restarting or overwriting an authored request.
func (sep *SoundEffectPlayer) RefreshVolume() {
	sep.UpdateFading()
}

func (sep *SoundEffectPlayer) closeSoundLocked(soundID string) {
	sound := sep.activeSounds[soundID]
	if sound.Player != nil {
		if err := sound.Player.Close(); err != nil {
			logging.Warn(logging.CategoryAudio, "Close sound %s: %v", GetSoundEffectName(sound.Effect), err)
		}
	}
	delete(sep.activeSounds, soundID)
}

func (sep *SoundEffectPlayer) retireCompletedLocked() {
	for id, sound := range sep.activeSounds {
		if sound.Mode == game_state.SoundPlaybackOneShot && !sound.Player.IsPlaying() {
			sep.closeSoundLocked(id)
		}
	}
}

// GetActiveSoundCount returns the number of currently playing sounds
func (sep *SoundEffectPlayer) GetActiveSoundCount() int {
	sep.mutex.RLock()
	defer sep.mutex.RUnlock()

	return len(sep.activeSounds)
}

// Close shuts down the sound effect player and releases resources
func (sep *SoundEffectPlayer) Close() error {
	sep.mutex.Lock()
	defer sep.mutex.Unlock()

	var closeErrors []error
	// Stop all active sounds even when one player fails to close.
	for _, sound := range sep.activeSounds {
		if sound.Player != nil {
			if err := sound.Player.Close(); err != nil {
				closeErrors = append(closeErrors, fmt.Errorf("closing sound %s: %w", GetSoundEffectName(sound.Effect), err))
			}
		}
	}

	sep.activeSounds = make(map[string]*ActiveSound)
	sep.soundAssets = make(map[game_state.SoundEffect][]*SoundAsset)
	clear(sep.nextVariant)
	sep.assetsLoaded = false

	return errors.Join(closeErrors...)
}

// removeOldestOneShot makes room without evicting a workflow-owned loop.
func (sep *SoundEffectPlayer) removeOldestOneShot() bool {
	var oldestID string
	var oldestSequence uint64

	for id, sound := range sep.activeSounds {
		if sound.Mode == game_state.SoundPlaybackOneShot && (oldestID == "" || sound.Sequence < oldestSequence) {
			oldestSequence = sound.Sequence
			oldestID = id
		}
	}

	if oldestID != "" {
		sep.closeSoundLocked(oldestID)

		return true
	}

	return false
}

// getCurrentTimeMillis returns current time in milliseconds
// TODO: Consider using game clock for deterministic behavior
// Note: This is used for audio management (sound ID generation and cleanup),
// not gameplay logic, so non-deterministic time is less critical here
func getCurrentTimeMillis() int64 {
	return time.Now().UnixMilli()
}

// silentStream is a placeholder stream that produces silence
type silentStream struct {
	position int64
}

func (s *silentStream) Read(p []byte) (n int, err error) {
	// Fill with silence (zeros)
	for i := range p {
		p[i] = 0
	}
	s.position += int64(len(p))

	return len(p), nil
}

func (s *silentStream) Seek(offset int64, whence int) (int64, error) {
	switch whence {
	case io.SeekStart:
		s.position = offset
	case io.SeekCurrent:
		s.position += offset
	case io.SeekEnd:
		s.position = offset // For infinite silence, end is same as start
	}

	return s.position, nil
}

// GetSoundEffectName returns human-readable name for a sound effect
func GetSoundEffectName(effect game_state.SoundEffect) string {
	if metadata, exists := GetSoundEffectMetadata(effect); exists {
		return metadata.Name
	}

	return "Unknown Sound Effect"
}
