package game_state

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/map_state"
	"github.com/bradhannah/UltimaVRedux/internal/party_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
)

func TestCombatActorProjectionExactClonePossessionAndSaduj(t *testing.T) {
	gs, _ := newCombatAttackPreviewTest(t, references.MorningStar)
	source := gs.CombatScene.GetCurrentTurnIdentity()
	clone := gs.CombatScene.addPCCloneFromSource(source, source.EntityIndex, references.Position{X: 3, Y: 3}, NewCombatPCProvider(gs.PartyState.GetCharacter(source.EntityIndex)))
	gs.CombatScene.setCombatPCPossessed(clone, true)
	for _, identity := range []CombatTurnIdentity{source, clone} {
		actor, ok := gs.ProjectCombatActor(identity, 0)
		require.True(t, ok)
		assert.Equal(t, identity, actor.Identity)
		assert.Equal(t, identity == clone, actor.Possessed)
		assert.Equal(t, gs.combatIdentityLoyalty(identity), actor.Loyalty)
	}
	gs.CombatScene.setCombatPCUnderground(clone, true)
	actor, ok := gs.ProjectCombatActor(clone, 0)
	assert.False(t, ok)
	assert.Empty(t, actor.Name)
	// Intrinsic hostility is neither possession nor charm provenance.
	gs.CombatScene.PartyActive[1] = true
	character := gs.PartyState.GetCharacter(1)
	character.PartyMap = party_state.PartyMapActive
	character.SetStatus(party_state.Good)
	gs.CombatScene.SetPartyMemberPosition(1, gs.CombatScene.GetCombatantPosition(source))
	clear(character.Name[:])
	copy(character.Name[:], "Saduj")
	actor, ok = gs.ProjectCombatActor(CombatTurnIdentity{IsPartyMember: true, EntityIndex: 1}, 0)
	require.True(t, ok)
	assert.Equal(t, CombatMonsterLoyaltyMonster, actor.Loyalty)
	assert.False(t, actor.Possessed)
	assert.False(t, actor.Charmed)
}

func TestCombatActorProjectionDisclosureIdentityAndPurity(t *testing.T) {
	gs, mock := newCombatAttackPreviewTest(t, references.Bow)
	monster, index := addSyntheticTriggerMonster(gs, references.Position{X: 5, Y: 2}, CombatMonsterLoyaltyParty, references.MissileNone)
	identity := CombatTurnIdentity{EntityIndex: index}
	rng, elapsed := gs.GetRNG().Calls(), gs.ElapsedMs
	actor, ok := gs.ProjectCombatActor(identity, monster.CombatantID())
	require.True(t, ok)
	assert.False(t, actor.Charmed, "loyalty alone is not provenance")
	assert.False(t, actor.Summoned)
	_, ok = gs.ProjectCombatActor(identity, monster.CombatantID()+1)
	assert.False(t, ok)
	for _, conceal := range []func(){
		func() { monster.SetInvisible(true) },
		func() {
			monster.SetInvisible(false)
			gs.GetCurrentLayeredMap().SetTileByLayer(map_state.MapLayer, &actor.Position, indexes.BlackSquare)
		},
		func() {
			gs.GetCurrentLayeredMap().SetTileByLayer(map_state.MapLayer, &actor.Position, indexes.BrickFloor)
			monster.SetActive(false)
		},
	} {
		conceal()
		view, disclosed := gs.ProjectCombatActor(identity, monster.CombatantID())
		assert.False(t, disclosed)
		assert.Equal(t, CombatActorPresentation{}, view)
	}
	assert.Equal(t, rng, gs.GetRNG().Calls())
	assert.Equal(t, elapsed, gs.ElapsedMs)
	assert.Empty(t, mock.Messages)
}

func TestCombatAttackHelmMorningStarNamesAndSafeEmptyProjection(t *testing.T) {
	gs, _ := newCombatAttackPreviewTest(t, references.MorningStar)
	gs.PartyState.GetCharacter(0).SetEquippedItem(references.SlotHelm, references.SpikedHelm)
	monster, _ := addSyntheticTriggerMonster(gs, references.Position{X: 3, Y: 2}, CombatMonsterLoyaltyMonster, references.MissileNone)
	require.True(t, gs.BeginCombatAttack())
	assert.Equal(t, CombatShotChooseTarget, gs.PreviewCombatAttack().Aim.Status)
	for step, equipment := range []references.Equipment{references.SpikedHelm, references.MorningStar} {
		gs.CombatScene.AttackTargeting.Cursor = monster.Position()
		preview := gs.PreviewCombatAttack()
		assert.Equal(t, equipment, preview.Sources[0].Equipment)
		assert.Equal(t, step+1, preview.Sources[0].MaxRange)
		assert.Equal(t, "Orc", preview.Aim.Name)
		monster.SetLoyalty(CombatMonsterLoyaltyParty)
		assert.Equal(t, "Orc", gs.PreviewCombatAttack().Aim.Name)
		assert.Equal(t, CombatShotFriendly, gs.PreviewCombatAttack().Aim.Status)
		monster.SetInvisible(true)
		hidden := *gs.PreviewCombatAttack().Aim
		monster.SetActive(false)
		assert.Equal(t, hidden, *gs.PreviewCombatAttack().Aim, "no invisible-occupancy detector")
		monster.SetActive(true)
		monster.SetInvisible(false)
		monster.SetLoyalty(CombatMonsterLoyaltyMonster)
		gs.CancelCombatAttackSource()
	}
}

func TestCombatActorProvenanceComesFromSuccessfulProducers(t *testing.T) {
	gs, _ := newCombatSummonTestState(t)
	result := gs.castCombatKalXen(newStrictSummonRNG(t, 0, 5, 5, 0))
	require.Len(t, result.Created, 1)
	created := result.Created[0]
	actor, ok := gs.ProjectCombatActor(CombatTurnIdentity{EntityIndex: created.Index}, created.CombatantID)
	require.True(t, ok)
	assert.True(t, actor.Summoned)
	assert.False(t, actor.Charmed)
	monster := gs.CombatScene.MonsterManager.GetMonster(created.Index)
	monster.SetLoyalty(CombatMonsterLoyaltyMonster)
	gs.PartyState.GetCharacter(0).Intelligence = 255
	gs.SetRandomSeed(42)
	require.True(t, gs.CastCombatAnXenEx(0, actor.Identity))
	actor, ok = gs.ProjectCombatActor(actor.Identity, created.CombatantID)
	require.True(t, ok)
	assert.True(t, actor.Charmed)
	assert.True(t, actor.Summoned)
}

func TestCombatActorInvisibleDungeonAssistanceKeepsLegalityWithoutDisclosure(t *testing.T) {
	gs, _ := newCombatAttackPreviewTest(t, references.MorningStar)
	monster, _ := addSyntheticTriggerMonster(gs, references.Position{X: 4, Y: 2}, CombatMonsterLoyaltyMonster, references.MissileNone)
	monster.SetInvisible(true)
	gs.CombatScene.DungeonRoomOrigin = &DungeonRoomEncounterOrigin{Location: references.Doom}
	setBlockedCombatMissileMap(gs, references.Position{X: 3, Y: 2})
	rng := gs.GetRNG().Calls()
	preview := gs.PreviewCombatAttack()
	require.Len(t, preview.Targets, 1)
	assert.Nil(t, preview.Targets[0].Actor)
	assert.Empty(t, preview.Targets[0].Name)
	assert.Equal(t, CombatAutoActionAttack, gs.PlanCombatAutoAction().Kind)
	require.True(t, gs.BeginCombatAttack())
	gs.CombatScene.AttackTargeting.Cursor = monster.Position()
	hidden := *gs.PreviewCombatAttack().Aim
	monster.SetActive(false)
	assert.Equal(t, hidden, *gs.PreviewCombatAttack().Aim, "readout must not reveal an invisible target through terrain-bypass wording")
	assert.Equal(t, rng, gs.GetRNG().Calls())
}
