package game_state

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/party_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
)

func TestAnXenCorpIsImplementedNoTargetCombatSpell(t *testing.T) {
	assert.True(t, IsImplementedCombatSpell(references.AnXenCorp))
	assert.Equal(t, references.TargetAutomatic, references.GetSpellInfo(references.AnXenCorp).Targeting)
}

func TestCastCombatAnXenCorpFiltersTargetsAndAppliesFailedSaves(t *testing.T) {
	gs, mock, affected := setupCombatWithMonster(t, references.MonsterSkeleton, references.Down, 42)
	if gs == nil {
		return
	}
	casterIdx := gs.CombatScene.GetActiveCombatantIndex()
	caster := gs.PartyState.GetCharacter(casterIdx)
	caster.Intelligence = 255
	affected.SetCurrentHP(min(affected.MaxHP(), 8))

	resistedRef := *affected.EnemyRef()
	resistedRef.Intelligence = 600
	resisted := NewCombatMonster(references.MonsterSkeleton, &resistedRef, references.Position{X: 2, Y: 2}, newFixedRNG(1))
	resisted.SetCurrentHP(min(resisted.MaxHP(), 7))
	resistedIdx := gs.CombatScene.AddMonsterToQueue(resisted)

	enemyRefs := *gs.GameReferences.EnemyReferences
	orc := NewCombatMonster(references.MonsterOrc, &enemyRefs[references.MonsterOrc], references.Position{X: 3, Y: 3}, newFixedRNG(1))
	gs.CombatScene.AddMonsterToQueue(orc)
	immune := NewCombatMonster(references.MonsterShadowlord, &enemyRefs[references.MonsterShadowlord], references.Position{X: 4, Y: 4}, newFixedRNG(1))
	immuneIdx := gs.CombatScene.AddMonsterToQueue(immune)
	pcHP := caster.GetCurrentHp()
	mock.Reset()

	result := gs.CastCombatAnXenCorp(casterIdx)

	require.Len(t, result.Affected, 1)
	assert.Equal(t, affected.CombatantID(), result.Affected[0].CombatantID)
	assert.Equal(t, affected.Position(), result.Affected[0].Position)
	assert.Equal(t, CombatHitNormal, result.Affected[0].Save.Outcome, "target failed its intelligence save")
	assert.Equal(t, 1, affected.CurrentHP())
	assert.True(t, affected.IsFleeing())

	require.Len(t, result.Resisted, 1)
	assert.Equal(t, resistedIdx, result.Resisted[0].Index)
	assert.Equal(t, CombatHitMiss, result.Resisted[0].Save.Outcome, "target passed its intelligence save")
	assert.Equal(t, min(resisted.MaxHP(), 7), resisted.CurrentHP())
	assert.False(t, resisted.IsFleeing())

	require.Len(t, result.Immune, 1)
	assert.Equal(t, immuneIdx, result.Immune[0].Index)
	assert.Equal(t, immune.Position(), result.Immune[0].Position)
	assert.False(t, immune.IsFleeing())
	assert.Equal(t, pcHP, caster.GetCurrentHp(), "the no-target spell must not inspect PCs")
	assert.False(t, orc.IsFleeing(), "non-undead monsters are skipped")
	assert.Len(t, result.Affected, 1)
	assert.Len(t, result.Resisted, 1)
	assert.Len(t, result.Immune, 1)

	require.Len(t, mock.CombatEvents, 1)
	event := mock.CombatEvents[0]
	assert.Equal(t, CombatEventRepelUndead, event.Kind)
	assert.Equal(t, references.AnXenCorp, event.Diagnostics.Spell)
	assert.Equal(t, result, event.AnXenCorp.Result)
	assert.Equal(t, casterIdx, event.AnXenCorp.Caster.Index)
}

func TestFleeingMonsterRecoveryUsesOriginalUpperBoundary(t *testing.T) {
	gs, _, monster := setupCombatWithMonster(t, references.MonsterOrc, references.Down, 42)
	if gs == nil {
		return
	}
	monster.SetCurrentHP(monster.MaxHP() - 2)

	recoverFleeingMonsterHP(monster, newSequenceRNG(2))
	assert.Equal(t, monster.MaxHP()-2, monster.CurrentHP(), "roll 3 of 4 does not recover")
	recoverFleeingMonsterHP(monster, newSequenceRNG(3))
	assert.Equal(t, monster.MaxHP()-1, monster.CurrentHP(), "roll 4 of 4 recovers exactly one HP")
}

func TestAnXenCorpRepelsPartyLoyalUndeadSummonLikeOriginalQueueScan(t *testing.T) {
	gs, _, hostile := setupCombatWithMonster(t, references.MonsterOrc, references.Down, 42)
	if gs == nil {
		return
	}
	hostile.SetActive(false)
	casterIndex := gs.CombatScene.GetActiveCombatantIndex()
	gs.PartyState.GetCharacter(casterIndex).Intelligence = 255
	daemon := NewCombatMonster(
		references.MonsterDevil,
		mustCombatEnemyReference(t, gs, references.MonsterDevil),
		references.Position{X: 5, Y: 5},
		newFixedRNG(1),
	)
	daemon.SetLoyalty(CombatMonsterLoyaltyParty)
	gs.CombatScene.AddMonsterToQueue(daemon)

	result := gs.CastCombatAnXenCorp(casterIndex)

	require.Len(t, result.Affected, 1)
	assert.Equal(t, daemon.CombatantID(), result.Affected[0].CombatantID)
	assert.Equal(t, 1, daemon.CurrentHP())
	assert.True(t, daemon.IsFleeing(), "original An Xen Corp scans undead monster entries regardless of loyalty")
}

func TestFleeingMonsterMovesAwayFromNearestOpponent(t *testing.T) {
	gs, mock, monster := setupCombatWithMonster(t, references.MonsterOrc, references.Down, 42)
	if gs == nil {
		return
	}
	activeIdx := isolateCombatPC(gs)
	from, away, opponent := findStraightFleeTestPositions(t, gs, monster, 0)
	gs.CombatScene.SetPartyMemberPosition(activeIdx, opponent)
	monster.SetPosition(from)
	monster.SetCurrentHP(1)
	monster.SetFleeing(true)
	mock.Reset()

	gs.executeFleeingMonsterTurnWithRNG(monster, 0, newSequenceRNG(0, 255))

	assert.Equal(t, away, monster.Position())
	require.Len(t, mock.CombatEvents, 1)
	assert.Equal(t, CombatEventMove, mock.CombatEvents[0].Kind)
	assert.Greater(t,
		integerDistance(monster.Position(), gs.CombatScene.GetPartyMemberPosition(activeIdx)),
		integerDistance(from, gs.CombatScene.GetPartyMemberPosition(activeIdx)),
	)
}

func TestFleeingMonsterEscapesAcrossEdgeWithoutDeathOrXP(t *testing.T) {
	gs, mock, monster := setupCombatWithMonster(t, references.MonsterOrc, references.Down, 42)
	if gs == nil {
		return
	}
	activeIdx := isolateCombatPC(gs)
	gs.CombatScene.SetPartyMemberPosition(activeIdx, references.Position{X: 5, Y: 5})
	monster.SetPosition(references.Position{X: 0, Y: 5})
	monster.SetCurrentHP(1)
	monster.SetFleeing(true)
	mock.Reset()

	gs.executeFleeingMonsterTurnWithRNG(monster, 0, newSequenceRNG(0, 255))

	assert.Equal(t, references.Position{X: -1, Y: 5}, monster.Position())
	assert.False(t, monster.IsActive())
	assert.False(t, monster.IsDead())
	assert.Equal(t, 0, gs.CombatScene.MonsterManager.GetTotalExperience())
	entry := gs.CombatScene.TurnQueue.GetEntry(0, false)
	require.NotNil(t, entry)
	assert.Equal(t, turnQueueFled, entry.status)
	require.Len(t, mock.CombatEvents, 1)
	assert.Equal(t, CombatEventFled, mock.CombatEvents[0].Kind)
	assert.False(t, mock.CombatEvents[0].Attacker.IsPartyMember)
}

func TestCorneredFleeingMonsterAttacksNearestOpponent(t *testing.T) {
	gs, mock, monster := setupCombatWithMonster(t, references.MonsterOrc, references.Down, 42)
	if gs == nil {
		return
	}
	activeIdx := isolateCombatPC(gs)
	center := references.Position{X: 5, Y: 5}
	gs.CombatScene.SetPartyMemberPosition(activeIdx, references.Position{X: 5, Y: 4})
	monster.SetPosition(center)
	monster.SetCurrentHP(1)
	monster.SetFleeing(true)
	for _, position := range []references.Position{{X: 5, Y: 6}, {X: 4, Y: 5}, {X: 6, Y: 5}} {
		require.True(t, gs.CombatScene.AddField(NewCombatField(CombatFieldProtection, position)))
	}
	mock.Reset()

	gs.executeFleeingMonsterTurnWithRNG(monster, 0, newSequenceRNG(0, 0, 0, 1, 2, 3))

	assert.Equal(t, center, monster.Position())
	require.NotEmpty(t, mock.CombatEvents)
	assert.Equal(t, CombatEventAttack, mock.CombatEvents[len(mock.CombatEvents)-1].Kind)
	assert.Equal(t, activeIdx, mock.CombatEvents[len(mock.CombatEvents)-1].Target.Index)
}

func TestHostileFleeingMonsterBlocksVictoryUntilEscape(t *testing.T) {
	gs, _, monster := setupCombatWithMonster(t, references.MonsterOrc, references.Down, 42)
	if gs == nil {
		return
	}
	activeIdx := isolateCombatPC(gs)
	gs.CombatScene.SetPartyMemberPosition(activeIdx, references.Position{X: 5, Y: 5})
	monster.SetPosition(references.Position{X: 0, Y: 5})
	monster.SetCurrentHP(1)
	monster.SetFleeing(true)

	assert.Equal(t, CombatNotEnded, gs.checkCombatEndConditions())
	assert.False(t, gs.CombatScene.Victory)
	gs.executeFleeingMonsterTurnWithRNG(monster, 0, newSequenceRNG(0, 255))
	assert.Equal(t, CombatVictory, gs.checkCombatEndConditions())
	assert.True(t, gs.CombatScene.Victory)
}

func TestOrdinaryMonsterCannotLeaveCombatMap(t *testing.T) {
	gs, _, monster := setupCombatWithMonster(t, references.MonsterOrc, references.Down, 42)
	if gs == nil {
		return
	}
	monster.SetPosition(references.Position{X: 0, Y: 5})
	assert.False(t, isCombatPositionValidForMonster(gs, references.Position{X: -1, Y: 5}, monster, 0))
	monster.SetFleeing(true)
	assert.True(t, isCombatPositionValidForMonster(gs, references.Position{X: -1, Y: 5}, monster, 0))
}

func TestFleeingMonsterReevaluatesConditionAfterRecovery(t *testing.T) {
	gs, mock, monster := setupCombatWithMonster(t, references.MonsterGazer, references.Down, 42)
	if gs == nil {
		return
	}
	monster.SetCurrentHP(9)
	monster.SetFleeing(true)
	activeIdx := isolateCombatPC(gs)
	center := references.Position{X: 5, Y: 5}
	monster.SetPosition(center)
	gs.CombatScene.SetPartyMemberPosition(activeIdx, references.Position{X: 5, Y: 3})
	for _, position := range []references.Position{
		{X: 5, Y: 6}, {X: 6, Y: 5}, {X: 5, Y: 4}, {X: 4, Y: 5},
	} {
		require.True(t, gs.CombatScene.AddField(NewCombatField(CombatFieldProtection, position)))
	}
	require.True(t, gs.monsterBoxedIn(monster, 0))
	rng := newSequenceRNG(3, 0, 29)
	mock.Reset()

	handled, teleported := gs.executeFleeingMonsterTurnWithRNG(monster, 0, rng)

	assert.True(t, handled)
	assert.False(t, teleported)
	assert.Equal(t, 10, monster.CurrentHP())
	assert.False(t, monster.IsFleeing())
	assert.Equal(t, center, monster.Position())
	assert.Equal(t, 1, rng.index, "a recovered boxed monster skips special and attack RNG")
	assert.Empty(t, mock.CombatEvents)
}

func isolateCombatPC(gs *GameState) int {
	activeIdx := gs.CombatScene.GetActiveCombatantIndex()
	for index := range party_state.MaxActiveParty {
		gs.CombatScene.PartyActive[index] = index == activeIdx
	}

	return activeIdx
}

func findStraightFleeTestPositions(
	t *testing.T,
	gs *GameState,
	monster *CombatMonster,
	monsterIdx int,
) (references.Position, references.Position, references.Position) {
	t.Helper()
	directions := []references.Position{{X: 1}, {X: -1}, {Y: 1}, {Y: -1}}
	for y := references.Coordinate(1); y < references.YCombatMapTiles-1; y++ {
		for x := references.Coordinate(1); x < references.XCombatMapTiles-1; x++ {
			from := references.Position{X: x, Y: y}
			for _, direction := range directions {
				away := references.Position{X: x + direction.X, Y: y + direction.Y}
				opponent := references.Position{X: x - direction.X, Y: y - direction.Y}
				if isCombatPositionValidForMonster(gs, away, monster, monsterIdx) {
					return from, away, opponent
				}
			}
		}
	}
	t.Fatal("combat fixture has no straight passable fleeing route")

	return references.Position{}, references.Position{}, references.Position{}
}
