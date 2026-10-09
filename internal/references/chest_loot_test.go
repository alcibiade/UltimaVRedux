package references

import (
	"testing"

	"github.com/bradhannah/UltimaVRedux/internal/deterministic_random"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type chestDraw struct{ bound, result int }
type chestDrawScript struct {
	t     *testing.T
	draws []chestDraw
	calls int
}

func (script *chestDrawScript) Intn(bound int) int {
	script.t.Helper()
	require.Less(script.t, script.calls, len(script.draws), "unexpected loot RNG draw")
	draw := script.draws[script.calls]
	script.calls++
	require.Equal(script.t, draw.bound, bound, "draw %d range", script.calls)
	require.GreaterOrEqual(script.t, draw.result, 0)
	require.Less(script.t, draw.result, bound)
	return draw.result
}

func TestChestLootMiscellaneousSourceOrderAndReplacementDraws(t *testing.T) {
	script := &chestDrawScript{t: t, draws: []chestDraw{
		{30, 6}, {2, 1}, // food threshold 7, quantity 2
		{30, 6}, {2, 0}, // torches
		{30, 14}, {2, 1}, // gems
		{30, 8}, {2, 0}, // keys
		{30, 16}, {8, 7}, // scroll type 7, quantity one
		{30, 16}, {8, 6}, // potion type 6, quantity one
		{30, 2}, {90, 89}, {90, 0}, // discarded 90 replaced with one gold
		{30, 24}, {10, 9}, {30, 6}, // discarded 10 replaced with chest value 7
	}}
	for range 16 {
		script.draws = append(script.draws, chestDraw{48, int(Ankh)})
	}
	loot := RollChestLoot(30, script)
	require.Len(t, loot, 8)
	want := []ItemAndQuantity{
		{Item: Food, Quantity: 2}, {Item: Torches, Quantity: 1},
		{Item: Gem, Quantity: 2}, {Item: Key, Quantity: 1},
		{Item: Scroll(7), Quantity: 1}, {Item: Potion(6), Quantity: 1},
		{Item: Gold, Quantity: 1},
	}
	for index, item := range want {
		assert.Equal(t, item, loot[index].Item)
		assert.Zero(t, loot[index].ChestValue)
	}
	assert.Nil(t, loot[7].Item.Item)
	assert.Equal(t, byte(7), loot[7].ChestValue)
	assert.Equal(t, len(script.draws), script.calls)
}

func TestChestLootArmamentExclusionsThresholdAndAttemptCount(t *testing.T) {
	script := &chestDrawScript{t: t}
	for range 8 { // all miscellaneous checks fail, with no quantity draws
		script.draws = append(script.draws, chestDraw{30, 0})
	}
	for _, item := range []Equipment{JewelShield, MysticArmour, ChaosSword, GlassSword, JeweledSword, MysticSword, Ankh} {
		script.draws = append(script.draws, chestDraw{48, int(item)})
	}
	script.draws = append(script.draws,
		chestDraw{48, int(MagicShield)}, chestDraw{30, 26}, // 27 fails
		chestDraw{48, int(MagicShield)}, chestDraw{30, 27}, // 28 accepts
	)
	for range 7 {
		script.draws = append(script.draws, chestDraw{48, int(Ankh)})
	}
	loot := RollChestLoot(30, script)
	require.Equal(t, []ChestLoot{{Item: ItemAndQuantity{Item: MagicShield, Quantity: 1}}}, loot)
	assert.Equal(t, len(script.draws), script.calls)
}

func TestChestLootValueShortCircuitAndEmptyResult(t *testing.T) {
	// Value two skips every misc gate and both equipment acceptance draws.
	script := &chestDrawScript{t: t, draws: []chestDraw{{48, int(MagicShield)}, {48, int(Ankh)}}}
	assert.Empty(t, RollChestLoot(2, script))
	assert.Equal(t, 2, script.calls)
}

func TestChestLootMiscellaneousFailuresDoNotDrawQuantities(t *testing.T) {
	script := &chestDrawScript{t: t, draws: []chestDraw{
		{30, 5}, {30, 5}, {30, 13}, {30, 7},
		{30, 15}, {30, 15}, {30, 1}, {30, 23},
	}}
	for range 16 {
		script.draws = append(script.draws, chestDraw{48, int(Ankh)})
	}
	assert.Empty(t, RollChestLoot(30, script))
	assert.Equal(t, len(script.draws), script.calls)
}

func TestChestLootUsesInjectedDeterministicRNG(t *testing.T) {
	first := deterministic_random.New(8675309)
	second := deterministic_random.New(8675309)
	assert.Equal(t, RollChestLoot(30, first), RollChestLoot(30, second))
	assert.Positive(t, first.Calls())
	assert.Equal(t, first.Calls(), second.Calls())
}

func TestChestLootGoldAndChildReachParentValueMaximums(t *testing.T) {
	script := &chestDrawScript{t: t}
	for range 6 {
		script.draws = append(script.draws, chestDraw{30, 0})
	}
	script.draws = append(script.draws,
		chestDraw{30, 2}, chestDraw{90, 0}, chestDraw{90, 89},
		chestDraw{30, 24}, chestDraw{10, 0}, chestDraw{30, 29},
	)
	for range 16 {
		script.draws = append(script.draws, chestDraw{48, int(Ankh)})
	}
	assert.Equal(t, []ChestLoot{
		{Item: ItemAndQuantity{Item: Gold, Quantity: 90}},
		{ChestValue: 30},
	}, RollChestLoot(30, script))
	assert.Equal(t, len(script.draws), script.calls)
}
