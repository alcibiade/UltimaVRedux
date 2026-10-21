package game_state

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/references"
)

func (g *GameState) addCombatLooseItem(position references.Position, item references.Item, quantity uint16) bool {
	itemReference := g.GameReferences.InventoryItemReferences.GetReferenceByItem(item)

	return g.CombatScene.AddObject(NewCombatLooseItem(position, item, quantity, itemReference.ItemSprite))
}

func TestActionGetCombatMapCollectsStackedLooseItems(t *testing.T) {
	gs, _ := NewIntegrationTestBuilder(t).
		WithLocation(references.Britain).
		WithSystemCallbacks().
		Build()

	require.NoError(t, gs.TransitionToCombatMap(0, references.BritanniaCombatTerritory, CombatEntryNorth))
	activePosition := gs.CombatScene.GetPartyMemberPosition(gs.CombatScene.GetActiveCombatantIndex())
	targetPosition := references.Down.GetNewPositionInDirection(&activePosition)
	require.NotNil(t, targetPosition)
	gs.addCombatLooseItem(*targetPosition, references.Gold, 3)
	gs.addCombatLooseItem(*targetPosition, references.Blue, 1)
	goldBefore := gs.PartyState.Inventory.Gold.Get()
	potionBefore := gs.PartyState.Inventory.Potions.Get(references.Blue)

	assert.True(t, gs.ActionGetCombatMap(references.Down))
	assert.Equal(t, goldBefore+3, gs.PartyState.Inventory.Gold.Get())
	assert.Equal(t, 1, gs.CombatScene.ObjectCount())

	assert.True(t, gs.ActionGetCombatMap(references.Down))
	assert.Equal(t, potionBefore+1, gs.PartyState.Inventory.Potions.Get(references.Blue))
	assert.Zero(t, gs.CombatScene.ObjectCount())
}

func TestActionGetCombatMapRequiresOpeningChest(t *testing.T) {
	gs, mock := NewIntegrationTestBuilder(t).
		WithLocation(references.Britain).
		WithSystemCallbacks().
		Build()

	require.NoError(t, gs.TransitionToCombatMap(0, references.BritanniaCombatTerritory, CombatEntryNorth))
	activePosition := gs.CombatScene.GetPartyMemberPosition(gs.CombatScene.GetActiveCombatantIndex())
	targetPosition := references.Down.GetNewPositionInDirection(&activePosition)
	require.NotNil(t, targetPosition)
	gs.CombatScene.AddObject(NewCombatChest(*targetPosition, 10, false))
	mock.Reset()

	assert.True(t, gs.ActionGetCombatMap(references.Down))
	assert.Contains(t, mock.Messages, "Open it first!")
	assert.Equal(t, 1, gs.CombatScene.ObjectCount())
}

func TestActionGetCombatMapUsesOriginalStackOrder(t *testing.T) {
	gs, _ := NewIntegrationTestBuilder(t).
		WithLocation(references.Britain).
		WithSystemCallbacks().
		Build()

	require.NoError(t, gs.TransitionToCombatMap(0, references.BritanniaCombatTerritory, CombatEntryNorth))
	activePosition := gs.CombatScene.GetPartyMemberPosition(gs.CombatScene.GetActiveCombatantIndex())
	targetPosition := references.Down.GetNewPositionInDirection(&activePosition)
	require.NotNil(t, targetPosition)
	gs.addCombatLooseItem(*targetPosition, references.Gold, 3)
	gs.CombatScene.AddObject(NewCombatChest(*targetPosition, 10, false))
	goldBefore := gs.PartyState.Inventory.Gold.Get()

	assert.True(t, gs.ActionGetCombatMap(references.Down))
	assert.Equal(t, goldBefore+3, gs.PartyState.Inventory.Gold.Get())
	assert.Equal(t, 1, gs.CombatScene.ObjectCount(), "later nested chest must remain")

	assert.True(t, gs.ActionGetCombatMap(references.Down))
	assert.Equal(t, 1, gs.CombatScene.ObjectCount(), "unopened nested chest must not be collected")
}

func TestActionGetCombatMapCollectsUniqueFixedRoomQuestItems(t *testing.T) {
	tests := []struct {
		name       string
		descriptor byte
		item       references.QuestItem
	}{
		{name: "shard", descriptor: 0xb4, item: references.ShardOfFalsehood},
		{name: "crown", descriptor: 0xb5, item: references.Crown},
		{name: "sceptre", descriptor: 0xb6, item: references.Sceptre},
		{name: "amulet", descriptor: 0xb7, item: references.Amulet},
		{name: "sandalwood box", descriptor: 0x0e, item: references.WoodenBox},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			gs, mock := NewIntegrationTestBuilder(t).
				WithLocation(references.Britain).
				WithSystemCallbacks().
				Build()
			require.NoError(t, gs.TransitionToCombatMap(0, references.BritanniaCombatTerritory, CombatEntryNorth))
			activePosition := gs.CombatScene.GetPartyMemberPosition(gs.CombatScene.GetActiveCombatantIndex())
			targetPosition := references.Down.GetNewPositionInDirection(&activePosition)
			require.NotNil(t, targetPosition)
			object, err := gs.fixedDungeonLootObject(test.descriptor, 0, *targetPosition)
			require.NoError(t, err)
			gs.CombatScene.AddObject(object)
			mock.Reset()

			require.True(t, gs.ActionGetCombatMap(references.Down))

			assert.Equal(t, uint16(1), gs.PartyState.Inventory.QuestItems.Get(test.item))
			assert.Zero(t, gs.CombatScene.ObjectCount())
			assert.Contains(t, mock.Messages, gs.GameReferences.InventoryItemReferences.GetQuestItemPickupMessage(test.item))
			if test.item == references.WoodenBox {
				assert.True(t, gs.lordBritishPassageOpen)
				assert.True(t, gs.PartyState.IsNPCDead(references.LordBritishsCastle, lordBritishSandalwoodBoxNPC))
			}
		})
	}
}

func TestActionGetCombatMapDiscardsRespawnedUniqueItemAlreadyOwned(t *testing.T) {
	gs, mock := NewIntegrationTestBuilder(t).
		WithLocation(references.Britain).
		WithSystemCallbacks().
		Build()
	require.NoError(t, gs.TransitionToCombatMap(0, references.BritanniaCombatTerritory, CombatEntryNorth))
	activePosition := gs.CombatScene.GetPartyMemberPosition(gs.CombatScene.GetActiveCombatantIndex())
	targetPosition := references.Down.GetNewPositionInDirection(&activePosition)
	require.NotNil(t, targetPosition)
	object, err := gs.fixedDungeonLootObject(0xb5, 0, *targetPosition)
	require.NoError(t, err)
	gs.CombatScene.AddObject(object)
	gs.PartyState.Inventory.QuestItems.SetHasOne(references.Crown)
	mock.Reset()

	require.True(t, gs.ActionGetCombatMap(references.Down))

	assert.Equal(t, uint16(1), gs.PartyState.Inventory.QuestItems.Get(references.Crown))
	assert.Zero(t, gs.CombatScene.ObjectCount())
	assert.Contains(t, mock.Messages, "Already have that!")
}
