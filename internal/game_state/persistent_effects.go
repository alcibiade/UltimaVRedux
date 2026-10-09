package game_state

import (
	"fmt"

	"github.com/bradhannah/UltimaVRedux/internal/references"
)

// PersistentEffectID identifies a durable party-wide condition independently of its artwork.
type PersistentEffectID string

const (
	// PersistentEffectTorch is carried mundane illumination.
	PersistentEffectTorch PersistentEffectID = "torch"
	// PersistentEffectMagicLight is spell-created illumination.
	PersistentEffectMagicLight PersistentEffectID = "magic_light"
	// PersistentEffectCrownSuppression is permanent Crown-sourced suppression, not In An.
	PersistentEffectCrownSuppression PersistentEffectID = "crown_suppression"
	// PersistentEffectProtection is the active defensive duration spell.
	PersistentEffectProtection PersistentEffectID = "protection"
	// PersistentEffectNegateMagic is the active magic-suppression duration spell.
	PersistentEffectNegateMagic PersistentEffectID = "negate_magic"
	// PersistentEffectTimeStop is the active frozen-time duration spell.
	PersistentEffectTimeStop PersistentEffectID = "time_stop"
	// PersistentEffectQuickness is the active speed duration spell.
	PersistentEffectQuickness PersistentEffectID = "quickness"
	// PersistentEffectConfusion is the active monster-confusion duration spell.
	PersistentEffectConfusion PersistentEffectID = "confusion"
	// PersistentEffectIntoxication is the active town-command intoxication consequence.
	PersistentEffectIntoxication PersistentEffectID = "intoxication"
	// PersistentEffectShadowlord is automatic Shadowlord town influence.
	PersistentEffectShadowlord PersistentEffectID = "shadowlord"
	// PersistentEffectStarvation is the party-wide zero-food condition.
	PersistentEffectStarvation PersistentEffectID = "starvation"
	// PersistentEffectHMSCape is active frigate rigging.
	PersistentEffectHMSCape PersistentEffectID = "hms_cape"
)

// PersistentEffectAtmosphere describes nonnumeric presentation emphasis.
type PersistentEffectAtmosphere byte

const (
	// EffectAtmosphereSteady has no urgency treatment.
	EffectAtmosphereSteady PersistentEffectAtmosphere = iota
	// EffectAtmosphereFading indicates a condition is weakening.
	EffectAtmosphereFading
	// EffectAtmosphereCritical indicates a condition is nearly exhausted.
	EffectAtmosphereCritical
	// EffectAtmosphereOminous indicates hostile world influence.
	EffectAtmosphereOminous
	// EffectAtmosphereDanger indicates an active party hazard.
	EffectAtmosphereDanger
)

// PersistentEffectStatus is a detached view of one active party-wide condition.
type PersistentEffectStatus struct {
	ID         PersistentEffectID
	Atmosphere PersistentEffectAtmosphere
}

// GetPersistentEffectStatuses returns active conditions in deterministic display order.
// REDUX DEVIATION: Original U5 original party-effect rules displayed one active effect marker and no
// torch or magic-light status icon. Redux presents concurrent party-wide state
// without exposing exact countdowns so effects remain readable off the map.
// See ref:SPELL-0004.
func (g *GameState) GetPersistentEffectStatuses() []PersistentEffectStatus {
	effects := make([]PersistentEffectStatus, 0, 7)

	if minutes := g.MapState.Lighting.TorchMinutesRemaining(); minutes > 0 {
		effects = append(effects, PersistentEffectStatus{
			ID:         PersistentEffectTorch,
			Atmosphere: remainingMinutesAtmosphere(minutes),
		})
	}
	if g.MagicLightMinutesRemaining > 0 {
		effects = append(effects, PersistentEffectStatus{
			ID:         PersistentEffectMagicLight,
			Atmosphere: remainingMinutesAtmosphere(g.MagicLightMinutesRemaining),
		})
	}
	if g.IsCrownActive() {
		effects = append(effects, PersistentEffectStatus{
			ID:         PersistentEffectCrownSuppression,
			Atmosphere: EffectAtmosphereSteady,
		})
	} else if g.HasAnyDurationEffect() {
		effects = append(effects, PersistentEffectStatus{
			ID:         persistentDurationEffectID(g.ActiveDurationEffect.EffectType),
			Atmosphere: remainingTurnsAtmosphere(g.ActiveDurationEffect.TurnsRemaining),
		})
	}
	if g.smashed > 0 {
		effects = append(effects, PersistentEffectStatus{
			ID:         PersistentEffectIntoxication,
			Atmosphere: EffectAtmosphereDanger,
		})
	}
	if g.ShadowlordState != nil && g.ShadowlordState.HasAutomaticPresence() {
		effects = append(effects, PersistentEffectStatus{
			ID:         PersistentEffectShadowlord,
			Atmosphere: EffectAtmosphereOminous,
		})
	}
	if g.PartyState.Inventory.Provisions.Food.Get() == 0 {
		effects = append(effects, PersistentEffectStatus{
			ID:         PersistentEffectStarvation,
			Atmosphere: EffectAtmosphereDanger,
		})
	}
	if g.IsHMSCapeActivated() && g.PartyVehicle.GetVehicleDetails().VehicleType == references.FrigateVehicle {
		effects = append(effects, PersistentEffectStatus{
			ID:         PersistentEffectHMSCape,
			Atmosphere: EffectAtmosphereSteady,
		})
	}

	return effects
}

func persistentDurationEffectID(effectType DurationEffectType) PersistentEffectID {
	switch effectType {
	case DurationEffectProtection:
		return PersistentEffectProtection
	case DurationEffectNegateMagic:
		return PersistentEffectNegateMagic
	case DurationEffectTimeStop:
		return PersistentEffectTimeStop
	case DurationEffectQuickness:
		return PersistentEffectQuickness
	case DurationEffectConfusion:
		return PersistentEffectConfusion
	case DurationEffectNone, DurationEffectLight:
		panic(fmt.Sprintf("unsupported persistent duration effect %q", effectType))
	default:
		panic(fmt.Sprintf("unsupported persistent duration effect %q", effectType))
	}
}

func remainingMinutesAtmosphere(remaining int) PersistentEffectAtmosphere {
	switch {
	case remaining <= 10:
		return EffectAtmosphereCritical
	case remaining <= 25:
		return EffectAtmosphereFading
	default:
		return EffectAtmosphereSteady
	}
}

func remainingTurnsAtmosphere(remaining int) PersistentEffectAtmosphere {
	switch {
	case remaining <= 3:
		return EffectAtmosphereCritical
	case remaining <= 5:
		return EffectAtmosphereFading
	default:
		return EffectAtmosphereSteady
	}
}
