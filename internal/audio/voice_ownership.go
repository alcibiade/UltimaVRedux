package audio

import (
	"errors"

	"github.com/bradhannah/UltimaVRedux/internal/game_state"
)

// VoiceHandle identifies one presentation instance in its originating manager.
// Zero, completed, evicted and closed-manager handles safely stop nothing. Handles
// never serialize into gameplay/save state and never decide authoritative timing.
type VoiceHandle struct {
	manager  *AudioManager
	sequence uint64
}

// Stop releases this instance only. Repeated fades cannot extend its lifetime;
// zero/negative fade durations stop immediately, including an existing release.
func (handle VoiceHandle) Stop(fadeOutMs int64) {
	if handle.manager != nil {
		handle.manager.StopVoice(handle, fadeOutMs)
	}
}

// VoiceOwner groups a bounded set of live voices without retaining completed
// handles. Its manager serializes creation, release and terminal closure.
type VoiceOwner struct {
	manager *AudioManager
	closed  bool
}

// ErrVoiceOwnerClosed rejects new playback through an expired lifetime.
var ErrVoiceOwnerClosed = errors.New("audio voice owner is closed")

// NewVoiceOwner creates an independent presentation lifetime. Close or Stop it
// from the workflow/scene finalizer even if the workflow is cancelled before play.
func (am *AudioManager) NewVoiceOwner() *VoiceOwner {
	am.mutex.RLock()
	defer am.mutex.RUnlock()

	return &VoiceOwner{manager: am, closed: am.closed}
}

// Play starts an instance in this lifetime; ordinary finite admission policy applies.
func (owner *VoiceOwner) Play(request game_state.SoundPlaybackRequest) (VoiceHandle, error) {
	if owner == nil || owner.manager == nil {
		return VoiceHandle{}, ErrVoiceOwnerClosed
	}

	return owner.manager.playVoice(request, owner)
}

// PlayCue starts a neutral cue in this lifetime, using its originating library.
func (owner *VoiceOwner) PlayCue(request CueRequest) (VoiceHandle, error) {
	if owner == nil || owner.manager == nil {
		return VoiceHandle{}, ErrVoiceOwnerClosed
	}

	return owner.manager.playCueVoice(request, owner)
}

// Stop closes the lifetime to new playback and releases only its current voices.
// An immediate stop can still finish a previously requested fade.
func (owner *VoiceOwner) Stop(fadeOutMs int64) {
	if owner == nil || owner.manager == nil {
		return
	}
	manager := owner.manager
	manager.mutex.Lock()
	defer manager.mutex.Unlock()
	if manager.closed || (owner.closed && fadeOutMs > 0) {
		return
	}
	owner.closed = true
	manager.soundEffectPlayer.stopOwned(func(sound *ActiveSound) bool { return sound.owner == owner }, fadeOutMs)
}

// Close immediately releases this lifetime, including voices already fading out.
func (owner *VoiceOwner) Close() error {
	owner.Stop(0)

	return nil
}

// PlayVoice returns a manager-scoped instance handle. Silence, mute and dropped
// finite requests return the safe zero handle without allocating an instance ID.
func (am *AudioManager) PlayVoice(request game_state.SoundPlaybackRequest) (VoiceHandle, error) {
	return am.playVoice(request, am.NewVoiceOwner())
}

func (am *AudioManager) playVoice(request game_state.SoundPlaybackRequest, owner *VoiceOwner) (VoiceHandle, error) {
	return am.playSelectedVoice(request, owner, nil)
}

func (am *AudioManager) playCueVoice(request CueRequest, owner *VoiceOwner) (VoiceHandle, error) {
	mode := game_state.SoundPlaybackOneShot
	if request.Loop {
		mode = game_state.SoundPlaybackLoop
	}
	legacy := game_state.SoundPlaybackRequest{Effect: game_state.SoundEffect(-1), Mode: mode,
		Volume: request.Volume, FadeInMs: request.FadeInMs}

	return am.playSelectedVoice(legacy, owner, &request)
}

func (am *AudioManager) playSelectedVoice(request game_state.SoundPlaybackRequest,
	owner *VoiceOwner, cue *CueRequest) (VoiceHandle, error) {
	am.mutex.RLock()
	closed := am.closed
	expired := owner != nil && owner.closed
	am.mutex.RUnlock()
	if closed {
		return VoiceHandle{}, ErrAudioClosed
	}
	if expired {
		return VoiceHandle{}, ErrVoiceOwnerClosed
	}
	if err := am.ensureAssetsLoaded(); err != nil {
		return VoiceHandle{}, err
	}
	am.mutex.RLock()
	defer am.mutex.RUnlock()
	if am.closed {
		return VoiceHandle{}, ErrAudioClosed
	}
	if owner != nil && owner.closed {
		return VoiceHandle{}, ErrVoiceOwnerClosed
	}
	if !am.enabled {
		return VoiceHandle{manager: nil, sequence: 0}, nil
	}
	var sequence uint64
	var err error
	if cue == nil {
		sequence, err = am.soundEffectPlayer.play(request, owner)
	} else {
		sequence, err = am.soundEffectPlayer.playCue(request, cue.CueID, owner, cue.ReplacePrevious, false)
	}
	if err != nil || sequence == 0 {
		return VoiceHandle{}, err
	}
	if cue == nil && isInstrumentEffect(request.Effect) && am.soundEffectPlayer.hasInstrumentMusic() {
		am.musicManager.LocalMusic()
	}

	return VoiceHandle{manager: am, sequence: sequence}, nil
}

// StopVoice rejects foreign-manager handles and reports whether the instance was
// still owned. Legacy StopSound remains explicitly effect-wide across owners.
func (am *AudioManager) StopVoice(handle VoiceHandle, fadeOutMs int64) bool {
	am.mutex.RLock()
	defer am.mutex.RUnlock()
	if am.closed || handle.manager != am || handle.sequence == 0 {
		return false
	}

	return am.soundEffectPlayer.stopOwned(func(sound *ActiveSound) bool {
		return sound.Sequence == handle.sequence
	}, fadeOutMs)
}

func (sep *SoundEffectPlayer) stopOwned(matches func(*ActiveSound) bool, fadeOutMs int64) bool {
	sep.mutex.Lock()
	defer sep.mutex.Unlock()
	now, found := sep.nowMillis(), false
	for soundID, sound := range sep.activeSounds {
		if !matches(sound) {
			continue
		}
		found = true
		if fadeOutMs <= 0 {
			sep.closeSoundLocked(soundID)

			continue
		}
		if sound.FadeOutDuration > 0 && sound.FadeOutStart <= now &&
			sound.FadeOutDuration-(now-sound.FadeOutStart) <= fadeOutMs {
			continue
		}
		sep.updateEnvelopeLocked(sound, now)
		sound.FadeOutStart, sound.FadeOutDuration = now, fadeOutMs
		sound.FadeOutGain, sound.FadeOutVolume = sound.FadeGain, sound.Volume
		sound.FadeInDuration = 0
	}

	return found
}
