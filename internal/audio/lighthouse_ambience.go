package audio

import "github.com/bradhannah/UltimaVRedux/internal/references"

const lighthouseEmitter = "lighthouse"

// LighthouseAmbientSource follows the location, independent of floor or visible
// coastal tiles. The manager shares its voice with nearby lighthouse tile cues.
func (am *AudioManager) LighthouseAmbientSource(
	location references.Location, listener references.Position,
) (AmbientSoundSource, bool) {
	if location != references.Fogsbane && location != references.Stormcrow &&
		location != references.Greyhaven && location != references.Waveguide {
		return AmbientSoundSource{}, false //nolint:exhaustruct // No lighthouse location source.
	}
	definition, exists := am.GetAmbientSoundDefinition(AmbientLighthouse)
	if !exists {
		panic("lighthouse ambient cue has no definition")
	}

	return AmbientSoundSource{
		ID:      "",
		Emitter: lighthouseEmitter, Position: listener, SoundType: definition.SoundType,
		IsActive: true, IsDynamic: true, SoundFile: definition.FileName, MaxDistance: 0,
		baseVolume: definition.BaseVolume, VolumeModifier: 1,
		FadeInTime: definition.FadeInTime, FadeOutTime: definition.FadeOutTime,
	}, true
}
