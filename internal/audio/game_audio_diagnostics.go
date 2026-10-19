package audio

import (
	"cmp"
	"slices"
	"strconv"

	berad "github.com/bradhannah/berad-sdk/modules/berad-audio"
)

// PlaybackSnapshot adapts detached module diagnostics to the game's existing UI
// contract. It performs no audio servicing, source discovery or clock sampling.
func (manager *GameAudioManager) PlaybackSnapshot() PlaybackSnapshot {
	manager.mutex.RLock()
	defer manager.mutex.RUnlock()
	result := PlaybackSnapshot{Enabled: manager.enabled, Closed: manager.closed,
		EncodedBytes: manager.playback.EncodedBytes(),
		Voices:       nil, AmbientDesired: 0, AmbientFailures: make(map[AmbientSoundType]AmbientFailure)}
	for _, voice := range manager.sfx.Snapshot() {
		entry := manager.voiceDescription("sfx", strconv.FormatUint(voice.ID, 10),
			voice.Cue, voice.Recording, voice.State, voice.Loop)
		entry.PositionMs = voice.Position.Milliseconds()
		entry.Gain.Request, entry.Gain.Category, entry.Gain.Envelope, entry.Gain.Effective =
			voice.RequestGain, manager.config.SoundEffectVolume, voice.Envelope, voice.Gain
		result.Voices = append(result.Voices, entry)
	}
	music := manager.music.Snapshot()
	if music.Voice.State == berad.VoicePlaying || music.Voice.State == berad.VoicePaused {
		entry := manager.voiceDescription("music", string(music.Voice.Cue),
			music.Voice.Cue, music.Voice.Recording, music.Voice.State, false)
		entry.PositionMs = music.Position.Milliseconds()
		entry.Gain.Category, entry.Gain.Modifier = music.CategoryGain, music.ContextGain
		entry.Gain.Effective = music.Voice.Gain
		result.Voices = append(result.Voices, entry)
	}
	ambience := manager.ambience.Snapshot()
	result.AmbientDesired = ambience.Desired
	for _, source := range ambience.Sources {
		if source.Voice.State == berad.VoicePlaying || source.Voice.State == berad.VoiceReleasing {
			entry := manager.voiceDescription("ambient", string(source.ID),
				source.Voice.Cue, source.Voice.Recording, source.Voice.State, true)
			entry.Gain.Category, entry.Gain.Effective = manager.config.AmbientVolume, source.Voice.Gain
			mix := manager.sourceMix[source.ID]
			entry.Gain.Request, entry.Gain.Distance, entry.Gain.Modifier = mix.SourceGain, mix.DistanceGain, mix.Intensity
			entry.Gain.Envelope = source.Voice.Envelope
			entry.PositionMs = source.Voice.Position.Milliseconds()
			result.Voices = append(result.Voices, entry)
		}
		if source.Failure != "" {
			result.AmbientFailures[AmbientSoundType(source.ID)] = AmbientFailure{
				Attempts: source.Attempts, RetryAt: ambience.At.Add(source.RetryAfter), Message: source.Failure}
		}
	}
	for _, voice := range manager.preview.Snapshot() {
		entry := manager.voiceDescription("preview", "ambient-preview", voice.Cue, voice.Recording, voice.State, true)
		entry.PositionMs = voice.Position.Milliseconds()
		entry.Gain.Category, entry.Gain.Effective = manager.config.AmbientVolume, voice.Gain
		result.Voices = append(result.Voices, entry)
	}
	slices.SortFunc(result.Voices, func(left, right PlaybackVoice) int {
		if order := cmp.Compare(left.Lane, right.Lane); order != 0 {
			return order
		}

		return cmp.Compare(left.ID, right.ID)
	})

	return result
}

func (manager *GameAudioManager) voiceDescription(lane, identity string, cueID berad.CueID,
	recordingID berad.RecordingID, state berad.VoiceState, loop bool,
) PlaybackVoice {
	recording := manager.library.document.Recordings[string(recordingID)]

	return PlaybackVoice{ID: identity, Lane: lane, CueID: string(cueID), RecordingID: string(recordingID),
		Asset: manager.library.recordingPath(string(recordingID)), State: string(state), Loop: loop, PositionMs: 0,
		Window: cloneAudioValue(recording.PlaybackWindow), Failure: "",
		Gain: GainBreakdown{Recording: manager.library.document.recordingCalibration(string(recordingID)),
			Cue:     manager.library.document.Cues[string(cueID)].VolumeModifier,
			Request: 1, Distance: 1, Modifier: 1, Category: 1, Envelope: 1, Effective: 0}}
}
