package audio

import (
	"fmt"
	"slices"
)

// GainBreakdown explains linear amplitude before device/PCM gain splitting.
type GainBreakdown struct {
	Recording float64
	Cue       float64
	Request   float64
	Distance  float64
	Modifier  float64
	Category  float64
	Envelope  float64
	Effective float64
}

// PlaybackVoice describes actual playback, never the UI's selected recording.
type PlaybackVoice struct {
	ID          string
	Lane        string
	CueID       string
	RecordingID string
	Asset       string
	State       string
	Loop        bool
	PositionMs  int64
	Window      *PlaybackWindow
	Gain        GainBreakdown
	Failure     string
}

// PlaybackSnapshot is Redux's detached diagnostics contract. Capture never
// services audio, advances selection or spends a gameplay operation.
type PlaybackSnapshot struct {
	Enabled         bool
	Closed          bool
	EncodedBytes    int64
	Voices          []PlaybackVoice
	AmbientDesired  int
	AmbientFailures map[AmbientSoundType]AmbientFailure
}

// Lines explains the captured source identities and composed gain factors.
func (snapshot PlaybackSnapshot) Lines() []string {
	lines := []string{fmt.Sprintf("Audio: %d voices; enabled=%t closed=%t; %d encoded bytes",
		len(snapshot.Voices), snapshot.Enabled, snapshot.Closed, snapshot.EncodedBytes),
		"Gain: recording * cue * request * distance * source/context * category * fade -> applied"}
	for _, voice := range snapshot.Voices {
		gain := voice.Gain
		lines = append(lines, fmt.Sprintf("%s %s -> %s [%s] %d ms | %.3g * %.3g * %.3g * %.3g * %.3g * %.3g * %.3g -> %.3g",
			voice.Lane, voice.CueID, voice.RecordingID, voice.State, voice.PositionMs, gain.Recording, gain.Cue, gain.Request,
			gain.Distance, gain.Modifier, gain.Category, gain.Envelope, gain.Effective))
		if voice.Asset != "" {
			detail := "  asset: " + voice.Asset
			if voice.Window != nil {
				detail += fmt.Sprintf(" [%d..%d ms]", voice.Window.StartMs, voice.Window.EndMs)
			}
			lines = append(lines, detail)
		}
		if voice.Failure != "" {
			lines = append(lines, "  "+voice.Failure)
		}
	}
	keys := make([]AmbientSoundType, 0, len(snapshot.AmbientFailures))
	for id := range snapshot.AmbientFailures {
		keys = append(keys, id)
	}
	slices.Sort(keys)
	for _, id := range keys {
		failure := snapshot.AmbientFailures[id]
		lines = append(lines, fmt.Sprintf("ambient %s: retry level %d · %s", id, failure.Attempts, failure.Message))
	}

	return lines
}
