package game_state

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/map_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
)

func TestCombatGroundDisplayMatchesOriginalDescendingBufferWrites(t *testing.T) {
	state, _ := newSyntheticDungeonTriggerCombatState(t)
	position := references.Position{X: 4, Y: 4}
	state.GetCurrentLayeredMap().SetTileByLayer(map_state.MapLayer, &position, indexes.BrickFloor)
	// A small independent buff0/buff1 oracle follows the original descending
	// scan, including its conditional write, rather than the Redux selector.
	type sourceObject struct {
		sprite      indexes.SpriteIndex
		conditional bool
		field       bool
	}
	for _, sprites := range [][]indexes.SpriteIndex{
		{indexes.Chest, indexes.ItemGem, indexes.DeadBody},
		{indexes.ItemPotion, indexes.Splat, indexes.FireField},
		{indexes.DeadBody, indexes.Splat, indexes.FireField},
		{indexes.FireField, indexes.MagicField, indexes.PoisonField},
	} {
		for _, slots := range [][3]int{{12, 20, 28}, {12, 28, 20}, {20, 12, 28}, {20, 28, 12}, {28, 12, 20}, {28, 20, 12}} {
			t.Run(fmt.Sprintf("sprites=%v/slots=%v", sprites, slots), func(t *testing.T) {
				state.CombatScene.objects, state.CombatScene.fields = nil, nil
				var original [32]sourceObject
				for index, sprite := range sprites {
					object := sourceObject{sprite: sprite, conditional: sprite == indexes.DeadBody || sprite == indexes.Splat}
					kind, field := map[indexes.SpriteIndex]CombatFieldKind{
						indexes.FireField: CombatFieldFire, indexes.MagicField: CombatFieldSleep, indexes.PoisonField: CombatFieldPoison,
					}[sprite]
					object.field = field
					object.conditional = object.conditional || field
					original[slots[index]] = object
					if field {
						entry := NewCombatField(kind, position)
						entry.originalObjectSlot = encodeOriginalObjectSlot(slots[index])
						state.CombatScene.fields = append(state.CombatScene.fields, entry)
					} else {
						entry := NewCombatDungeonObject(position, sprite, 0)
						entry.originalObjectSlot = encodeOriginalObjectSlot(slots[index])
						state.CombatScene.insertObjectByOriginalSlot(entry)
					}
				}
				buff0, winner := indexes.SpriteIndex(indexes.BrickFloor), sourceObject{}
				winnerSlot := -1
				for slot := 31; slot >= 0; slot-- {
					object := original[slot]
					if object.sprite == 0 || (object.conditional && buff0 == 0) {
						continue
					}
					winner, winnerSlot, buff0 = object, slot, 0
				}
				objects := append([]CombatObject(nil), state.CombatScene.objects...)
				fields := append([]CombatField(nil), state.CombatScene.fields...)
				calls := state.GetRNG().Calls()
				for _, elapsed := range []int64{0, 999, 1000, 2000, 60000} {
					state.ElapsedMs = elapsed
					sprite, found := state.CombatPileSprite(position)
					require.True(t, found)
					want := winner.sprite
					if winner.field {
						want = indexes.BrickFloor
					}
					assert.Equal(t, want, sprite)
					for _, field := range state.CombatScene.fields {
						assert.Equal(t, winner.field && decodeOriginalObjectSlot(field.originalObjectSlot) == winnerSlot,
							state.CombatFieldIsDisplayed(field))
					}
				}
				assert.Equal(t, objects, state.CombatScene.objects)
				assert.Equal(t, fields, state.CombatScene.fields)
				assert.Equal(t, calls, state.GetRNG().Calls())
			})
		}
	}
}

func TestCombatGroundDisplayConcealmentAndRemovalExposeNextWinner(t *testing.T) {
	state, _ := newSyntheticDungeonTriggerCombatState(t)
	position := references.Position{X: 4, Y: 4}
	field := NewCombatField(CombatFieldFire, position)
	field.originalObjectSlot = encodeOriginalObjectSlot(28)
	state.CombatScene.fields = []CombatField{field}
	body := NewCombatDeadPC(position)
	body.originalObjectSlot = encodeOriginalObjectSlot(20)
	state.CombatScene.insertObjectByOriginalSlot(body)
	chest := NewCombatChest(position, 30, true)
	chest.originalObjectSlot = encodeOriginalObjectSlot(12)
	state.CombatScene.insertObjectByOriginalSlot(chest)
	layer := state.GetCurrentLayeredMap()
	for _, terrain := range []indexes.SpriteIndex{indexes.BlackSquare, indexes.BrickWallArchway} {
		layer.SetTileByLayer(map_state.MapLayer, &position, terrain)
		_, found := state.CombatPileSprite(position)
		assert.False(t, found)
		assert.False(t, state.CombatFieldIsDisplayed(field))
	}
	layer.SetTileByLayer(map_state.MapLayer, &position, indexes.BrickFloor)
	sprite, found := state.CombatPileSprite(position)
	require.True(t, found)
	assert.Equal(t, indexes.SpriteIndex(indexes.Chest), sprite)
	assert.False(t, state.CombatFieldIsDisplayed(field))
	state.CombatScene.takeFirstObjectAt(position, CombatObject.IsChest)
	sprite, found = state.CombatPileSprite(position)
	require.True(t, found)
	assert.Equal(t, indexes.SpriteIndex(indexes.BrickFloor), sprite, "field has terrain underneath, not a duplicate body")
	assert.True(t, state.CombatFieldIsDisplayed(field))
	state.CombatScene.removeFirstFieldAt(position)
	sprite, found = state.CombatPileSprite(position)
	require.True(t, found)
	assert.Equal(t, indexes.SpriteIndex(indexes.DeadBody), sprite)
}
