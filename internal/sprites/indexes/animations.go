// Package indexes defines pure sprite identities and shared atlas membership.
package indexes

// AnimationFamily describes atlas-frame identity, not generated pixel effects or
// directional variants. Audio uses Base for all members; rendering also uses the
// timing and input policy without rewriting the stored map tile.
type AnimationFamily struct {
	Base           SpriteIndex
	Frames         int
	IntervalMs     int64
	NormalizeInput bool
}

// AnimationFor returns membership in a verified atlas sequence. The bounded
// original actor ranges exclude artifacts, fields, directions and Redux sprites.
func AnimationFor(sprite SpriteIndex) (AnimationFamily, bool) {
	const (
		objectInterval = 125
		clockInterval  = 250
		actorInterval  = 175
	)
	switch {
	case sprite >= Waterfall1 && sprite <= Waterfall4:
		return AnimationFamily{Waterfall1, 4, objectInterval, true}, true
	case sprite >= Fountain1 && sprite <= Fountain4:
		return AnimationFamily{Fountain1, 4, objectInterval, false}, true
	case sprite == Clock1 || sprite == Clock2:
		return AnimationFamily{Clock1, 2, clockInterval, true}, true
	case sprite == Bellows1 || sprite == Bellows2:
		// Original terrain animation toggles clocks and bellows together; ref:TILE-0007.
		// Reuse the established clock presentation interval for this pair.
		return AnimationFamily{Bellows1, 2, clockInterval, true}, true
	case sprite >= SitChairDownEat1 && sprite < Shard,
		sprite >= StoneGargoyle1 && sprite < PoisonField,
		sprite >= Whirpool1 && sprite <= ShadowLord4:
		return AnimationFamily{sprite - sprite%StandardNumberOfAnimationFrames, 4, actorInterval, false}, true
	default:
		return AnimationFamily{Base: sprite, Frames: 0, IntervalMs: 0, NormalizeInput: false}, false
	}
}

// CanonicalAnimationSprite gives animated members one stable authoring identity.
func CanonicalAnimationSprite(sprite SpriteIndex) SpriteIndex {
	if family, ok := AnimationFor(sprite); ok {
		return family.Base
	}

	return sprite
}
