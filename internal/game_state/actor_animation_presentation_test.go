package game_state

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/map_units"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
)

func TestCombatActorAnimationUsesExactAllegianceWithoutDisclosureOrRNG(t *testing.T) {
	state, _ := newCombatAttackPreviewTest(t, references.Bow)
	combat := state.CombatScene
	pc := combat.GetCurrentTurnIdentity()
	clone := combat.addPCCloneFromSource(pc, pc.EntityIndex, references.Position{X: 3, Y: 3},
		NewCombatPCProvider(state.PartyState.GetCharacter(pc.EntityIndex)))
	require.NotZero(t, clone.Slot)
	combat.setCombatPCPossessed(clone, true)
	monster, index := addSyntheticTriggerMonster(state, references.Position{X: 5, Y: 2}, CombatMonsterLoyaltyMonster, references.MissileNone)
	identity := CombatTurnIdentity{EntityIndex: index}
	state.ActivateTimeStopEffect(20)
	rng := state.GetRNG().Calls()
	pcKey, slow, valid := state.CombatActorAnimation(pc, 0)
	require.True(t, valid)
	assert.False(t, slow)
	cloneKey, slow, valid := state.CombatActorAnimation(clone, 0)
	require.True(t, valid)
	assert.True(t, slow)
	assert.NotEqual(t, pcKey, cloneKey)
	monster.SetInvisible(true)
	_, disclosed := state.ProjectCombatActor(identity, monster.CombatantID())
	assert.False(t, disclosed)
	monsterKey, slow, valid := state.CombatActorAnimation(identity, monster.CombatantID())
	require.True(t, valid)
	assert.True(t, slow, "invisibility is not immunity to Time Stop")
	monster.SetLoyalty(CombatMonsterLoyaltyParty)
	controlledKey, slow, valid := state.CombatActorAnimation(identity, monster.CombatantID())
	require.True(t, valid)
	assert.False(t, slow)
	assert.Equal(t, monsterKey, controlledKey, "allegiance changes rate, not actor lifetime")
	combat.setCombatPCUnderground(clone, true)
	_, slow, valid = state.CombatActorAnimation(clone, 0)
	assert.True(t, valid)
	assert.True(t, slow)
	state.ActivateQuicknessEffect(30)
	for range 20 {
		state.ForEachActorAnimation(func(ActorAnimationIdentity, bool) {})
	}
	assert.Equal(t, rng, state.GetRNG().Calls(), "eligibility must not call temporal action suppression")
	_, _, valid = state.CombatActorAnimation(identity, monster.CombatantID()+1)
	assert.False(t, valid)
	replacement, replacementIndex := addSyntheticTriggerMonster(state, references.Position{X: 5, Y: 2}, CombatMonsterLoyaltyMonster, references.MissileNone)
	beforeMove, _, valid := state.CombatActorAnimation(CombatTurnIdentity{EntityIndex: replacementIndex}, replacement.CombatantID())
	require.True(t, valid)
	combat.MonsterManager.RemoveMonster(index)
	replacementKey, _, valid := state.CombatActorAnimation(CombatTurnIdentity{EntityIndex: replacementIndex - 1}, replacement.CombatantID())
	require.True(t, valid)
	assert.NotEqual(t, monsterKey, replacementKey)
	assert.Equal(t, beforeMove, replacementKey, "moving storage index does not change surviving monster identity")
}

func TestSurfaceActorAnimationExcludesObjectsVehiclesAndHazards(t *testing.T) {
	state, _ := NewIntegrationTestBuilder(t).WithLocation(references.Britain).WithSystemCallbacks().Build()
	state.ActivateTimeStopEffect(20)
	var reference references.NPCReference
	reference.SetKeyIndex(indexes.Rat_KeyIndex)
	actor := map_units.NewNPCFriendly(reference, 7)
	actor.SetFloor(state.MapState.PlayerLocation.Floor)
	key, slow, valid := state.SurfaceActorAnimation(actor)
	require.True(t, valid)
	assert.True(t, slow)
	actor.SetVisible(false)
	actor.SetPos(references.Position{X: 31, Y: 31})
	second, _, valid := state.SurfaceActorAnimation(actor)
	assert.True(t, valid)
	assert.Equal(t, key, second)
	for _, sprite := range []indexes.SpriteIndex{indexes.Chest, indexes.SkiffUp, indexes.Carpet2, indexes.ElectricField} {
		reference.SetKeyIndex(sprite)
		object := map_units.NewNPCFriendly(reference, 7)
		object.SetFloor(actor.Floor())
		_, _, valid = state.SurfaceActorAnimation(object)
		assert.False(t, valid, "object sprite %d", sprite)
	}
	reference.SetKeyIndex(indexes.Rat_KeyIndex)
	replacement := map_units.NewNPCFriendly(reference, 7)
	replacement.SetFloor(actor.Floor())
	replacementKey, _, valid := state.SurfaceActorAnimation(replacement)
	require.True(t, valid)
	assert.NotEqual(t, key, replacementKey, "NPCNum is not a lifetime key")
}

func TestDungeonActorAnimationIdentitySurvivesMovementButNotReplacement(t *testing.T) {
	state := newDungeonStateTestGame(t, references.DungeonCells{})
	wanderer := &DungeonWanderer{Monster: references.MonsterRat, Floor: 0, Position: references.Position{X: 2, Y: 1}}
	state.activeDungeon.replaceWanderer(wanderer)
	key, _, valid := state.DungeonActorAnimation()
	require.True(t, valid)
	wanderer.Position.Y++
	state.activeDungeon.setFacingAndLastDirection(references.DungeonFacingWest)
	moved, _, valid := state.DungeonActorAnimation()
	assert.True(t, valid)
	assert.Equal(t, key, moved)
	state.activeDungeon.replaceWanderer(&DungeonWanderer{Monster: references.MonsterBat, Floor: 0, Position: wanderer.Position})
	replacement, _, valid := state.DungeonActorAnimation()
	assert.True(t, valid)
	assert.NotEqual(t, key, replacement)
	state.CombatScene = &CombatState{}
	_, _, valid = state.DungeonActorAnimation()
	assert.False(t, valid)
}
