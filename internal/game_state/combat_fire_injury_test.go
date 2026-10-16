package game_state

import (
	"fmt"
	"maps"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/map_state"
	"github.com/bradhannah/UltimaVRedux/internal/party_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
)

func setFireInjuryTerrain(state *GameState, position references.Position, tile indexes.SpriteIndex) {
	state.CombatScene.CombatMapRef.SetTileNumber(position, tile)
	state.GetCurrentLayeredMap().SetTileByLayer(map_state.MapLayer, &position, tile)
}

func lastResolvedFireInjury(t *testing.T, events []CombatEvent) CombatFireInjury {
	t.Helper()
	var injuries []CombatFireInjury
	for _, event := range events {
		if event.FireInjury != nil {
			injuries = append(injuries, *event.FireInjury)
		}
	}
	require.Len(t, injuries, 1)

	return injuries[0]
}

func addMortalFireInjuryMonster(t *testing.T, state *GameState, position references.Position) (*CombatMonster, int) {
	t.Helper()
	// The trigger/attack-preview fixture deliberately makes its Orc immortal.
	// Injury tests need real HP loss; detach abilities before removing immunity
	// so other actors and the shared trigger fixture retain their original rules.
	reference := (*state.GameReferences.EnemyReferences)[references.MonsterOrc]
	reference.EnemyAbilities = maps.Clone(reference.EnemyAbilities)
	delete(reference.EnemyAbilities, references.Immortal)
	monster := NewCombatMonster(references.MonsterOrc, &reference, position, newFixedRNG(1))
	index := state.CombatScene.AddMonsterToQueue(monster)
	require.NotZero(t, monster.CombatantID())
	entry := state.CombatScene.TurnQueue.GetEntryByIdentity(CombatTurnIdentity{EntityIndex: index})
	require.NotNil(t, entry)
	require.Same(t, monster, entry.Combatant)
	require.True(t, monster.IsAlive())
	require.False(t, monster.IsInvisible() || monster.IsCorpserHidden())
	require.Equal(t, 7, monster.ApplyDamageModifiers(7, references.NoEquipment))
	require.True(t, (*state.GameReferences.EnemyReferences)[references.MonsterOrc].HasAbility(references.Immortal),
		"the shared trigger reference must remain unchanged")

	return monster, index
}

func TestCombatFireInjuryMovementPublishesActualHPOnce(t *testing.T) {
	for _, tile := range []indexes.SpriteIndex{indexes.Fireplace, indexes.Lava} {
		state, mock := newCombatAttackPreviewTest(t, references.NoEquipment)
		actor := state.CombatScene.GetCurrentTurnIdentity()
		from := references.Position{X: 2, Y: 2}
		destination := references.Position{X: 3, Y: 2}
		state.CombatScene.SetPartyMemberPosition(actor.EntityIndex, from)
		setFireInjuryTerrain(state, destination, tile)
		character := state.PartyState.GetCharacter(actor.EntityIndex)
		character.SetMaxHp(100)
		character.SetHp(100)
		character.SetEquippedItem(references.SlotRing, references.NoEquipment)
		state.SetRandomSeed(findCombatWakeSeed(t, state, 10, func(roll int) bool { return roll == 7 }))
		before := state.GetRNG().Calls()
		mock.Reset()

		require.Equal(t, CombatMoveMoved, state.MoveCombatCharacter(references.Right))

		injury := lastResolvedFireInjury(t, mock.CombatEvents)
		owner := state.BindCombatFireInjury(injury)
		assert.Equal(t, 7, injury.HPLost)
		assert.Equal(t, uint16(93), character.GetCurrentHp())
		assert.Equal(t, before+1, state.GetRNG().Calls())
		assert.Equal(t, actor.EntityIndex, injury.Target.Index)
		assert.Equal(t, actor.Slot, injury.Target.Slot)
		assert.Equal(t, destination, injury.Target.Position)
		assert.True(t, injury.Disclosed)
		position, disclosed := state.ProjectCombatFireInjury(injury, owner)
		assert.True(t, disclosed)
		assert.Equal(t, destination, position)
		assert.Equal(t, before+1, state.GetRNG().Calls(), "projection is RNG-free")
	}
}

func TestCombatFireInjuryZeroDeadInactiveAndTerrainPrecedence(t *testing.T) {
	state, mock := newCombatAttackPreviewTest(t, references.NoEquipment)
	actor := state.CombatScene.GetCurrentTurnIdentity()
	position := state.CombatScene.GetCombatantPosition(actor)
	character := state.PartyState.GetCharacter(actor.EntityIndex)
	setFireInjuryTerrain(state, position, indexes.Fireplace)
	require.True(t, state.CombatScene.AddField(NewCombatField(CombatFieldSleep, position)))
	state.SetRandomSeed(findCombatWakeSeed(t, state, 10, func(roll int) bool { return roll == 0 }))
	before := state.GetRNG().Calls()
	mock.Reset()
	state.resolveCombatPCStandingEffects(actor)
	injury := lastResolvedFireInjury(t, mock.CombatEvents)
	assert.Zero(t, injury.HPLost)
	owner := state.BindCombatFireInjury(injury)
	_, disclosed := state.ProjectCombatFireInjury(injury, owner)
	assert.False(t, disclosed, "zero HP loss is not an injury")
	assert.Equal(t, before+1, state.GetRNG().Calls())
	assert.False(t, state.CombatScene.isCombatPCSleeping(actor), "terrain retains precedence over fields")
	for _, retire := range []func(){
		func() { character.SetStatus(party_state.Dead) },
		func() {
			character.SetStatus(party_state.Good)
			state.CombatScene.PartyActive[actor.EntityIndex] = false
		},
	} {
		retire()
		mock.Reset()
		before = state.GetRNG().Calls()
		state.resolveCombatPCStandingEffects(actor)
		assert.Empty(t, mock.CombatEvents)
		assert.Equal(t, before, state.GetRNG().Calls())
	}
}

func TestCombatFireInjuryFatalMonsterSurvivesRemovalButNotReuse(t *testing.T) {
	state, mock := newCombatAttackPreviewTest(t, references.NoEquipment)
	position := references.Position{X: 5, Y: 2}
	monster, index := addMortalFireInjuryMonster(t, state, position)
	require.Equal(t, 3, monster.SetCurrentHP(3))
	setFireInjuryTerrain(state, position, indexes.Fireplace)
	view, disclosed := state.ProjectCombatActor(CombatTurnIdentity{EntityIndex: index}, monster.CombatantID())
	require.True(t, disclosed, "the exact living enemy must be disclosed before injury")
	require.Equal(t, CombatMonsterLoyaltyMonster, view.Loyalty)
	state.SetRandomSeed(findCombatWakeSeed(t, state, 10, func(roll int) bool { return roll == 7 }))
	mock.Reset()
	state.resolveCombatMonsterFieldEffects(monster, index)
	injury := lastResolvedFireInjury(t, mock.CombatEvents)
	require.True(t, injury.Fatal)
	owner := state.BindCombatFireInjury(injury)
	require.True(t, injury.Disclosed)
	assert.Equal(t, 3, injury.HPLost, "fatal loss is capped at actual pre-hit HP")
	projected, disclosed := state.ProjectCombatFireInjury(injury, owner)
	require.True(t, disclosed)
	assert.Equal(t, position, projected)
	monster.SetInvisible(true)
	_, disclosed = state.ProjectCombatFireInjury(injury, owner)
	assert.False(t, disclosed, "a retained fatal result cannot bypass hidden-actor policy")
	monster.SetInvisible(false)
	state.CombatScene.MonsterManager.RemoveMonster(index)
	state.CombatScene.TurnQueue.RemoveAllMonsters()
	projected, disclosed = state.ProjectCombatFireInjury(injury, owner)
	assert.True(t, disclosed, "the fatal label does not need a sprite or queue entry")
	assert.Equal(t, position, projected)
	replacement, replacementIndex := addMortalFireInjuryMonster(t, state, position)
	require.Equal(t, index, replacementIndex)
	require.NotEqual(t, injury.CombatantID, replacement.CombatantID())
	_, disclosed = state.ProjectCombatFireInjury(injury, owner)
	assert.False(t, disclosed, "recycled manager index must not inherit a fatal injury")
}

func TestCombatFireInjuryLiveCloneFollowsOnlyExactEntry(t *testing.T) {
	state, mock := newCombatAttackPreviewTest(t, references.NoEquipment)
	source := state.CombatScene.GetCurrentTurnIdentity()
	position := references.Position{X: 4, Y: 4}
	character := state.PartyState.GetCharacter(source.EntityIndex)
	character.SetMaxHp(100)
	character.SetHp(100)
	clone := state.CombatScene.addPCCloneFromSource(source, source.EntityIndex, position, NewCombatPCProvider(character))
	state.SetRandomSeed(findCombatWakeSeed(t, state, 10, func(roll int) bool { return roll == 7 }))
	mock.Reset()
	state.applyCombatFieldToPC(NewCombatField(CombatFieldFire, position), clone, false)
	injury := lastResolvedFireInjury(t, mock.CombatEvents)
	require.False(t, injury.Fatal)
	owner := state.BindCombatFireInjury(injury)
	to := references.Position{X: 5, Y: 4}
	setFireInjuryTerrain(state, to, indexes.BrickFloor)
	state.CombatScene.setCombatantPosition(clone, to)
	projected, disclosed := state.ProjectCombatFireInjury(injury, owner)
	assert.True(t, disclosed)
	assert.Equal(t, to, projected)
	assert.Equal(t, position, injury.Target.Position, "projection must not mutate the resolved square")
	state.CombatScene.setCombatPCUnderground(clone, true)
	_, disclosed = state.ProjectCombatFireInjury(injury, owner)
	assert.False(t, disclosed)
	state.CombatScene.setCombatPCUnderground(clone, false)
	state.CombatScene.TurnQueue.ReplaceCombatant(clone, clone, NewCombatPCProvider(character))
	_, disclosed = state.ProjectCombatFireInjury(injury, owner)
	assert.False(t, disclosed)
}

func TestCombatFireInjuryCloneFatalDisclosureAndLifetime(t *testing.T) {
	state, mock := newCombatAttackPreviewTest(t, references.NoEquipment)
	source := state.CombatScene.GetCurrentTurnIdentity()
	position := references.Position{X: 4, Y: 4}
	character := state.PartyState.GetCharacter(source.EntityIndex)
	clone := state.CombatScene.addPCCloneFromSource(source, source.EntityIndex, position, NewCombatPCProvider(character))
	setFireInjuryTerrain(state, position, indexes.Fireplace)
	character.SetHp(3)
	state.SetRandomSeed(findCombatWakeSeed(t, state, 10, func(roll int) bool { return roll == 7 }))
	mock.Reset()
	state.resolveCombatPCStandingEffects(clone)
	injury := lastResolvedFireInjury(t, mock.CombatEvents)
	assert.Equal(t, clone.Slot, injury.Target.Slot)
	owner := state.BindCombatFireInjury(injury)
	assert.Equal(t, 3, injury.HPLost)
	assert.True(t, injury.Fatal)
	assert.False(t, state.CombatScene.IsCombatPCActive(clone))
	projected, disclosed := state.ProjectCombatFireInjury(injury, owner)
	assert.True(t, disclosed)
	assert.Equal(t, position, projected, "retired clone label must not fall back to the source PC")
	setFireInjuryTerrain(state, position, indexes.BrickWallArchway)
	_, disclosed = state.ProjectCombatFireInjury(injury, owner)
	assert.False(t, disclosed)
	setFireInjuryTerrain(state, position, indexes.BrickFloor)
	state.CombatScene.TurnQueue.ReplaceCombatant(clone, clone, NewCombatPCProvider(character))
	_, disclosed = state.ProjectCombatFireInjury(injury, owner)
	assert.False(t, disclosed, "a new entry provider cannot inherit the fatal label")
}

func TestCombatFireInjuryMonsterBirthHiddenAndEncounterReplacement(t *testing.T) {
	state, mock := newCombatAttackPreviewTest(t, references.NoEquipment)
	position := references.Position{X: 5, Y: 2}
	monster, index := addMortalFireInjuryMonster(t, state, position)
	setFireInjuryTerrain(state, position, indexes.Fireplace)
	view, disclosed := state.ProjectCombatActor(CombatTurnIdentity{EntityIndex: index}, monster.CombatantID())
	require.True(t, disclosed, "the exact living enemy must be disclosed before injury")
	require.Equal(t, CombatMonsterLoyaltyMonster, view.Loyalty)
	state.SetRandomSeed(findCombatWakeSeed(t, state, 10, func(roll int) bool { return roll == 7 }))
	mock.Reset()
	state.resolveCombatMonsterFieldEffects(monster, index)
	injury := lastResolvedFireInjury(t, mock.CombatEvents)
	require.Equal(t, 7, injury.HPLost)
	owner := state.BindCombatFireInjury(injury)
	require.False(t, injury.Fatal)
	require.True(t, injury.Disclosed)
	assert.Equal(t, monster.CombatantID(), injury.CombatantID)
	_, disclosed = state.ProjectCombatFireInjury(injury, owner)
	require.True(t, disclosed)
	monster.SetInvisible(true)
	_, disclosed = state.ProjectCombatFireInjury(injury, owner)
	assert.False(t, disclosed)
	monster.SetInvisible(false)
	old := state.CombatScene
	replacement := *old
	state.CombatScene = &replacement
	_, disclosed = state.ProjectCombatFireInjury(injury, owner)
	assert.False(t, disclosed, "equal encounter number on another state is not the same encounter")
	state.CombatScene = old
	monster.combatantID++
	_, disclosed = state.ProjectCombatFireInjury(injury, owner)
	assert.False(t, disclosed, "same manager index cannot substitute a different birth identity")
}

func TestCombatFireInjuryHiddenResolutionCannotRevealLater(t *testing.T) {
	state, mock := newCombatAttackPreviewTest(t, references.NoEquipment)
	actor := state.CombatScene.GetCurrentTurnIdentity()
	position := state.CombatScene.GetCombatantPosition(actor)
	setFireInjuryTerrain(state, position, indexes.BrickWallArchway)
	state.SetRandomSeed(findCombatWakeSeed(t, state, 10, func(roll int) bool { return roll == 7 }))
	mock.Reset()
	state.applyCombatFieldToPC(NewCombatField(CombatFieldFire, position), actor, false)
	injury := lastResolvedFireInjury(t, mock.CombatEvents)
	assert.False(t, injury.Disclosed)
	owner := state.BindCombatFireInjury(injury)
	setFireInjuryTerrain(state, position, indexes.BrickFloor)
	_, disclosed := state.ProjectCombatFireInjury(injury, owner)
	assert.False(t, disclosed)
}

func TestCombatFireInjuryDirectFieldHasOneImpactOwner(t *testing.T) {
	state, mock := newCombatAttackPreviewTest(t, references.NoEquipment)
	position := references.Position{X: 5, Y: 2}
	monster, index := addMortalFireInjuryMonster(t, state, position)
	require.Equal(t, 1, monster.EnemyRef().Armour)
	// Direct fire rolls 1..21, then subtracts the fixture's one point of armor.
	state.SetRandomSeed(findCombatWakeSeed(t, state, 20, func(roll int) bool { return roll == 7 }))
	mock.Reset()
	impacts := 0
	state.SystemCallbacks.Visual.ShowCombatImpact = func(CombatAttackFeedback) { impacts++ }
	hpBefore := monster.CurrentHP()
	state.applyCombatFieldToMonster(NewCombatField(CombatFieldFire, position), monster, index, 0, true)
	injury := lastResolvedFireInjury(t, mock.CombatEvents)
	assert.Equal(t, 7, injury.HPLost, "the single-owner assertion must exercise a damaging impact")
	assert.Equal(t, hpBefore-monster.CurrentHP(), injury.HPLost)
	assert.Zero(t, impacts, "field event owns the label; generic magic impact must not duplicate it")
	assert.Equal(t, 0, countCombatEvents(mock.CombatEvents, CombatEventAttack))
}

func TestCombatFireInjuryImmortalTargetHasNoFalseInjury(t *testing.T) {
	state, mock := newCombatAttackPreviewTest(t, references.NoEquipment)
	position := references.Position{X: 5, Y: 2}
	monster, index := addSyntheticTriggerMonster(state, position, CombatMonsterLoyaltyMonster, references.MissileNone)
	setFireInjuryTerrain(state, position, indexes.Fireplace)
	require.True(t, monster.HasAbility(references.Immortal))
	_, disclosed := state.ProjectCombatActor(CombatTurnIdentity{EntityIndex: index}, monster.CombatantID())
	require.True(t, disclosed)
	state.SetRandomSeed(findCombatWakeSeed(t, state, 10, func(roll int) bool { return roll == 7 }))
	hpBefore, callsBefore := monster.CurrentHP(), state.GetRNG().Calls()
	mock.Reset()
	state.resolveCombatMonsterFieldEffects(monster, index)
	injury := lastResolvedFireInjury(t, mock.CombatEvents)
	assert.True(t, injury.Disclosed, "immunity is not concealment")
	owner := state.BindCombatFireInjury(injury)
	assert.Zero(t, injury.HPLost)
	assert.False(t, injury.Fatal)
	assert.Equal(t, hpBefore, monster.CurrentHP())
	assert.Equal(t, callsBefore+1, state.GetRNG().Calls())
	_, disclosed = state.ProjectCombatFireInjury(injury, owner)
	assert.False(t, disclosed, "a positive raw roll is not an injury when immunity prevents HP loss")
}

func TestCombatFireInjuryTapeIsDetachedFromLaterCombatMutation(t *testing.T) {
	resolve := func() (*GameState, *MockSystemCallbacks, CombatFireInjury) {
		state, mock := newCombatAttackPreviewTest(t, references.NoEquipment)
		actor := state.CombatScene.GetCurrentTurnIdentity()
		character := state.PartyState.GetCharacter(actor.EntityIndex)
		character.SetMaxHp(100)
		character.SetHp(100)
		position := state.CombatScene.GetCombatantPosition(actor)
		setFireInjuryTerrain(state, position, indexes.Fireplace)
		state.SetRandomSeed(findCombatWakeSeed(t, state, 10, func(roll int) bool { return roll == 7 }))
		mock.Reset()
		state.resolveCombatPCStandingEffects(actor)

		return state, mock, lastResolvedFireInjury(t, mock.CombatEvents)
	}
	state, events, injury := resolve()
	_, expectedEvents, expectedInjury := resolve()
	require.Equal(t, expectedEvents.CombatEvents, events.CombatEvents)
	// Value equality across independently allocated states also guards against
	// comparable pointer fields sneaking back into the resolved injury record.
	require.True(t, injury == expectedInjury)
	actor := state.CombatScene.GetCurrentTurnIdentity()
	state.CombatScene.SetPartyMemberPosition(actor.EntityIndex, references.Position{X: 8, Y: 8})
	state.CombatScene.setCombatPCSleeping(actor, true)
	state.PartyState.GetCharacter(actor.EntityIndex).SetHp(1)
	state.CombatScene.TurnQueue.ReplaceCombatant(actor, actor,
		NewCombatPCProvider(state.PartyState.GetCharacter(actor.EntityIndex)))
	state.CombatScene.EncounterID++
	assert.Equal(t, expectedEvents.CombatEvents, events.CombatEvents, "durable tape equality cannot traverse the mutated live graph")
	assert.Equal(t, expectedInjury, *events.CombatEvents[0].FireInjury)
}

func TestCombatFireInjuryRejectsSameProviderReplacementBeforeAdmission(t *testing.T) {
	for _, hp := range []uint16{3, 100} {
		t.Run(fmt.Sprint(hp), func(t *testing.T) {
			state, mock := newCombatAttackPreviewTest(t, references.NoEquipment)
			actor := state.CombatScene.GetCurrentTurnIdentity()
			character := state.PartyState.GetCharacter(actor.EntityIndex)
			character.SetMaxHp(100)
			character.SetHp(hp)
			position := state.CombatScene.GetCombatantPosition(actor)
			setFireInjuryTerrain(state, position, indexes.Fireplace)
			state.SetRandomSeed(findCombatWakeSeed(t, state, 10, func(roll int) bool { return roll == 7 }))
			mock.Reset()
			state.resolveCombatPCStandingEffects(actor)
			injury := lastResolvedFireInjury(t, mock.CombatEvents)
			owner := state.BindCombatFireInjury(injury)
			_, disclosed := state.ProjectCombatFireInjury(injury, owner)
			require.True(t, disclosed)
			entry := state.CombatScene.TurnQueue.GetEntryByIdentity(actor)
			require.NotNil(t, entry)
			require.Equal(t, injury.EntryID, entry.lifetimeID)
			state.CombatScene.TurnQueue.ReplaceCombatant(actor, actor, entry.Combatant)
			require.NotEqual(t, injury.EntryID, entry.lifetimeID, "even the same provider starts a new entry lifetime")
			_, disclosed = state.ProjectCombatFireInjury(injury, owner)
			assert.False(t, disclosed, "an already admitted injury cannot follow replacement")
			lateOwner := state.BindCombatFireInjury(injury)
			_, disclosed = state.ProjectCombatFireInjury(injury, lateOwner)
			assert.False(t, disclosed, "late admission cannot recapture a replacement as the original actor")
		})
	}
}
