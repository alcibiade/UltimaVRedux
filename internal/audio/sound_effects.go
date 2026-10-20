package audio

import (
	"fmt"
	"time"

	"github.com/bradhannah/UltimaVRedux/internal/game_state"
	berad "github.com/bradhannah/berad-sdk/modules/berad-audio"
)

const bardEmitter = "bard"

// PlaySoundEffect requests one full-gain finite game cue.
func (manager *GameAudioManager) PlaySoundEffect(effect game_state.SoundEffect) error {
	return manager.PlaySound(game_state.SoundPlaybackRequest{
		Effect: effect, Mode: game_state.SoundPlaybackOneShot, Volume: 1, FadeInMs: 0})
}

// PlaySound preserves the legacy effect-wide game lifetime and replacement rules.
func (manager *GameAudioManager) PlaySound(request game_state.SoundPlaybackRequest) error {
	_, err := manager.play(request, false)

	return err
}

// PlayVoice creates a separately owned voice for workflow cleanup.
func (manager *GameAudioManager) PlayVoice(request game_state.SoundPlaybackRequest) (GameVoice, error) {
	return manager.play(request, true)
}

func (manager *GameAudioManager) play(request game_state.SoundPlaybackRequest, owned bool) (GameVoice, error) {
	manager.mutex.RLock()
	defer manager.mutex.RUnlock()
	if manager.closed {
		return GameVoice{}, ErrAudioClosed
	}
	if !manager.enabled {
		return GameVoice{manager: nil, voice: berad.Voice{}}, nil
	}
	cue, options, err := gamePlaybackOptions(request)
	if err != nil {
		return GameVoice{}, err
	}
	var result berad.PlayResult
	if owned {
		result, err = manager.sfx.Play(cue, options)
	} else {
		if request.Effect == game_state.SoundFireBurn || request.Effect == game_state.SoundWalkMetal1 {
			options.Slot = string(cue)
		}
		result, err = manager.legacyOwner.Play(cue, options)
	}
	if err != nil {
		return GameVoice{}, fmt.Errorf("play game cue %s: %w", cue, err)
	}
	if result.Outcome != berad.Admitted {
		return GameVoice{manager: nil, voice: berad.Voice{}}, nil
	}
	if isInstrumentEffect(request.Effect) && result.Voice.Status().TargetGain > 0 {
		manager.policy.LocalMusic()
	}

	return GameVoice{manager: manager, voice: result.Voice}, nil
}

func gamePlaybackOptions(request game_state.SoundPlaybackRequest) (berad.CueID, berad.PlayOptions, error) {
	symbol, known := catalogEffectNames()[request.Effect]
	if !known || request.FadeInMs < 0 ||
		(request.Mode != game_state.SoundPlaybackOneShot && request.Mode != game_state.SoundPlaybackLoop) {
		return "", berad.PlayOptions{}, fmt.Errorf("%w: invalid game sound request", berad.ErrInvalidRequest)
	}
	options := berad.DefaultPlayOptions()
	options.Gain, options.FadeIn = request.Volume, gameFade(request.FadeInMs)
	options.Loop = request.Mode == game_state.SoundPlaybackLoop
	if request.Effect == game_state.SoundFireBurn && !options.Loop {
		options.ReleaseTail = fireBurnReleaseMs * time.Millisecond
	}

	return berad.CueID("sfx/" + symbol), options, nil
}

// StopSound deliberately reaches every instance of the effect across owners.
func (manager *GameAudioManager) StopSound(request game_state.SoundStopRequest) {
	manager.mutex.RLock()
	defer manager.mutex.RUnlock()
	if manager.closed {
		return
	}
	if symbol, known := catalogEffectNames()[request.Effect]; known {
		manager.report(manager.sfx.StopCue(berad.CueID("sfx/"+symbol), gameFade(request.FadeOutMs)))
	}
}

func (manager *GameAudioManager) hasLocalMusic() bool {
	local := false
	for _, voice := range manager.sfx.Snapshot() {
		if effect, known := manager.effects[string(voice.Cue)]; known && isInstrumentEffect(effect) && voice.Gain > 0 {
			local = true
		}
	}
	snapshot := manager.ambience.Snapshot()
	retained := make(map[berad.SourceID]bool, len(snapshot.Sources))
	for _, source := range snapshot.Sources {
		retained[source.ID] = true
		if manager.roles[source.ID] == bardEmitter && source.Voice.TargetGain > 0 &&
			(source.Voice.State == berad.VoicePlaying || source.Voice.State == berad.VoiceReleasing) {
			local = true
		}
	}
	for id := range manager.roles {
		if !retained[id] {
			delete(manager.roles, id)
			delete(manager.sourceMix, id)
		}
	}

	return local
}
