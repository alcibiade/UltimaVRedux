package game_state

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
)

type strictDungeonCombatRNG struct {
	t      *testing.T
	values []int
	bounds []int
	index  int
}

func (r *strictDungeonCombatRNG) Intn(bound int) int {
	r.t.Helper()
	require.Less(r.t, r.index, len(r.values), "unexpected RNG call with bound %d", bound)
	require.Equal(r.t, r.bounds[r.index], bound, "RNG call %d bound", r.index)
	value := r.values[r.index]
	require.GreaterOrEqual(r.t, value, 0)
	require.Less(r.t, value, bound)
	r.index++

	return value
}

func (r *strictDungeonCombatRNG) requireExhausted() {
	r.t.Helper()
	assert.Equal(r.t, len(r.values), r.index, "scripted RNG calls consumed")
}

func TestDungeonWandererCombatGroupRangesAndExactRandomOrder(t *testing.T) {
	tests := []struct {
		monster references.MonsterType
		maximum int
		forced  bool
	}{
		{monster: references.MonsterRat, maximum: 10},
		{monster: references.MonsterBat, maximum: 16, forced: true},
		{monster: references.MonsterSpider, maximum: 4},
		{monster: references.MonsterGhost, maximum: 6},
		{monster: references.MonsterSlime, maximum: 16, forced: true},
		{monster: references.MonsterGremlin, maximum: 13},
		{monster: references.MonsterGazer, maximum: 4},
		{monster: references.MonsterReaper, maximum: 3},
	}
	for _, test := range tests {
		t.Run(test.monster.Name(), func(t *testing.T) {
			for _, countRoll := range []int{0, test.maximum - 1} {
				values := make([]int, 17)
				bounds := make([]int, 17)
				for index := range references.MaxMonstersPerMap {
					values[index] = (index*7 + 3) % references.MaxMonstersPerMap
					bounds[index] = references.MaxMonstersPerMap
				}
				values[16] = countRoll
				bounds[16] = test.maximum
				rng := &strictDungeonCombatRNG{t: t, values: values, bounds: bounds}

				selected, count := dungeonWandererCombatSlots(test.monster, rng)

				wantCount := countRoll + 1
				if test.forced {
					wantCount = test.maximum
				}
				assert.Equal(t, wantCount, count)
				selectedCount := 0
				for _, occupied := range selected {
					if occupied {
						selectedCount++
					}
				}
				assert.Equal(t, wantCount, selectedCount)
				rng.requireExhausted()
			}
		})
	}
}

func TestPrepareDungeonWandererCombatStateUsesSelectedSlotsAndSharedDexJitter(t *testing.T) {
	var cells references.DungeonCells
	state := newDungeonStateTestGame(t, cells)
	state.activeDungeon.facing = references.DungeonFacingNorth
	enemyReferences := make(references.EnemyReferences, int(references.MonsterShadowlord)+1)
	enemyReferences[references.MonsterRat] = references.EnemyReference{
		MonsterType: references.MonsterRat,
		Dexterity:   20,
		HitPoints:   10,
		MaxPerMap:   1,
		Friend:      &enemyReferences[references.MonsterTroll],
	}
	state.GameReferences = &references.GameReferences{EnemyReferences: &enemyReferences}

	values := append(make([]int, 0, 24),
		15, 15, 15, 15, 15, 15, 15, 15, 15, 15, 15, 15, 15, 15, 15, 15,
		2,          // Count roll produces three Rats.
		7, 6, 5, 4, // Unused dng_precom replacement-monster rolls.
		0, 7, 4, // DEX jitter for slots 0, 1, and 15 in ascending slot order.
	)
	bounds := append(make([]int, 0, 24),
		16, 16, 16, 16, 16, 16, 16, 16, 16, 16, 16, 16, 16, 16, 16, 16,
		10,
		8, 8, 8, 8,
		8, 8, 8,
	)
	rng := &strictDungeonCombatRNG{t: t, values: values, bounds: bounds}

	combat, err := state.prepareDungeonWandererCombatState(references.MonsterRat, rng)
	require.NoError(t, err)
	rng.requireExhausted()

	require.Equal(t, 3, combat.MonsterManager.Count())
	wantSlots := []int{0, 1, 15}
	wantDexterity := []int{16, 23, 20}
	for monsterIndex, slot := range wantSlots {
		monster := combat.MonsterManager.GetMonster(monsterIndex)
		require.NotNil(t, monster)
		assert.Equal(t, references.MonsterRat, monster.MonsterType(), "groups remain homogeneous")
		assert.Equal(t, combat.CombatMapRef.MonsterPositions[slot], monster.Position())
		assert.Equal(t, wantDexterity[monsterIndex], monster.CurrentDex())
	}
	_, triggerSlotsPresent := combat.CombatMapRef.DungeonTriggerSlots()
	assert.False(t, triggerSlotsPresent)
	assert.Equal(t, CombatEntryNorth, combat.EntryDirection)
	assert.Equal(t, combat.CombatMapRef.PlayerPositions[int(CombatEntryNorth)][0], combat.PartyPositions[0])
	_, queuedMonsters := combat.TurnQueue.CountRemaining()
	assert.Equal(t, 3, queuedMonsters)
}

func TestPrepareDungeonWandererMaterializesSourceDepthCenterChestBeforeMonsters(t *testing.T) {
	var cells references.DungeonCells
	position := references.Position{X: 1, Y: 1}
	cells.SetCell(7, position, 0x40)
	state := newDungeonStateTestGame(t, cells)
	state.MapState.PlayerLocation.Floor = 7
	enemyReferences := make(references.EnemyReferences, int(references.MonsterShadowlord)+1)
	enemyReferences[references.MonsterRat] = references.EnemyReference{
		MonsterType: references.MonsterRat, Dexterity: 20, HitPoints: 10, MaxPerMap: 1,
	}
	state.GameReferences = &references.GameReferences{EnemyReferences: &enemyReferences}
	values := append(make([]int, references.MaxMonstersPerMap),
		0,          // One Rat.
		0, 0, 0, 0, // Unused replacement choices.
		0, // Rat DEX jitter.
	)
	bounds := append(make([]int, references.MaxMonstersPerMap),
		10,
		8, 8, 8, 8,
		8,
	)
	for index := range references.MaxMonstersPerMap {
		bounds[index] = references.MaxMonstersPerMap
	}
	rng := &strictDungeonCombatRNG{t: t, values: values, bounds: bounds}

	combat, err := state.prepareDungeonWandererCombatState(references.MonsterRat, rng)
	require.NoError(t, err)
	rng.requireExhausted()

	center := references.Position{X: 5, Y: 5}
	chest, found := combat.objectAtMatching(center, CombatObject.IsChest)
	require.True(t, found)
	assert.Equal(t, 28, chest.TreasureValue())
	assert.False(t, chest.IsLockedAndTrapped())
	assert.Equal(t, indexes.SpriteIndex(0x45), combat.CombatMapRef.GetTileNumber(center))
	assert.Equal(t, uint8(2), chest.originalObjectSlot, "the chest follows the one living PC")
	monsterEntry := combat.TurnQueue.GetEntry(0, false)
	require.NotNil(t, monsterEntry)
	assert.Equal(t, uint8(3), monsterEntry.originalObjectSlot, "monsters are admitted after the chest")
}
