package audio

import (
	"fmt"
	"math"
	"slices"

	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
)

type ambientCandidate struct {
	emitter            string
	definition         AmbientSoundDefinition
	position           references.Position
	distance, modifier float64
}

func sameAmbientAsset(left, right AmbientSoundDefinition) bool {
	return left.FileName == right.FileName && left.SoundType == right.SoundType && left.LoopSound == right.LoopSound
}

func (asm *AmbientSoundManager) collectSources(
	position references.Position, tiles []TileWithPosition, dynamic []AmbientSoundSource,
) map[AmbientSoundType]ambientCandidate {
	return collectAmbientSources(asm.soundDefinitions, asm.soundTypeDefinitions, position, tiles, dynamic)
}

// collectAmbientSources is Redux's world-to-soundscape policy. Both the existing
// manager and the staged module adapter use this same grouping/identity logic.
func collectAmbientSources(tileDefinitions map[indexes.SpriteIndex]AmbientSoundDefinition,
	definitions map[AmbientSoundType]AmbientSoundDefinition, position references.Position,
	tiles []TileWithPosition, dynamic []AmbientSoundSource,
) map[AmbientSoundType]ambientCandidate {
	sources := make(map[AmbientSoundType]ambientCandidate)
	for _, tile := range tiles {
		definition, exists := tileDefinitions[tile.Tile.Index]
		if !exists {
			continue
		}
		distance := ambientSourceDistance(position, tile.Position)
		previous, exists := sources[definition.SoundType]
		if strongerAmbientSource(definition, distance, previous.definition, previous.distance, exists) {
			sources[definition.SoundType] = ambientCandidate{
				emitter:    "",
				definition: definition, position: tile.Position, distance: distance, modifier: 1,
			}
		}
	}
	for _, source := range dynamic {
		collectDynamicAmbientSource(sources, definitions, position, source)
	}

	return sources
}

func collectDynamicAmbientSource(
	sources map[AmbientSoundType]ambientCandidate, definitions map[AmbientSoundType]AmbientSoundDefinition,
	listener references.Position, source AmbientSoundSource,
) {
	if !source.IsActive {
		return
	}
	base := definitions[source.SoundType]
	definition := AmbientSoundDefinition{
		SoundType: source.SoundType, FileName: source.SoundFile, MaxDistance: source.MaxDistance,
		BaseVolume: source.GetBaseVolume(), LoopSound: base.LoopSound,
		FadeInTime: source.FadeInTime, FadeOutTime: source.FadeOutTime,
	}
	distance := ambientSourceDistance(listener, source.Position)
	if definition.MaxDistance == 0 {
		distance = 0 // A background source is non-spatial, not tied to the last listener snapshot.
	}
	if distance > definition.MaxDistance {
		return
	}
	modifier := source.VolumeModifier
	if !source.IsDynamic && modifier == 0 && definition.BaseVolume > 0 {
		modifier = 1
	}
	key := AmbientSoundType(fmt.Sprintf("source/%s/%s/%d/%d",
		source.Emitter, source.SoundType, source.Position.X, source.Position.Y))
	if source.ID != "" {
		key = AmbientSoundType("source/" + source.ID)
	}
	if definition.MaxDistance == 0 {
		key = AmbientSoundType("background/" + string(source.SoundType))
	}
	if source.Emitter == lighthouseEmitter {
		key = source.SoundType // Preserve the coastal/interior lighthouse voice.
	}
	previous, exists := sources[key]
	if !exists || distance < previous.distance || source.Emitter == lighthouseEmitter {
		sources[key] = ambientCandidate{
			emitter:    source.Emitter,
			definition: definition, position: source.Position, distance: distance, modifier: modifier,
		}
	}
}

func ambientSourceDistance(left, right references.Position) float64 {
	dx, dy := float64(left.X-right.X), float64(left.Y-right.Y)

	return math.Sqrt(dx*dx + dy*dy)
}

func (asm *AmbientSoundManager) reconcileSourcesLocked(sources map[AmbientSoundType]ambientCandidate) {
	asm.desired = sources
	for key, attempt := range asm.attempts {
		candidate, exists := sources[key]
		if !exists || !sameAmbientAsset(candidate.definition, attempt.definition) {
			delete(asm.attempts, key)
		}
	}
}

func (asm *AmbientSoundManager) reconcileAdmittedLocked(admitted []AmbientSoundType) {
	for key, instance := range asm.activeSounds {
		candidate, exists := asm.desired[key]
		if exists && candidate.definition.FileName == "" {
			asm.retireAmbientLocked(key)

			continue
		}
		if !exists || !slices.Contains(admitted, key) {
			asm.fadeOutAmbientSound(instance)

			continue
		}
		if !sameAmbientAsset(instance.Definition, candidate.definition) {
			continue // Prepare a replacement before retiring the existing voice.
		}
		if instance.FadeState == AmbientFadeOut {
			asm.updateFade(instance)
			instance.FadeState, instance.FadeFrom, instance.FadeStart = AmbientFadeIn, instance.FadeGain, asm.now()
		}
		instance.Definition, instance.Position, instance.VolumeModifier =
			candidate.definition, candidate.position, candidate.modifier
		instance.emitter = candidate.emitter
		asm.updateAmbientSound(instance, candidate.distance)
	}
}

func (asm *AmbientSoundManager) startDesiredSourcesLocked(admitted []AmbientSoundType) {
	for _, key := range admitted {
		candidate := asm.desired[key]
		if candidate.definition.FileName == "" {
			continue
		}
		if active := asm.activeSounds[key]; active != nil && sameAmbientAsset(active.Definition, candidate.definition) {
			continue
		}
		if asm.activeSounds[key] == nil && len(asm.activeSounds) >= AmbientVoiceLimit {
			continue // Release tails count against the budget; do not overlap new arrivals.
		}
		attempt := asm.attempts[key]
		if attempt.completed || asm.now().Before(attempt.next) {
			continue
		}
		instance, err := asm.prepareAmbientVoice(
			candidate.definition, candidate.position, candidate.distance, candidate.modifier)
		if err != nil {
			asm.failAmbientLocked(key, candidate.definition, err)

			continue
		}
		if asm.activeSounds[key] != nil {
			asm.retireAmbientLocked(key)
		}
		asm.activeSounds[key] = instance
		instance.emitter = candidate.emitter
		instance.Player.Play()
		attempt.definition, attempt.err = candidate.definition, nil
		asm.attempts[key] = attempt
	}
}
