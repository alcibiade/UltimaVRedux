package game_state

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
)

func TestCombatAnSanctUnlocksDoorAndDisarmsChest(t *testing.T) {
	gs, mock, _ := setupCombatWithMonster(t, references.MonsterOrc, references.Down, 42)
	casterIdx := gs.CombatScene.GetActiveCombatantIndex()
	origin := references.Position{X: 5, Y: 5}
	door := references.Position{X: 6, Y: 5}
	chest := references.Position{X: 4, Y: 5}
	gs.CombatScene.SetPartyMemberPosition(casterIdx, origin)
	gs.CombatScene.CombatMapRef.SetTileNumber(door, indexes.LockedDoorView)
	gs.CombatScene.AddObject(NewCombatChest(chest, 10, true))
	mock.Reset()

	assert.Equal(t, CombatUtilityDoorUnlocked, gs.CastCombatAnSanct(casterIdx, door))
	assert.Equal(t, indexes.SpriteIndex(indexes.RegularDoorView), gs.CombatScene.CombatMapRef.GetTileNumber(door))
	assert.Equal(t, CombatUtilityChestDisarmed, gs.CastCombatAnSanct(casterIdx, chest))
	object, found := gs.CombatScene.ObjectAt(chest)
	require.True(t, found)
	assert.False(t, object.IsLockedAndTrapped())
	require.Len(t, mock.CombatEvents, 2)
	assert.Equal(t, CombatEventUtilitySpell, mock.CombatEvents[0].Kind)
	assert.Equal(t, CombatUtilityChestDisarmed, mock.CombatEvents[1].UtilityResult)
}

func TestCombatMagicLockSpellsAreOriginalSemanticInverses(t *testing.T) {
	gs, mock, _ := setupCombatWithMonster(t, references.MonsterOrc, references.Down, 42)
	casterIdx := gs.CombatScene.GetActiveCombatantIndex()
	target := references.Position{X: 6, Y: 5}
	gs.CombatScene.CombatMapRef.SetTileNumber(target, indexes.RegularDoor)
	mock.Reset()

	gs.CastCombatAnExPor(casterIdx, target)
	assert.Equal(t, indexes.SpriteIndex(indexes.MagicLockDoor), gs.CombatScene.CombatMapRef.GetTileNumber(target))
	gs.CastCombatInExPor(casterIdx, target)
	assert.Equal(t, indexes.SpriteIndex(indexes.RegularDoor), gs.CombatScene.CombatMapRef.GetTileNumber(target))
	require.Len(t, mock.CombatEvents, 2)
	assert.Equal(t, references.AnExPor, mock.CombatEvents[0].Diagnostics.Spell)
	assert.Equal(t, references.InExPor, mock.CombatEvents[1].Diagnostics.Spell)
}

func TestCombatUtilityAdjacentTargetingDefersResources(t *testing.T) {
	tests := []struct {
		spell references.Spell
		tile  indexes.SpriteIndex
	}{
		{references.AnSanct, indexes.LockedDoor},
		{references.AnExPor, indexes.RegularDoor},
		{references.InExPor, indexes.MagicLockDoor},
	}
	for _, test := range tests {
		t.Run(references.GetSpellName(test.spell), func(t *testing.T) {
			gs, _, _ := setupCombatWithMonster(t, references.MonsterOrc, references.Down, 42)
			casterIdx := gs.CombatScene.GetActiveCombatantIndex()
			caster := gs.PartyState.GetCharacter(casterIdx)
			caster.Level = 8
			caster.SetMp(30)
			gs.PartyState.Inventory.Spells.Set(test.spell, 1)
			origin := references.Position{X: 5, Y: 5}
			target := references.Position{X: 6, Y: 5}
			gs.CombatScene.SetPartyMemberPosition(casterIdx, origin)
			gs.CombatScene.CombatMapRef.SetTileNumber(target, test.tile)
			mpBefore := caster.GetCurrentMp()

			require.True(t, gs.BeginCombatAdjacentSpellTargeting(CastRequest{Spell: test.spell, CasterIdx: casterIdx}).Valid)
			gs.SetCombatAdjacentSpellDirection(references.Right)
			assert.Equal(t, mpBefore, caster.GetCurrentMp())
			result := gs.ConfirmCombatAdjacentSpellTarget()

			assert.Equal(t, CombatAdjacentSpellTargetResolved, result.Outcome)
			assert.Equal(t, mpBefore-byte(references.GetSpellCircle(test.spell)), caster.GetCurrentMp())
			assert.Zero(t, gs.PartyState.Inventory.Spells.Get(test.spell))
		})
	}
}

func TestPlanAndApplyCombatInPorUsesValidRandomDestination(t *testing.T) {
	gs, mock, _ := setupCombatWithMonster(t, references.MonsterOrc, references.Down, 42)
	gs.CombatScene.CombatMapRef = newOpenCombatConeMap()
	casterIdx := gs.CombatScene.GetActiveCombatantIndex()
	gs.SetRandomSeed(1)
	mock.Reset()

	result := gs.PlanCombatInPor(casterIdx)
	require.Equal(t, CombatInPorResolved, result.Outcome)
	assert.True(t, gs.isCombatPositionValid(&result.To, casterIdx))
	assert.Equal(t, result.From, gs.CombatScene.GetPartyMemberPosition(casterIdx))

	gs.ApplyCombatInPor(result)
	assert.Equal(t, result.To, gs.CombatScene.GetPartyMemberPosition(casterIdx))
	require.Len(t, mock.CombatEvents, 1)
	assert.Equal(t, CombatUtilityTeleported, mock.CombatEvents[0].UtilityResult)
}

func TestPlanAndApplyCombatInPorMovesExactCloneCaster(t *testing.T) {
	gs, mock, _ := setupCombatWithMonster(t, references.MonsterOrc, references.Down, 42)
	gs.CombatScene.CombatMapRef = newOpenCombatConeMap()
	source := gs.CombatScene.GetCurrentTurnIdentity()
	basePosition := references.Position{X: 1, Y: 1}
	clonePosition := references.Position{X: 2, Y: 2}
	gs.CombatScene.SetPartyMemberPosition(source.EntityIndex, basePosition)
	clone := gs.CombatScene.addPCCloneFromSource(
		source,
		source.EntityIndex,
		clonePosition,
		NewCombatPCProvider(gs.PartyState.GetCharacter(source.EntityIndex)),
	)
	gs.CombatScene.CurrentTurnIdentity = clone
	gs.CombatScene.currentTurnIdentitySet = true
	gs.SetRandomSeed(1)
	mock.Reset()

	result := gs.PlanCombatInPor(source.EntityIndex)
	require.Equal(t, CombatInPorResolved, result.Outcome)
	assert.Equal(t, clone, result.Caster)
	assert.Equal(t, clonePosition, result.From)
	gs.ApplyCombatInPor(result)

	assert.Equal(t, basePosition, gs.CombatScene.GetPartyMemberPosition(source.EntityIndex))
	assert.Equal(t, result.To, gs.CombatScene.GetCombatantPosition(clone))
	require.Len(t, mock.CombatEvents, 1)
	assert.Equal(t, clone.Slot, mock.CombatEvents[0].Attacker.Slot)
}

func TestCombatInPorRejectsDungeonBeforeConsumption(t *testing.T) {
	gs, _, _ := setupCombatWithMonster(t, references.MonsterOrc, references.Down, 42)
	gs.CombatScene.ExitContext = CombatExitDungeonEncounter
	casterIdx := gs.CombatScene.GetActiveCombatantIndex()
	caster := gs.PartyState.GetCharacter(casterIdx)
	caster.Level = 8
	caster.SetMp(30)
	gs.PartyState.Inventory.Spells.Set(references.InPor, 1)
	mpBefore := caster.GetCurrentMp()

	validation := gs.ValidateAndConsumeSpellResources(CastRequest{Spell: references.InPor, CasterIdx: casterIdx})

	assert.False(t, validation.Valid)
	assert.Equal(t, "Not here!", validation.ErrorMsg)
	assert.Equal(t, mpBefore, caster.GetCurrentMp())
	assert.Equal(t, uint16(1), gs.PartyState.Inventory.Spells.Get(references.InPor))
}

func TestCombatInPorFailsAfterSevenBlockedAttempts(t *testing.T) {
	gs, _, _ := setupCombatWithMonster(t, references.MonsterOrc, references.Down, 42)
	for x := range references.XCombatMapTiles {
		for y := range references.YCombatMapTiles {
			gs.setCombatTerrain(references.Position{
				X: references.Coordinate(x), Y: references.Coordinate(y),
			}, indexes.BlackSquare)
		}
	}
	casterIdx := gs.CombatScene.GetActiveCombatantIndex()
	from := gs.CombatScene.GetPartyMemberPosition(casterIdx)

	result := gs.PlanCombatInPor(casterIdx)

	assert.Equal(t, CombatInPorFailed, result.Outcome)
	assert.Equal(t, from, result.To)
}
