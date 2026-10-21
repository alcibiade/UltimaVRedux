package game_state

import (
	"github.com/bradhannah/UltimaVRedux/internal/party_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
)

type pendingCombatAttack struct {
	attacker        CombatTurnIdentity
	source          CombatAttackSource
	launch          CombatProjectileLaunch
	primaryTarget   CombatTurnIdentity
	primarySet      bool
	impactTarget    CombatTurnIdentity
	impactTargetSet bool
	diagnostics     CombatDiagnostics
}

// SkipUnavailableCombatAttackSources rejects unusable ranged weapons before
// Aim begins while preserving any later attack sources.
func (g *GameState) SkipUnavailableCombatAttackSources() CombatAttackTargetResult {
	if g.CombatScene == nil || g.CombatScene.AttackTargeting == nil {
		return combatAttackResult(CombatAttackTargetIgnored)
	}
	targeting := g.CombatScene.AttackTargeting
	skipped := false
	for targeting.Attacker.IsPartyMember {
		source := targeting.CurrentSource()
		status := g.combatAttackSourceAvailability(targeting.Attacker, source)
		if status == CombatShotClear {
			break
		}
		switch status { //nolint:exhaustive // Only availability statuses can originate here.
		case CombatShotNoAmmo:
			g.SystemCallbacks.Message.AddRowStr("No " + source.Equipment.GetRequiredAmmo().String() + "!")
		case CombatShotInterference:
			engager, _ := g.validRememberedMeleeEngager(targeting.Attacker)
			g.publishCombatInterference(targeting.Attacker, engager, CombatInterruptedAttack, source.Equipment)
		default:
			g.SystemCallbacks.Message.AddRowStr("Not readied!")
		}
		skipped = true
		if g.advanceCombatAttackSource() {
			return combatAttackResult(CombatAttackSequenceComplete)
		}
	}
	if skipped {
		return combatAttackResult(CombatAttackSourceResolved)
	}

	return combatAttackResult(CombatAttackTargetIgnored)
}

func (g *GameState) preparePendingCombatAttack(
	targeting *CombatAttackTargeting,
	source CombatAttackSource,
	primaryTarget CombatTurnIdentity,
	primarySet bool,
) CombatAttackTargetResult {
	if g.CombatScene.pendingAttack != nil {
		panic("combat attack committed while another physical attack is pending")
	}
	if targeting.Attacker.IsPartyMember && source.Equipment.IsSubjectToCombatInterference() {
		if engager, found := g.validRememberedMeleeEngager(targeting.Attacker); found {
			g.publishCombatInterference(targeting.Attacker, engager, CombatInterruptedAttack, source.Equipment)

			return g.completeCombatAttackSource(CombatAttackSourceResolved)
		}
	}
	if targeting.Attacker.IsPartyMember &&
		!g.commitPlayerRangedResource(targeting.Attacker, source, primarySet, targeting.Cursor) {
		return g.completeCombatAttackSource(CombatAttackSourceResolved)
	}
	// REDUX DEVIATION: player projectile launch uses the same glide for all launches.
	// Reach weapons use physical swing foley rather than a flying-projectile cue.
	// See ref:COMBAT-0004.
	if source.Missile == references.MissileNone ||
		source.Equipment == references.MorningStar || source.Equipment == references.Halberd {
		g.SystemCallbacks.Audio.PlaySoundEffect(SoundCombatSwing)
	} else {
		g.SystemCallbacks.Audio.PlaySoundEffect(SoundCombatRangedLaunch)
	}

	diagnostics := CombatDiagnostics{
		Source:  source.Equipment,
		Missile: source.Missile,
	}
	if !targeting.Attacker.IsPartyMember {
		attacker := g.CombatScene.MonsterManager.GetMonster(targeting.AttackerIndex)
		diagnostics.UsedStrength = attacker.HasAbility(references.Bludgeons)
	}
	aim := targeting.Cursor
	hit := CombatHitResult{}
	if primarySet {
		hit = g.commitPhysicalRangedHit(targeting.Attacker, source, primaryTarget)
		diagnostics.Hit = hit
		defense := g.combatTargetDefense(primaryTarget)
		diagnostics.TargetBaseDex = defense.BaseDex
		if hit.Outcome == CombatHitMiss {
			aim = randomAdjacentCombatAttackMissAimWithRNG(
				targeting.Cursor, targeting.Origin, source.MaxRange, g.AsRandomProvider(),
			)
		}
	}

	launch := traceCombatProjectileLaunch(g.combatMissileTerrainAt, targeting.Origin, aim, source.Missile)
	if primarySet && hit.Outcome != CombatHitMiss && source.Equipment.BypassesTerrainOnSuccessfulHit() {
		launch = CombatProjectileLaunch{
			From: targeting.Origin, Aim: targeting.Cursor, Impact: targeting.Cursor,
			Missile: source.Missile, Animate: false,
		}
	}
	// Projectile launch omits projectile travel for both reach weapons even on a miss.
	// Keep tracing/scatter and successful-hit bypass independent of visual admission.
	// See ref:COMBAT-0004.
	if source.Equipment == references.MorningStar || source.Equipment == references.Halberd {
		launch.Animate = false
	}
	diagnostics.ProjectileBlocked = launch.Blocked

	pending := &pendingCombatAttack{
		attacker: targeting.Attacker, source: source, launch: launch,
		primaryTarget: primaryTarget, primarySet: primarySet, diagnostics: diagnostics,
	}
	if !launch.Blocked {
		identity, found := g.combatMissileImpactOccupantAt(launch.Impact)
		if found && identity != targeting.Attacker {
			pending.impactTarget = identity
			pending.impactTargetSet = true
		}
	}
	g.CombatScene.pendingAttack = pending

	result := combatAttackResult(CombatAttackSourcePending)
	result.Launch = &launch
	if source.Equipment.HasAbility(references.AbilityReturns) {
		returnLaunch := CombatProjectileLaunch{
			From: launch.Impact, Aim: targeting.Origin, Impact: targeting.Origin,
			Missile: source.Missile, Animate: true,
		}
		result.ReturnLaunch = &returnLaunch
	}

	return result
}

// ResolvePendingCombatAttack applies one committed physical impact and advances its source.
func (g *GameState) ResolvePendingCombatAttack() CombatAttackTargetResult {
	if g.CombatScene == nil || g.CombatScene.pendingAttack == nil {
		return combatAttackResult(CombatAttackTargetIgnored)
	}

	pending := *g.CombatScene.pendingAttack
	g.CombatScene.pendingAttack = nil
	if pending.primarySet {
		defer g.finishDirectedCombatAttack(pending.primaryTarget, g.CombatScene.isCombatPCSleeping(pending.primaryTarget))
	}
	if pending.impactTargetSet && !pending.launch.Blocked {
		defer g.finishDirectedCombatAttack(pending.impactTarget, g.CombatScene.isCombatPCSleeping(pending.impactTarget))
	}
	if pending.launch.Blocked {
		g.publishBlockedPhysicalAttack(pending)
	} else {
		if physicalCombatAttackTriggersDungeonMap(pending.source) {
			g.CheckTrap(&pending.launch.Impact)
		}
		if pending.source.Equipment == references.FlamingOil {
			g.createFlamingOilField(pending)
		}
		switch {
		case pending.impactTargetSet:
			g.resolvePendingPhysicalImpact(pending)
		case pending.primarySet:
			g.publishPendingPhysicalMiss(pending)
		default:
			g.publishEmptyPhysicalAttack(pending)
		}
	}

	result := g.completeCombatAttackSource(CombatAttackSourceResolved)
	result.ShowImpact = !pending.launch.Blocked && pending.impactTargetSet

	return result
}

func physicalCombatAttackTriggersDungeonMap(source CombatAttackSource) bool {
	return source.Equipment != references.MorningStar && source.Equipment != references.Halberd
}

func (g *GameState) commitPhysicalRangedHit(
	attacker CombatTurnIdentity,
	source CombatAttackSource,
	target CombatTurnIdentity,
) CombatHitResult {
	defense := g.combatTargetDefense(target)
	attackerStat := g.combatPhysicalAttackAttribute(attacker, source.Equipment)
	if !attacker.IsPartyMember {
		return CombatHitCheck(attackerStat, defense.EffectiveDex, false, g.AsRandomProvider())
	}

	neverMiss := source.Equipment.HasAbility(references.AbilityShatters) ||
		source.Equipment.HasAbility(references.AbilityNeverMisses) ||
		source.Equipment.HasAbility(references.AbilityCharmsWielder)

	return CombatHitCheck(attackerStat, defense.EffectiveDex, neverMiss, g.AsRandomProvider())
}

func randomAdjacentCombatAttackMissAimWithRNG(
	aim references.Position,
	attacker references.Position,
	maxRange int,
	rng references.RandomProvider,
) references.Position {
	// REDUX DEVIATION: projectile launch (91-96) scatters without rechecking
	// range. Keep physical misses within normal attack range so neither their
	// presentation nor collateral damage can extend the attack's reach.
	// See ref:COMBAT-0004.
	for {
		candidate := randomAdjacentCombatMissAimWithRNG(aim, attacker, rng)
		if integerDistance(attacker, candidate) <= maxRange {
			return candidate
		}
	}
}

func randomAdjacentCombatMissAimWithRNG(
	aim references.Position,
	attacker references.Position,
	rng references.RandomProvider,
) references.Position {
	for {
		candidate := references.Position{
			X: aim.X + references.Coordinate(rng.Intn(3)-1),
			Y: aim.Y + references.Coordinate(rng.Intn(3)-1),
		}
		if candidate == aim || candidate == attacker || candidate.X < 0 ||
			candidate.X >= references.XCombatMapTiles || candidate.Y < 0 ||
			candidate.Y >= references.YCombatMapTiles {
			continue
		}

		return candidate
	}
}

func (g *GameState) commitPlayerRangedResource(
	attackerIdentity CombatTurnIdentity,
	source CombatAttackSource,
	primarySet bool,
	aim references.Position,
) bool {
	attackerIndex := attackerIdentity.EntityIndex
	character := g.PartyState.GetCharacter(attackerIndex)
	inventory := &g.PartyState.Inventory.Equipment
	if source.Equipment.IsConsumedOnRangedAttack() {
		g.consumeReadiedOrSpareEquipment(character, source, false)

		return true
	}
	if !primarySet {
		return true
	}
	if ammo := source.Equipment.GetRequiredAmmo(); ammo != references.NoEquipment {
		if !inventory.DecrementByOne(ammo) {
			g.SystemCallbacks.Message.AddRowStr("No " + ammo.String() + "!")

			return false
		}
		if inventory.Get(ammo) == 0 {
			g.unreadyAllCombatWeapons(source.Equipment)
		}

		return true
	}
	if source.Equipment.IsRecoverableThrownWeapon() &&
		integerDistance(g.CombatScene.GetCombatantPosition(attackerIdentity), aim) > 1 {
		g.consumeReadiedOrSpareEquipment(character, source, true)
	}

	return true
}

func (g *GameState) consumeReadiedOrSpareEquipment(
	character *party_state.PlayerCharacter,
	source CombatAttackSource,
	useSpare bool,
) {
	if g.PartyState.Inventory.Equipment.HasSome(source.Equipment) {
		g.PartyState.Inventory.Equipment.DecrementByOne(source.Equipment)

		return
	}
	if useSpare || character.GetEquippedItem(source.Slot) == source.Equipment {
		character.SetEquippedItem(source.Slot, references.NoEquipment)
	}
}

func (g *GameState) unreadyAllCombatWeapons(weapon references.Equipment) {
	count := uint16(0)
	g.PartyState.ForEachCharacter(func(_ int, character *party_state.PlayerCharacter) {
		if !character.IsInParty() {
			return
		}
		for _, slot := range [...]references.EquipmentSlot{references.SlotWeapon, references.SlotShield} {
			if character.GetEquippedItem(slot) == weapon {
				character.SetEquippedItem(slot, references.NoEquipment)
				count++
			}
		}
	})
	// REDUX DEVIATION: EQUIP-0015: the original last-ammunition return has
	// no local stack cap. Redux discards overflow above 99, retaining bounded
	// inventory rather than spawning recoverable ground equipment.
	g.PartyState.Inventory.Equipment.IncrementBy(weapon, count)
}

func (g *GameState) resolvePendingPhysicalImpact(pending pendingCombatAttack) {
	if pending.attacker.IsPartyMember {
		if pending.impactTarget.IsPartyMember {
			g.resolveCommittedPCAttackAgainstPC(pending, CombatEventRangedAttack)
		} else {
			g.resolveCommittedPCAttackAgainstMonster(pending, CombatEventRangedAttack)
		}

		return
	}
	if pending.impactTarget.IsPartyMember {
		g.resolveCommittedControlledAttackAgainstPC(pending, CombatEventRangedAttack)
	} else {
		g.resolveCommittedControlledAttackAgainstMonster(pending, CombatEventRangedAttack)
	}
}

func (g *GameState) resolveCommittedPCAttackAgainstMonster(
	pending pendingCombatAttack,
	eventKind CombatEventKind,
) {
	monster := g.CombatScene.MonsterManager.GetMonster(pending.impactTarget.EntityIndex)
	if monster == nil || !monster.IsActive() || monster.IsDead() {
		return
	}
	diagnostics := pending.diagnostics
	diagnostics.TargetBaseDex = monster.EnemyRef().Dexterity
	g.resolvePlayerWeaponHitAgainstMonster(
		g.PartyState.GetCharacter(pending.attacker.EntityIndex),
		pending.source.Equipment,
		pending.source.Slot,
		monster,
		pending.impactTarget.EntityIndex,
		pending.attacker.EntityIndex,
		diagnostics,
		eventKind,
	)
}

func (g *GameState) resolveCommittedPCAttackAgainstPC(
	pending pendingCombatAttack,
	eventKind CombatEventKind,
) {
	targetIndex := pending.impactTarget.EntityIndex
	diagnostics := pending.diagnostics
	diagnostics.TargetBaseDex = int(g.PartyState.GetCharacter(targetIndex).Dexterity)
	result, hpLost, fatal, diagnostics := g.applyPlayerWeaponHitToPC(
		pending.attacker.EntityIndex, pending.source, targetIndex, diagnostics,
	)
	g.publishPartyLoyalAttackResult(
		eventKind,
		combatFeedbackIdentity(pending.attacker, pending.launch.From),
		combatFeedbackIdentity(pending.impactTarget, pending.launch.Impact),
		result, hpLost, fatal, diagnostics,
	)
}

func (g *GameState) resolveCommittedControlledAttackAgainstMonster(
	pending pendingCombatAttack,
	eventKind CombatEventKind,
) {
	target := g.CombatScene.MonsterManager.GetMonster(pending.impactTarget.EntityIndex)
	if target == nil || !target.IsActive() || target.IsDead() {
		return
	}
	diagnostics := pending.diagnostics
	diagnostics.TargetBaseDex = target.EnemyRef().Dexterity
	g.resolveControlledMonsterHitAgainstMonster(
		pending.attacker.EntityIndex, target, pending.impactTarget.EntityIndex, diagnostics, eventKind,
	)
}

func (g *GameState) resolveCommittedControlledAttackAgainstPC(
	pending pendingCombatAttack,
	eventKind CombatEventKind,
) {
	attacker := g.CombatScene.MonsterManager.GetMonster(pending.attacker.EntityIndex)
	targetIndex := pending.impactTarget.EntityIndex
	diagnostics := pending.diagnostics
	diagnostics.TargetBaseDex = int(g.PartyState.GetCharacter(targetIndex).Dexterity)
	result, hpLost, fatal, diagnostics := g.applyControlledMonsterHitToPC(
		attacker, pending.impactTarget, false, diagnostics,
	)
	g.publishPartyLoyalAttackResult(
		eventKind,
		combatFeedbackIdentity(pending.attacker, pending.launch.From),
		combatFeedbackIdentity(pending.impactTarget, pending.launch.Impact),
		result, hpLost, fatal, diagnostics,
	)
}

func (g *GameState) publishBlockedPhysicalAttack(pending pendingCombatAttack) {
	diagnostics := pending.diagnostics
	diagnostics.ProjectileBlocked = true
	event := CombatEvent{
		Kind:     CombatEventRangedAttack,
		Attacker: combatFeedbackIdentity(pending.attacker, pending.launch.From),
		Result:   CombatFeedbackMiss, From: pending.launch.From, To: pending.launch.Impact,
		Diagnostics: diagnostics,
	}
	if pending.primarySet {
		event.Target = combatFeedbackIdentity(pending.primaryTarget, g.combatTurnPosition(pending.primaryTarget))
	}
	g.SystemCallbacks.Combat.PublishEvent(event)
}

func (g *GameState) publishPendingPhysicalMiss(pending pendingCombatAttack) {
	target := combatFeedbackIdentity(pending.primaryTarget, g.combatTurnPosition(pending.primaryTarget))
	attacker := combatFeedbackIdentity(pending.attacker, pending.launch.From)
	g.SystemCallbacks.Visual.ShowCombatImpact(CombatAttackFeedback{
		Attacker: attacker, Target: target, Result: CombatFeedbackMiss,
		Presentation: combatAttackPresentationForEvent(
			CombatEventRangedAttack,
			pending.diagnostics,
			pending.attacker.IsPartyMember,
		),
	})
	g.SystemCallbacks.Combat.PublishEvent(CombatEvent{
		Kind: CombatEventRangedAttack, Attacker: attacker, Target: target,
		Result: CombatFeedbackMiss, From: pending.launch.From, To: pending.launch.Impact,
		Diagnostics: pending.diagnostics,
	})
}

func (g *GameState) publishEmptyPhysicalAttack(pending pendingCombatAttack) {
	g.SystemCallbacks.Combat.PublishEvent(CombatEvent{
		Kind:        CombatEventRangedAttack,
		Attacker:    combatFeedbackIdentity(pending.attacker, pending.launch.From),
		Result:      CombatFeedbackMiss,
		From:        pending.launch.From,
		To:          pending.launch.Impact,
		Diagnostics: pending.diagnostics,
	})
}

func (g *GameState) createFlamingOilField(pending pendingCombatAttack) {
	field := NewCombatField(CombatFieldFire, pending.launch.Impact)
	field.createdThisAction = true
	if !g.CombatScene.AddField(field) {
		g.SystemCallbacks.Message.AddRowStr("No room for field!")

		return
	}
	g.SystemCallbacks.Combat.PublishEvent(CombatEvent{
		Kind:     CombatEventFieldCreated,
		Attacker: combatFeedbackIdentity(pending.attacker, pending.launch.From),
		To:       pending.launch.Impact, FieldKind: CombatFieldFire,
		Diagnostics: CombatDiagnostics{Source: references.FlamingOil},
	})
}

func (g *GameState) combatIdentityName(identity CombatTurnIdentity) string {
	if identity.IsPartyMember {
		return g.PartyState.GetCharacter(identity.EntityIndex).GetNameAsString()
	}

	return g.CombatScene.MonsterManager.GetMonster(identity.EntityIndex).MonsterType().Name()
}
