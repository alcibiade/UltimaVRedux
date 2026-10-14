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
	library           *Library
	backend           PlaybackBackend
	redux             bool
	cueAssets         map[string][]*SoundAsset
	nextCue           map[string]int
	assetErrors       map[string]error
	cueEffects        map[string]game_state.SoundEffect
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
	CueID       string
	RequestGain float64
	owner       *VoiceOwner
	Effect      game_state.SoundEffect
	RecordingID string
	Mode        game_state.SoundPlaybackMode
	Player      *gainPlayer
	Volume      float64
	BaseGain    float64 // Request and authored cue gain, before category and fade.
	FadeGain    float64 // Presentation envelope independent of category preference.
	StartTime   int64   // Elapsed presentation milliseconds, never civil/gameplay time.
	Duration    int64   // Finite decoded duration for diagnostics (not a stop timer).
	Sequence    uint64

	// Fade settings
	FadeInDuration  int64   // Fade-in duration in milliseconds (0 = no fade-in)
	FadeOutStart    int64   // Elapsed start; zero is a valid presentation instant.
	FadeOutDuration int64   // Fade-out duration in milliseconds (0 = no fade-out).
	FadeOutVolume   float64 // Volume captured when fade-out begins
	FadeOutGain     float64 // Fade envelope captured when fade-out begins.
	TargetVolume    float64 // Target volume after fade-in
}

// SoundAsset represents a loaded sound effect asset
type SoundAsset struct {
	Effect        game_state.SoundEffect
	RecordingID   string
	StreamFactory func(mode game_state.SoundPlaybackMode) (io.ReadSeeker, error)
	Duration      int64 // Duration in milliseconds
	SampleRate    int   // Sample rate in Hz
}

// NewSoundEffectPlayerWithContext creates a new sound effect player with specific audio context
func NewSoundEffectPlayerWithContext(
	config *AudioConfiguration, audioContext *audio.Context,
) (*SoundEffectPlayer, error) {
	if config == nil {
		return nil, fmt.Errorf("%w: configuration cannot be nil", ErrAudioLibrary)
	}

	if audioContext == nil {
		return nil, fmt.Errorf("%w: audio context cannot be nil", ErrAudioLibrary)
	}

	return newProjectSoundPlayer(config, defaultRuntimeLibrary(),
		EbitengineBackend{Context: audioContext}, audioContext, true), nil
}

func newProjectSoundPlayer(config *AudioConfiguration, library *Library, backend PlaybackBackend,
	context *audio.Context, redux bool) *SoundEffectPlayer {
	return &SoundEffectPlayer{
		library: library, backend: backend, redux: redux, cueAssets: make(map[string][]*SoundAsset),
		nextCue: make(map[string]int), assetErrors: make(map[string]error),
		cueEffects:   make(map[string]game_state.SoundEffect),
		config:       config,
		audioContext: context,
		activeSounds: make(map[string]*ActiveSound),
		mutex:        sync.RWMutex{}, nextSoundSequence: 0,
		nowMillis:    newPresentationClock(time.Now).Milliseconds,
		soundAssets:  make(map[game_state.SoundEffect][]*SoundAsset),
		nextVariant:  make(map[game_state.SoundEffect]int),
		assetsLoaded: false,
	}
}

// LoadAssets loads all sound effect assets from embedded files
func (sep *SoundEffectPlayer) LoadAssets() error {
	sep.mutex.Lock()
	defer sep.mutex.Unlock()

	if sep.assetsLoaded {
		return nil // Already loaded
	}

	// Use data-driven sound file mapping from metadata
	if sep.redux {
		for effect, name := range catalogEffectNames() {
			cue := sep.library.document.Cues["sfx/"+name]
			metadata := SoundEffectMetadata{Name: cue.Name, RecordingID: cue.PrimaryRecording(),
				VolumeModifier: cue.VolumeModifier}
			if err := sep.loadConfiguredSoundAsset(effect, metadata); err != nil {
				return err
			}
		}
	}

	sep.assetsLoaded = true

	return nil
}

func (sep *SoundEffectPlayer) loadConfiguredSoundAsset(
	effect game_state.SoundEffect, metadata SoundEffectMetadata,
) error {
	cueID := "sfx/" + catalogEffectNames()[effect]
	assets, err := sep.prepareCueAssets(cueID, effect)
	if err != nil {
		return fmt.Errorf("load sound effect %s: %w", metadata.Name, err)
	}
	if len(assets) == 0 {
		return nil
	}
	sep.soundAssets[effect], sep.cueAssets[cueID] = assets, assets
	sep.cueEffects[cueID] = effect

	return nil
}

func (sep *SoundEffectPlayer) prepareCueAssets(id string, effect game_state.SoundEffect) ([]*SoundAsset, error) {
	cue := sep.library.document.Cues[id]
	assets := make([]*SoundAsset, 0, len(cue.Recordings()))
	for _, recordingID := range cue.Recordings() {
		filePath := sep.library.recordingPath(recordingID)
		if filePath == "" {
			continue
		} // Only single-clip cues may be empty slots.
		window := sep.library.document.Recordings[recordingID].PlaybackWindow
		asset, err := sep.prepareSoundAsset(effect, recordingID, filePath, window)
		if err != nil {
			return nil, err
		}
		assets = append(assets, asset)
	}

	return assets, nil
}

func (sep *SoundEffectPlayer) prepareSoundAsset(
	effect game_state.SoundEffect, recordingID, filePath string, window *PlaybackWindow,
) (*SoundAsset, error) {
	fileData, err := sep.library.readFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("read sound %s: %w", filePath, err)
	}
	stream, err := DecodeAudioStream(fileData, filePath, sep.backend.SampleRate())
	if err != nil {
		return nil, fmt.Errorf("decode sound %s: %w", filePath, err)
	}

	if window != nil {
		asset, err := excerptSoundAsset(effect, stream, sep.backend.SampleRate(), *window)
		if err != nil {
			return nil, fmt.Errorf("prepare %s: %w", recordingID, err)
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
	originalData := fileData // Library owns immutable encoded bytes shared by independent decoders.

	asset := &SoundAsset{
		Effect:      effect,
		RecordingID: recordingID,
		StreamFactory: func(mode game_state.SoundPlaybackMode) (io.ReadSeeker, error) {
			freshStream, err := DecodeAudioStream(originalData, filePath, sep.backend.SampleRate())
			if err != nil {
				return nil, fmt.Errorf("recreate sound %s: %w", filePath, err)
			}
			if mode == game_state.SoundPlaybackLoop {
				return audio.NewInfiniteLoop(freshStream, streamLength), nil
			}

			return freshStream, nil
		},
		Duration:   streamLength * millisecondsPerSecond / (int64(sep.backend.SampleRate()) * decodedPCMBytesPerFrame),
		SampleRate: sep.backend.SampleRate(),
	}

	return asset, nil
}

// Play starts one explicitly configured sound instance.
func (sep *SoundEffectPlayer) Play(request game_state.SoundPlaybackRequest) error {
	_, err := sep.play(request, nil)

	return err
}

func (sep *SoundEffectPlayer) play(request game_state.SoundPlaybackRequest, owner *VoiceOwner) (uint64, error) {
	return sep.playCue(request, "sfx/"+catalogEffectNames()[request.Effect], owner,
		request.Effect == game_state.SoundWalkMetal1 || request.Effect == game_state.SoundFireBurn, true)
}

func (sep *SoundEffectPlayer) playCue(request game_state.SoundPlaybackRequest, cueID string,
	owner *VoiceOwner, replace, legacy bool) (uint64, error) {
	sep.mutex.Lock()
	defer sep.mutex.Unlock()
	asset, skip, err := sep.assetForCue(request, cueID, legacy)
	if err != nil {
		return 0, err
	}
	if skip {
		return 0, nil
	}

	// Calculate effective volume with sound-specific volume modifier
	baseVolume := request.Volume
	if cue, exists := sep.library.document.Cues[cueID]; exists {
		baseVolume *= cue.VolumeModifier * sep.library.document.recordingCalibration(asset.RecordingID)
	}
	effectiveVolume := sep.config.GetEffectiveVolume(baseVolume)
	if request.Mode == game_state.SoundPlaybackOneShot && effectiveVolume == 0 {
		return 0, nil // An inaudible finite request cannot evict an audible voice.
	}

	// Create a fresh stream for this player to avoid reuse issues
	freshStream, err := asset.StreamFactory(request.Mode)
	if err != nil {
		return 0, fmt.Errorf("prepare cue %s: %w", cueID, err)
	}
	if freshStream == nil {
		return 0, fmt.Errorf("%w: %s", errSoundStreamUnavailable, cueID)
	}

	// Create audio player
	player, err := newGainPlayerWithBackend(sep.backend, freshStream)
	if err != nil {
		return 0, fmt.Errorf("failed to create audio player: %w", err)
	}
	// Preparation is transactional: failed decoding/player creation cannot evict
	// a sounding voice or advance variant/instance sequence selection.
	admitted, err := sep.admitPreparedLocked(request, cueID, owner, player, replace)
	if err != nil || !admitted {
		return 0, err
	}

	initialVolume := effectiveVolume
	if request.FadeInMs > 0 {
		initialVolume = 0
	}
	player.SetVolume(initialVolume)
	sep.trackActiveSound(request, player, baseVolume, initialVolume, effectiveVolume, asset, owner, cueID)
	player.Play()
	if legacy {
		sep.nextVariant[request.Effect] = (sep.nextVariant[request.Effect] + 1) % len(sep.soundAssets[request.Effect])
		sep.nextCue[cueID] = sep.nextVariant[request.Effect]
	} else {
		sep.nextCue[cueID] = (sep.nextCue[cueID] + 1) % len(sep.cueAssets[cueID])
		if effect, exists := sep.cueEffects[cueID]; exists {
			sep.nextVariant[effect] = sep.nextCue[cueID]
		}
	}

	return sep.nextSoundSequence, nil
}

func (sep *SoundEffectPlayer) admitPreparedLocked(
	request game_state.SoundPlaybackRequest, cueID string, owner *VoiceOwner, player *gainPlayer, replace bool,
) (bool, error) {
	if replace {
		for id, sound := range sep.activeSounds {
			if sound.CueID == cueID && sound.owner == owner {
				sep.closeSoundLocked(id)
			}
		}
	}
	sep.retireCompletedLocked()
	const voiceLimit = 8
	if len(sep.activeSounds) >= voiceLimit && !sep.removeOldestOneShot() {
		if err := player.Close(); err != nil {
			return false, fmt.Errorf("retire unadmitted sound: %w", err)
		}
		if request.Mode == game_state.SoundPlaybackLoop {
			return false, errSoundVoiceLimit
		}
		logging.Debug(logging.CategoryAudio, "Dropping %s: all SFX voices are owned loops", cueID)

		return false, nil
	}

	return true, nil
}

func (sep *SoundEffectPlayer) assetForRequest(
	request game_state.SoundPlaybackRequest,
) (*SoundAsset, bool, error) {
	if err := sep.validateRequest(request); err != nil {
		return nil, false, err
	}
	assets := sep.soundAssets[request.Effect]
	if len(assets) > 0 {
		return assets[sep.nextVariant[request.Effect]%len(assets)], false, nil
	}
	cue, exists := sep.library.document.Cues["sfx/"+catalogEffectNames()[request.Effect]]
	if exists && sep.library.document.Recordings[cue.PrimaryRecording()].File == nil {
		return nil, true, nil
	}

	return nil, false, fmt.Errorf("%w: %d", errSoundEffectNotFound, request.Effect)
}

func (sep *SoundEffectPlayer) assetForCue(request game_state.SoundPlaybackRequest, cueID string,
	legacy bool) (*SoundAsset, bool, error) {
	if legacy {
		return sep.assetForRequest(request)
	}
	if err := sep.validateRequest(request); err != nil {
		return nil, false, err
	}
	if _, exists := sep.library.document.Cues[cueID]; !exists {
		return nil, false, fmt.Errorf("%w: unknown cue %s", ErrAudioLibrary, cueID)
	}
	if err := sep.assetErrors[cueID]; err != nil {
		return nil, false, err
	}
	assets, prepared := sep.cueAssets[cueID]
	if !prepared {
		var err error
		assets, err = sep.prepareCueAssets(cueID, game_state.SoundEffect(-1))
		if err != nil {
			sep.assetErrors[cueID] = err

			return nil, false, err
		}
		sep.cueAssets[cueID] = assets
	}
	if len(assets) == 0 {
		return nil, true, nil
	}

	return assets[sep.nextCue[cueID]%len(assets)], false, nil
}

func (sep *SoundEffectPlayer) validateRequest(request game_state.SoundPlaybackRequest) error {
	if request.Mode != game_state.SoundPlaybackOneShot && request.Mode != game_state.SoundPlaybackLoop {
		return fmt.Errorf("%w: %d", errUnsupportedSoundPlaybackMode, request.Mode)
	}
	if err := validateVolumeRange(request.Volume); err != nil {
		return fmt.Errorf("invalid sound request volume: %w", err)
	}
	if request.FadeInMs < 0 {
		return fmt.Errorf("%w: %d", errInvalidSoundFade, request.FadeInMs)
	}
	if !sep.assetsLoaded {
		return errSoundAssetsNotLoaded
	}

	return nil
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
	owner *VoiceOwner,
	cueID string,
) {
	soundID, startTime, sequence := sep.nextSoundID(request.Effect)
	sep.activeSounds[soundID] = &ActiveSound{
		CueID:           cueID,
		RequestGain:     request.Volume,
		owner:           owner,
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
		FadeOutGain:     0,
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
				logging.Warn(logging.CategoryAudio, "Stopping sound %s: %v", sound.CueID, err)
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
					logging.Warn(logging.CategoryAudio, "Stopping sound %s: %v", sound.CueID, err)
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
			logging.Debug(logging.CategoryAudio, "Starting fade-out for %s: %.3f -> 0.0 over %dms",
				sound.CueID, sound.Volume, fadeOutDurationMs)
		}
	}
}

// UpdateFading services envelopes, decoder failures, and completed voices every frame.
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
			logging.Warn(logging.CategoryAudio, "Close sound %s: %v", sound.CueID, err)
		}
	}
	delete(sep.activeSounds, soundID)
}

func (sep *SoundEffectPlayer) retireCompletedLocked() {
	for id, sound := range sep.activeSounds {
		if err := sound.Player.Failure(); err != nil {
			logging.Warn(logging.CategoryAudio, "Retire failed sound %s: %v", sound.CueID, err)
			sep.closeSoundLocked(id)
		} else if !sound.Player.IsPlaying() {
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
				closeErrors = append(closeErrors, fmt.Errorf("closing sound %s: %w", sound.CueID, err))
			}
		}
	}

	sep.activeSounds = make(map[string]*ActiveSound)
	sep.soundAssets = make(map[game_state.SoundEffect][]*SoundAsset)
	clear(sep.nextVariant)
	clear(sep.nextCue)
	clear(sep.cueAssets)
	clear(sep.assetErrors)
	clear(sep.cueEffects)
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

// GetSoundEffectName returns human-readable name for a sound effect
func GetSoundEffectName(effect game_state.SoundEffect) string {
	if metadata, exists := GetSoundEffectMetadata(effect); exists {
		return metadata.Name
	}

	return "Unknown Sound Effect"
}
