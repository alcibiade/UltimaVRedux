package audio

import (
	"math"

	"github.com/bradhannah/UltimaVRedux/internal/references"
)

// AmbientScanRadius follows authored ranges rather than a fixed scene constant.
func (am *AudioManager) AmbientScanRadius() int {
	var radius float64
	for _, profile := range productionLibrary().Routing.Ambient {
		radius = max(radius, profile.RadiusTiles)
	}
	for _, tile := range productionLibrary().Routing.Tiles {
		if tile.RadiusTiles != nil {
			radius = max(radius, *tile.RadiusTiles)
		}
	}

	return int(math.Ceil(radius))
}

// AmbientEmitterDefinition resolves the two code-owned dynamic emitter roles.
func (am *AudioManager) AmbientEmitterDefinition(emitter string) AmbientSoundDefinition {
	id, exists := productionLibrary().Routing.Emitters[emitter]
	if !exists {
		panic("unknown ambient emitter: " + emitter)
	}
	definition, exists := am.GetAmbientSoundDefinition(AmbientSoundType(id))
	if !exists {
		panic("validated ambient emitter has no definition")
	}

	return definition
}

// DungeonAmbientSource returns a stable non-spatial source at the listener.
// Its playback key is the cue, not the listener's changing coordinates.
//
//nolint:godox // The owner intentionally deferred recordings for these wired slots.
func (am *AudioManager) DungeonAmbientSource(
	location references.Location, listener references.Position,
) (AmbientSoundSource, bool) {
	// TODO(audio): Supply cave/brick recordings; loop through floors/combat, fade on exit.
	id := productionLibrary().Routing.dungeonCue(location)
	if id == "" {
		return AmbientSoundSource{}, false //nolint:exhaustruct // Explicit or inherited silence.
	}
	definition, exists := am.GetAmbientSoundDefinition(AmbientSoundType(id))
	if !exists {
		panic("validated dungeon background has no definition")
	}

	return AmbientSoundSource{
		Emitter: "", Position: listener, SoundType: definition.SoundType, IsActive: true, IsDynamic: true,
		SoundFile: definition.FileName, MaxDistance: 0, baseVolume: definition.BaseVolume,
		VolumeModifier: 1, FadeInTime: definition.FadeInTime, FadeOutTime: definition.FadeOutTime,
	}, true
}

// NavalAmbientSource follows the boarded vessel, not the visible water tiles.
// Its stable cue key retains the loop through travel, turns and combat.
func (am *AudioManager) NavalAmbientSource(vessel string, listener references.Position) (AmbientSoundSource, bool) {
	return navalAmbientSource(productionLibrary().Routing, am.ambientSounds.soundTypeDefinitions, vessel, listener)
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
		Emitter: vessel, Position: listener, SoundType: definition.SoundType, IsActive: true, IsDynamic: true,
		SoundFile: definition.FileName, MaxDistance: 0, baseVolume: definition.BaseVolume,
		VolumeModifier: 1, FadeInTime: definition.FadeInTime, FadeOutTime: definition.FadeOutTime,
	}, true
}
