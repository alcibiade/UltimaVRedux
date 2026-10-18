package audio

import (
	"cmp"
	"slices"
)

// Snapshot returns the neutral runtime's actual voices and gain factors.
func (runtime *Runtime) Snapshot() PlaybackSnapshot { return runtime.manager.PlaybackSnapshot() }

// PlaybackSnapshot reports a coherent facade snapshot under established lock order.
func (am *AudioManager) PlaybackSnapshot() PlaybackSnapshot {
	am.mutex.Lock()
	defer am.mutex.Unlock()
	result := PlaybackSnapshot{Enabled: am.enabled, Closed: am.closed, EncodedBytes: am.library.EncodedBytes(),
		Voices: am.soundEffectPlayer.snapshotVoices(), AmbientDesired: 0, AmbientFailures: am.ambientSounds.Failures()}
	music := am.backgroundMusic.snapshotVoice()
	am.musicManager.mutex.RLock()
	policy := am.musicManager.policy
	if len(music) > 0 && policy.track == music[0].ID && !policy.suspended {
		music[0].Gain.Modifier, music[0].Gain.Envelope = policy.gain, policy.envelope
	}
	am.musicManager.mutex.RUnlock()
	result.Voices = append(result.Voices, music...)
	ambient, desired := am.ambientSounds.snapshotVoices()
	result.Voices = append(result.Voices, ambient...)
	result.AmbientDesired = desired
	if am.ambientPreviewPlayer != nil {
		cue := am.library.document.Cues["ambient/AmbientForest"]
		voice := am.library.voiceDescription("preview", "ambient-preview", "ambient/AmbientForest", cue.RecordingID,
			am.ambientPreviewPlayer, true)
		voice.Gain.Category = am.config.AmbientVolume
		voice.Gain.Effective = am.ambientPreviewPlayer.Volume()
		result.Voices = append(result.Voices, voice)
	}
	slices.SortFunc(result.Voices, func(left, right PlaybackVoice) int {
		if order := cmp.Compare(left.Lane, right.Lane); order != 0 {
			return order
		}

		return cmp.Compare(left.ID, right.ID)
	})

	return result
}

func (library *Library) voiceDescription(
	lane, voiceID, cueID, recordingID string, player *gainPlayer, loop bool,
) PlaybackVoice {
	state, failure := "paused", ""
	if player.IsPlaying() {
		state = "playing"
	}
	if err := player.Failure(); err != nil {
		state, failure = "failed", err.Error()
	}
	recording := library.document.Recordings[recordingID]
	asset := ""
	if recording.File != nil {
		asset = library.recordingPath(recordingID)
	}

	return PlaybackVoice{ID: voiceID, Lane: lane, CueID: cueID, RecordingID: recordingID,
		Asset: asset, State: state, Loop: loop,
		PositionMs: player.Position().Milliseconds(), Window: cloneAudioValue(recording.PlaybackWindow), Failure: failure,
		Gain: GainBreakdown{Recording: library.document.recordingCalibration(recordingID),
			Cue:     library.document.Cues[cueID].VolumeModifier,
			Request: 1, Distance: 1, Modifier: 1, Category: 1, Envelope: 1, Effective: player.Volume()}}
}

func (sep *SoundEffectPlayer) snapshotVoices() []PlaybackVoice {
	sep.mutex.RLock()
	defer sep.mutex.RUnlock()
	voices := make([]PlaybackVoice, 0, len(sep.activeSounds))
	for id, sound := range sep.activeSounds {
		voice := sep.library.voiceDescription("sfx", id, sound.CueID, sound.RecordingID, sound.Player, sound.Mode != 0)
		voice.Gain.Request, voice.Gain.Category = sound.RequestGain, sep.config.SoundEffectVolume
		voice.Gain.Envelope = sound.FadeGain
		if !sound.Player.IsPlaying() && voice.Failure == "" {
			voice.State = "completed"
		}
		voices = append(voices, voice)
	}

	return voices
}

func (bms *BackgroundMusicSystem) snapshotVoice() []PlaybackVoice {
	bms.mutex.RLock()
	defer bms.mutex.RUnlock()
	if bms.player == nil {
		return nil
	}
	cueID := "music/track/" + bms.currentTrack
	voice := bms.library.voiceDescription("music", bms.currentTrack, cueID,
		bms.library.document.Cues[cueID].RecordingID, bms.player, false)
	voice.Gain.Category, voice.Gain.Modifier = bms.config.MusicVolume, bms.volumeGain
	if bms.isPlaying && !bms.player.IsPlaying() && voice.Failure == "" {
		voice.State = "completed"
	}

	return []PlaybackVoice{voice}
}

func (asm *AmbientSoundManager) snapshotVoices() ([]PlaybackVoice, int) {
	asm.mu.RLock()
	defer asm.mu.RUnlock()
	voices := make([]PlaybackVoice, 0, len(asm.activeSounds))
	library := asm.audioManager.library
	for id, sound := range asm.activeSounds {
		cueID := string(sound.Definition.SoundType)
		cue := library.document.Cues[cueID]
		voice := library.voiceDescription("ambient", string(id), cueID, cue.RecordingID,
			sound.Player, sound.Definition.LoopSound)
		voice.Asset = sound.Definition.FileName
		voice.Gain.Distance = AmbientDistanceGain(sound.LastDistance, sound.Definition.MaxDistance)
		voice.Gain.Modifier, voice.Gain.Category = sound.VolumeModifier, asm.audioManager.config.AmbientVolume
		voice.Gain.Envelope = sound.FadeGain
		if authored := voice.Gain.Recording * voice.Gain.Cue; authored > 0 {
			voice.Gain.Request = sound.Definition.BaseVolume / authored
		}
		if sound.FadeState == AmbientFadeOut && voice.Failure == "" {
			voice.State = "releasing"
		}
		voices = append(voices, voice)
	}

	return voices, len(asm.desired)
}
