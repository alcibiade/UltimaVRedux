package game_state

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/references"
)

// TestApplyCombatDamageToMonster_SurvivingSlimeDivides prevents ordinary hits from
// silently omitting the original SPLITS ability or desynchronizing the turn queue.
func TestApplyCombatDamageToMonster_SurvivingSlimeDivides(t *testing.T) {
	gs, mock, slime := setupCombatWithMonster(t, references.MonsterSlime, references.Down, 42)
	setRepeatedMonsterDivisionForTest(t, gs, false)
	fillCombatMapForDivisionTest(gs)
	gs.CombatScene.fields = nil
	managerCountBefore := gs.CombatScene.MonsterManager.Count()
	queueCountBefore := gs.CombatScene.TurnQueue.TotalEntryCount()
	activeIdx := gs.CombatScene.GetActiveCombatantIndex()

	gs.applyCombatDamageToMonster(
		slime,
		managerCountBefore-1,
		1,
		false,
		gs.PartyState.GetCharacter(activeIdx),
		activeIdx,
		CombatDiagnostics{},
	)

	require.Equal(t, managerCountBefore+1, gs.CombatScene.MonsterManager.Count())
	assert.Equal(t, queueCountBefore+1, gs.CombatScene.TurnQueue.TotalEntryCount())
	child := gs.CombatScene.MonsterManager.GetMonster(managerCountBefore)
	require.NotNil(t, child)
	assert.Equal(t, references.MonsterSlime, child.MonsterType())
	assert.Equal(t, slime.CurrentHP(), child.CurrentHP(), "clone inherits the parent's post-hit HP")
	assert.LessOrEqual(t, absCoordinateDifference(slime.Position().X, child.Position().X), references.Coordinate(1))
	assert.LessOrEqual(t, absCoordinateDifference(slime.Position().Y, child.Position().Y), references.Coordinate(1))
	assert.NotEqual(t, slime.Position(), child.Position())
	assert.Contains(t, mock.Messages, "Slime divides!")
	require.Len(t, mock.CombatEvents, 2, "damage and division must both be retained in the event tape")
	division := mock.CombatEvents[1]
	assert.Equal(t, CombatEventDivision, division.Kind)
	assert.Equal(t, managerCountBefore-1, division.Attacker.Index)
	assert.Equal(t, managerCountBefore, division.Target.Index)
	assert.Equal(t, slime.Position(), division.From)
	assert.Equal(t, child.Position(), division.To)

	gs.tryDivideCombatMonster(slime, managerCountBefore-1)
	gs.tryDivideCombatMonster(child, managerCountBefore)
	assert.Equal(t, managerCountBefore+1, gs.CombatScene.MonsterManager.Count(),
		"Redux mode allows only one child per encounter-created dividing monster")
}

func TestGargoyleUsesReduxSingleDivisionRule(t *testing.T) {
	gs, _, gargoyle := setupCombatWithMonster(t, references.MonsterGargoyle, references.Down, 42)
	setRepeatedMonsterDivisionForTest(t, gs, false)
	fillCombatMapForDivisionTest(gs)
	gs.CombatScene.fields = nil
	managerCountBefore := gs.CombatScene.MonsterManager.Count()
	gargoyle.SetCurrentHP(32)

	gs.tryDivideCombatMonster(gargoyle, managerCountBefore-1)

	require.Equal(t, managerCountBefore+1, gs.CombatScene.MonsterManager.Count())
	child := gs.CombatScene.MonsterManager.GetMonster(managerCountBefore)
	require.NotNil(t, child)
	assert.Equal(t, references.MonsterGargoyle, child.MonsterType())
	assert.Equal(t, 32, child.CurrentHP(), "original post-hit HP inheritance remains intact")

	gs.tryDivideCombatMonster(gargoyle, managerCountBefore-1)
	gs.tryDivideCombatMonster(child, managerCountBefore)
	assert.Equal(t, managerCountBefore+1, gs.CombatScene.MonsterManager.Count())
}

func TestRepeatedMonsterDivisionSettingRestoresOriginalBehavior(t *testing.T) {
	gs, _, slime := setupCombatWithMonster(t, references.MonsterSlime, references.Down, 42)
	fillCombatMapForDivisionTest(gs)
	setRepeatedMonsterDivisionForTest(t, gs, true)
	managerCountBefore := gs.CombatScene.MonsterManager.Count()

	gs.tryDivideCombatMonster(slime, managerCountBefore-1)
	require.Equal(t, managerCountBefore+1, gs.CombatScene.MonsterManager.Count())
	child := gs.CombatScene.MonsterManager.GetMonster(managerCountBefore)

	gs.tryDivideCombatMonster(slime, managerCountBefore-1)
	gs.tryDivideCombatMonster(child, managerCountBefore)

	assert.Equal(t, managerCountBefore+3, gs.CombatScene.MonsterManager.Count(),
		"original mode permits both parent and child to divide repeatedly")
}

func TestFailedPlacementDoesNotConsumeReduxDivision(t *testing.T) {
	gs, _, slime := setupCombatWithMonster(t, references.MonsterSlime, references.Down, 42)
	setRepeatedMonsterDivisionForTest(t, gs, false)
	fillCombatMapForDivisionTest(gs)
	gs.CombatScene.fields = nil
	managerCountBefore := gs.CombatScene.MonsterManager.Count()
	for yOffset := -1; yOffset <= 1; yOffset++ {
		for xOffset := -1; xOffset <= 1; xOffset++ {
			if xOffset == 0 && yOffset == 0 {
				continue
			}
			position := references.Position{
				X: slime.Position().X + references.Coordinate(xOffset),
				Y: slime.Position().Y + references.Coordinate(yOffset),
			}
			if position.X >= 0 && position.X < references.XCombatMapTiles &&
				position.Y >= 0 && position.Y < references.YCombatMapTiles {
				require.True(t, gs.CombatScene.AddField(NewCombatField(CombatFieldProtection, position)))
			}
		}
	}

	gs.tryDivideCombatMonster(slime, managerCountBefore-1)
	assert.Equal(t, managerCountBefore, gs.CombatScene.MonsterManager.Count())
	assert.False(t, slime.divisionConsumed)

	gs.CombatScene.fields = nil
	gs.tryDivideCombatMonster(slime, managerCountBefore-1)
	assert.Equal(t, managerCountBefore+1, gs.CombatScene.MonsterManager.Count())
	assert.True(t, slime.divisionConsumed)
}

// TestApplyCombatDamageToMonster_LethalHitDoesNotDivide prevents a killed Slime
// from creating a replacement after the death path has run.
func TestApplyCombatDamageToMonster_LethalHitDoesNotDivide(t *testing.T) {
	gs, _, slime := setupCombatWithMonster(t, references.MonsterSlime, references.Down, 42)
	managerCountBefore := gs.CombatScene.MonsterManager.Count()
	activeIdx := gs.CombatScene.GetActiveCombatantIndex()

	gs.applyCombatDamageToMonster(
		slime,
		managerCountBefore-1,
		slime.CurrentHP(),
		false,
		gs.PartyState.GetCharacter(activeIdx),
		activeIdx,
		CombatDiagnostics{},
	)

	assert.Equal(t, managerCountBefore, gs.CombatScene.MonsterManager.Count())
}

// TestResolveCombatProjectileSpell_SurvivingSlimeDivides prevents spell damage from bypassing
// the shared post-hit behavior used by the original combat damage routine.
func TestResolveCombatProjectileSpell_SurvivingSlimeDivides(t *testing.T) {
	gs, mock, slime := setupCombatWithMonster(t, references.MonsterSlime, references.Down, 42)
	fillCombatMapForDivisionTest(gs)
	slime.enemyRef.HitPoints = 100
	slime.SetCurrentHP(100)
	gs.SetRandomSeed(42)
	managerCountBefore := gs.CombatScene.MonsterManager.Count()
	activeIdx := gs.CombatScene.GetActiveCombatantIndex()

	gs.resolveCombatProjectileSpell(pendingCombatSpell{
		spell:          references.VasFlam,
		caster:         CombatTurnIdentity{IsPartyMember: true, EntityIndex: activeIdx},
		target:         CombatTurnIdentity{EntityIndex: managerCountBefore - 1},
		targetPosition: slime.Position(),
		targetSet:      true,
		hit:            CombatHitResult{Outcome: CombatHitNeverMiss, AutomaticHit: true},
	})

	require.False(t, slime.IsDead(), "test seed must leave the Slime alive")
	assert.Equal(t, managerCountBefore+1, gs.CombatScene.MonsterManager.Count())
	assert.Contains(t, mock.Messages, "Slime divides!")
	require.Len(t, mock.CombatEvents, 2, "spell and division must both be retained in the event tape")
	assert.Equal(t, CombatEventSpell, mock.CombatEvents[0].Kind)
	assert.Equal(t, CombatEventDivision, mock.CombatEvents[1].Kind)
}

func fillCombatMapForDivisionTest(gs *GameState) {
	rawData := brickFloorCombatRaw()
	gs.CombatScene.CombatMapRef.SetRawData(rawData)
}

func absCoordinateDifference(a, b references.Coordinate) references.Coordinate {
	return max(a-b, b-a)
}

func setRepeatedMonsterDivisionForTest(t *testing.T, gs *GameState, allow bool) {
	t.Helper()
	require.NotNil(t, gs.GameConfig)
	require.NotNil(t, gs.GameConfig.GameplayConfiguration)
	previous := gs.GameConfig.GameplayConfiguration.AllowRepeatedMonsterDivision
	gs.GameConfig.GameplayConfiguration.AllowRepeatedMonsterDivision = allow
	t.Cleanup(func() {
		gs.GameConfig.GameplayConfiguration.AllowRepeatedMonsterDivision = previous
	})
}
