package game_state

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/party_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
)

func TestRemainingDurationSpellsUseOriginalDurations(t *testing.T) {
	gs, _, _ := setupCombatWithMonster(t, references.MonsterOrc, references.Down, 42)
	if gs == nil {
		return
	}

	assert.True(t, gs.CastInSanct())
	assert.Equal(t, inSanctDurationTurns, gs.ActiveDurationEffect.TurnsRemaining)
	gs.ClearDurationEffect()
	assert.True(t, gs.CastCombatQuasAnWis())
	assert.Equal(t, quasAnWisDurationTurns, gs.ActiveDurationEffect.TurnsRemaining)
	gs.ClearDurationEffect()
	assert.True(t, gs.CastCombatInAn())
	assert.Equal(t, inAnDurationTurns, gs.ActiveDurationEffect.TurnsRemaining)
}

func TestInAnActivatesOutsideCombat(t *testing.T) {
	gs, _, _ := setupCombatWithMonster(t, references.MonsterOrc, references.Down, 42)
	if gs == nil {
		return
	}
	gs.TransitionFromCombatMap()
	require.NotEqual(t, references.ContextCombat, gs.GetCurrentSpellContext())
	assert.True(t, gs.CastCombatInAn())
	assert.True(t, gs.HasActiveDurationEffect(DurationEffectNegateMagic))
}

func TestCombatInSanctAddsThreeDefense(t *testing.T) {
	gs, _, _ := setupCombatWithMonster(t, references.MonsterOrc, references.Down, 42)
	if gs == nil {
		return
	}
	character := gs.PartyState.GetCharacter(0)
	before := gs.combatPartyDefense(character)
	require.True(t, gs.CastInSanct())
	assert.Equal(t, before+combatInSanctDefenseBonus, gs.combatPartyDefense(character))
}

func TestCombatAnTymRejectsShadowlordAndLeavesDurationUnchanged(t *testing.T) {
	gs, _, _ := setupCombatWithMonster(t, references.MonsterShadowlord, references.Down, 42)
	if gs == nil {
		return
	}

	assert.False(t, gs.CastAnTym())
	assert.False(t, gs.HasActiveDurationEffect(DurationEffectTimeStop))
}

func TestCombatAnXenExFreesPossessedPartyMember(t *testing.T) {
	gs, _, _ := setupCombatWithMonster(t, references.MonsterOrc, references.Down, 42)
	if gs == nil {
		return
	}
	target := 0
	gs.CombatScene.PartyPossessed[target] = true
	gs.PartyState.GetCharacter(target).SetStatus(party_state.Charmed)

	assert.True(t, gs.CastCombatAnXenEx(gs.CombatScene.GetActiveCombatantIndex(), CombatTurnIdentity{IsPartyMember: true, EntityIndex: target}))
	assert.False(t, gs.CombatScene.PartyPossessed[target])
	assert.Equal(t, party_state.Good, gs.PartyState.GetCharacter(target).GetStatus())
}

func TestCombatInQuasCorpIncludesPartyLoyalMonster(t *testing.T) {
	gs, _, monster := setupCombatWithMonster(t, references.MonsterOrc, references.Down, 42)
	if gs == nil {
		return
	}
	monster.SetLoyalty(CombatMonsterLoyaltyParty)
	gs.PartyState.GetCharacter(gs.CombatScene.GetActiveCombatantIndex()).Intelligence = 255

	result := gs.CastCombatInQuasCorp(gs.CombatScene.GetActiveCombatantIndex())

	assert.Len(t, result.Affected, 1)
	assert.True(t, monster.IsFleeing())
	assert.Equal(t, 1, monster.CurrentHP())
}

func TestCombatPCCloneUsesIndependentQueueSlotAndSharedCharacterState(t *testing.T) {
	gs, _, _ := setupCombatWithMonster(t, references.MonsterOrc, references.Down, 42)
	if gs == nil {
		return
	}
	partyIndex := gs.CombatScene.GetActiveCombatantIndex()
	original := CombatTurnIdentity{IsPartyMember: true, EntityIndex: partyIndex}
	clone := gs.CombatScene.addPCCloneFromSource(
		original,
		partyIndex, references.Position{X: 1, Y: 1}, NewCombatPCProvider(gs.PartyState.GetCharacter(partyIndex)),
	)
	require.NotZero(t, clone.Slot)
	assert.NotEqual(t, gs.CombatScene.GetCombatantPosition(original), gs.CombatScene.GetCombatantPosition(clone))
	assert.NotNil(t, gs.CombatScene.TurnQueue.GetEntryByIdentity(original))
	cloneEntry := gs.CombatScene.TurnQueue.GetEntryByIdentity(clone)
	require.NotNil(t, cloneEntry)
	assert.Equal(t, gs.PartyState.GetCharacter(partyIndex), cloneEntry.Combatant.(*CombatPCProvider).Character())

	gs.PartyState.GetCharacter(partyIndex).SetHp(1)
	gs.setCombatPCSleeping(CombatTurnIdentity{IsPartyMember: true, EntityIndex: partyIndex}, true)
	assert.Equal(t, uint16(1), cloneEntry.Combatant.(*CombatPCProvider).Character().GetCurrentHp())
	assert.Equal(t, party_state.Sleep, cloneEntry.Combatant.(*CombatPCProvider).Character().GetStatus())
}

func TestCombatPCCloneCopiesThenOwnsEntryConditions(t *testing.T) {
	gs, _, _ := setupCombatWithMonster(t, references.MonsterOrc, references.Down, 42)
	if gs == nil {
		return
	}
	partyIndex := gs.CombatScene.GetActiveCombatantIndex()
	source := CombatTurnIdentity{IsPartyMember: true, EntityIndex: partyIndex}
	gs.CombatScene.setCombatPCPossessed(source, true)
	gs.CombatScene.setCombatPCInvisible(source, true)
	gs.setCombatPCSleeping(source, true)
	clone := gs.CombatScene.addPCCloneFromSource(
		source,
		partyIndex,
		references.Position{X: 1, Y: 1},
		NewCombatPCProvider(gs.PartyState.GetCharacter(partyIndex)),
	)

	assert.True(t, gs.CombatScene.isCombatPCPossessed(clone))
	assert.True(t, gs.CombatScene.isCombatPCInvisible(clone))
	assert.True(t, gs.CombatScene.isCombatPCSleeping(clone))
	gs.CombatScene.setCombatPCPossessed(clone, false)
	gs.CombatScene.setCombatPCInvisible(clone, false)
	gs.setCombatPCSleeping(clone, false)
	assert.True(t, gs.CombatScene.isCombatPCPossessed(source))
	assert.True(t, gs.CombatScene.isCombatPCInvisible(source))
	assert.True(t, gs.CombatScene.isCombatPCSleeping(source))
}

func TestPCOnPCCombatUsesExactSleepingClonePosition(t *testing.T) {
	gs, mock, _ := setupCombatWithMonster(t, references.MonsterOrc, references.Down, 42)
	if gs == nil {
		return
	}
	attacker := gs.CombatScene.GetCurrentTurnIdentity()
	targetIndex := (attacker.EntityIndex + 1) % party_state.MaxActiveParty
	targetSource := CombatTurnIdentity{IsPartyMember: true, EntityIndex: targetIndex}
	clonePosition := references.Position{X: 2, Y: 2}
	clone := gs.CombatScene.addPCCloneFromSource(
		targetSource,
		targetIndex,
		clonePosition,
		NewCombatPCProvider(gs.PartyState.GetCharacter(targetIndex)),
	)
	gs.setCombatPCSleeping(clone, true)
	gs.setCombatPCSleeping(targetSource, false)
	mock.Reset()

	gs.resolveCombatPCAttackSourceAgainstPC(
		attacker, CombatAttackSource{Equipment: references.NoEquipment}, clone,
	)

	assert.Equal(t, []SoundEffect{SoundCombatSwing}, mock.SoundEffectsPlayed)
	require.Len(t, mock.CombatEvents, 2)
	event := mock.CombatEvents[0]
	assert.Equal(t, 1, event.Diagnostics.Hit.DefenderEffectiveDex)
	assert.Equal(t, clonePosition, event.Target.Position)
	assert.Equal(t, CombatConditionWoke, mock.CombatEvents[1].Condition)
}

func TestCombatPCCloneFleeRemovesOnlyExactEntry(t *testing.T) {
	gs, _, _ := setupCombatWithMonster(t, references.MonsterOrc, references.Down, 42)
	if gs == nil {
		return
	}
	partyIndex := gs.CombatScene.GetActiveCombatantIndex()
	clone := gs.CombatScene.addPCCloneFromSource(
		CombatTurnIdentity{IsPartyMember: true, EntityIndex: partyIndex},
		partyIndex,
		references.Position{X: 0, Y: 5},
		NewCombatPCProvider(gs.PartyState.GetCharacter(partyIndex)),
	)
	gs.CombatScene.CurrentTurnIdentity = clone
	gs.CombatScene.currentTurnIdentitySet = true

	assert.Equal(t, CombatMoveFled, gs.MoveCombatCharacter(references.Left))
	assert.True(t, gs.CombatScene.PartyActive[partyIndex])
	assert.Empty(t, gs.CombatScene.partyClones)
	cloneEntry := gs.CombatScene.TurnQueue.GetEntryByIdentity(clone)
	require.NotNil(t, cloneEntry)
	assert.Equal(t, turnQueueFled, cloneEntry.status)
	assert.True(t, cloneEntry.sourceSlotReleased)
}

func TestSharedPCDeathFinalizesSourceAndEveryClonePosition(t *testing.T) {
	gs, _, monster := setupCombatWithMonster(t, references.MonsterOrc, references.Down, 42)
	if gs == nil {
		return
	}
	partyIndex := gs.CombatScene.GetActiveCombatantIndex()
	source := CombatTurnIdentity{IsPartyMember: true, EntityIndex: partyIndex}
	clone := gs.CombatScene.addPCCloneFromSource(
		source,
		partyIndex,
		references.Position{X: 1, Y: 1},
		NewCombatPCProvider(gs.PartyState.GetCharacter(partyIndex)),
	)
	character := gs.PartyState.GetCharacter(partyIndex)
	character.SetHp(1)
	objectsBefore := gs.CombatScene.ObjectCount()

	gs.applyCombatDamageToPC(character, clone, 20, monster, 0, CombatDiagnostics{})

	assert.False(t, gs.CombatScene.PartyActive[partyIndex])
	assert.Empty(t, gs.CombatScene.partyClones)
	assert.Equal(t, turnQueueDead, gs.CombatScene.TurnQueue.GetEntryByIdentity(source).status)
	assert.Equal(t, turnQueueDead, gs.CombatScene.TurnQueue.GetEntryByIdentity(clone).status)
	assert.Equal(t, objectsBefore+2, gs.CombatScene.ObjectCount())
}

func TestCombatInQuasXenClonesPCIntoNewQueueSlot(t *testing.T) {
	gs, _, _ := setupCombatWithMonster(t, references.MonsterOrc, references.Down, 42)
	if gs == nil {
		return
	}
	partyIndex := gs.CombatScene.GetActiveCombatantIndex()
	entriesBefore := gs.CombatScene.TurnQueue.TotalEntryCount()
	require.True(t, gs.CastCombatInQuasXen(partyIndex, CombatTurnIdentity{IsPartyMember: true, EntityIndex: partyIndex}))
	require.Len(t, gs.CombatScene.partyClones, 1)
	assert.Equal(t, entriesBefore+1, gs.CombatScene.TurnQueue.TotalEntryCount())
	assert.NotEqual(t, gs.CombatScene.GetPartyMemberPosition(partyIndex), gs.CombatScene.partyClones[0].position)
}

func TestCombatInQuasXenMonsterCloneCopiesSourceSlotState(t *testing.T) {
	gs, _, source := setupCombatWithMonster(t, references.MonsterOrc, references.Down, 42)
	if gs == nil {
		return
	}
	sourceIdentity := CombatTurnIdentity{EntityIndex: 0}
	sourceEntry := gs.CombatScene.TurnQueue.GetEntryByIdentity(sourceIdentity)
	require.NotNil(t, sourceEntry)
	sourceEntry.turnCountdown = 13
	expectedSourceSlot, found := gs.CombatScene.TurnQueue.firstFreeSourceSlot(0)
	require.True(t, found)
	monstersBefore := gs.CombatScene.MonsterManager.Count()

	require.True(t, gs.CastCombatInQuasXen(
		gs.CombatScene.GetActiveCombatantIndex(), sourceIdentity,
	))

	clone := gs.CombatScene.MonsterManager.GetMonster(monstersBefore)
	require.NotNil(t, clone)
	assert.Equal(t, source.MonsterType(), clone.MonsterType())
	cloneEntry := gs.CombatScene.TurnQueue.GetEntryByIdentity(CombatTurnIdentity{EntityIndex: monstersBefore})
	require.NotNil(t, cloneEntry)
	assert.Equal(t, expectedSourceSlot, cloneEntry.sourceSlot)
	assert.Equal(t, sourceEntry.turnCountdown, cloneEntry.turnCountdown)
}

func TestRelXenBetDeletesPCEntryAndCreatesMonsterRegionEntry(t *testing.T) {
	gs, _, _ := setupCombatWithMonster(t, references.MonsterOrc, references.Down, 42)
	if gs == nil {
		return
	}
	partyIndex := gs.CombatScene.GetActiveCombatantIndex()
	target := CombatTurnIdentity{IsPartyMember: true, EntityIndex: partyIndex}
	entriesBefore := gs.CombatScene.TurnQueue.TotalEntryCount()
	monstersBefore := gs.CombatScene.MonsterManager.Count()

	require.True(t, gs.CastCombatRelXenBet(partyIndex, target))
	assert.False(t, gs.CombatScene.PartyActive[partyIndex])
	assert.Equal(t, monstersBefore+1, gs.CombatScene.MonsterManager.Count())
	assert.Equal(t, entriesBefore+1, gs.CombatScene.TurnQueue.TotalEntryCount())
	retired := gs.CombatScene.TurnQueue.GetEntryByIdentity(target)
	require.NotNil(t, retired)
	assert.Equal(t, turnQueueFled, retired.status)
	assert.True(t, retired.sourceSlotReleased)
	ratEntry := gs.CombatScene.TurnQueue.GetEntry(monstersBefore, false)
	require.NotNil(t, ratEntry)
	assert.GreaterOrEqual(t, ratEntry.sourceSlot, sourceMonsterSlotStart)
}

func TestPurplePotionTransformsActivePCWithoutSpellEvent(t *testing.T) {
	gs, mock, _ := setupCombatWithMonster(t, references.MonsterOrc, references.Down, 42)
	if gs == nil {
		return
	}
	partyIndex := gs.CombatScene.GetActiveCombatantIndex()
	monstersBefore := gs.CombatScene.MonsterManager.Count()
	mock.Reset()

	require.True(t, gs.TransformActivePartyMemberIntoRat(partyIndex))

	assert.True(t, gs.CombatScene.PartyActive[partyIndex])
	assert.True(t, gs.CombatScene.IsCombatPCRat(gs.CombatScene.GetCurrentTurnIdentity()))
	assert.Equal(t, monstersBefore, gs.CombatScene.MonsterManager.Count())
	assert.Empty(t, mock.CombatEvents, "a potion mutation must not be labeled as Rel Xen Bet")
}

func TestCombatPCCloneCapacityRejectsWithoutMutation(t *testing.T) {
	gs, _, _ := setupCombatWithMonster(t, references.MonsterOrc, references.Down, 42)
	if gs == nil {
		return
	}
	partyIndex := gs.CombatScene.GetActiveCombatantIndex()
	for gs.CombatScene.TurnQueue.TotalEntryCount() < 32 {
		gs.CombatScene.addPCCloneFromSource(
			CombatTurnIdentity{IsPartyMember: true, EntityIndex: partyIndex},
			partyIndex,
			references.Position{},
			NewCombatPCProvider(gs.PartyState.GetCharacter(partyIndex)),
		)
	}
	entriesBefore := gs.CombatScene.TurnQueue.TotalEntryCount()
	clonesBefore := len(gs.CombatScene.partyClones)
	assert.False(t, gs.CastCombatInQuasXen(partyIndex, CombatTurnIdentity{IsPartyMember: true, EntityIndex: partyIndex}))
	assert.Equal(t, entriesBefore, gs.CombatScene.TurnQueue.TotalEntryCount())
	assert.Len(t, gs.CombatScene.partyClones, clonesBefore)
}

func TestQuasAnWisPlanDoesNotConsumeRNG(t *testing.T) {
	withPlan, _, _ := setupCombatWithMonster(t, references.MonsterOrc, references.Down, 77)
	withoutPlan, _, _ := setupCombatWithMonster(t, references.MonsterOrc, references.Down, 77)
	if withPlan == nil || withoutPlan == nil {
		return
	}
	require.True(t, withPlan.CastCombatQuasAnWis())
	require.True(t, withoutPlan.CastCombatQuasAnWis())
	entry := withPlan.CombatScene.TurnQueue.GetEntry(0, false)
	withPlan.planMonsterTurn(entry)
	assert.Equal(t, withoutPlan.RollDice(30), withPlan.RollDice(30),
		"presentation planning must not consume confusion RNG")
}

func TestRelXenBetReplacesCloneAndCleansUpProxy(t *testing.T) {
	gs, mock, _ := setupCombatWithMonster(t, references.MonsterOrc, references.Down, 42)
	if gs == nil {
		return
	}
	partyIndex := gs.CombatScene.GetActiveCombatantIndex()
	clone := gs.CombatScene.addPCCloneFromSource(
		CombatTurnIdentity{IsPartyMember: true, EntityIndex: partyIndex},
		partyIndex, references.Position{X: 1, Y: 1}, NewCombatPCProvider(gs.PartyState.GetCharacter(partyIndex)),
	)
	entriesBefore := gs.CombatScene.TurnQueue.TotalEntryCount()
	mock.Reset()
	require.True(t, gs.CastCombatRelXenBet(partyIndex, clone))
	assert.Empty(t, gs.CombatScene.partyClones)
	assert.Equal(t, entriesBefore+1, gs.CombatScene.TurnQueue.TotalEntryCount())
	retired := gs.CombatScene.TurnQueue.GetEntryByIdentity(clone)
	require.NotNil(t, retired)
	assert.Equal(t, turnQueueFled, retired.status)
	assert.True(t, retired.sourceSlotReleased)
	require.Len(t, mock.CombatEvents, 1)
	assert.Equal(t, clone.Slot, mock.CombatEvents[0].Target.Slot)
	assert.Equal(t, references.Position{X: 1, Y: 1}, mock.CombatEvents[0].Target.Position)
}

func TestRelXenBetUsesFreeSourceSlotBeyondInitialMonsterLimit(t *testing.T) {
	gs, _, _ := setupCombatWithMonster(t, references.MonsterOrc, references.Down, 42)
	if gs == nil {
		return
	}
	ref := &(*gs.GameReferences.EnemyReferences)[references.MonsterOrc]
	for gs.CombatScene.MonsterManager.Count() < references.MaxMonstersInCombat {
		monster := NewCombatMonster(
			references.MonsterOrc, ref, references.Position{X: 8, Y: 8}, newFixedRNG(1),
		)
		monster.SetActive(false)
		gs.CombatScene.MonsterManager.AddMonster(monster)
	}
	target := gs.CombatScene.GetCurrentTurnIdentity()
	require.True(t, target.IsPartyMember)

	assert.True(t, gs.isCombatSpellTargetValid(references.RelXenBet, nil, target))
	assert.True(t, gs.CastCombatRelXenBet(target.EntityIndex, target))
	assert.Equal(t, references.MaxMonstersInCombat+1, gs.CombatScene.MonsterManager.Count())
}

func TestCloneOnlyPartyCannotClaimVictoryWhileHostileRemains(t *testing.T) {
	gs, _, _ := setupCombatWithMonster(t, references.MonsterOrc, references.Down, 42)
	if gs == nil {
		return
	}
	source := gs.CombatScene.GetCurrentTurnIdentity()
	clone := gs.CombatScene.addPCCloneFromSource(
		source,
		source.EntityIndex,
		references.Position{X: 1, Y: 1},
		NewCombatPCProvider(gs.PartyState.GetCharacter(source.EntityIndex)),
	)
	require.NotZero(t, clone.Slot)
	gs.CombatScene.PartyActive[source.EntityIndex] = false
	gs.CombatScene.TurnQueue.markFledIdentity(source)

	assert.False(t, gs.CanEscapeCombatMap())
	assert.NotNil(t, gs.CombatScene)
}
