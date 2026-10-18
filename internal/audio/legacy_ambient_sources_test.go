package audio

import (
	"slices"

	"github.com/bradhannah/UltimaVRedux/internal/references"
)

func sameAmbientAsset(left, right AmbientSoundDefinition) bool {
	return left.FileName == right.FileName && left.SoundType == right.SoundType && left.LoopSound == right.LoopSound
}

func (asm *AmbientSoundManager) collectSources(position references.Position, tiles []TileWithPosition,
	dynamic []AmbientSoundSource,
) map[AmbientSoundType]ambientCandidate {
	return collectAmbientSources(asm.soundDefinitions, asm.soundTypeDefinitions, position, tiles, dynamic)
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
			continue
		}
		if instance.FadeState == AmbientFadeOut {
			asm.updateFade(instance)
			instance.FadeState, instance.FadeFrom, instance.FadeStart = AmbientFadeIn, instance.FadeGain, asm.now()
		}
		instance.Definition, instance.Position, instance.VolumeModifier = candidate.definition, candidate.position, candidate.modifier
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
			continue
		}
		attempt := asm.attempts[key]
		if attempt.completed || asm.now().Before(attempt.next) {
			continue
		}
		instance, err := asm.prepareAmbientVoice(candidate.definition, candidate.position, candidate.distance, candidate.modifier)
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
