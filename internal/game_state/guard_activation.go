package game_state

import (
	"errors"
	"fmt"

	"github.com/bradhannah/UltimaVRedux/internal/ai"
	"github.com/bradhannah/UltimaVRedux/internal/map_units"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
)

// ErrInvalidGuardActivation indicates that guard hostility is unavailable in the current context.
var ErrInvalidGuardActivation = errors.New("invalid guard activation")

const (
	guardActivationFleeRollMaximum = 255
	guardActivationFleeThreshold   = 128
)

// GuardActivationResult reports the state changes and ordered RNG work performed by ActivateGuards.
// CriminalBefore and CriminalAfter remain equal because criminal status is caller-owned Redux policy,
// not part of the original activate_guards operation.
type GuardActivationResult struct {
	CriminalBefore      bool
	CriminalAfter       bool
	KarmaAtActivation   int
	ActiveNPCCount      int
	PotentialGuardCount int
	HostileBefore       int
	HostileAfter        int
	CivilianRollCount   int
	FleeAttemptCount    int
	FleeAcceptedCount   int
	FleeRejectedCount   int
	FleeBefore          int
	FleeAfter           int
}

// ActivateGuards applies the original town rules guard activation operation in NPC slot order.
// See ref:CRIME-0002.
func (g *GameState) ActivateGuards() (GuardActivationResult, error) {
	result := GuardActivationResult{CriminalBefore: g.IsCriminal(), CriminalAfter: g.IsCriminal()}
	if g.IsInCombat() {
		return result, fmt.Errorf("%w: unavailable during combat", ErrInvalidGuardActivation)
	}
	if g.CurrentNPCAIController == nil {
		return result, fmt.Errorf("%w: NPC AI controller is unavailable", ErrInvalidGuardActivation)
	}
	controller, ok := g.CurrentNPCAIController.(*ai.NPCAIControllerSmallMap)
	if !ok {
		return result, nil
	}
	result.KarmaAtActivation = g.PartyState.Karma.Value

	for _, unit := range guardActivationSlots(controller) {
		if unit == nil || unit.IsEmptyMapUnit() || !unit.IsVisible() {
			continue
		}

		result.ActiveNPCCount++
		if guardActivationNPCIsFleeing(unit) {
			result.FleeBefore++
		}

		if guardActivationHostileShape(unit) {
			result.PotentialGuardCount++
			if guardActivationNPCIsHostile(unit) {
				result.HostileBefore++
			}
			if friendly, isFriendly := unit.(*map_units.NPCFriendly); isFriendly {
				// The original hostility override compares unlike tile/type encodings
				// for its Shadowlord branch. Preserve the current override representation
				// without treating that comparison as universal shape equivalence. ref:CRIME-0002
				friendly.MapUnitDetails().SetLawAIOverride(references.DrudgeWorthThing)
				friendly.MapUnitDetails().SetCurrentPath(nil)
				friendly.MapUnitDetails().PendingInteraction = map_units.PendingNPCInteractionNone
			}
			if guardActivationNPCIsHostile(unit) {
				result.HostileAfter++
			}
		} else {
			// Original random(0,255) is consumed before fleeing schedule override applies its own eligibility checks.
			// See ref:CRIME-0002.
			result.CivilianRollCount++
			if g.RandomIntInRange(0, guardActivationFleeRollMaximum) < guardActivationFleeThreshold {
				result.FleeAttemptCount++
				friendly, isFriendly := unit.(*map_units.NPCFriendly)
				if isFriendly && controller.SetFleeFromAvatar(friendly) {
					result.FleeAcceptedCount++
				} else {
					result.FleeRejectedCount++
				}
			}
		}

		if guardActivationNPCIsFleeing(unit) {
			result.FleeAfter++
		}
	}

	return result, nil
}

func guardActivationSlots(controller *ai.NPCAIControllerSmallMap) [map_units.MaximumNpcsPerMap]map_units.MapUnit {
	var slots [map_units.MaximumNpcsPerMap]map_units.MapUnit
	for _, unit := range *controller.GetNpcs() {
		if unit == nil {
			continue
		}
		slot := unit.MapUnitDetails().NPCNum
		if slot >= 0 && slot < len(slots) && slots[slot] == nil {
			slots[slot] = unit
		}
	}

	return slots
}

func guardActivationHostileShape(unit map_units.MapUnit) bool {
	sprite := unit.GetKeySpriteIndex()

	return sprite == indexes.GuardKeyIndex ||
		sprite == indexes.DaemonKeyIndex ||
		sprite == indexes.ShadowLordKeyIndex
}

func guardActivationNPCIsHostile(unit map_units.MapUnit) bool {
	if _, isEnemy := unit.(*map_units.NPCEnemy); isEnemy {
		return true
	}

	action := unit.MapUnitDetails().GetOverriddenAiType()

	return action == references.ExtortOrAttackOrFollow || action == references.DrudgeWorthThing
}

func guardActivationNPCIsFleeing(unit map_units.MapUnit) bool {
	return unit.MapUnitDetails().GetOverriddenAiType() == references.FleeFromAvatar
}
