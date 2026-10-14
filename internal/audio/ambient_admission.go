package audio

import (
	"cmp"
	"math"
	"slices"
)

// AmbientVoiceLimit bounds sounding and releasing proximity voices together.
// Music, preview and the eight SFX voices have independent ownership/budgets.
const AmbientVoiceLimit = 16

type ambientRank struct {
	key      AmbientSoundType
	gain     float64
	retained bool
}

// rankSourcesLocked admits the strongest source/distance mix, retaining existing
// voices on equal gain, then breaking ties by stable identity. User category mute
// does not change this ranking or restart loops when unmuted. Failed sources in
// backoff and completed finite sources cannot reserve otherwise usable capacity.
func (asm *AmbientSoundManager) rankSourcesLocked() []AmbientSoundType {
	ranks := make([]ambientRank, 0, len(asm.desired))
	now := asm.now()
	for key, candidate := range asm.desired {
		active := asm.activeSounds[key] != nil
		attempt := asm.attempts[key]
		if candidate.definition.FileName == "" || candidate.definition.BaseVolume <= 0 || attempt.completed ||
			(!active && attempt.err != nil && now.Before(attempt.next)) {
			continue
		}
		gain := candidate.definition.BaseVolume * candidate.modifier *
			AmbientDistanceGain(candidate.distance, candidate.definition.MaxDistance)
		if math.IsNaN(gain) || math.IsInf(gain, 0) || gain < 0 {
			continue
		}
		ranks = append(ranks, ambientRank{key: key, gain: gain, retained: active})
	}
	slices.SortFunc(ranks, func(left, right ambientRank) int {
		if order := cmp.Compare(right.gain, left.gain); order != 0 {
			return order
		}
		if left.retained != right.retained {
			if left.retained {
				return -1
			}

			return 1
		}

		return cmp.Compare(left.key, right.key)
	})
	keys := make([]AmbientSoundType, 0, min(len(ranks), AmbientVoiceLimit))
	for _, rank := range ranks[:min(len(ranks), AmbientVoiceLimit)] {
		keys = append(keys, rank.key)
	}

	return keys
}
