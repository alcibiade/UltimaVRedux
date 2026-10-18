package audio

import (
	"math"

	"github.com/bradhannah/UltimaVRedux/internal/references"
)

func (am *AudioManager) AmbientScanRadius() int {
	var radius float64
	for _, profile := range am.library.document.Routing.Ambient {
		radius = max(radius, profile.RadiusTiles)
	}
	for _, tile := range am.library.document.Routing.Tiles {
		if tile.RadiusTiles != nil {
			radius = max(radius, *tile.RadiusTiles)
		}
	}
	return int(math.Ceil(radius))
}

func (am *AudioManager) AmbientEmitterDefinition(emitter string) AmbientSoundDefinition {
	id, exists := am.library.document.Routing.Emitters[emitter]
	if !exists {
		panic("unknown ambient emitter: " + emitter)
	}
	definition, exists := am.GetAmbientSoundDefinition(AmbientSoundType(id))
	if !exists {
		panic("validated ambient emitter has no definition")
	}
	return definition
}

func (am *AudioManager) DungeonAmbientSource(location references.Location, listener references.Position) (AmbientSoundSource, bool) {
	return dungeonAmbientSource(am.library, am.ambientSounds.soundTypeDefinitions, location, listener)
}

func (am *AudioManager) NavalAmbientSource(vessel string, listener references.Position) (AmbientSoundSource, bool) {
	return navalAmbientSource(am.library.document.Routing, am.ambientSounds.soundTypeDefinitions, vessel, listener)
}

func (am *AudioManager) LighthouseAmbientSource(location references.Location, listener references.Position) (AmbientSoundSource, bool) {
	return lighthouseAmbientSource(am.ambientSounds.soundTypeDefinitions, location, listener)
}
