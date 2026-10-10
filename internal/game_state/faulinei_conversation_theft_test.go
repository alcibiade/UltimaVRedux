package game_state

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/deterministic_random"
	"github.com/bradhannah/UltimaVRedux/internal/party_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
)

func TestFaulineiConversationTheftRequiresAutomaticPresence(t *testing.T) {
	for _, test := range []struct {
		name      string
		configure func(*references.ShadowlordState)
	}{
		{name: "no Shadowlord", configure: func(state *references.ShadowlordState) {
			state.AutomaticPresence = references.NoShadowlord
		}},
		{name: "Yell-summoned Faulinei", configure: func(state *references.ShadowlordState) {
			state.AutomaticPresence = references.NoShadowlord
			state.Summoned = true
			state.LastCalled = references.Faulinei
		}},
		{name: "automatic Astaroth", configure: func(state *references.ShadowlordState) {
			state.AutomaticPresence = references.Astaroth
		}},
		{name: "automatic Nosfentor", configure: func(state *references.ShadowlordState) {
			state.AutomaticPresence = references.Nosfentor
		}},
	} {
		t.Run(test.name, func(t *testing.T) {
			state := newFaulineiTheftTestState(11)
			test.configure(state.ShadowlordState)
			state.PartyState.Inventory.Provisions.Keys.Set(1)
			callsBefore := state.GetRNG().Calls()

			result := state.ApplyFaulineiConversationTheft()

			assert.False(t, result.Occurred())
			assert.Equal(t, uint16(1), state.PartyState.Inventory.Provisions.Keys.Get())
			assert.Equal(t, callsBefore, state.GetRNG().Calls())
		})
	}
}

func TestFaulineiConversationTheftProvisionSelectionConsumesOrderedRetries(t *testing.T) {
	const seed = uint64(1)
	expected := deterministic_random.New(seed)
	expectedCalls := 0
	for {
		expectedCalls++
		if expected.Intn(3) == 2 {
			break
		}
	}
	require.Greater(t, expectedCalls, 1, "seed must exercise empty-category retries")

	state := newFaulineiTheftTestState(seed)
	state.PartyState.Inventory.Provisions.Torches.Set(1)
	state.PartyState.Inventory.Equipment.Set(references.Ankh, 1)

	result := state.ApplyFaulineiConversationTheft()

	assert.Equal(t, FaulineiConversationTheftResult{Kind: FaulineiConversationTheftTorch, Quantity: 1}, result)
	assert.Zero(t, state.PartyState.Inventory.Provisions.Torches.Get())
	assert.Equal(t, uint16(1), state.PartyState.Inventory.Equipment.Get(references.Ankh))
	assert.Equal(t, uint64(expectedCalls), state.GetRNG().Calls())
	assert.Equal(t, expected.Intn(1000), state.RandomIntInRange(0, 999))
}

func TestFaulineiConversationTheftUsesInventoryPriorityAndHighestIndex(t *testing.T) {
	tests := []struct {
		name      string
		configure func(*party_state.Inventory)
		want      FaulineiConversationTheftResult
		remaining func(*party_state.Inventory) uint16
	}{
		{
			name: "equipment", configure: func(inventory *party_state.Inventory) {
				inventory.Equipment.Set(references.LeatherHelm, 1)
				inventory.Equipment.Set(references.Ankh, 1)
				inventory.Potions.Set(references.White, 1)
			},
			want: FaulineiConversationTheftResult{Kind: FaulineiConversationTheftEquipment, ItemID: int(references.Ankh), Quantity: 1},
			remaining: func(inventory *party_state.Inventory) uint16 {
				return inventory.Equipment.Get(references.Ankh)
			},
		},
		{
			name: "potion", configure: func(inventory *party_state.Inventory) {
				inventory.Potions.Set(references.Blue, 1)
				inventory.Potions.Set(references.White, 1)
				inventory.Scrolls.Set(references.ScrollAnTym, 1)
			},
			want: FaulineiConversationTheftResult{Kind: FaulineiConversationTheftPotion, ItemID: int(references.White), Quantity: 1},
			remaining: func(inventory *party_state.Inventory) uint16 {
				return inventory.Potions.Get(references.White)
			},
		},
		{
			name: "scroll", configure: func(inventory *party_state.Inventory) {
				inventory.Scrolls.Set(references.ScrollVasLor, 1)
				inventory.Scrolls.Set(references.ScrollAnTym, 1)
			},
			want: FaulineiConversationTheftResult{Kind: FaulineiConversationTheftScroll, ItemID: int(references.ScrollAnTym), Quantity: 1},
			remaining: func(inventory *party_state.Inventory) uint16 {
				return inventory.Scrolls.Get(references.ScrollAnTym)
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			state := newFaulineiTheftTestState(19)
			test.configure(&state.PartyState.Inventory)

			result := state.ApplyFaulineiConversationTheft()

			assert.Equal(t, test.want, result)
			assert.Zero(t, test.remaining(&state.PartyState.Inventory))
			assert.Zero(t, state.GetRNG().Calls())
		})
	}
}

func TestFaulineiConversationTheftGoldFallbackClampsAndAlwaysConsumesOneRoll(t *testing.T) {
	const seed = uint64(27)
	expected := deterministic_random.New(seed)
	wantRoll := uint16(expected.Intn(15) + 1)

	state := newFaulineiTheftTestState(seed)
	state.PartyState.Inventory.Gold.Set(2)
	result := state.ApplyFaulineiConversationTheft()

	assert.Equal(t, FaulineiConversationTheftGold, result.Kind)
	assert.Equal(t, min(wantRoll, uint16(2)), result.Quantity)
	assert.Equal(t, uint16(2)-result.Quantity, state.PartyState.Inventory.Gold.Get())
	assert.Equal(t, uint64(1), state.GetRNG().Calls())

	empty := newFaulineiTheftTestState(seed)
	emptyResult := empty.ApplyFaulineiConversationTheft()
	assert.True(t, emptyResult.Occurred())
	assert.Equal(t, FaulineiConversationTheftResult{Kind: FaulineiConversationTheftGold}, emptyResult)
	assert.Equal(t, uint64(1), empty.GetRNG().Calls())
}

func newFaulineiTheftTestState(seed uint64) *GameState {
	state := &GameState{
		PartyState:      party_state.PartyState{Inventory: *party_state.NewInventory()},
		ShadowlordState: references.NewShadowlordState(),
	}
	state.ShadowlordState.AutomaticPresence = references.Faulinei
	state.SetRandomSeed(seed)

	return state
}
