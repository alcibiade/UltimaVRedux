package audio

import (
	"github.com/bradhannah/UltimaVRedux/internal/references"
)

//nolint:godox // The owner intentionally deferred recordings for these wired slots.
func dungeonAmbientSource(library *Library, definitions map[AmbientSoundType]AmbientSoundDefinition,
	location references.Location, listener references.Position,
) (AmbientSoundSource, bool) {
	// TODO(audio): Supply cave/brick recordings; loop through floors/combat, fade on exit.
	id := library.document.Routing.dungeonCue(location)
	if id == "" {
		return AmbientSoundSource{}, false //nolint:exhaustruct // Explicit or inherited silence.
	}
	definition, exists := definitions[AmbientSoundType(id)]
	if !exists {
		panic("validated dungeon background has no definition")
	}

	return AmbientSoundSource{
		ID:      "",
		Emitter: "", Position: listener, SoundType: definition.SoundType, IsActive: true, IsDynamic: true,
		SoundFile: definition.FileName, MaxDistance: 0, baseVolume: definition.BaseVolume,
		VolumeModifier: 1, FadeInTime: definition.FadeInTime, FadeOutTime: definition.FadeOutTime,
	}, true
}

func navalAmbientSource(
	routing Routing, definitions map[AmbientSoundType]AmbientSoundDefinition,
	vessel string, listener references.Position,
) (AmbientSoundSource, bool) {
	if vessel != NavalFrigate && vessel != NavalSkiff {
		return AmbientSoundSource{}, false //nolint:exhaustruct // No supported boarded vessel.
	}
	cue := routing.Naval[vessel]
	if cue == nil {
		return AmbientSoundSource{}, false //nolint:exhaustruct // Authored silence.
	}
	definition, exists := definitions[AmbientSoundType(*cue)]
	if !exists {
		panic("validated naval background has no definition")
	}

	return AmbientSoundSource{
		ID:      "",
		Emitter: vessel, Position: listener, SoundType: definition.SoundType, IsActive: true, IsDynamic: true,
		SoundFile: definition.FileName, MaxDistance: 0, baseVolume: definition.BaseVolume,
		VolumeModifier: 1, FadeInTime: definition.FadeInTime, FadeOutTime: definition.FadeOutTime,
	}, true
}
