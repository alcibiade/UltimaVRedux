package game_state

import "github.com/bradhannah/UltimaVRedux/internal/references"

// CombatShotStatus describes an intended attack, never a prediction of its random hit roll.
type CombatShotStatus string

// CombatShotClear and the remaining statuses describe deterministic attack eligibility.
const (
	CombatShotClear         CombatShotStatus = "Clear"
	CombatShotTerrainBypass CombatShotStatus = "Bypasses terrain on hit"
	CombatShotOutOfRange    CombatShotStatus = "Out of range"
	CombatShotBlocked       CombatShotStatus = "Blocked"
	CombatShotNoAmmo        CombatShotStatus = "No ammo"
	CombatShotInterference  CombatShotStatus = "Interference"
	CombatShotUnready       CombatShotStatus = "Not readied"
	CombatShotChooseTarget  CombatShotStatus = "Choose target"
	CombatShotFriendly      CombatShotStatus = "Friendly"
)

// Attackable reports whether the intended shot may be offered by assistance.
func (status CombatShotStatus) Attackable() bool {
	return status == CombatShotClear || status == CombatShotTerrainBypass
}

// CombatAttackTargetPreview contains only visible identity and deterministic shot facts.
type CombatAttackTargetPreview struct {
	Position     references.Position
	MonsterIndex int
	Name         string
	Distance     int
	Status       CombatShotStatus
	Actor        *CombatActorPresentation
}

// CombatAttackPreview is a detached, current-state projection. During physical
// aiming Sources contains only the current source; otherwise it contains the loadout.
type CombatAttackPreview struct {
	Sources []CombatAttackSource
	Targets []CombatAttackTargetPreview
	Aim     *CombatAttackTargetPreview
}

// PreviewCombatAttack shares Tab's eligibility without moving cursors, remembering
// targets, emitting output, or consuming RNG/resources. No preview is cached.
func (g *GameState) PreviewCombatAttack() CombatAttackPreview {
	preview := CombatAttackPreview{}
	actor, active := g.activeCombatPCIdentity()
	if !active || !g.canPreviewCombatAttack(actor) {
		return preview
	}
	origin := g.CombatScene.GetCombatantPosition(actor)
	preview.Sources = collectCombatAttackSources(g.PartyState.GetCharacter(actor.EntityIndex))
	if targeting := g.CombatScene.AttackTargeting; targeting != nil {
		if targeting.Attacker != actor || targeting.autoTarget != nil {
			return CombatAttackPreview{}
		}
		preview.Sources = []CombatAttackSource{targeting.CurrentSource()}
		aim := g.reviewCombatAttackPositionDisclosure(actor, origin, preview.Sources[0], targeting.Cursor, true)
		preview.Aim = &aim
	}
	g.CombatScene.MonsterManager.ForEachActive(func(index int, monster *CombatMonster) {
		if !g.isVisibleCombatAutoTarget(monster) {
			return
		}
		for _, source := range preview.Sources {
			target := g.reviewCombatAttackPosition(actor, origin, source, monster.Position())
			if target.Status.Attackable() {
				target.MonsterIndex = index
				preview.Targets = append(preview.Targets, target)

				break
			}
		}
	})

	return preview
}

func (g *GameState) canPreviewCombatAttack(actor CombatTurnIdentity) bool {
	combat := g.CombatScene

	return !combat.HasScheduledCombatStep() && !combat.isCombatPCSleeping(actor) &&
		!combat.isCombatPCPossessed(actor) && !combat.IsPartyTurnAutoPass(actor.EntityIndex) &&
		combat.SpellTargeting == nil && combat.ConeTargeting == nil &&
		combat.AdjacentSpellTargeting == nil && combat.pendingSpell == nil && combat.pendingAttack == nil
}

func (g *GameState) isVisibleCombatAutoTarget(monster *CombatMonster) bool {
	return monster != nil && monster.IsActive() && !monster.IsDead() && monster.IsHostile() &&
		g.CombatTerrainAllowsObjectDisplay(monster.Position()) &&
		!monster.IsCorpserHidden() && (!monster.IsInvisible() || !g.CombatScene.invisibilityProtectsTargets())
}

func (g *GameState) combatAttackSourceAvailability(actor CombatTurnIdentity, source CombatAttackSource) CombatShotStatus {
	if ammo := source.Equipment.GetRequiredAmmo(); ammo != references.NoEquipment &&
		!g.PartyState.Inventory.Equipment.HasSome(ammo) {
		return CombatShotNoAmmo
	}
	if source.Slot != references.SlotNone &&
		g.PartyState.GetCharacter(actor.EntityIndex).GetEquippedItem(source.Slot) != source.Equipment {
		return CombatShotUnready
	}
	if source.Equipment.IsSubjectToCombatInterference() {
		if _, found := g.validRememberedMeleeEngager(actor); found {
			return CombatShotInterference
		}
	}

	return CombatShotClear
}

func (g *GameState) reviewCombatAttackPosition(
	actor CombatTurnIdentity, origin references.Position, source CombatAttackSource, position references.Position,
) CombatAttackTargetPreview {
	return g.reviewCombatAttackPositionDisclosure(actor, origin, source, position, false)
}

// Assistance keeps its existing dungeon invisibility/occupancy rules. Manual
// readouts instead classify only the disclosed exact actor, so neither a name
// nor a friendly/bypass status can become an invisible-occupancy detector.
func (g *GameState) reviewCombatAttackPositionDisclosure(
	actor CombatTurnIdentity, origin references.Position, source CombatAttackSource, position references.Position, disclosedOnly bool,
) CombatAttackTargetPreview {
	result := CombatAttackTargetPreview{
		Position: position, MonsterIndex: NoCombatTarget, Distance: integerDistance(origin, position),
		Name: "", Status: CombatShotClear,
	}
	if view, disclosed := g.CombatActorAt(position); disclosed && position != origin {
		result.Name, result.Actor = view.Name, &view
		if !view.Identity.IsPartyMember {
			result.MonsterIndex = view.Identity.EntityIndex
		}
	}
	disposition := g.GetCombatAimDisposition(position)
	monster, _ := g.CombatScene.MonsterManager.GetMonsterIndexAtPosition(position)
	bypassTarget := disposition == CombatAimEnemy && g.isVisibleCombatAutoTarget(monster)
	if disclosedOnly {
		disposition, bypassTarget = CombatAimEmpty, false
		if result.Actor != nil {
			if result.Actor.Loyalty == CombatMonsterLoyaltyParty {
				disposition = CombatAimFriendly
			} else {
				disposition, bypassTarget = CombatAimEnemy, true
			}
		}
	}
	switch {
	case position == origin:
		result.Status = CombatShotChooseTarget
	case !isCombatAttackPositionInRange(origin, source, position):
		result.Status = CombatShotOutOfRange
	case disposition == CombatAimFriendly:
		result.Status = CombatShotFriendly
	default:
		result.Status = g.combatAttackSourceAvailability(actor, source)
		if !result.Status.Attackable() || source.Equipment.GetAttackRange() == 0 {
			return result
		}
		if bypassTarget && source.Equipment.BypassesTerrainOnSuccessfulHit() {
			result.Status = CombatShotTerrainBypass
		} else if launch := traceCombatProjectileLaunch(g.combatMissileTerrainAt, origin, position, source.Missile); launch.Blocked {
			result.Status = CombatShotBlocked
		}
	}

	return result
}
