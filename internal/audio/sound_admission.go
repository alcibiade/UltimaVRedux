package audio

import (
	"fmt"

	"github.com/bradhannah/UltimaVRedux/internal/game_state"
)

// skipInaudibleCue validates identity before the metadata-only mute decision.
// It must run before lazy preparation, which can decode every variant of a cue.
func (sep *SoundEffectPlayer) skipInaudibleCue(request game_state.SoundPlaybackRequest, cueID string) (bool, error) {
	if err := sep.validateRequest(request); err != nil {
		return false, err
	}
	cue, exists := sep.library.document.Cues[cueID]
	if !exists {
		return false, fmt.Errorf("%w: unknown cue %s", ErrAudioLibrary, cueID)
	}
	if cue.RecordingID != "" && sep.library.document.Recordings[cue.RecordingID].File == nil {
		return true, nil // Explicit empty slots need neither preparation nor mix state.
	}

	return request.Mode == game_state.SoundPlaybackOneShot &&
		sep.config.GetEffectiveVolume(request.Volume*cue.VolumeModifier) == 0, nil
}
