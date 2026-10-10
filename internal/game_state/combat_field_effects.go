package game_state

import (
	"github.com/bradhannah/UltimaVRedux/internal/party_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
)

func combatFieldKindForSpell(spell references.Spell) CombatFieldKind {
	//nolint:exhaustive // Only field-creating spells are valid callers.
	switch spell {
	case references.InNoxGrav:
		return CombatFieldPoison
	case references.InZuGrav:
		return CombatFieldSleep
	case references.InFlamGrav:
		return CombatFieldFire
	case references.InSanctGrav:
		return CombatFieldProtection
	default:
		panic("spell does not create a combat field")
	}
}

func (g *GameState) resolveCombatFieldSpell(pending pendingCombatSpell) {
	field := NewCombatField(combatFieldKindForSpell(pending.spell), pending.launch.Impact)
	field.createdThisAction = true
	installed := g.CombatScene.AddField(field)
	if !installed {
		g.SystemCallbacks.Message.AddRowStr("No room for field!")
	} else {
		g.SystemCallbacks.Combat.PublishEvent(CombatEvent{ //nolint:exhaustruct // Field creation has no attack result.
			Kind:        CombatEventFieldCreated,
			Attacker:    combatFeedbackIdentity(pending.caster, pending.launch.From),
			To:          pending.launch.Impact,
			FieldKind:   field.Kind(),
			Diagnostics: CombatDiagnostics{Spell: pending.spell},
		})
	}
	g.resolveCombatFieldAtPosition(field, pending.launch.Impact, pending.casterIdx, true)
}

func (g *GameState) resolveCombatFieldAtPosition(field CombatField, position references.Position, casterIdx int, direct bool) {
	if field.Kind() == CombatFieldProtection {
		return
	}
	if direct {
		identity, found := g.combatMissileImpactOccupantAt(position)
		if !found {
			return
		}
		if identity.IsPartyMember {
			g.applyCombatFieldToPC(field, identity, true)

			return
		}
		monster := g.CombatScene.MonsterManager.GetMonster(identity.EntityIndex)
		g.applyCombatFieldToMonster(field, monster, identity.EntityIndex, casterIdx, true)

		return
	}
	if monster, monsterIdx := g.CombatScene.MonsterManager.GetMonsterIndexAtPosition(position); monster != nil {
		g.applyCombatFieldToMonster(field, monster, monsterIdx, casterIdx, false)

		return
	}
	if identity, found := g.CombatScene.combatPCAtPosition(position); found {
		g.applyCombatFieldToPC(field, identity, false)
	}
}

func (g *GameState) applyCombatFieldToMonster(field CombatField, monster *CombatMonster, monsterIdx, casterIdx int, direct bool) {
	if field.Kind() == CombatFieldPoison && !direct &&
		monster.MonsterType() >= references.MonsterSeaHorse {
		return
	}
	hpBefore := monster.CurrentHP()
	switch field.Kind() {
	case CombatFieldSleep:
		monster.SetAsleep(true)
	case CombatFieldPoison:
		damage := g.RandomIntInRange(0, 20)
		if direct {
			finalDamage := CombatApplyMagicDamageModifiers(
				damage, monster.MonsterType().IsUndead(), monster.HasAbility(references.Immortal),
			)
			g.applyCombatDamageToMonster(
				monster, monsterIdx, finalDamage, false, g.PartyState.GetCharacter(casterIdx), casterIdx,
				CombatDiagnostics{
					FinalDamage:  finalDamage,
					TargetUndead: monster.MonsterType().IsUndead(), TargetImmortal: monster.HasAbility(references.Immortal),
					MagicDamage: true,
				},
			)
		} else {
			g.applyCombatFieldDamageToMonster(monster, monsterIdx, damage)
		}
	case CombatFieldFire:
		if direct {
			damage := CombatCalcMagicDamage(21, monster.EnemyRef().Armour, g.AsRandomProvider())
			finalDamage := CombatApplyMagicDamageModifiers(
				damage.Damage, monster.MonsterType().IsUndead(), monster.HasAbility(references.Immortal),
			)
			g.applyCombatDamageToMonster(
				monster, monsterIdx, finalDamage, false, g.PartyState.GetCharacter(casterIdx), casterIdx,
				CombatDiagnostics{
					Damage: damage, FinalDamage: finalDamage,
					TargetUndead: monster.MonsterType().IsUndead(), TargetImmortal: monster.HasAbility(references.Immortal),
					MagicDamage: true,
				},
			)
		} else {
			g.applyCombatFieldDamageToMonster(monster, monsterIdx, g.RandomIntInRange(0, 10))
		}
	case CombatFieldProtection:
		panic("protection fields do not trigger")
	}
	g.publishCombatFieldTrigger(
		field,
		false,
		monsterIdx,
		0,
		monster.Position(),
		max(hpBefore-monster.CurrentHP(), 0),
		monster.IsDead(),
	)
}

func (g *GameState) applyCombatFieldDamageToMonster(monster *CombatMonster, monsterIdx, damage int) {
	hpBefore := monster.CurrentHP()
	damage = monster.ApplyDamageModifiers(damage, references.NoEquipment)
	monster.TakeDamage(damage)
	if hpBefore > 0 && monster.IsDead() {
		g.finalizeCombatMonsterDeathWithPolicy(monster, monsterIdx, nil, NoTargetPC, false, true)

		return
	}
	g.tryDivideCombatMonster(monster, monsterIdx)
}

func (g *GameState) applyCombatFieldToPC(field CombatField, identity CombatTurnIdentity, direct bool) {
	if direct && field.Kind() != CombatFieldSleep {
		defer g.finishDirectedCombatAttack(identity, g.CombatScene.isCombatPCSleeping(identity))
	}
	pcIdx := identity.EntityIndex
	character := g.PartyState.GetCharacter(pcIdx)
	position := g.CombatScene.GetCombatantPosition(identity)
	hpBefore := character.GetCurrentHp()
	statusBefore := character.GetStatus()
	switch field.Kind() {
	case CombatFieldSleep:
		g.setCombatPCSleeping(identity, true)
	case CombatFieldPoison:
		if character.IsGood() {
			character.SetStatus(party_state.Poisoned)
			g.SystemCallbacks.Message.AddRowStr(character.GetNameAsString() + " is poisoned!")
		} else {
			character.SubtractHp(uint16(g.RandomIntInRange(0, 20)))
		}
	case CombatFieldFire:
		if direct {
			damage := CombatCalcMagicDamage(21, g.combatPartyDefense(character), g.AsRandomProvider())
			character.SubtractHp(uint16(damage.Damage))
		} else {
			character.SubtractHp(uint16(g.RandomIntInRange(0, 10)))
		}
	case CombatFieldProtection:
		panic("protection fields do not trigger")
	}
	changes := transientStatusChanges{}
	changes.observePoison(statusBefore, character)
	g.publishTransientStatusChanges(changes)
	if character.GetCurrentHp() == 0 {
		character.MarkAsDead()
		g.markCombatPCDead(pcIdx)
	}
	g.publishCombatFieldTrigger(
		field,
		true,
		pcIdx,
		identity.Slot,
		position,
		int(hpBefore-character.GetCurrentHp()),
		character.GetCurrentHp() == 0,
	)
}

func (g *GameState) publishCombatFieldTrigger(
	field CombatField,
	partyMember bool,
	index, slot int,
	position references.Position,
	hpLost int,
	fatal bool,
) {
	g.SystemCallbacks.Combat.PublishEvent(CombatEvent{ //nolint:exhaustruct // Field trigger has no physical attack diagnostics.
		Kind: CombatEventFieldTriggered,
		Target: CombatFeedbackEntity{
			IsPartyMember: partyMember, Index: index, Slot: slot, Position: position,
		},
		HPLost:    hpLost,
		Fatal:     fatal,
		To:        position,
		FieldKind: field.Kind(),
	})
}

func (g *GameState) resolveCombatFieldExpiration() {
	for _, field := range g.CombatScene.expireFields(g.GetRNG().Intn) {
		g.SystemCallbacks.Combat.PublishEvent(CombatEvent{ //nolint:exhaustruct // Field expiration has no actor or roll diagnostics.
			Kind:         CombatEventFieldExpired,
			To:           field.Position(),
			FieldKind:    field.Kind(),
			FieldFizzled: field.createdThisAction,
		})
	}
}

func (g *GameState) resolveCombatPCStandingEffects(identity CombatTurnIdentity) {
	if !g.CombatScene.isCombatPCActive(identity) {
		return
	}
	character := g.PartyState.GetCharacter(identity.EntityIndex)
	if character.IsDead() {
		return
	}
	position := g.CombatScene.GetCombatantPosition(identity)
	if field, found := combatTerrainEffectAt(g.CombatScene.CombatMapRef.GetTileNumber(position), position); found {
		g.applyCombatFieldToPC(field, identity, false)

		return
	}
	if field, found := g.CombatScene.firstHarmfulFieldAt(position); found {
		g.applyCombatFieldToPC(field, identity, false)
	}
}

func combatTerrainEffectAt(tile indexes.SpriteIndex, position references.Position) (CombatField, bool) {
	//nolint:exhaustive // Only terrain with an original infield effect is relevant.
	switch tile {
	case indexes.Lava, indexes.Fireplace:
		return NewCombatField(CombatFieldFire, position), true
	case indexes.Swamp:
		return NewCombatField(CombatFieldPoison, position), true
	default:
		return CombatField{}, false
	}
}

func (g *GameState) resolveCombatMonsterActionEffects(monster *CombatMonster, monsterIndex int) {
	g.resolveCombatControlledEndAction(CombatTurnIdentity{EntityIndex: monsterIndex})
	g.resolveCombatMonsterFieldEffects(monster, monsterIndex)
}

func (g *GameState) resolveCombatMonsterFieldEffects(monster *CombatMonster, monsterIndex int) {
	if monster == nil || monster.IsDead() || !monster.IsActive() {
		return
	}
	position := monster.Position()
	if field, found := combatTerrainEffectAt(g.CombatScene.CombatMapRef.GetTileNumber(position), position); found {
		g.applyCombatFieldToMonster(field, monster, monsterIndex, NoTargetPC, false)

		return
	}
	if field, found := g.CombatScene.firstHarmfulFieldAt(position); found {
		g.applyCombatFieldToMonster(field, monster, monsterIndex, NoTargetPC, false)
	}
}

func (g *GameState) dispelCombatFieldFrom(origin references.Position, direction references.Direction) (
	CombatField,
	references.Position,
	bool,
) {
	target := direction.GetNewPositionInDirection(&origin)
	if target == nil {
		return CombatField{}, references.Position{}, false
	}
	field, removed := g.CombatScene.removeFirstFieldAt(*target)
	if !removed {
		return CombatField{}, *target, false
	}

	return field, *target, true
}

// CastCombatAnGrav removes the first encounter-local field in the selected direction.
func (g *GameState) CastCombatAnGrav(casterIdx int, direction references.Direction) bool {
	identity := g.combatPartyAttackerIdentity(casterIdx)
	origin := g.CombatScene.GetCombatantPosition(identity)
	field, target, removed := g.dispelCombatFieldFrom(origin, direction)
	if !removed {
		return false
	}
	g.SystemCallbacks.Combat.PublishEvent(CombatEvent{ //nolint:exhaustruct // Dispel has no damage diagnostics.
		Kind:        CombatEventFieldDispelled,
		Attacker:    combatFeedbackIdentity(identity, origin),
		To:          target,
		FieldKind:   field.Kind(),
		Diagnostics: CombatDiagnostics{Spell: references.AnGrav},
	})

	return true
}

// UseCombatSceptre removes the first encounter-local field beside the active party member.
func (g *GameState) UseCombatSceptre(direction references.Direction) bool {
	if direction != references.Up && direction != references.Down && direction != references.Left && direction != references.Right {
		return false
	}
	g.CommitCastUseAttempt()
	identity := g.CombatScene.GetCurrentTurnIdentity()
	if !identity.IsPartyMember {
		panic("UseCombatSceptre requires an active party member")
	}
	origin := g.CombatScene.GetCombatantPosition(identity)
	field, target, removed := g.dispelCombatFieldFrom(origin, direction)
	if !removed {
		g.SystemCallbacks.Message.AddRowStr("No effect!")

		return false
	}
	g.SystemCallbacks.Combat.PublishEvent(CombatEvent{ //nolint:exhaustruct // Dispel has no damage diagnostics.
		Kind:      CombatEventFieldDispelled,
		Attacker:  combatFeedbackIdentity(identity, origin),
		To:        target,
		FieldKind: field.Kind(),
	})
	g.SystemCallbacks.Message.AddRowStr("Field dissolved!")

	return true
}

// CastCombatInZu applies the combat-only directional mass-sleep cone.
func (g *GameState) CastCombatInZu(casterIdx int, direction references.Direction) []references.Position {
	return g.castCommittedCombatCone(references.InZu, casterIdx, direction)
}

func (g *GameState) resolveCombatInZu(
	casterIdx int,
	candidates []references.Position,
) []references.Position {
	caster := g.PartyState.GetCharacter(casterIdx)
	affected := make([]references.Position, 0)
	processed := make(map[CombatTurnIdentity]struct{})
	for _, position := range candidates {
		target, found := g.CombatScene.combatConeVictimAt(position, processed)
		if found && g.applyCombatInZuToTarget(target, caster.Intelligence) {
			affected = append(affected, position)
		}
	}

	return affected
}

func (g *GameState) applyCombatInZuToTarget(target CombatTurnIdentity, casterIntelligence uint8) bool {
	position := g.CombatScene.GetCombatantPosition(target)
	if !target.IsPartyMember {
		monsterIdx := target.EntityIndex
		monster := g.CombatScene.MonsterManager.GetMonster(monsterIdx)
		// Original spell projection rules evaluates Intelligence saving throw before immunity, including for immune targets.
		// See ref:COMBAT-0007.
		if g.RollOriginalD30() < (30+monster.EnemyRef().Intelligence-int(casterIntelligence))/2 {
			return false
		}
		if isCombatSpellImmuneMonster(monster) {
			return false
		}
		monster.SetAsleep(true)
		g.publishCombatFieldTrigger(
			NewCombatField(CombatFieldSleep, position), false, monsterIdx, 0, position, 0, false,
		)

		return true
	}
	if target.IsPartyMember {
		character := g.PartyState.GetCharacter(target.EntityIndex)
		if g.RollOriginalD30() < (30+int(character.Intelligence)-int(casterIntelligence))/2 {
			return false
		}
		g.setCombatPCSleeping(target, true)
		g.publishCombatFieldTrigger(
			NewCombatField(CombatFieldSleep, position), true, target.EntityIndex, target.Slot, position, 0, false,
		)

		return true
	}

	return false
}

func isCombatSpellImmuneMonster(monster *CombatMonster) bool {
	//nolint:exhaustive // Only original spell-immune monsters need explicit handling.
	switch monster.MonsterType() {
	case references.MonsterShadowlord, references.MonsterBlackthorn, references.MonsterLordBritish:
		return true
	default:
		return false
	}
}
