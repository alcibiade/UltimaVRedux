package game_state

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/party_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
)

func TestCombatPilePreviewRetainsSlotsRNGAndPickupAuthority(t *testing.T) {
	state, _, _, _, position := setupPlayerRangedAttackTest(t, references.MorningStar, references.MonsterOrc, 1)
	state.CombatScene.AttackTargeting = nil
	require.True(t, state.CombatScene.AddObject(NewCombatLooseItem(position, references.Blue, 1, indexes.ItemPotion)))
	require.True(t, state.CombatScene.AddObject(NewCombatLooseItem(position, references.Gem, 2, indexes.ItemGem)))
	before := append([]CombatObject(nil), state.CombatScene.objects...)
	calls := state.GetRNG().Calls()
	for _, elapsed := range []int64{0, 999, 1000, 2000, 60000} {
		state.ElapsedMs = elapsed
		got, found := state.CombatPileSprite(position)
		require.True(t, found)
		assert.Equal(t, indexes.SpriteIndex(indexes.ItemPotion), got)
	}
	assert.Equal(t, before, state.CombatScene.objects)
	assert.Equal(t, calls, state.GetRNG().Calls())
	state.PartyState.Inventory.Potions.Set(references.Blue, 98)
	result := state.GetCombatItem(references.Right)
	assert.Equal(t, uint16(1), result.Acquired)
	assert.Equal(t, indexes.SpriteIndex(indexes.ItemPotion), result.Sprite)
	got, found := state.CombatPileSprite(position)
	require.True(t, found)
	assert.Equal(t, indexes.SpriteIndex(indexes.ItemGem), got)
	state.PartyState.Inventory.Provisions.Gems.Set(98)
	result = state.GetCombatItem(references.Right)
	assert.Equal(t, uint16(1), result.Acquired, "receipt uses capped committed quantity")
	assert.Contains(t, result.Message, "+1")
}

func TestCombatOpenPublishesOneIconPerAdmittedDescription(t *testing.T) {
	state, _, _, _, position := setupPlayerRangedAttackTest(t, references.MorningStar, references.MonsterOrc, 1)
	state.CombatScene.AttackTargeting = nil
	require.True(t, state.CombatScene.AddObject(NewCombatChest(position, 30, false)))
	state.SetRandomSeed(624)
	result := state.ActionOpenCombatMap(references.Right)
	require.True(t, result.Handled)
	require.Greater(t, len(result.RevealItems), 1)
	require.Len(t, result.RevealItems, len(result.RevealLines))
	for index, entry := range result.RevealItems {
		assert.Equal(t, result.RevealLines[index], entry.Description)
		object := state.CombatScene.objects[len(state.CombatScene.objects)-1-index]
		assert.Equal(t, object.Sprite(), entry.Sprite)
	}
}

func TestCombatMixedPilePrecedenceAndMarkerFallback(t *testing.T) {
	for _, reverse := range []bool{false, true} {
		state, _, _, _, position := setupPlayerRangedAttackTest(t, references.MorningStar, references.MonsterOrc, 1)
		state.CombatScene.AttackTargeting = nil
		objects := []CombatObject{
			NewCombatDeadPC(position), NewCombatCorpse(position),
			NewCombatLooseItem(position, references.Blue, 1, indexes.ItemPotion),
			NewCombatLooseItem(position, references.Gem, 2, indexes.ItemGem),
			NewCombatChest(position, 30, false),
		}
		// Deliberately admit out of order. Display uses original object slots slots,
		// not insertion order, ObjectAt's reverse scan, or a chest-first policy.
		// See ref:COMBAT-0010.
		for index := range objects {
			objects[index].originalObjectSlot = encodeOriginalObjectSlot([]int{3, 25, 10, 20, 15}[index])
		}
		for i := range objects {
			if reverse {
				state.CombatScene.insertObjectByOriginalSlot(objects[len(objects)-1-i])
			} else {
				state.CombatScene.insertObjectByOriginalSlot(objects[i])
			}
		}
		before := append([]CombatObject(nil), state.CombatScene.objects...)
		calls := state.GetRNG().Calls()
		for _, elapsed := range []int64{-1, 0, 999, 1000, 2000, 3000} {
			state.ElapsedMs = elapsed
			sprite, found := state.CombatPileSprite(position)
			require.True(t, found)
			assert.Equal(t, indexes.SpriteIndex(indexes.ItemPotion), sprite)
			assert.True(t, state.CombatScene.HasBlockingObjectAt(position))
		}
		assert.Equal(t, before, state.CombatScene.objects)
		assert.Equal(t, calls, state.GetRNG().Calls())
		state.CombatScene.takeFirstObjectAt(position, func(o CombatObject) bool { return o.IsChest() })
		for _, elapsed := range []int64{0, 999, 1000, 2000} {
			state.ElapsedMs = elapsed
			sprite, found := state.CombatPileSprite(position)
			require.True(t, found)
			assert.Equal(t, indexes.SpriteIndex(indexes.ItemPotion), sprite)
		}
		for range 2 {
			require.True(t, state.GetCombatItem(references.Right).Handled)
		}
		for _, elapsed := range []int64{0, 999, 1000, 2000} {
			state.ElapsedMs = elapsed
			sprite, found := state.CombatPileSprite(position)
			require.True(t, found)
			assert.Equal(t, indexes.SpriteIndex(indexes.Splat), sprite)
		}
		assert.False(t, state.CombatScene.HasBlockingObjectAt(position))
		state.CombatScene.takeFirstObjectAt(position, func(o CombatObject) bool { return o.IsCorpse() })
		sprite, found := state.CombatPileSprite(position)
		require.True(t, found)
		assert.Equal(t, indexes.SpriteIndex(indexes.DeadBody), sprite)
	}
}

func TestCombatPileDisplayUsesLowestOrdinarySlotWithoutChangingObjectAt(t *testing.T) {
	state, _, _, _, position := setupPlayerRangedAttackTest(t, references.MorningStar, references.MonsterOrc, 1)
	state.CombatScene.AttackTargeting = nil
	potion := NewCombatLooseItem(position, references.Blue, 1, indexes.ItemPotion)
	potion.originalObjectSlot = encodeOriginalObjectSlot(25)
	chest := NewCombatChest(position, 30, false)
	chest.originalObjectSlot = encodeOriginalObjectSlot(10)
	state.CombatScene.insertObjectByOriginalSlot(potion)
	state.CombatScene.insertObjectByOriginalSlot(chest)
	selected, found := state.CombatScene.ObjectAt(position)
	require.True(t, found)
	require.NotZero(t, selected.LifetimeID())
	potion.lifetimeID = selected.LifetimeID() // Admission supplies the encounter-local lifetime.
	assert.Equal(t, potion, selected, "existing action lookup is not display authority")
	for _, elapsed := range []int64{0, 999, 1000, 2000, 60000} {
		state.ElapsedMs = elapsed
		sprite, found := state.CombatPileSprite(position)
		require.True(t, found)
		assert.Equal(t, indexes.SpriteIndex(indexes.Chest), sprite)
	}
	state.CombatScene.takeFirstObjectAt(position, CombatObject.IsChest)
	sprite, found := state.CombatPileSprite(position)
	require.True(t, found)
	assert.Equal(t, indexes.SpriteIndex(indexes.ItemPotion), sprite)
}

func TestCombatGetOpensAndCollectsLootAboveDeadPartyMarker(t *testing.T) {
	for _, bodyFirst := range []bool{false, true} {
		state, _, _, _, position := setupPlayerRangedAttackTest(t, references.MorningStar, references.MonsterOrc, 1)
		state.CombatScene.AttackTargeting = nil
		body := NewCombatDeadPC(position)
		chest := NewCombatChest(position, 30, false)
		if bodyFirst {
			require.True(t, state.CombatScene.AddObject(body))
			require.True(t, state.CombatScene.AddObject(chest))
		} else {
			require.True(t, state.CombatScene.AddObject(chest))
			require.True(t, state.CombatScene.AddObject(body))
		}
		body, _ = state.CombatScene.objectAtMatching(position, func(o CombatObject) bool { return o.Sprite() == indexes.DeadBody })
		before := state.GetCombatItem(references.Right)
		assert.True(t, before.Handled)
		assert.Equal(t, "Open it first!", before.Message)
		assert.Equal(t, indexes.SpriteIndex(indexes.Chest), before.Sprite)
		state.SetRandomSeed(624)
		opened := state.ActionOpenCombatMap(references.Right)
		require.True(t, opened.Handled)
		require.Greater(t, len(opened.RevealItems), 1)
		for range len(opened.RevealItems) {
			state.CombatPileSprite(position)
			picked := state.GetCombatItem(references.Right)
			require.True(t, picked.Handled)
			assert.NotEqual(t, indexes.SpriteIndex(indexes.DeadBody), picked.Sprite)
			assert.NotEqual(t, "Nothing to get!", picked.Message)
		}
		assert.Equal(t, []CombatObject{body}, state.CombatScene.objects)
		inventory := state.PartyState.Inventory.LockerSnapshot()
		for range 2 {
			picked := state.GetCombatItem(references.Right)
			assert.False(t, picked.Handled)
			assert.Equal(t, "Nothing to get!", picked.Message)
			assert.Zero(t, picked.Acquired)
		}
		assert.Equal(t, inventory, state.PartyState.Inventory.LockerSnapshot())
		assert.Equal(t, []CombatObject{body}, state.CombatScene.objects)
	}
}

func TestCombatGetPreservesActualPartyDeathAndOriginalMarkerSlot(t *testing.T) {
	state, _, _, _, position := setupPlayerRangedAttackTest(t, references.MorningStar, references.MonsterOrc, 1)
	state.CombatScene.AttackTargeting = nil
	deadIndex := (state.CombatScene.GetActiveCombatantIndex() + 1) % party_state.MaxActiveParty
	deadIdentity := CombatTurnIdentity{IsPartyMember: true, EntityIndex: deadIndex}
	state.CombatScene.SetPartyMemberPosition(deadIndex, position)
	character := state.PartyState.GetCharacter(deadIndex)
	character.SetHp(0)
	character.MarkAsDead()
	state.markCombatPCDead(deadIndex)
	body, found := state.CombatScene.objectAtMatching(position, func(o CombatObject) bool { return o.Sprite() == indexes.DeadBody })
	require.True(t, found)
	require.True(t, state.CombatScene.AddObject(NewCombatLooseItem(position, references.Gold, 3, indexes.ItemMoney)))
	goldBefore := state.PartyState.Inventory.Gold.Get()
	result := state.GetCombatItem(references.Right)
	require.True(t, result.Handled)
	assert.Equal(t, uint16(3), result.Acquired)
	assert.Equal(t, goldBefore+3, state.PartyState.Inventory.Gold.Get())
	assert.Equal(t, []CombatObject{body}, state.CombatScene.objects)
	assert.True(t, character.IsDead())
	assert.False(t, state.CombatScene.isCombatPCActive(deadIdentity))
	assert.False(t, state.GetCombatItem(references.Right).Handled)
	assert.Equal(t, goldBefore+3, state.PartyState.Inventory.Gold.Get())
}

func TestCombatCorpseOnlyPileIsStableAndNotCollectible(t *testing.T) {
	state, _, _, _, position := setupPlayerRangedAttackTest(t, references.MorningStar, references.MonsterOrc, 1)
	state.CombatScene.AttackTargeting = nil
	require.True(t, state.CombatScene.AddObject(NewCombatCorpse(position)))
	for _, elapsed := range []int64{0, 999, 1000, 2000} {
		state.ElapsedMs = elapsed
		sprite, found := state.CombatPileSprite(position)
		require.True(t, found)
		assert.Equal(t, indexes.SpriteIndex(indexes.Splat), sprite)
	}
	assert.False(t, state.GetCombatItem(references.Right).Handled)
	assert.Equal(t, 1, state.CombatScene.ObjectCount())
}
