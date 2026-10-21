package game_state

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
)

func TestCastCombatAnYlemRemovesOnlyOriginalObjectTiles(t *testing.T) {
	gs, _, _ := setupCombatWithMonster(t, references.MonsterOrc, references.Down, 42)
	activeIdx := gs.CombatScene.GetActiveCombatantIndex()
	origin := references.Position{X: 5, Y: 5}
	target := references.Position{X: 5, Y: 6}
	gs.CombatScene.SetPartyMemberPosition(activeIdx, origin)
	gs.CombatScene.CombatMapRef.SetTileNumber(target, indexes.Plant)

	assert.True(t, gs.CastCombatAnYlem(activeIdx, references.Down))
	assert.Equal(t, indexes.SpriteIndex(indexes.BrickFloor), gs.CombatScene.CombatMapRef.GetTileNumber(target))

	gs.CombatScene.CombatMapRef.SetTileNumber(target, indexes.TableLeft)
	assert.False(t, gs.CastCombatAnYlem(activeIdx, references.Down))
	assert.Equal(t, indexes.SpriteIndex(indexes.TableLeft), gs.CombatScene.CombatMapRef.GetTileNumber(target))
}

func TestCastCombatAnYlemUsesExactCloneOrigin(t *testing.T) {
	gs, _, _ := setupCombatWithMonster(t, references.MonsterOrc, references.Down, 42)
	source := gs.CombatScene.GetCurrentTurnIdentity()
	clonePosition := references.Position{X: 2, Y: 2}
	target := references.Position{X: 2, Y: 3}
	clone := gs.CombatScene.addPCCloneFromSource(
		source,
		source.EntityIndex,
		clonePosition,
		NewCombatPCProvider(gs.PartyState.GetCharacter(source.EntityIndex)),
	)
	gs.CombatScene.CurrentTurnIdentity = clone
	gs.CombatScene.currentTurnIdentitySet = true
	gs.CombatScene.CombatMapRef.SetTileNumber(target, indexes.Plant)

	require.True(t, gs.CastCombatAnYlem(source.EntityIndex, references.Down))
	assert.Equal(t, indexes.SpriteIndex(indexes.BrickFloor), gs.CombatScene.CombatMapRef.GetTileNumber(target))
}

func TestCastCombatWisQuasRevealsActiveInvisibleMonsters(t *testing.T) {
	gs, _, monster := setupCombatWithMonster(t, references.MonsterOrc, references.Down, 42)
	monster.SetInvisible(true)

	assert.Equal(t, []references.Position{monster.Position()}, gs.CastCombatWisQuas())
	assert.False(t, monster.IsInvisible())
	assert.Empty(t, gs.CastCombatWisQuas())
}

func TestCombatAdjacentSpellTargetingAnYlemDefersResourcesUntilValidConfirmation(t *testing.T) {
	gs, _, _ := setupCombatWithMonster(t, references.MonsterOrc, references.Down, 42)
	activeIdx := gs.CombatScene.GetActiveCombatantIndex()
	caster := gs.PartyState.GetCharacter(activeIdx)
	caster.Level = 1
	caster.SetMp(10)
	gs.PartyState.Inventory.Spells.Set(references.AnYlem, 1)
	origin := references.Position{X: 5, Y: 5}
	target := references.Position{X: 5, Y: 6}
	gs.CombatScene.SetPartyMemberPosition(activeIdx, origin)
	gs.CombatScene.CombatMapRef.SetTileNumber(target, indexes.Plant)
	mpBefore := caster.GetCurrentMp()

	require.True(t, gs.BeginCombatAdjacentSpellTargeting(CastRequest{Spell: references.AnYlem, CasterIdx: activeIdx}).Valid)
	assert.Equal(t, mpBefore, caster.GetCurrentMp())
	assert.Equal(t, uint16(1), gs.PartyState.Inventory.Spells.Get(references.AnYlem))
	gs.SetCombatAdjacentSpellDirection(references.Down)
	result := gs.ConfirmCombatAdjacentSpellTarget()

	assert.Equal(t, CombatAdjacentSpellTargetResolved, result.Outcome)
	assert.Equal(t, target, result.Target)
	assert.Equal(t, mpBefore-1, caster.GetCurrentMp())
	assert.Zero(t, gs.PartyState.Inventory.Spells.Get(references.AnYlem))
}

func TestCastCombatInVasPorYlemDamagesEligibleHostileMonster(t *testing.T) {
	gs, mock, monster := setupCombatWithMonster(t, references.MonsterOrc, references.Down, 42)
	activeIdx := gs.CombatScene.GetActiveCombatantIndex()
	monster.currentDex = 1
	gs.SetRandomSeed(1)
	mock.Reset()
	hpBefore := monster.CurrentHP()

	affected := gs.CastCombatInVasPorYlem(activeIdx)

	assert.Contains(t, affected, monster.Position())
	require.Len(t, mock.CombatEvents, 1)
	event := mock.CombatEvents[0]
	assert.Equal(t, references.InVasPorYlem, event.Diagnostics.Spell)
	assert.GreaterOrEqual(t, event.Diagnostics.Damage.AttackRoll, 1)
	assert.LessOrEqual(t, event.Diagnostics.Damage.AttackRoll, 20)
	assert.Equal(t, hpBefore-event.HPLost, monster.CurrentHP())
}
